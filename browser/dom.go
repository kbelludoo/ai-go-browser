package browser

// This bounded observation model uses x/net/html for HTML parsing and exposes
// a subset of DOM operations for agents. It has no scripting event loop or CSS
// layout engine; Chromium supplies those when a live session is active.

import (
	"bytes"
	"errors"
	"html"
	"net/url"
	"strconv"
	"strings"

	xhtml "golang.org/x/net/html"
)

const maxDOMNodes = 50000
const maxDOMDepth = 128
const maxQueryResults = 32

type Node struct {
	Tag      string
	Text     string
	Attr     map[string]string
	Parent   *Node
	Children []*Node
	valueSet bool
	ref      string
}
type Document struct {
	Root    *Node
	URL     string
	Status  int
	Bytes   int
	links   map[string]*Node
	fields  map[string]*Node
	buttons map[string]*Node
}
type Element struct {
	Ref      string   `json:"ref,omitempty"`
	Tag      string   `json:"tag"`
	ID       string   `json:"id,omitempty"`
	Role     string   `json:"role,omitempty"`
	Label    string   `json:"label,omitempty"`
	Text     string   `json:"text,omitempty"`
	Href     string   `json:"href,omitempty"`
	Name     string   `json:"name,omitempty"`
	Value    string   `json:"value,omitempty"`
	Type     string   `json:"type,omitempty"`
	Disabled bool     `json:"disabled,omitempty"`
	ReadOnly bool     `json:"readonly,omitempty"`
	Checked  *bool    `json:"checked,omitempty"`
	Multiple bool     `json:"multiple,omitempty"`
	Values   []string `json:"values,omitempty"`
	Options  []Option `json:"options,omitempty"`
	Required bool     `json:"required,omitempty"`
	Pattern  string   `json:"pattern,omitempty"`
	Min      string   `json:"min,omitempty"`
	Max      string   `json:"max,omitempty"`
	Step     string   `json:"step,omitempty"`
}

func (n *Node) add(child *Node) {
	child.Parent = n
	n.Children = append(n.Children, child)
}
func (n *Node) prop(name string) string { return n.Attr[name] }

func (n *Node) hasRole(role string) bool {
	for _, value := range strings.Fields(n.prop("role")) {
		if strings.EqualFold(value, role) {
			return true
		}
	}
	return false
}

func (n *Node) visible() bool {
	for current := n; current != nil; current = current.Parent {
		if current.hidden() {
			return false
		}
	}
	return n != nil
}

func (n *Node) contentEditable() bool {
	for current := n; current != nil; current = current.Parent {
		if value, present := current.Attr["contenteditable"]; present {
			switch strings.ToLower(value) {
			case "", "true", "plaintext-only":
				return true
			case "false":
				return false
			}
		}
	}
	return false
}

func (n *Node) editableControl() bool {
	switch n.Tag {
	case "#text", "#document", "input", "textarea", "select", "button":
		return false
	}
	if n.hasRole("textbox") {
		return true
	}
	return n.contentEditable() && (n.Parent == nil || !n.Parent.contentEditable())
}

func (n *Node) genericButton() bool {
	switch n.Tag {
	case "#text", "#document", "input", "textarea", "select", "button":
		return false
	}
	return n.hasRole("button") || (n.Tag == "a" && !has(n.Attr, "href"))
}

