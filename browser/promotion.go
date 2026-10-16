package browser

import (
	"encoding/json"
	"errors"
	"fmt"
)

type controlLocator struct {
	ID   string `json:"id,omitempty"`
	Path []int  `json:"path"`
	Tag  string `json:"tag"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type controlTransfer struct {
	Locator controlLocator `json:"locator"`
	Value   string         `json:"value"`
	Values  []string       `json:"values,omitempty"`
	Options []Option       `json:"options,omitempty"`
	Checked bool           `json:"checked"`
}

func (d *Document) locator(node *Node) controlLocator {
	out := controlLocator{Tag: node.Tag, Name: node.prop("name"), Type: node.controlType()}
	if id := node.prop("id"); id != "" {
		count := 0
		controlWalk(d.Root, func(n *Node) {
			if n.prop("id") == id {
				count++
			}
		})
		if count == 1 {
			out.ID = id
		}
	}
	for n := node; n.Parent != nil; n = n.Parent {
		index := 0
		for _, sibling := range n.Parent.Children {
			if sibling == n {
				break
			}
			if sibling.Tag != "#text" {
				index++
			}
		}
		out.Path = append(out.Path, index)
	}
	for left, right := 0, len(out.Path)-1; left < right; left, right = left+1, right-1 {
		out.Path[left], out.Path[right] = out.Path[right], out.Path[left]
	}
	return out
}

// promoteForm reopens the observed page in Chromium and transfers editable
// controls, while retaining fresh hidden tokens from the new response. Every
// locator is checked before any values are applied; it never submits a form.
func (b *Browser) promoteForm(formRef, buttonRef string) (string, string, error) {
	if b.Document == nil || b.Current == nil {
		return "", "", errors.New("no page to promote")
	}
	var formNode *Node
	controlWalk(b.Document.Root, func(n *Node) {
		if n.Tag == "form" && n.ref == formRef {
			formNode = n
		}
	})
	if formNode == nil {
		return "", "", errors.New("form to promote not found")
	}
	payload := struct {
		Form     controlLocator    `json:"form"`
		Button   *controlLocator   `json:"button,omitempty"`
		Controls []controlTransfer `json:"controls"`
	}{Form: b.Document.locator(formNode)}
	if buttonRef != "" {
		button := b.Document.buttons[buttonRef]
		if button == nil {
			return "", "", errors.New("submitter to promote not found")
		}
		locator := b.Document.locator(button)
		payload.Button = &locator
	}
	// Restore this form's controls only. Standalone editors and other forms
	// may rerender independently and must not prevent a valid submission.
	for _, form := range b.Current.Forms {
		if form.Ref != formRef {
			continue
		}
		for _, field := range form.Fields {
			node := b.Document.fields[field.Ref]
			if node == nil || field.Disabled || field.ReadOnly || field.Type == "hidden" || field.Type == "file" {
				continue
			}
			payload.Controls = append(payload.Controls, controlTransfer{Locator: b.Document.locator(node), Value: field.Value, Values: field.Values, Options: field.Options, Checked: field.Checked})
		}
	}
	data, _ := json.Marshal(payload)
	if err := b.openLive(b.Current.URL, false); err != nil {
		return "", "", fmt.Errorf("promote form to Chromium: %w", err)
	}
	ctx, cancel := b.liveContext()
	defer cancel()
	raw, err := b.live.evaluateAction(ctx, `(() => {
		`+liveDOMHelpers+`
		const state = `+string(data)+`;
		const locate = locator => {
			if (locator.id && document.querySelectorAll('#'+CSS.escape(locator.id)).length !== 1) throw new Error('control identity changed during Chromium promotion');
			let element = locator.id ? document.getElementById(locator.id) : document;
			if (!locator.id) for (const index of locator.path) element = element && element.children[index];
			const type = element instanceof HTMLSelectElement ? 'select' : element && (element.type || element.tagName.toLowerCase());
			if (!element || !element.isConnected || element.tagName.toLowerCase() !== locator.tag || (element.name || '') !== locator.name || type !== locator.type)
				throw new Error('page controls changed during Chromium promotion; take a new snapshot');
			return element;
		};
		const form = locate(state.form), button = state.button ? locate(state.button) : null;
		if (!(form instanceof HTMLFormElement) || (button && button.form !== form)) throw new Error('form association changed during Chromium promotion');
		const controls = state.controls.map(control => ({control, element: locate(control.locator)}));
		for (const {control, element} of controls) {
			if (element.form !== form) throw new Error('form association changed during Chromium promotion');
			if (liveDOM.disabled(element) || liveDOM.readonly(element)) throw new Error('control became disabled during Chromium promotion');
			if (element instanceof HTMLSelectElement) {
				if (element.options.length !== (control.options || []).length) throw new Error('select options changed during Chromium promotion');
				for (let i=0; i<element.options.length; i++) if (element.options[i].value !== control.options[i].value || element.options[i].matches(':disabled') !== !!control.options[i].disabled)
					throw new Error('select options changed during Chromium promotion');
			}
		}
		for (const {control, element} of controls) {
			if (element instanceof HTMLSelectElement) {
				for (let i=0; i<element.options.length; i++) element.options[i].selected = !!control.options[i].selected;
			} else if (element.type === 'checkbox' || element.type === 'radio') element.checked = control.checked;
			else {
				const prototype = element instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
				Object.getOwnPropertyDescriptor(prototype, 'value').set.call(element, control.value);
			}
		}
		for (const {element} of controls) {
			element.dispatchEvent(new Event('input', {bubbles:true}));
			element.dispatchEvent(new Event('change', {bubbles:true}));
		}
		if (!form.isConnected || (button && (!button.isConnected || button.form !== form || locate(state.button) !== button))) throw new Error('form association changed while restoring values; take a new snapshot');
		for (const {control, element} of controls) {
			if (!element.isConnected || element.form !== form || locate(control.locator) !== element || liveDOM.disabled(element) || liveDOM.readonly(element)) throw new Error('controls changed while restoring values; take a new snapshot');
		}
		return {form:form.getAttribute('data-aigobrowser'), button:button && button.getAttribute('data-aigobrowser')};
	})()`)
	if err != nil {
		return "", "", err
	}
	var markers struct {
		Form   string `json:"form"`
		Button string `json:"button"`
	}
	if err := json.Unmarshal(raw, &markers); err != nil {
		return "", "", err
	}
	if err := b.refreshLiveContext(ctx); err != nil {
		return "", "", err
	}
	refs := b.liveRefs()
	if refs[markers.Form] == "" || (buttonRef != "" && refs[markers.Button] == "") {
		return "", "", errors.New("form changed while restoring controls; take a new snapshot")
	}
	return refs[markers.Form], refs[markers.Button], nil
}
