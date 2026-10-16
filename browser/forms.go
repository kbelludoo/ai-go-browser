package browser

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func (b *Browser) fieldByRef(ref string) *Field {
	if b.Current == nil {
		return nil
	}
	for i := range b.Current.Inputs {
		if b.Current.Inputs[i].Ref == ref {
			return &b.Current.Inputs[i]
		}
	}
	for i := range b.Current.Forms {
		for j := range b.Current.Forms[i].Fields {
			if b.Current.Forms[i].Fields[j].Ref == ref {
				return &b.Current.Forms[i].Fields[j]
			}
		}
	}
	return nil
}

func (b *Browser) refreshStaticPage() {
	if b.Document == nil || b.Current == nil {
		return
	}
	p := b.Document.PageWithMarkdown(b.MaxText, !b.CompactSnapshots)
	p.Stack = b.Current.Stack
	for key, value := range b.Current.Metadata {
		if _, exists := p.Metadata[key]; !exists {
			p.Metadata[key] = value
		}
	}
	b.Current = &p
}

func (b *Browser) fillControl(ref, value string) error {
	f := b.fieldByRef(ref)
	if f == nil {
		return fmt.Errorf("field %q not found", ref)
	}
	if f.Disabled || f.ReadOnly || f.Type == "hidden" || f.Type == "file" || f.Type == "submit" {
		return errors.New("field cannot be edited")
	}
	if f.Type == "checkbox" || f.Type == "radio" {
		return errors.New("use check for checkbox or radio fields")
	}
	if f.Type == "select" {
		valid := false
		for _, option := range f.Options {
			if option.Value == value {
				valid = !option.Disabled
				break
			}
		}
		if !valid {
			return errors.New("select value is missing or disabled")
		}
	}
	f.Value = value
	if b.Document != nil {
		if n := b.Document.fields[ref]; n != nil {
			n.setValue(value)
			if id := n.prop("id"); id != "" && b.jsSession != nil {
				_, err := b.jsSession.exchange(map[string]string{"cmd": "setvalue", "id": id, "value": value}, b.JSTimeout)
				if err != nil {
					b.closeJSSession()
					return fmt.Errorf("JS input sync failed: %w", err)
				}
			}
		}
	}
	b.refreshStaticPage()
	return nil
}

func (b *Browser) checkControl(ref string, checked bool) error {
	f := b.fieldByRef(ref)
	if f == nil {
		return fmt.Errorf("field %q not found", ref)
	}
	if f.Disabled || (f.Type != "checkbox" && f.Type != "radio") {
		return errors.New("check requires an enabled checkbox or radio field")
	}
	set := func(field *Field, value bool) {
		field.Checked = value
		if b.Document != nil {
			if n := b.Document.fields[field.Ref]; n != nil {
				if n.Attr == nil {
					n.Attr = map[string]string{}
				}
				if value {
					n.Attr["checked"] = ""
				} else {
					delete(n.Attr, "checked")
				}
			}
		}
	}
	if f.Type == "radio" && checked && f.Name != "" {
		group := b.Current.Inputs
		for _, form := range b.Current.Forms {
			for _, field := range form.Fields {
				if field.Ref == ref {
					group = form.Fields
				}
			}
		}
		for _, other := range group {
			if other.Type == "radio" && other.Name == f.Name && other.Ref != ref {
				if field := b.fieldByRef(other.Ref); field != nil {
					set(field, false)
				}
			}
		}
	}
	set(f, checked)
	b.refreshStaticPage()
	return nil
}

// formParams follows successful-control rules, including repeated select values
// and a named submitter. Disabled and unchecked controls do not contribute.
func formParams(form Form, submitter *Button) url.Values {
	params := url.Values{}
	add := func(name, value string) {
		params.Add(normalizeFormNewlines(name), normalizeFormNewlines(value))
	}
	for _, f := range form.Fields {
		if f.Disabled || f.Name == "" || f.Type == "file" || f.Type == "button" || f.Type == "submit" || f.Type == "reset" || f.Type == "image" {
			continue
		}
		if (f.Type == "checkbox" || f.Type == "radio") && !f.Checked {
			continue
		}
		if f.Type == "select" {
			for _, value := range f.Values {
				add(f.Name, value)
			}
			continue
		}
		add(f.Name, f.Value)
	}
	if submitter != nil && !submitter.Disabled && submitter.Type == "submit" && submitter.Name != "" {
		add(submitter.Name, submitter.Value)
	}
	return params
}

// URL-encoded HTML submissions normalize lone CR/LF in names and values.
func normalizeFormNewlines(value string) string {
	if !strings.ContainsAny(value, "\r\n") {
		return value
	}
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.ReplaceAll(value, "\n", "\r\n")
}