func (n *Node) hidden() bool {
	if n.Tag == "script" || n.Tag == "style" || n.Tag == "noscript" || n.Tag == "template" || n.Tag == "svg" || n.Tag == "head" || n.Tag == "title" {
		return true
	}
	if _, ok := n.Attr["hidden"]; ok {
		return true
	}
	if strings.EqualFold(n.prop("aria-hidden"), "true") {
		return true
	}
	style := strings.ToLower(strings.ReplaceAll(n.prop("style"), " ", ""))
	return strings.Contains(style, "display:none") || strings.Contains(style, "visibility:hidden")
}
func (n *Node) innerText(limit int) string {
	if limit <= 0 {
		return ""
	}
	text := pageText{limit: limit}
	var visit func(*Node)
	visit = func(p *Node) {
		if p == nil || text.runes >= limit || p.hidden() {
			return
		}
		if p.Tag == "#text" {
			text.add(p.Text)
			return
		}
		if p.Tag == "img" {
			text.add(p.prop("alt"))
		}
		for _, child := range p.Children {
			visit(child)
			if text.runes >= limit {
				return
			}
		}
	}
	visit(n)
	return text.text.String()
}
func autoCloses(newTag, top string) bool {
	if newTag == "li" && top == "li" {
		return true
	}
	if (newTag == "dt" || newTag == "dd") && (top == "dt" || top == "dd") {
		return true
	}
	if newTag == "option" && top == "option" {
		return true
	}
	if newTag == "tr" && top == "tr" {
		return true
	}
	if (newTag == "td" || newTag == "th") && (top == "td" || top == "th") {
		return true
	}
	if top == "p" {
		switch newTag {
		case "address", "article", "aside", "blockquote", "div", "dl", "fieldset", "footer", "form", "h1", "h2", "h3", "h4", "h5", "h6", "header", "hr", "main", "nav", "ol", "p", "pre", "section", "table", "ul":
			return true
		}
	}
	return false
}
func ParseDocument(raw []byte, baseURL string, status, bytesRead int) (*Document, error) {
	d := &Document{Root: &Node{Tag: "#document"}, URL: baseURL, Status: status, Bytes: bytesRead}

	doc, err := xhtml.Parse(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	nodeCount := 0
	var buildTree func(*xhtml.Node, *Node, int) error
	buildTree = func(src *xhtml.Node, dstParent *Node, depth int) error {
		if nodeCount > maxDOMNodes {
			return errors.New("HTML node limit reached")
		}
		if depth > maxDOMDepth {
			return errors.New("HTML nesting limit reached")
		}

		var currentDst *Node
		switch src.Type {
		case xhtml.ElementNode:
			attrMap := make(map[string]string, len(src.Attr))
			for _, a := range src.Attr {
				attrMap[strings.ToLower(a.Key)] = a.Val
			}
			currentDst = &Node{
				Tag:  strings.ToLower(src.Data),
				Attr: attrMap,
			}
			dstParent.add(currentDst)
			nodeCount++

		case xhtml.TextNode:
			text := src.Data
			if strings.TrimSpace(text) != "" {
				currentDst = &Node{
					Tag:  "#text",
					Text: text,
				}
				dstParent.add(currentDst)
				nodeCount++
			}
			return nil

		case xhtml.DocumentNode:
			currentDst = dstParent

		case xhtml.CommentNode, xhtml.DoctypeNode:
			return nil
		}

		nextDepth := depth
		if currentDst != dstParent {
			nextDepth++
		}

		for c := src.FirstChild; c != nil; c = c.NextSibling {
			if err := buildTree(c, currentDst, nextDepth); err != nil {
				return err
			}
		}
		return nil
	}

	if err := buildTree(doc, d.Root, 0); err != nil {
		return nil, err
	}

	return d, nil
}
func domWalk(n *Node, hidden bool, fn func(*Node)) {
	if n == nil || hidden || n.hidden() {
		return
	}
	fn(n)
	for _, ch := range n.Children {
		domWalk(ch, false, fn)
	}
}

// controlWalk includes hidden controls, which still participate in submission,
// while excluding inert template contents and controls inside datalists.
func controlWalk(n *Node, fn func(*Node)) {
	if n == nil {
		return
	}
	switch n.Tag {
	case "template", "script", "style", "noscript", "head", "datalist":
		return
	}
	fn(n)
	for _, child := range n.Children {
		controlWalk(child, fn)
	}
}

func (n *Node) controlType() string {
	if n.editableControl() {
		return "contenteditable"
	}
	if n.genericButton() {
		return "button"
	}
	if n.Tag != "input" && n.Tag != "button" {
		return n.Tag
	}
	typ := strings.ToLower(n.prop("type"))
	if n.Tag == "button" {
		if typ == "button" || typ == "reset" {
			return typ
		}
		return "submit"
	}
	switch typ {
	case "hidden", "text", "search", "tel", "url", "email", "password", "date", "month", "week", "time", "datetime-local", "number", "range", "color", "checkbox", "radio", "file", "submit", "image", "reset", "button":
		return typ
	}
	return "text"
}

func isDescendant(n, ancestor *Node) bool {
	for ; n != nil; n = n.Parent {
		if n == ancestor {
			return true
		}
	}
	return false
}

func (n *Node) disabled() bool {
	if has(n.Attr, "disabled") {
		return true
	}
	for current := n; current != nil; current = current.Parent {
		if strings.EqualFold(strings.TrimSpace(current.prop("aria-disabled")), "true") {
			return true
		}
	}
	for parent := n.Parent; parent != nil; parent = parent.Parent {
		if parent.Tag != "fieldset" || !has(parent.Attr, "disabled") {
			continue
		}
		var legend *Node
		for _, child := range parent.Children {
			if child.Tag == "legend" {
				legend = child
				break
			}
		}
		if legend == nil || !isDescendant(n, legend) {
			return true
		}
	}
	return false
}

func (n *Node) readOnly() bool {
	// A textbox role describes semantics; it does not make an element editable.
	if n.editableControl() && !n.contentEditable() {
		return true
	}
	for current := n; current != nil; current = current.Parent {
		if strings.EqualFold(strings.TrimSpace(current.prop("aria-readonly")), "true") {
			return true
		}
	}
	if !has(n.Attr, "readonly") {
		return false
	}
	switch n.controlType() {
	case "contenteditable", "textarea", "text", "search", "tel", "url", "email", "password", "date", "month", "week", "time", "datetime-local", "number":
		return true
	}
	return false
}

func formOwner(n *Node, ids, markers map[string]*Node) *Node {
	// Live snapshots flatten the composed tree. Preserve the browser's actual
	// ownership instead of inferring it from that serialized ancestry or IDs.
	if marker, specified := n.Attr["data-aigobrowser-form"]; specified {
		return markers[marker]
	}
	if id, specified := n.Attr["form"]; specified {
		owner := ids[id]
		if owner != nil && owner.Tag == "form" {
			return owner
		}
		return nil
	}
	for parent := n.Parent; parent != nil; parent = parent.Parent {
		if parent.Tag == "form" {
			return parent
		}
	}
	return nil
}

func (n *Node) snapshotLabel() (string, bool) {
	label, present := n.Attr["data-aigobrowser-label"]
	if !present {
		return "", false
	}
	if label = norm(label); label != "" {
		return label, true
	}
	// An empty name from the native tree must not match unrelated labels after
	// shadow roots have been flattened into the observation.
	return n.prop("placeholder"), true
}

func controlLabel(n *Node, ids map[string]*Node, labels map[string][]*Node) string {
	if label, present := n.snapshotLabel(); present {
		return label
	}
	if label := norm(n.prop("aria-label")); label != "" {
		return label
	}
	text := pageText{limit: 512}
	for _, id := range strings.Fields(n.prop("aria-labelledby")) {
		if source := ids[id]; source != nil {
			text.add(source.innerText(512))
		}
	}
	if text.text.Len() > 0 {
		return text.text.String()
	}
	for _, label := range labels[n.prop("id")] {
		text.add(label.innerText(512))
	}
	for parent := n.Parent; parent != nil; parent = parent.Parent {
		if parent.Tag == "label" && !has(parent.Attr, "for") {
			text.add(parent.innerText(512))
			break
		}
	}
	if text.text.Len() > 0 {
		return text.text.String()
	}
	return n.prop("placeholder")
}

func (d *Document) collectForms(base *url.URL) ([]Form, []Button, []Field) {
	d.fields = map[string]*Node{}
	d.buttons = map[string]*Node{}
	forms := []Form{}
	buttons := []Button{}
	inputs := []Field{}
	ids := map[string]*Node{}
	formMarkers := map[string]*Node{}
	labels := map[string][]*Node{}
	formIndexes := map[*Node]int{}
	controlWalk(d.Root, func(n *Node) {
		if id := n.prop("id"); id != "" && ids[id] == nil {
			ids[id] = n
		}
		if n.Tag == "label" && n.prop("for") != "" {
			labels[n.prop("for")] = append(labels[n.prop("for")], n)
		}
		if n.Tag != "form" {
			return
		}
		if marker := n.prop("data-aigobrowser"); marker != "" {
			formMarkers[marker] = n
		}
		method := strings.ToUpper(n.prop("method"))
		if method != "POST" && method != "DIALOG" {
			method = "GET"
		}
		enctype := strings.ToLower(n.prop("enctype"))
		if enctype != "multipart/form-data" && enctype != "text/plain" {
			enctype = "application/x-www-form-urlencoded"
		}
		action := d.URL
		if n.prop("action") != "" {
			if resolved := resolve(base, n.prop("action")); resolved != "" {
				action = resolved
			}
		}
		formIndexes[n] = len(forms)
		ref := "f" + strconv.Itoa(len(forms)+1)
		n.ref = ref
		forms = append(forms, Form{Ref: ref, Method: method, Action: action, Fields: []Field{}, SubmitLabel: "Enviar", Enctype: enctype, NoValidate: has(n.Attr, "novalidate")})
	})
	type radioField struct {
		form, field int
		node        *Node
	}
	radios := map[string]radioField{}
	controlWalk(d.Root, func(n *Node) {
		editable, genericButton := n.editableControl(), n.genericButton()
		if n.Tag != "input" && n.Tag != "textarea" && n.Tag != "select" && n.Tag != "button" && !editable && !genericButton {
			return
		}
		if (editable || genericButton) && !n.visible() {
			return
		}
		owner := formOwner(n, ids, formMarkers)
		formIndex, associated := formIndexes[owner]
		if editable || genericButton {
			associated = false
		}
		typ := n.controlType()
		if n.Tag == "button" || typ == "button" || typ == "submit" || typ == "reset" || typ == "image" {
			label := controlLabel(n, ids, labels)
			if label == "" {
				label = n.innerText(256)
			}
			if label == "" {
				label = n.prop("value")
			}
			if label == "" {
				label = "(button)"
			}
			button := Button{Ref: "b" + strconv.Itoa(len(buttons)+1), ID: n.prop("id"), Label: label, Type: typ, Name: n.prop("name"), Value: n.prop("value"), Disabled: n.disabled()}
			var formPtr *Form
			if associated {
				button.FormRef = forms[formIndex].Ref
				if typ == "submit" && !button.Disabled && forms[formIndex].SubmitRef == "" {
					forms[formIndex].SubmitRef = button.Ref
					forms[formIndex].SubmitLabel = label
				}
				formPtr = &forms[formIndex]
			}
			meta := AnalyzeElementAction(button.Ref, n.Tag, label, n.Attr, formPtr)
			button.Action = &meta
			buttons = append(buttons, button)
			n.ref = button.Ref
			d.buttons[button.Ref] = n
			return
		}
		field := Field{Ref: "i" + strconv.Itoa(len(d.fields)+1), ID: n.prop("id"), Type: typ, Name: n.prop("name"), Label: controlLabel(n, ids, labels), Disabled: n.disabled(), ReadOnly: n.readOnly(), Required: has(n.Attr, "required"), Pattern: n.prop("pattern"), Min: n.prop("min"), Max: n.prop("max"), Step: n.prop("step")}
		if typ == "checkbox" || typ == "radio" {
			field.Checked = has(n.Attr, "checked")
		}
		if n.Tag == "select" {
			field.Multiple = has(n.Attr, "multiple")
			field.Options, field.Values, field.Value = n.selectState()
		} else {
			field.Value = n.value()
		}
		if typ == "radio" && field.Checked && field.Name != "" {
			group := "standalone\x00" + field.Name
			if associated {
				group = forms[formIndex].Ref + "\x00" + field.Name
			}
			if previous, exists := radios[group]; exists {
				if previous.form < 0 {
					inputs[previous.field].Checked = false
				} else {
					forms[previous.form].Fields[previous.field].Checked = false
				}
				delete(previous.node.Attr, "checked")
			}
			if associated {
				radios[group] = radioField{form: formIndex, field: len(forms[formIndex].Fields), node: n}
			} else {
				radios[group] = radioField{form: -1, field: len(inputs), node: n}
			}
		}
		if associated {
			forms[formIndex].Fields = append(forms[formIndex].Fields, field)
		} else {
			inputs = append(inputs, field)
		}
		d.fields[field.Ref] = n
		n.ref = field.Ref
	})
	return forms, buttons, inputs
}

func (d *Document) Page(textLimit int) Page {
	return d.PageWithMarkdown(textLimit, true)
}

// PageWithMarkdown lets callers avoid allocating a second text representation
// on every observation. It still extracts all controls and navigable links.
func (d *Document) PageWithMarkdown(textLimit int, includeMarkdown bool) Page {
	p := Page{URL: d.URL, Status: d.Status, Bytes: d.Bytes, Links: []Link{}, Headings: []Heading{}, Forms: []Form{}, Metadata: map[string]string{}}
	d.links = map[string]*Node{}
	d.fields = map[string]*Node{}
	d.buttons = map[string]*Node{}
	base, _ := url.Parse(d.URL)
	hasControls := false
	// Collect title, meta and base href.
	var findHead func(*Node)
	findHead = func(n *Node) {
		if n.Tag == "form" || n.Tag == "input" || n.Tag == "select" || n.Tag == "textarea" || n.Tag == "button" || n.editableControl() || n.genericButton() {
			hasControls = true
		}
		if n.Tag == "title" && p.Title == "" {
			p.Title = norm(n.rawText(4096))
		}
		if n.Tag == "meta" {
			name := strings.ToLower(n.prop("name"))
			prop := strings.ToLower(n.prop("property"))
			content := strings.TrimSpace(n.prop("content"))
			if content != "" {
				if name != "" {
					p.Metadata[name] = content
					if name == "description" && p.Description == "" {
						p.Description = content
					}
				}
				if prop != "" {
					p.Metadata[prop] = content
					if (prop == "og:description" || prop == "twitter:description") && p.Description == "" {
						p.Description = content
					}
				}
			}
		}
		if n.Tag == "base" && n.prop("href") != "" {
			if next := resolve(base, n.prop("href")); next != "" {
				base, _ = url.Parse(next)
			}
		}
		for _, child := range n.Children {
			findHead(child)
		}
	}
	findHead(d.Root)
	text := pageText{limit: textLimit}
	// A document tree has no layout; a bounded text-only traversal is best
	// suited to agent observations. CSS is not applied.
	domWalk(d.Root, false, func(n *Node) {
		if n.Tag == "#text" {
			text.add(n.Text)
			return
		}
		if n.Tag == "a" && has(n.Attr, "href") {
			if href := resolve(base, n.prop("href")); href != "" {
				ref := "l" + strconv.Itoa(len(p.Links)+1)
				label := n.prop("aria-label")
				if label == "" {
					label = n.innerText(1200)
				}
				if label == "" {
					label = "(link)"
				}
				p.Links = append(p.Links, Link{Ref: ref, Label: label, URL: href})
				n.ref = ref
				d.links[ref] = n
			}
		}
		if len(n.Tag) == 2 && n.Tag[0] == 'h' && n.Tag[1] >= '1' && n.Tag[1] <= '6' {
			p.Headings = append(p.Headings, Heading{Level: int(n.Tag[1] - '0'), Text: n.innerText(1200)})
		}
	})
	if hasControls {
		p.Forms, p.Buttons, p.Inputs = d.collectForms(base)
	}
	if includeMarkdown {
		p.Markdown = d.Markdown()
	}
	p.Text = text.text.String()
	p.Truncated = text.truncated
	return p
}

// Markdown converts the DOM into clean, structured Markdown for AI agents.
func (d *Document) Markdown() string {
	if d == nil || d.Root == nil {
		return ""
	}
	base, _ := url.Parse(d.URL)
	var b strings.Builder

	var walk func(n *Node, inPre bool)
	walk = func(n *Node, inPre bool) {
		if n == nil || n.hidden() {
			return
		}
		if n.Tag == "#text" {
			txt := n.Text
			if !inPre {
				txt = norm(html.UnescapeString(txt))
			}
			if txt != "" {
				b.WriteString(txt)
			}
			return
		}

		switch n.Tag {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			level := int(n.Tag[1] - '0')
			b.WriteString("\n\n" + strings.Repeat("#", level) + " ")
			for _, ch := range n.Children {
				walk(ch, inPre)
			}
			b.WriteString("\n\n")
		case "p", "section", "article":
			b.WriteString("\n\n")
			for _, ch := range n.Children {
				walk(ch, inPre)
			}
			b.WriteString("\n\n")
		case "br":
			b.WriteString("\n")
		case "hr":
			b.WriteString("\n\n---\n\n")
		case "ul", "ol":
			b.WriteString("\n")
			for _, ch := range n.Children {
				walk(ch, inPre)
			}
			b.WriteString("\n")
		case "li":
			b.WriteString("\n- ")
			for _, ch := range n.Children {
				walk(ch, inPre)
			}
		case "blockquote":
			b.WriteString("\n> ")
			for _, ch := range n.Children {
				walk(ch, inPre)
			}
			b.WriteString("\n")
		case "pre":
			b.WriteString("\n```\n")
			for _, ch := range n.Children {
				walk(ch, true)
			}
			b.WriteString("\n```\n")
		case "code":
			if !inPre {
				b.WriteString("`")
				for _, ch := range n.Children {
					walk(ch, true)
				}
				b.WriteString("`")
			} else {
				for _, ch := range n.Children {
					walk(ch, true)
				}
			}
		case "strong", "b":
			b.WriteString("**")
			for _, ch := range n.Children {
				walk(ch, inPre)
			}
			b.WriteString("**")
		case "em", "i":
			b.WriteString("*")
			for _, ch := range n.Children {
				walk(ch, inPre)
			}
			b.WriteString("*")
		case "a":
			href := ""
			if has(n.Attr, "href") {
				href = resolve(base, n.prop("href"))
			}
			label := strings.TrimSpace(html.UnescapeString(n.innerText(256)))
			if label == "" {
				label = n.prop("aria-label")
			}
			if label == "" {
				label = n.prop("title")
			}
			if label == "" {
				for _, ch := range n.Children {
					if ch.Tag == "img" && ch.prop("alt") != "" {
						label = ch.prop("alt")
						break
					}
				}
			}
			if href != "" && label != "" {
				b.WriteString(" [" + label + "](" + href + ") ")
			} else {
				for _, ch := range n.Children {
					walk(ch, inPre)
				}
			}
		case "table":
			b.WriteString("\n\n")
			for _, ch := range n.Children {
				walk(ch, inPre)
			}
			b.WriteString("\n\n")
		case "tr":
			b.WriteString("\n")
			for _, ch := range n.Children {
				walk(ch, inPre)
			}
		case "td", "th":
			b.WriteString(" ")
			for _, ch := range n.Children {
				walk(ch, inPre)
			}
			b.WriteString(" ")
		default:
			for _, ch := range n.Children {
				walk(ch, inPre)
			}
		}
	}

	walk(d.Root, false)
	lines := strings.Split(b.String(), "\n")
	var cleaned []string
	blank := false
	for _, l := range lines {
		trimmed := strings.TrimRight(l, " \t\r")
		if strings.TrimSpace(trimmed) == "" {
			if !blank && len(cleaned) > 0 {
				cleaned = append(cleaned, "")
				blank = true
			}
		} else {
			cleaned = append(cleaned, trimmed)
			blank = false
		}
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}
func (n *Node) rawText(limit int) string {
	var b strings.Builder
	var walk func(*Node)
	walk = func(x *Node) {
		if limit > 0 && b.Len() > limit {
			return
		}
		if x.Tag == "#text" {
			b.WriteString(x.Text)
		}
		for _, ch := range x.Children {
			walk(ch)
		}
	}
	walk(n)
	return b.String()
}
func (n *Node) value() string {
	if n.editableControl() && n.prop(liveMarker) != "" {
		if value, present := n.Attr["data-aigobrowser-editable-value"]; present {
			return value
		}
	}
	if n.Tag == "textarea" || n.editableControl() {
		return n.rawText(0)
	}
	if n.Tag == "select" {
		_, _, value := n.selectState()
		return value
	}
	if v, ok := n.Attr["value"]; ok {
		return v
	}
	if n.controlType() == "checkbox" || n.controlType() == "radio" {
		return "on"
	}
	return ""
}

func (n *Node) optionNodes() []*Node {
	var options []*Node
	var walk func(*Node)
	walk = func(cur *Node) {
		if cur != n && cur.Tag == "select" {
			return
		}
		if cur.Tag == "option" {
			options = append(options, cur)
			return
		}
		for _, child := range cur.Children {
			walk(child)
		}
	}
	walk(n)
	return options
}

func optionValue(n *Node) string {
	if value, present := n.Attr["value"]; present {
		return value
	}
	return norm(n.rawText(0))
}

func (n *Node) selectState() ([]Option, []string, string) {
	options := []Option{}
	selected := -1
	firstEnabled := -1
	multiple := has(n.Attr, "multiple")
	for _, node := range n.optionNodes() {
		disabled := has(node.Attr, "disabled")
		for parent := node.Parent; parent != nil && parent != n; parent = parent.Parent {
			if parent.Tag == "optgroup" && has(parent.Attr, "disabled") {
				disabled = true
			}
		}
		option := Option{Value: optionValue(node), Label: node.prop("label"), Selected: has(node.Attr, "selected"), Disabled: disabled}
		if option.Label == "" {
			option.Label = node.innerText(512)
		}
		if !disabled && firstEnabled < 0 {
			firstEnabled = len(options)
		}
		if option.Selected && !multiple {
			if selected >= 0 {
				options[selected].Selected = false
			}
			selected = len(options)
		}
		options = append(options, option)
	}
	size, _ := strconv.Atoi(n.prop("size"))
	if !multiple && selected < 0 && !n.valueSet && firstEnabled >= 0 && size <= 1 {
		options[firstEnabled].Selected = true
	}
	values := []string{}
	value := ""
	found := false
	for _, option := range options {
		if !option.Selected {
			continue
		}
		if !found {
			value, found = option.Value, true
		}
		if !option.Disabled {
			values = append(values, option.Value)
		}
	}
	return options, values, value
}

func (n *Node) setValue(v string) {
	if n.Tag == "textarea" || n.editableControl() {
		delete(n.Attr, "data-aigobrowser-editable-value")
		n.Children = []*Node{{Tag: "#text", Text: v, Parent: n}}
		return
	}
	if n.Tag == "select" {
		n.valueSet = true
		selected := false
		for _, option := range n.optionNodes() {
			delete(option.Attr, "selected")
			if !selected && optionValue(option) == v {
				if option.Attr == nil {
					option.Attr = map[string]string{}
				}
				option.Attr["selected"] = ""
				selected = true
			}
		}
		return
	}
	if n.Attr == nil {
		n.Attr = map[string]string{}
	}
	n.Attr["value"] = v
}

// Query supports simple compound selectors (tag, #id, .class, [name=value]),
// and a single descendant relation. It is NOT a full CSS selector engine.
func (d *Document) Query(selector string) ([]Element, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" || len(selector) > 200 {
		return nil, errors.New("selector missing or too long")
	}
	parts := strings.Fields(selector)
	if len(parts) > 2 {
		return nil, errors.New("at most one descendant combinator is supported")
	}
	for _, part := range parts {
		if _, err := parseSimpleSelector(part); err != nil {
			return nil, err
		}
	}
	out := []Element{}
	var walk func(*Node, bool)
	walk = func(n *Node, hidden bool) {
		if n == nil || len(out) >= maxQueryResults {
			return
		}
		hidden = hidden || n.hidden()
		if !hidden && n.Tag != "#text" && n.Tag != "#document" {
			match := matchSimple(n, parts[len(parts)-1])
			if match && len(parts) == 2 {
				match = false
				for par := n.Parent; par != nil; par = par.Parent {
					if matchSimple(par, parts[0]) {
						match = true
						break
					}
				}
			}
			if match {
				el := Element{Ref: n.ref, Tag: n.Tag, ID: n.prop("id"), Role: n.prop("role"), Label: norm(n.prop("aria-label")), Text: n.innerText(700), Name: n.prop("name")}
				if label, present := n.snapshotLabel(); present {
					el.Label = label
				}
				if n.Tag == "a" && has(n.Attr, "href") {
					base, _ := url.Parse(d.URL)
					el.Href = resolve(base, n.prop("href"))
				}
				if n.genericButton() || n.Tag == "button" {
					el.Type = n.controlType()
					el.Disabled = n.disabled()
				}
				if n.Tag == "input" || n.Tag == "textarea" || n.Tag == "select" || n.editableControl() {
					el.Type = n.controlType()
					el.Disabled = n.disabled()
					el.ReadOnly = n.readOnly()
					el.Required = has(n.Attr, "required")
					el.Pattern, el.Min, el.Max, el.Step = n.prop("pattern"), n.prop("min"), n.prop("max"), n.prop("step")
					if el.Type == "checkbox" || el.Type == "radio" {
						checked := has(n.Attr, "checked")
						el.Checked = &checked
					}
					if el.Type == "password" || el.Type == "hidden" {
						el.Value = "[redacted]"
					} else if n.Tag == "select" {
						el.Multiple = has(n.Attr, "multiple")
						el.Options, el.Values, el.Value = n.selectState()
					} else {
						el.Value = n.value()
					}
				}
				out = append(out, el)
			}
		}
		if !hidden {
			for _, ch := range n.Children {
				walk(ch, false)
			}
		}
	}
	walk(d.Root, false)
	return out, nil
}
func parseSimpleSelector(s string) (bool, error) {
	if strings.ContainsAny(s, ":>+~,*") {
		return false, errors.New("unsupported selector syntax")
	}
	if strings.Count(s, "[") > 1 || strings.Count(s, "]") > 1 || strings.Count(s, "[") != strings.Count(s, "]") {
		return false, errors.New("invalid attribute selector")
	}
	if s == "" {
		return false, errors.New("empty selector")
	}
	return true, nil
}
func matchSimple(n *Node, s string) bool {
	if n.Tag == "#text" || n.Tag == "#document" {
		return false
	}
	if i := strings.IndexByte(s, '['); i >= 0 {
		a := strings.TrimSuffix(s[i+1:], "]")
		s = s[:i]
		parts := strings.SplitN(a, "=", 2)
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		if key != "name" && key != "type" && key != "role" && key != "id" && key != "href" {
			return false
		}
		if len(parts) == 2 {
			val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			if n.prop(key) != val {
				return false
			}
		} else {
			if _, ok := n.Attr[key]; !ok {
				return false
			}
		}
	}
	tagEnd := strings.IndexAny(s, ".#")
	if tagEnd < 0 {
		tagEnd = len(s)
	}
	if tag := s[:tagEnd]; tag != "" && tag != "*" && tag != n.Tag {
		return false
	}
	s = s[tagEnd:]
	for len(s) > 0 {
		symbol := s[0]
		s = s[1:]
		end := strings.IndexAny(s, ".#")
		if end < 0 {
			end = len(s)
		}
		tok := s[:end]
		if tok == "" {
			return false
		}
		if symbol == '#' && n.prop("id") != tok {
			return false
		}
		if symbol == '.' && !strings.Contains(" "+n.prop("class")+" ", " "+tok+" ") {
			return false
		}
		s = s[end:]
	}
	return true
}
