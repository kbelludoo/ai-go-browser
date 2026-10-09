package browser

// Action Resolver: static, conservative discovery of network-facing actions
// from the HTML DOM and literal JavaScript. It deliberately does NOT claim
// to emulate all JS or to know what remote APIs will return before requesting.

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const maxActions = 100
const maxExternalScripts = 4
const maxExternalScriptBytes = 256 << 10
const maxResourceBytes = 96 << 10

type Action struct {
	Ref                  string        `json:"ref"`
	Label                string        `json:"label"`
	Kind                 string        `json:"kind"`   // link, form, button, api
	Method               string        `json:"method"` // GET, POST, UNKNOWN
	URL                  string        `json:"url,omitempty"`
	Source               string        `json:"source"`
	Evidence             string        `json:"evidence,omitempty"`
	Preview              string        `json:"preview,omitempty"`
	Confidence           string        `json:"confidence"` // declared, inferred, unknown
	RequiresConfirmation bool          `json:"requires_confirmation"`
	Executable           bool          `json:"executable"`
	Fields               []ActionField `json:"fields,omitempty"`
}

type ActionField struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value,omitempty"`
}

type Resource struct {
	URL         string `json:"url"`
	Status      int    `json:"status"`
	ContentType string `json:"content_type"`
	Body        string `json:"body,omitempty"`
	Data        any    `json:"data,omitempty"`
	Truncated   bool   `json:"truncated,omitempty"`
}

type jsCall struct {
	method   string
	target   string
	evidence string
}

var (
	fetchPattern     = regexp.MustCompile(`(?s)\bfetch\s*\(\s*['"\x60]([^'"\x60]+)['"\x60]`)
	axiosPattern     = regexp.MustCompile(`(?s)\baxios\s*\.\s*(get|post)\s*\(\s*['"\x60]([^'"\x60]+)['"\x60]`)
	jqueryPattern    = regexp.MustCompile(`(?s)\$\s*\.\s*(get|post)\s*\(\s*['"\x60]([^'"\x60]+)['"\x60]`)
	xhrPattern       = regexp.MustCompile(`(?s)\.\s*open\s*\(\s*['"](GET|POST)['"]\s*,\s*['"]([^'"]+)['"]`)
	methodPattern    = regexp.MustCompile(`(?i)\bmethod\s*:\s*['"](GET|POST)['"]`)
	methodAnyPattern = regexp.MustCompile(`(?i)\bmethod\s*:`)
	byIDEvent        = regexp.MustCompile(`(?s)(?:getElementById\s*\(\s*['"]([^'"]+)['"]\s*\)|querySelector\s*\(\s*['"]#([^'"]+)['"]\s*\))\s*\.\s*addEventListener\s*\(\s*['"]click['"]`)
	functionPattern  = regexp.MustCompile(`\bfunction\s+([A-Za-z_$][\w$]*)\s*\(`)
	callNamePattern  = regexp.MustCompile(`^\s*([A-Za-z_$][\w$]*)\s*\(\s*\)\s*;?\s*$`)
)

