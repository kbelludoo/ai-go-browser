package browser

// This is a bounded, dependency-free DOM for agent navigation. It implements
// an intentionally LIMITED HTML subset, not the HTML Living Standard parser.
// In particular it does not implement the adoption agency algorithm,
// scripting event loop, custom elements, shadow DOM or CSS layout.

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
	Tag   string `json:"tag"`
	ID    string `json:"id,omitempty"`
	Role  string `json:"role,omitempty"`
	Text  string `json:"text,omitempty"`
	Href  string `json:"href,omitempty"`
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

func (n *Node) add(child *Node) {
	child.Parent = n
	n.Children = append(n.Children, child)
}
func (n *Node) prop(name string) string { return n.Attr[name] }
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
	var b strings.Builder
	var visit func(*Node, bool)
	visit = func(p *Node, hidden bool) {
		if b.Len() > limit || hidden || p.hidden() {
			return
		}
		if p.Tag == "#text" {
			b.WriteByte(' ')
			b.WriteString(p.Text)
			return
		}
		if p.Tag == "img" && p.prop("alt") != "" {
			b.WriteByte(' ')
			b.WriteString(p.prop("alt"))
		}
		for _, child := range p.Children {
			visit(child, false)
		}
	}
	visit(n, false)
	s := norm(b.String())
	if len(s) > limit {
		s = string([]rune(s)[:min(len([]rune(s)), limit)])
	}
	return s
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
func (d *Document) Page(textLimit int) Page {
	p := Page{URL: d.URL, Status: d.Status, Bytes: d.Bytes, Links: []Link{}, Headings: []Heading{}, Forms: []Form{}, Metadata: map[string]string{}}
	d.links = map[string]*Node{}
	d.fields = map[string]*Node{}
	d.buttons = map[string]*Node{}
	base, _ := url.Parse(d.URL)
	// Collect title, meta and base href.
	var findHead func(*Node)
	findHead = func(n *Node) {
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
	var parts []string
	// A document tree has no layout; a bounded text-only traversal is best
	// suited to agent observations. CSS is not applied.
	domWalk(d.Root, false, func(n *Node) {
		if n.Tag == "#text" {
			if t := norm(n.Text); t != "" {
				parts = append(parts, t)
			}
			return
		}
		if n.Tag == "a" {
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
				d.links[ref] = n
			}
		}
		if n.Tag == "button" || (n.Tag == "input" && n.prop("type") == "button") {
			ref := "b" + strconv.Itoa(len(p.Buttons)+1)
			label := n.prop("aria-label")
			if label == "" {
				label = n.innerText(256)
			}
			if label == "" {
				label = n.prop("value")
			}
			if label == "" {
				label = "(button)"
			}
			p.Buttons = append(p.Buttons, Button{Ref: ref, ID: n.prop("id"), Label: label, Type: n.prop("type")})
			d.buttons[ref] = n
		}
		if len(n.Tag) == 2 && n.Tag[0] == 'h' && n.Tag[1] >= '1' && n.Tag[1] <= '6' {
			p.Headings = append(p.Headings, Heading{Level: int(n.Tag[1] - '0'), Text: n.innerText(1200)})
		}
		if n.Tag == "form" {
			method := strings.ToUpper(n.prop("method"))
			if method != "POST" {
				method = "GET"
			}
			action := d.URL
			if n.prop("action") != "" {
				if v := resolve(base, n.prop("action")); v != "" {
					action = v
				}
			}
			f := Form{Ref: "f" + strconv.Itoa(len(p.Forms)+1), Method: method, Action: action, Fields: []Field{}, SubmitLabel: "Enviar"}
			var controls func(*Node)
			controls = func(cur *Node) {
				if cur != n && cur.Tag == "form" {
					return
				}
				switch cur.Tag {
				case "button":
					if v := cur.innerText(256); v != "" && (cur.prop("type") == "" || cur.prop("type") == "submit") {
						f.SubmitLabel = v
					}
				case "input", "textarea", "select":
					typ := cur.prop("type")
					if cur.Tag != "input" {
						typ = cur.Tag
					}
					if typ == "" {
						typ = "text"
					}
					typ = strings.ToLower(typ)
					if typ == "submit" {
						if v := cur.prop("value"); v != "" {
							f.SubmitLabel = v
						}
						break
					}
					if typ == "button" || typ == "reset" {
						break
					}
					val := cur.value()
					if (typ == "checkbox" || typ == "radio") && cur.prop("checked") == "" {
						if _, ok := cur.Attr["checked"]; !ok {
							val = ""
						}
					}
					ref := "i" + strconv.Itoa(len(d.fields)+1)
					label := cur.prop("aria-label")
					if label == "" {
						label = cur.prop("placeholder")
					}
					f.Fields = append(f.Fields, Field{Ref: ref, Type: typ, Name: cur.prop("name"), Value: val, Label: label})
					d.fields[ref] = cur
				}
				for _, ch := range cur.Children {
					controls(ch)
				}
			}
			controls(n)
			p.Forms = append(p.Forms, f)
		}
	})
	p.Markdown = d.Markdown()
	p.Text = norm(strings.Join(parts, " "))
	if textLimit > 0 && len([]rune(p.Text)) > textLimit {
		p.Text = string([]rune(p.Text)[:textLimit])
		p.Truncated = true
	}
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
			href := resolve(base, n.prop("href"))
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
		if b.Len() > limit {
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
	if v, ok := n.Attr["value"]; ok {
		return v
	}
	if n.Tag == "textarea" {
		return n.rawText(8192)
	}
	if n.Tag == "select" {
		var first, selected string
		var walk func(*Node)
		walk = func(x *Node) {
			if x.Tag == "option" {
				v := x.prop("value")
				if v == "" {
					v = norm(x.rawText(512))
				}
				if first == "" {
					first = v
				}
				if _, ok := x.Attr["selected"]; ok {
					selected = v
				}
			}
			for _, ch := range x.Children {
				walk(ch)
			}
		}
		walk(n)
		if selected != "" {
			return selected
		}
		return first
	}
	return ""
}
func (n *Node) setValue(v string) {
	if n.Tag == "textarea" {
		n.Children = []*Node{{Tag: "#text", Text: v, Parent: n}}
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
				el := Element{Tag: n.Tag, ID: n.prop("id"), Role: n.prop("role"), Text: n.innerText(700), Name: n.prop("name")}
				if n.Tag == "a" {
					base, _ := url.Parse(d.URL)
					el.Href = resolve(base, n.prop("href"))
				}
				if n.Tag == "input" || n.Tag == "textarea" || n.Tag == "select" {
					if n.prop("type") == "password" || n.prop("type") == "hidden" {
						el.Value = "[redacted]"
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
