package browser

import (
	"html"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

type Link struct {
	Ref   string `json:"ref"`
	Label string `json:"label"`
	URL   string `json:"url"`
}
type Button struct {
	Ref   string `json:"ref"`
	ID    string `json:"id,omitempty"`
	Label string `json:"label"`
	Type  string `json:"type,omitempty"`
}
type Heading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}
type Field struct {
	Ref   string `json:"ref"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value,omitempty"`
	Label string `json:"label,omitempty"`
}
type Form struct {
	Ref         string  `json:"ref"`
	Action      string  `json:"action"`
	Method      string  `json:"method"`
	Fields      []Field `json:"fields"`
	SubmitLabel string  `json:"submit_label,omitempty"`
}
type Page struct {
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Text      string    `json:"text"`
	Headings  []Heading `json:"headings"`
	Links     []Link    `json:"links"`
	Buttons   []Button  `json:"buttons,omitempty"`
	Forms     []Form    `json:"forms"`
	Status    int       `json:"status"`
	Bytes     int       `json:"bytes"`
	Truncated bool      `json:"truncated,omitempty"`
}

type token struct {
	tag, text      string
	attrs          map[string]string
	end, selfClose bool
}

func isNameChar(c byte) bool {
	return c != ' ' && c != '\r' && c != '\n' && c != '\t' && c != '/' && c != '>' && c != '='
}
func attrsFrom(s string) (string, map[string]string, bool) {
	m := map[string]string{}
	s = strings.TrimSpace(s)
	self := strings.HasSuffix(s, "/")
	s = strings.TrimSuffix(s, "/")
	i := 0
	for i < len(s) && !unicode.IsSpace(rune(s[i])) {
		i++
	}
	name := strings.ToLower(s[:i])
	for i < len(s) {
		for i < len(s) && (unicode.IsSpace(rune(s[i])) || s[i] == '/') {
			i++
		}
		a := i
		for i < len(s) && isNameChar(s[i]) {
			i++
		}
		if a == i {
			i++
			continue
		}
		key := strings.ToLower(s[a:i])
		for i < len(s) && unicode.IsSpace(rune(s[i])) {
			i++
		}
		value := ""
		if i < len(s) && s[i] == '=' {
			i++
			for i < len(s) && unicode.IsSpace(rune(s[i])) {
				i++
			}
			if i < len(s) && (s[i] == '"' || s[i] == '\'') {
				q := s[i]
				i++
				j := i
				for i < len(s) && s[i] != q {
					i++
				}
				value = s[j:i]
				if i < len(s) {
					i++
				}
			} else {
				j := i
				for i < len(s) && !unicode.IsSpace(rune(s[i])) && s[i] != '>' {
					i++
				}
				value = s[j:i]
			}
		}
		m[key] = html.UnescapeString(value)
	}
	return name, m, self
}
func nextToken(s string, pos *int) token {
	if *pos >= len(s) {
		return token{}
	}
	if s[*pos] != '<' {
		start := *pos
		i := strings.IndexByte(s[start:], '<')
		if i < 0 {
			*pos = len(s)
		} else {
			*pos = start + i
		}
		return token{text: html.UnescapeString(s[start:*pos])}
	}
	start := *pos
	if strings.HasPrefix(s[start:], "<!--") {
		p := strings.Index(s[start+4:], "-->")
		if p < 0 {
			*pos = len(s)
		} else {
			*pos = start + 4 + p + 3
		}
		return token{tag: "!comment"}
	}
	// Find >, respecting quoted attributes.
	q := byte(0)
	end := -1
	for i := start + 1; i < len(s); i++ {
		c := s[i]
		if q != 0 {
			if c == q {
				q = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			q = c
		} else if c == '>' {
			end = i
			break
		}
	}
	if end < 0 {
		*pos = start + 1
		return token{text: "<"}
	}
	*pos = end + 1
	raw := strings.TrimSpace(s[start+1 : end])
	if raw == "" {
		return token{}
	}
	if raw[0] == '!' || raw[0] == '?' {
		return token{tag: "!directive"}
	}
	closing := false
	if raw[0] == '/' {
		closing = true
		raw = strings.TrimSpace(raw[1:])
	}
	name, attrs, self := attrsFrom(raw)
	return token{tag: name, attrs: attrs, end: closing, selfClose: self}
}
func norm(s string) string { return strings.Join(strings.Fields(s), " ") }
func resolve(base *url.URL, ref string) string {
	u, e := url.Parse(strings.TrimSpace(ref))
	if e != nil {
		return ""
	}
	u = base.ResolveReference(u)
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return u.String()
}
func attr(m map[string]string, k string) string { return m[k] }
func has(m map[string]string, k string) bool    { _, yes := m[k]; return yes }
func voidTag(s string) bool {
	switch s {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	}
	return false
}
func skipTag(s string) bool {
	switch s {
	case "script", "style", "noscript", "template", "svg", "head":
		return true
	}
	return false
}

type frame struct {
	tag                                string
	hidden                             bool
	link, heading, form, field, button int
}

func Parse(raw []byte, baseURL string, status, bytesRead, textLimit int) Page {
	base, _ := url.Parse(baseURL)
	p := Page{URL: baseURL, Headings: []Heading{}, Links: []Link{}, Forms: []Form{}, Status: status, Bytes: bytesRead}
	stack := []frame{}
	parts := []string{}
	var title strings.Builder
	active := func() (bool, int, int, int, int, int) {
		hidden := false
		l, h, f, fi, b := -1, -1, -1, -1, -1
		for _, fr := range stack {
			if fr.hidden {
				hidden = true
			}
			if fr.link >= 0 {
				l = fr.link
			}
			if fr.heading >= 0 {
				h = fr.heading
			}
			if fr.form >= 0 {
				f = fr.form
			}
			if fr.field >= 0 {
				fi = fr.field
			}
			if fr.button >= 0 {
				b = fr.button
			}
		}
		return hidden, l, h, f, fi, b
	}
	getText := func(t string) {
		t = norm(t)
		if t == "" {
			return
		}
		hidden, l, h, f, fi, b := active()
		if len(stack) > 0 {
			for _, fr := range stack {
				if fr.tag == "title" {
					if title.Len() > 0 {
						title.WriteString(" ")
					}
					title.WriteString(t)
					return
				}
			}
		}
		if hidden {
			return
		}
		parts = append(parts, t)
		if l >= 0 {
			if p.Links[l].Label != "" {
				p.Links[l].Label += " "
			}
			p.Links[l].Label += t
		}
		if h >= 0 {
			if p.Headings[h].Text != "" {
				p.Headings[h].Text += " "
			}
			p.Headings[h].Text += t
		}
		if fi >= 0 && f >= 0 {
			for i := range p.Forms[f].Fields {
				if i == fi {
					if p.Forms[f].Fields[i].Value != "" {
						p.Forms[f].Fields[i].Value += " "
					}
					p.Forms[f].Fields[i].Value += t
					break
				}
			}
		}
		if b >= 0 && f >= 0 {
			if p.Forms[f].SubmitLabel != "" {
				p.Forms[f].SubmitLabel += " "
			}
			p.Forms[f].SubmitLabel += t
		}
	}
	src := string(raw)
	pos := 0
	for pos < len(src) {
		tok := nextToken(src, &pos) // parser intentionally supports an HTML subset, no CSS/JS execution
		if tok.tag == "" {
			getText(tok.text)
			continue
		}
		if strings.HasPrefix(tok.tag, "!") {
			continue
		}
		if tok.end {
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].tag == tok.tag {
					stack = stack[:i]
					break
				}
			}
			continue
		}
		hidden, _, _, f, _, _ := active()
		attrs := tok.attrs
		fr := frame{tag: tok.tag, link: -1, heading: -1, form: -1, field: -1, button: -1}
		fr.hidden = skipTag(tok.tag) || hidden || has(attrs, "hidden") || strings.EqualFold(attr(attrs, "aria-hidden"), "true") || strings.Contains(strings.ToLower(strings.ReplaceAll(attr(attrs, "style"), " ", "")), "display:none")
		if tok.tag == "title" {
			fr.hidden = false
		}
		if tok.tag == "a" && !fr.hidden {
			if href := resolve(base, attr(attrs, "href")); href != "" {
				fr.link = len(p.Links)
				p.Links = append(p.Links, Link{Ref: "l" + strconv.Itoa(len(p.Links)+1), Label: norm(attr(attrs, "aria-label")), URL: href})
			}
		}
		if len(tok.tag) == 2 && tok.tag[0] == 'h' && tok.tag[1] >= '1' && tok.tag[1] <= '6' && !fr.hidden {
			fr.heading = len(p.Headings)
			p.Headings = append(p.Headings, Heading{Level: int(tok.tag[1] - '0')})
		}
		if tok.tag == "form" && !fr.hidden {
			action := attr(attrs, "action")
			if action == "" {
				action = baseURL
			}
			resolved := resolve(base, action)
			if resolved == "" {
				resolved = baseURL
			}
			method := strings.ToUpper(attr(attrs, "method"))
			if method != "POST" {
				method = "GET"
			}
			fr.form = len(p.Forms)
			p.Forms = append(p.Forms, Form{Ref: "f" + strconv.Itoa(len(p.Forms)+1), Action: resolved, Method: method, Fields: []Field{}})
		}
		if tok.tag == "input" && !fr.hidden && f >= 0 {
			typ := strings.ToLower(attr(attrs, "type"))
			if typ == "" {
				typ = "text"
			}
			if typ == "submit" {
				if p.Forms[f].SubmitLabel == "" {
					p.Forms[f].SubmitLabel = attr(attrs, "value")
				}
			} else if typ != "button" && typ != "reset" {
				p.Forms[f].Fields = append(p.Forms[f].Fields, Field{Ref: "i" + strconv.Itoa(nextFieldID(p.Forms)), Name: attr(attrs, "name"), Type: typ, Value: attr(attrs, "value"), Label: attr(attrs, "aria-label")})
			}
		}
		if (tok.tag == "textarea" || tok.tag == "select") && !fr.hidden && f >= 0 {
			fr.field = len(p.Forms[f].Fields)
			p.Forms[f].Fields = append(p.Forms[f].Fields, Field{Ref: "i" + strconv.Itoa(nextFieldID(p.Forms)), Name: attr(attrs, "name"), Type: tok.tag, Label: attr(attrs, "aria-label")})
		}
		if tok.tag == "button" && !fr.hidden && f >= 0 {
			typ := strings.ToLower(attr(attrs, "type"))
			if typ == "" || typ == "submit" {
				fr.button = 1
			}
		}
		if tok.tag == "img" && !fr.hidden {
			_, l, _, _, _, _ := active()
			if l >= 0 && p.Links[l].Label == "" {
				p.Links[l].Label = attr(attrs, "alt")
			}
		}
		if !tok.selfClose && !voidTag(tok.tag) {
			stack = append(stack, fr)
		}
	}
	p.Title = norm(title.String())
	p.Text = norm(strings.Join(parts, " "))
	if textLimit > 0 && len(p.Text) > textLimit {
		p.Text = string([]rune(p.Text)[:min(textLimit, len([]rune(p.Text)))])
		p.Truncated = true
	}
	for i := range p.Links {
		p.Links[i].Label = norm(p.Links[i].Label)
		if p.Links[i].Label == "" {
			p.Links[i].Label = "(link)"
		}
	}
	for i := range p.Forms {
		if p.Forms[i].SubmitLabel == "" {
			p.Forms[i].SubmitLabel = "Enviar"
		}
	}
	return p
}
func nextFieldID(forms []Form) int {
	sum := 0
	for _, f := range forms {
		sum += len(f.Fields)
	}
	return sum + 1
}