// stripJSComments removes comments while preserving string literals and offsets.
// It does not parse template literal substitutions or regular expressions.
func stripJSComments(src string) string {
	dst := []byte(src)
	quote := byte(0)
	for i := 0; i < len(src); i++ {
		c := src[i]
		if quote != 0 {
			if c == '\\' {
				i++
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' || c == '`' {
			quote = c
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '/' {
			start := i
			for i < len(src) && src[i] != '\n' {
				i++
			}
			for j := start; j < i; j++ {
				dst[j] = ' '
			}
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '*' {
			start := i
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i++
			for j := start; j <= i && j < len(src); j++ {
				dst[j] = ' '
			}
		}
	}
	return string(dst)
}

// balancedEnd returns the closing delimiter position, ignoring quoted strings
// and JS comments. Limited lexer, not a JavaScript parser.
func balancedEnd(s string, start int, open, close byte) int {
	if start < 0 || start >= len(s) || s[start] != open {
		return -1
	}
	depth := 0
	quote := byte(0)
	for i := start; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == '\\' {
				i++
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '/' && i+1 < len(s) && s[i+1] == '/' {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(s) && s[i+1] == '*' {
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			i++
			continue
		}
		if c == '\'' || c == '"' || c == '`' {
			quote = c
			continue
		}
		if c == open {
			depth++
		}
		if c == close {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func extractJSCalls(s string) []jsCall {
	s = stripJSComments(s)
	calls := []jsCall{}
	for _, m := range fetchPattern.FindAllStringSubmatchIndex(s, 40) {
		target := s[m[2]:m[3]]
		if strings.Contains(target, "${") {
			continue
		}
		start := strings.Index(s[m[0]:m[1]], "(") + m[0]
		end := balancedEnd(s, start, '(', ')')
		if end < 0 {
			continue
		}
		args := s[m[1]:end]
		method := "GET"
		trimmed := strings.TrimSpace(args)
		if strings.HasPrefix(trimmed, ",") && !strings.HasPrefix(strings.TrimSpace(strings.TrimSpace(trimmed[1:])), "{") {
			method = "UNKNOWN" // variables/RequestInit are not statically resolved
		}
		if methodAnyPattern.MatchString(args) {
			method = "UNKNOWN"
			if found := methodPattern.FindStringSubmatch(args); len(found) == 2 {
				method = strings.ToUpper(found[1])
			}
		}
		calls = append(calls, jsCall{method: method, target: target, evidence: "fetch(literal URL)"})
	}
	for _, p := range []struct {
		re   *regexp.Regexp
		kind string
	}{{axiosPattern, "axios"}, {jqueryPattern, "jQuery"}, {xhrPattern, "XHR"}} {
		for _, m := range p.re.FindAllStringSubmatch(s, 40) {
			if strings.Contains(m[2], "${") {
				continue
			}
			calls = append(calls, jsCall{method: strings.ToUpper(m[1]), target: m[2], evidence: p.kind + "(literal URL)"})
		}
	}
	return calls
}

func getFunctionBody(script, name string) string {
	for _, idx := range functionPattern.FindAllStringSubmatchIndex(script, 100) {
		if script[idx[2]:idx[3]] != name {
			continue
		}
		open := strings.IndexByte(script[idx[1]:], '{')
		if open < 0 {
			continue
		}
		start := idx[1] + open
		end := balancedEnd(script, start, '{', '}')
		if end > start {
			return script[start+1 : end]
		}
	}
	return ""
}

func scriptHandlers(script string) map[string][]jsCall {
	out := map[string][]jsCall{}
	for _, idx := range byIDEvent.FindAllStringSubmatchIndex(script, 60) {
		id := ""
		if idx[2] >= 0 {
			id = script[idx[2]:idx[3]]
		} else if idx[4] >= 0 {
			id = script[idx[4]:idx[5]]
		}
		open := strings.LastIndex(script[idx[0]:idx[1]], "(")
		if open < 0 {
			continue
		}
		start := idx[0] + open
		end := balancedEnd(script, start, '(', ')')
		if end < 0 {
			continue
		}
		handler := script[idx[1]:end]
		out[id] = append(out[id], extractJSCalls(handler)...)
	}
	return out
}

func collectScriptText(d *Document) (string, []*Node) {
	var scripts []string
	external := []*Node{}
	if d == nil || d.Root == nil {
		return "", external
	}
	var walk func(*Node)
	walk = func(n *Node) {
		if n.Tag == "script" {
			if n.prop("src") != "" {
				external = append(external, n)
			} else if typ := strings.ToLower(n.prop("type")); typ == "" || typ == "text/javascript" || typ == "application/javascript" || typ == "module" {
				scripts = append(scripts, n.rawText(maxExternalScriptBytes))
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(d.Root)
	return strings.Join(scripts, "\n"), external
}

// externalScripts are fetched only when explicitly enabled; only same-origin
// script references are allowed and bounded. No scripts are executed here.
func (b *Browser) externalScripts(nodes []*Node, originURL string) string {
	if !b.ScanScripts {
		return ""
	}
	if b.scannedScripts {
		return b.scannedScriptText
	}
	b.scannedScripts = true
	origin, err := url.Parse(originURL)
	if err != nil {
		return ""
	}
	parts := []string{}
	for i, n := range nodes {
		if i >= maxExternalScripts {
			break
		}
		rawURL := n.prop("src")
		target := resolve(origin, rawURL)
		u, err := url.Parse(target)
		if err != nil || u.Scheme != origin.Scheme || u.Host != origin.Host || b.validate(u) != nil {
			continue
		}
		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			continue
		}
		res, err := b.client.Do(req)
		if err != nil {
			continue
		}
		if res.Request.URL.Scheme != origin.Scheme || res.Request.URL.Host != origin.Host || res.StatusCode != 200 {
			res.Body.Close()
			continue
		}
		content, err := io.ReadAll(io.LimitReader(res.Body, maxExternalScriptBytes+1))
		res.Body.Close()
		if err == nil && len(content) <= maxExternalScriptBytes {
			parts = append(parts, string(content))
		}
	}
	b.scannedScriptText = strings.Join(parts, "\n")
	return b.scannedScriptText
}

// Text already included in a page but hidden behind an accessible control can
// be returned directly without clicking or executing JavaScript.
func findNodeID(root *Node, id string) *Node {
	if id == "" || root == nil {
		return nil
	}
	if root.prop("id") == id {
		return root
	}
	for _, n := range root.Children {
		if hit := findNodeID(n, id); hit != nil {
			return hit
		}
	}
	return nil
}
func safeEmbeddedText(node *Node, maxLen int) string {
	var parts []string
	n := 0
	var walk func(*Node)
	walk = func(x *Node) {
		if n >= maxLen || x.Tag == "script" || x.Tag == "style" || x.Tag == "input" || x.Tag == "svg" || x.Tag == "template" {
			return
		}
		if x.Tag == "#text" {
			t := norm(x.Text)
			if t != "" {
				if len(t) > maxLen-n {
					t = t[:maxLen-n]
				}
				parts = append(parts, t)
				n += len(t)
			}
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	walk(node)
	return norm(strings.Join(parts, " "))
}

// httpMetadata describes declared HTTP actions independently of the language
// generating the HTML (PHP, Python, Go, Java, etc.). We do not infer server code.
// An arbitrary data-url is only a candidate; no method can be assumed.
func httpMetadata(n *Node) (method, target, source, confidence string) {
	for _, pair := range []struct{ attr, method string }{
		{"hx-get", "GET"}, {"hx-post", "POST"}, {"hx-put", "PUT"},
		{"hx-patch", "PATCH"}, {"hx-delete", "DELETE"},
	} {
		if target = n.prop(pair.attr); target != "" {
			return pair.method, target, "html:" + pair.attr, "declared"
		}
	}
	if target = n.prop("formaction"); target != "" {
		method = strings.ToUpper(n.prop("formmethod"))
		if method == "" {
			// A submit button inherits the surrounding form's method.
			// Outside a form, a `form` attribute may refer to another element;
			// do not pretend to know that association without resolving it.
			if n.prop("form") != "" {
				return "UNKNOWN", target, "html:formaction", "unknown"
			}
			method = "GET"
			for p := n.Parent; p != nil; p = p.Parent {
				if p.Tag == "form" {
					if strings.EqualFold(p.prop("method"), "POST") {
						method = "POST"
					}
					break
				}
			}
		}
		if method != "GET" && method != "POST" {
			return "UNKNOWN", target, "html:formaction", "unknown"
		}
		return method, target, "html:formaction", "declared"
	}
	if target = n.prop("data-url"); target != "" {
		method = "UNKNOWN"
		if m := strings.ToUpper(n.prop("data-method")); m == "GET" || m == "POST" || m == "PUT" || m == "PATCH" || m == "DELETE" {
			method = m
		}
		return method, target, "html:data-url", "inferred"
	}
	return "", "", "", ""
}

func (b *Browser) Discover() []Action {
	if b.Document == nil || b.Current == nil {
		return []Action{}
	}
	p := b.Current
	d := b.Document
	base, _ := url.Parse(d.URL)
	var actions []Action
	seen := map[string]bool{}
	add := func(a Action) {
		if len(actions) >= maxActions || a.Label == "" {
			return
		}
		key := a.Kind + "|" + a.Label + "|" + a.Method + "|" + a.URL
		if seen[key] {
			return
		}
		seen[key] = true
		a.Ref = "a" + strconv.Itoa(len(actions)+1)
		actions = append(actions, a)
	}
	for _, f := range p.Forms {
		fields := make([]ActionField, 0, len(f.Fields))
		for _, v := range f.Fields {
			field := ActionField{Name: v.Name, Type: v.Type}
			if v.Type != "hidden" && v.Type != "password" {
				field.Value = v.Value
			}
			fields = append(fields, field)
		}
		add(Action{Label: f.SubmitLabel, Kind: "form", Method: f.Method, URL: f.Action, Source: "html:form#" + f.Ref, Evidence: "form action/method", Confidence: "declared", Executable: true, RequiresConfirmation: f.Method != "GET", Fields: fields})
	}
	script, externals := collectScriptText(d)
	script += "\n" + b.externalScripts(externals, d.URL)
	script = stripJSComments(script)
	handlers := scriptHandlers(script)
	associated := map[string]bool{}
	for _, button := range p.Buttons {
		node := d.buttons[button.Ref]
		if node == nil {
			continue
		}
		// aria-controls/data-target identifies content already in the HTML.
		controlled := strings.TrimPrefix(strings.TrimSpace(node.prop("data-target")), "#")
		if controlled == "" {
			controlled = strings.TrimSpace(node.prop("aria-controls"))
		}
		if controlled != "" {
			if hiddenNode := findNodeID(d.Root, controlled); hiddenNode != nil {
				if preview := safeEmbeddedText(hiddenNode, 1800); preview != "" {
					add(Action{Label: button.Label, Kind: "embedded_data", Method: "NONE", Source: "html:aria-controls", Evidence: "associated existing DOM subtree #" + controlled, Confidence: "declared", Preview: preview, Executable: false})
				}
			}
		}
		// Declarative endpoints are protocol-level behavior, not JS-specific.
		method, target, source, confidence := httpMetadata(node)
		if target != "" {
			u := resolve(base, target)
			if u != "" {
				add(Action{Label: button.Label, Kind: "button", Method: method, URL: u, Source: source, Evidence: source + " attribute", Confidence: confidence, Executable: method == "GET" && source != "html:data-url" && source != "html:formaction", RequiresConfirmation: true})
			}
		}
		code := node.prop("onclick")
		if m := callNamePattern.FindStringSubmatch(code); len(m) == 2 {
			code += "\n" + getFunctionBody(script, m[1])
		}
		calls := append([]jsCall{}, extractJSCalls(code)...)
		if node.prop("id") != "" {
			calls = append(calls, handlers[node.prop("id")]...)
		}
		for _, call := range calls {
			u := resolve(base, html.UnescapeString(call.target))
			if u == "" {
				continue
			}
			associated[call.method+"|"+u] = true
			add(Action{Label: button.Label, Kind: "api", Method: call.method, URL: u, Source: "javascript:button#" + button.Ref, Evidence: call.evidence + " in click handler", Confidence: "inferred", Executable: call.method == "GET", RequiresConfirmation: true})
		}
	}
	for _, link := range p.Links {
		add(Action{Label: link.Label, Kind: "link", Method: "GET", URL: link.URL, Source: "html:a#" + link.Ref, Evidence: "href attribute", Confidence: "declared", Executable: true})
	}
	// htmx and other protocol declarations may be attached to arbitrary
	// elements, such as <div hx-get="/news">, not only to buttons.
	// This is particularly useful for server-rendered PHP/Python/Go apps.
	domWalk(d.Root, false, func(n *Node) {
		if n.Tag == "button" || (n.Tag == "input" && n.prop("type") == "button") {
			return // already handled above
		}
		method, target, source, confidence := httpMetadata(n)
		if target == "" {
			return
		}
		// formaction applies to submit controls; do not misattribute it.
		if source == "html:formaction" && n.Tag != "input" {
			return
		}
		u := resolve(base, target)
		if u == "" {
			return
		}
		label := n.prop("aria-label")
		if label == "" {
			label = n.innerText(120)
		}
		if label == "" {
			label = n.prop("value")
		}
		if label == "" {
			label = n.Tag + " action"
		}
		add(Action{Label: label, Kind: "http_action", Method: method, URL: u,
			Source: source, Evidence: source + " on <" + n.Tag + ">", Confidence: confidence,
			Executable: method == "GET" && source != "html:data-url" && source != "html:formaction", RequiresConfirmation: true})
	})
	// Hypermedia / alternate representations can be advertised in HTML head.
	// These expose data without guessing internal routes or backend language.
	var visitHead func(*Node)
	visitHead = func(n *Node) {
		if n.Tag == "link" && (strings.EqualFold(n.prop("rel"), "alternate") || strings.EqualFold(n.prop("rel"), "service-desc")) {
			if mime := strings.ToLower(n.prop("type")); strings.Contains(mime, "json") {
				if u := resolve(base, n.prop("href")); u != "" {
					label := n.prop("title")
					if label == "" {
						label = "Structured data (" + n.prop("rel") + ")"
					}
					add(Action{Label: label, Kind: "hypermedia", Method: "GET", URL: u,
						Source: "html:link", Evidence: "rel=" + n.prop("rel") + " type=" + mime,
						Confidence: "declared", Executable: true, RequiresConfirmation: true})
				}
			}
		}
		for _, ch := range n.Children {
			visitHead(ch)
		}
	}
	visitHead(d.Root)
	// Calls not reliably associated with a particular button are shown as
	// candidates; do not assert that any specific click triggers them.
	for _, call := range extractJSCalls(script) {
		u := resolve(base, call.target)
		if u == "" || associated[call.method+"|"+u] {
			continue
		}
		add(Action{Label: "Unassociated API call", Kind: "api", Method: call.method, URL: u, Source: "javascript:script", Evidence: call.evidence + "; button association unknown", Confidence: "inferred", Executable: call.method == "GET", RequiresConfirmation: true})
	}
	return actions
}

func actionByRef(actions []Action, ref string) (Action, bool) {
	for _, a := range actions {
		if a.Ref == ref {
			return a, true
		}
	}
	return Action{}, false
}

func (b *Browser) runAction(a Action, confirm bool) (*Resource, error) {
	if !a.Executable {
		return nil, errors.New("action has unknown or unsupported HTTP method")
	}
	if a.RequiresConfirmation && !confirm {
		return nil, errors.New("inferred or state-changing action requires confirm:true")
	}
	if a.Method == "POST" && (!b.AllowPost || !confirm) {
		return nil, errors.New("POST requires -allow-post and confirm:true")
	}
	u, err := url.Parse(a.URL)
	if err != nil {
		return nil, err
	}
	if err = b.validate(u); err != nil {
		return nil, err
	}
	var params url.Values
	if a.Kind == "form" {
		for _, form := range b.Current.Forms {
			if "html:form#"+form.Ref != a.Source {
				continue
			}
			params = url.Values{}
			for _, f := range form.Fields {
				if f.Name == "" || f.Type == "file" || f.Type == "submit" || f.Type == "button" {
					continue
				}
				if (f.Type == "radio" || f.Type == "checkbox") && f.Value == "" {
					continue
				}
				params.Add(f.Name, f.Value)
			}
			break
		}
	}
	// Semantic navigation for HTML links/forms, including normal cookies.
	if a.Kind == "link" || a.Kind == "form" {
		if err = b.request(a.Method, a.URL, params, true); err != nil {
			return nil, err
		}
		return nil, nil
	}
	// Other requests return JSON/text data rather than attempting to render.
	// Do not infer request bodies, authentication parameters or headers.
	if a.Method == "POST" {
		return nil, errors.New("inferred POST body not known; cannot safely reconstruct request")
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, text/html;q=0.8")
	req.Header.Set("User-Agent", "AIGoBrowser/0.5 (agent action resolver)")
	res, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	content, err := io.ReadAll(io.LimitReader(res.Body, maxResourceBytes+1))
	if err != nil {
		return nil, err
	}
	out := &Resource{URL: res.Request.URL.String(), Status: res.StatusCode, ContentType: res.Header.Get("Content-Type"), Truncated: len(content) > maxResourceBytes}
	if out.Truncated {
		content = content[:maxResourceBytes]
	}
	ct := strings.ToLower(res.Header.Get("Content-Type"))
	if strings.Contains(ct, "json") {
		var val any
		if !out.Truncated && json.Unmarshal(content, &val) == nil {
			out.Data = redactJSON(val, 0)
		} else {
			out.Body = "[JSON too large or invalid to parse safely]"
		}
	} else if strings.Contains(ct, "text/html") {
		pg := Parse(content, out.URL, res.StatusCode, len(content), 5000)
		out.Body = fmt.Sprintf("title: %s\ntext: %s", pg.Title, pg.Text)
	} else if strings.HasPrefix(ct, "text/") || ct == "" {
		out.Body = string(content)
	} else {
		out.Body = "[binary response omitted]"
	}
	return out, nil
}

var sensitiveKey = regexp.MustCompile(`(?i)(password|passwd|secret|token|api.?key|authorization|cookie|credential|session.?id|private.?key)`)

func redactJSON(v any, depth int) any {
	if depth > 30 {
		return "[depth limit]"
	}
	switch x := v.(type) {
	case map[string]any:
		dst := make(map[string]any, len(x))
		for k, val := range x {
			if sensitiveKey.MatchString(k) {
				dst[k] = "[redacted]"
			} else {
				dst[k] = redactJSON(val, depth+1)
			}
		}
		return dst
	case []any:
		dst := make([]any, len(x))
		for i, val := range x {
			dst[i] = redactJSON(val, depth+1)
		}
		return dst
	default:
		return v
	}
}