func (b *Browser) submitForm(ref, buttonRef string, confirm bool) error {
	explicitSubmit := buttonRef == ""
	if b.Current == nil {
		return errors.New("no page open")
	}
	for _, form := range b.Current.Forms {
		if form.Ref != ref {
			continue
		}
		if buttonRef == "" {
			buttonRef = form.SubmitRef
		}
		var submitter *Button
		for _, button := range b.Current.Buttons {
			if button.Ref == buttonRef {
				if button.Disabled || button.FormRef != ref || button.Type != "submit" {
					return errors.New("submitter is disabled or does not belong to this form")
				}
				copyButton := button
				submitter = &copyButton
			}
		}
		if buttonRef != "" && submitter == nil {
			return errors.New("submitter not found")
		}
		method, target, enctype := form.Method, form.Action, form.Enctype
		skipValidation := form.NoValidate
		if submitter != nil && b.Document != nil {
			n := b.Document.buttons[submitter.Ref]
			if n != nil {
				if action := n.prop("formaction"); action != "" {
					base, _ := url.Parse(b.Current.URL)
					target = resolve(base, action)
				}
				if m := strings.ToUpper(n.prop("formmethod")); m != "" {
					if m != "GET" && m != "POST" && m != "DIALOG" {
						return errors.New("unsupported form method")
					}
					method = m
				}
				if encoding := strings.ToLower(n.prop("formenctype")); encoding != "" {
					enctype = encoding
				}
				skipValidation = skipValidation || has(n.Attr, "formnovalidate")
			}
		}
		if method == "POST" && (!b.AllowPost || !confirm) {
			return errors.New("POST requires -allow-post and confirm:true")
		}
		needsLive := method == "DIALOG" || (method == "POST" && enctype != "" && enctype != "application/x-www-form-urlencoded")
		for _, field := range form.Fields {
			if field.Disabled || field.ReadOnly || field.Type == "hidden" {
				continue
			}
			if field.Type == "file" || (!skipValidation && (field.Required || field.Pattern != "" || field.Min != "" || field.Max != "" || field.Step != "" || field.Type == "email" || field.Type == "url" || field.Type == "number" || field.Type == "date" || field.Type == "time" || field.Type == "datetime-local" || field.Type == "month" || field.Type == "week")) {
				needsLive = true
			}
		}
		canPromote := b.Engine != "http" && b.AutoRenderJS
		if needsLive && !canPromote {
			return errors.New("form encoding or validation requires Chromium; use render or enable auto-js")
		}
		// Native handlers may be installed by inline or external scripts. Promote
		// only when an interaction needs them; static reading stays in Go.
		hasScripts := false
		if canPromote {
			controlWalk(b.Document.Root, func(node *Node) {
				if node.Tag == "script" && (node.prop("src") != "" || len(node.Children) > 0) {
					hasScripts = true
				}
			})
		}
		if needsLive || hasScripts {
			formRef, liveButton, err := b.promoteForm(ref, buttonRef)
			if err != nil {
				return err
			}
			command := Command{Cmd: "submit", Ref: formRef, Confirm: confirm}
			if liveButton != "" {
				// A submit command reports invalid fields even when promotion uses
				// a native button click. A press command lets the browser emit its
				// normal click/invalid events without a duplicate validity check.
				if explicitSubmit {
					button := b.Document.buttons[liveButton]
					if button == nil || button.prop(liveMarker) == "" {
						return errors.New("promoted submitter changed; take a new snapshot")
					}
					ctx, cancel := b.liveContext()
					err = b.validateLiveSubmission(ctx, "null", liveElementExpression(button.prop(liveMarker)), confirm, true)
					cancel()
					if err != nil {
						return err
					}
				}
				command.Cmd, command.Ref = "press", liveButton
			}
			_, err = b.executeLive(command)
			return err
		}
		return b.request(method, target, formParams(form, submitter), true)
	}
	return fmt.Errorf("form %q not found", ref)
}

func (b *Browser) pressControl(c Command) error {
	if b.Current == nil {
		return errors.New("no page open")
	}
	for _, button := range b.Current.Buttons {
		if button.Ref != c.Ref {
			continue
		}
		if button.Disabled {
			return errors.New("button is disabled")
		}
		if b.jsSession != nil {
			return b.dispatchButton(c.Ref)
		}
		if button.Type == "submit" && button.FormRef != "" {
			return b.submitForm(button.FormRef, button.Ref, c.Confirm)
		}
		if button.Type == "image" {
			return errors.New("image submitter requires coordinates; use Chromium")
		}
		return b.dispatchButton(c.Ref)
	}
	return fmt.Errorf("button %q not found", c.Ref)
}
