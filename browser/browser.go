package browser

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const MaxBody = 2 << 20

type Command struct {
	Cmd      string `json:"cmd"`
	URL      string `json:"url,omitempty"`
	Ref      string `json:"ref,omitempty"`
	Selector string `json:"selector,omitempty"`
	Value    string `json:"value,omitempty"`
	Confirm  bool   `json:"confirm,omitempty"`
}
type Memory struct {
	RSSMiB  float64 `json:"rss_mib"`
	HeapMiB float64 `json:"heap_mib"`
}
type Result struct {
	OK         bool      `json:"ok"`
	Error      string    `json:"error,omitempty"`
	Help       []string  `json:"help,omitempty"`
	Page       *Page     `json:"page,omitempty"`
	Matches    []Element `json:"matches,omitempty"`
	Actions    []Action  `json:"actions,omitempty"`
	Action     *Action   `json:"action,omitempty"`
	Resource   *Resource `json:"resource,omitempty"`
	Memory     Memory    `json:"memory"`
	DurationMs int64     `json:"duration_ms"`
	JS         *JSStats  `json:"js,omitempty"`
}

type Browser struct {
	client            *http.Client
	Current           *Page
	Document          *Document
	History           []string
	Allowed           map[string]bool
	AllowPost         bool
	MaxText           int
	ScanScripts       bool
	scannedScripts    bool
	scannedScriptText string
	JSWorker          string
	JSTimeout         time.Duration
	lastJS            *JSStats
	jsSession         *jsSession
	rawHTML           []byte
}

// Close terminates the page-scoped JavaScript worker, if running.
func (b *Browser) Close() { b.closeJSSession() }

func New(allowed []string, allowPost bool) *Browser {
	jar, _ := cookiejar.New(nil)
	b := &Browser{Allowed: map[string]bool{}, AllowPost: allowPost, MaxText: 5000}
	for _, h := range allowed {
		if h = strings.ToLower(strings.TrimSpace(h)); h != "" {
			b.Allowed[h] = true
		}
	}
	b.client = &http.Client{Timeout: 10 * time.Second, Jar: jar}
	b.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 8 {
			return errors.New("many redirects")
		}
		return b.validate(req.URL)
	}
	return b
}
func (b *Browser) validate(u *url.URL) error {
	if u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return errors.New("only http(s) URLs are allowed")
	}
	if u.User != nil {
		return errors.New("credentials in URL not allowed")
	}
	// Fail closed by default: an AI agent must explicitly opt in to network hosts.
	// This is NOT a complete SSRF defense (DNS rebinding/private address
	// resolution require controls at the network or container boundary).
	if len(b.Allowed) == 0 {
		return errors.New("network disabled: set -allow-hosts to authorize destinations")
	}
	if !b.Allowed[strings.ToLower(u.Hostname())] {
		return fmt.Errorf("host %s not on allowlist", u.Hostname())
	}
	return nil
}
func (b *Browser) request(method, target string, params url.Values, saveHistory bool) error {
	u, err := url.Parse(target)
	if err != nil {
		return err
	}
	if err = b.validate(u); err != nil {
		return err
	}
	var body io.Reader
	if method == "GET" && params != nil {
		q := u.Query()
		for k, values := range params {
			for _, v := range values {
				q.Add(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}
	if method == "POST" {
		body = strings.NewReader(params.Encode())
	}
	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "AIGoBrowser/0.1 (+headless-agent; HTML-only)")
	if method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	res, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if ct := res.Header.Get("Content-Type"); ct != "" && !strings.Contains(strings.ToLower(ct), "text/html") && !strings.Contains(strings.ToLower(ct), "application/xhtml+xml") {
		return fmt.Errorf("unsupported content type: %s", ct)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, MaxBody+1))
	if err != nil {
		return err
	}
	if len(data) > MaxBody {
		return fmt.Errorf("response exceeds %d bytes", MaxBody)
	}
	// Execute explicitly enabled inline JavaScript before extracting the agent snapshot.
	originalLen := len(data)
	data, b.lastJS = b.runInlineJS(data, res.Request.URL.String())
	d, err := ParseDocument(data, res.Request.URL.String(), res.StatusCode, originalLen)
	if err != nil {
		return err
	}
	p := d.Page(b.MaxText)
	b.rawHTML = append([]byte(nil), data...)
	b.scannedScripts = false
	b.scannedScriptText = ""
	b.Document = d
	if saveHistory && b.Current != nil {
		b.History = append(b.History, b.Current.URL)
	}
	b.Current = &p
	return nil
}
func (b *Browser) Execute(c Command) Result {
	start := time.Now()
	r := Result{}
	var err error
	switch strings.ToLower(strings.TrimSpace(c.Cmd)) {
	case "open":
		err = b.request("GET", c.URL, nil, true)
	case "discover":
		if b.Document == nil {
			err = errors.New("no page open")
			break
		}
		r.Actions = b.Discover()
	case "inspect":
		if b.Document == nil {
			err = errors.New("no page open")
			break
		}
		actions := b.Discover()
		a, found := actionByRef(actions, c.Ref)
		if !found {
			err = fmt.Errorf("action %q not found", c.Ref)
		} else {
			r.Action = &a
		}
	case "run":
		if b.Document == nil {
			err = errors.New("no page open")
			break
		}
		actions := b.Discover()
		a, found := actionByRef(actions, c.Ref)
		if !found {
			err = fmt.Errorf("action %q not found", c.Ref)
		} else {
			r.Resource, err = b.runAction(a, c.Confirm)
		}
	case "snapshot":
		if b.Current == nil {
			err = errors.New("no page open")
		}
	case "click":
		if b.Current == nil {
			err = errors.New("no page open")
			break
		}
		found := false
		for _, l := range b.Current.Links {
			if l.Ref == c.Ref {
				found = true
				err = b.request("GET", l.URL, nil, true)
				break
			}
		}
		if !found {
			err = fmt.Errorf("link %q not found", c.Ref)
		}
	case "press":
		err = b.dispatchButton(c.Ref)
	case "fill":
		if b.Current == nil {
			err = errors.New("no page open")
			break
		}
		found := false
		for i := range b.Current.Forms {
			for j := range b.Current.Forms[i].Fields {
				f := &b.Current.Forms[i].Fields[j]
				if f.Ref == c.Ref {
					found = true
					if f.Type == "hidden" || f.Type == "submit" || f.Type == "file" {
						err = errors.New("field cannot be edited")
					} else {
						f.Value = c.Value
						if b.Document != nil {
							if n := b.Document.fields[f.Ref]; n != nil {
								n.setValue(c.Value)
								if id := n.prop("id"); id != "" && b.jsSession != nil {
									_, syncErr := b.jsSession.exchange(map[string]string{"cmd": "setvalue", "id": id, "value": c.Value}, b.JSTimeout)
									if syncErr != nil {
										b.closeJSSession()
										err = fmt.Errorf("JS input sync failed: %w", syncErr)
									}
								}
							}
						}
					}
					break
				}
			}
		}
		if !found {
			err = fmt.Errorf("field %q not found", c.Ref)
		}
		if err == nil && b.Document != nil {
			p := b.Document.Page(b.MaxText)
			b.Current = &p
		}
	case "query":
		if b.Document == nil {
			err = errors.New("no page open")
			break
		}
		r.Matches, err = b.Document.Query(c.Selector)
	case "submit":
		if b.Current == nil {
			err = errors.New("no page open")
			break
		}
		found := false
		for _, form := range b.Current.Forms {
			if form.Ref != c.Ref {
				continue
			}
			found = true
			if form.Method == "POST" && (!b.AllowPost || !c.Confirm) {
				err = errors.New("POST requires -allow-post and confirm:true")
				break
			}
			vals := url.Values{}
			for _, f := range form.Fields {
				if f.Name == "" || f.Type == "file" || f.Type == "button" || f.Type == "submit" {
					continue
				}
				if (f.Type == "checkbox" || f.Type == "radio") && f.Value == "" {
					continue
				}
				vals.Add(f.Name, f.Value)
			}
			err = b.request(form.Method, form.Action, vals, true)
			break
		}
		if !found {
			err = fmt.Errorf("form %q not found", c.Ref)
		}
	case "back":
		if len(b.History) == 0 {
			err = errors.New("no history")
			break
		}
		last := b.History[len(b.History)-1]
		b.History = b.History[:len(b.History)-1]
		err = b.request("GET", last, nil, false)
	case "help":
		r.Help = []string{`{"cmd":"open","url":"http://..."}`, `{"cmd":"snapshot"}`, `{"cmd":"discover"}`, `{"cmd":"inspect","ref":"a1"}`, `{"cmd":"run","ref":"a1","confirm":true}`, `{"cmd":"query","selector":"form input[name=q]"}`, `{"cmd":"click","ref":"l1"}`, `{"cmd":"press","ref":"b1"}`, `{"cmd":"fill","ref":"i1","value":"value"}`, `{"cmd":"submit","ref":"f1","confirm":true}`, `{"cmd":"back"}`, `{"cmd":"quit"}`}
	case "quit":
		b.closeJSSession()
	default:
		err = fmt.Errorf("unknown command %q", c.Cmd)
	}
	r.OK = err == nil
	if err != nil {
		r.Error = err.Error()
	}
	// Query is deliberately compact: avoid repeating the entire page snapshot
	// for every agent lookup, which can dominate LLM input tokens.
	if b.Current != nil && strings.ToLower(strings.TrimSpace(c.Cmd)) != "query" && strings.ToLower(strings.TrimSpace(c.Cmd)) != "discover" && strings.ToLower(strings.TrimSpace(c.Cmd)) != "inspect" && !(strings.ToLower(strings.TrimSpace(c.Cmd)) == "run" && r.Resource != nil) {
		r.Page = safePage(b.Current)
	}
	r.JS = b.lastJS
	r.Memory = mem()
	r.DurationMs = time.Since(start).Milliseconds()
	return r
}
func mem() Memory {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	out := Memory{HeapMiB: float64(m.HeapAlloc) / 1048576}
	data, err := os.ReadFile("/proc/self/status")
	if err == nil {
		for _, line := range bytes.Split(data, []byte("\n")) {
			if bytes.HasPrefix(line, []byte("VmRSS:")) {
				fields := strings.Fields(string(line))
				if len(fields) > 1 {
					n, _ := strconv.ParseFloat(fields[1], 64)
					out.RSSMiB = n / 1024
				}
				break
			}
		}
	}
	return out
}
func Decode(line []byte) (Command, error) {
	var c Command
	err := json.Unmarshal(line, &c)
	return c, err
}

// safePage prevents browser-agent logs from displaying credentials and hidden tokens.
// Actual values remain in the browser session and are used when submitting forms.
func safePage(p *Page) *Page {
	if p == nil {
		return nil
	}
	copyPage := *p
	copyPage.Forms = make([]Form, len(p.Forms))
	for i, form := range p.Forms {
		copyPage.Forms[i] = form
		copyPage.Forms[i].Fields = append([]Field(nil), form.Fields...)
		for j := range copyPage.Forms[i].Fields {
			field := &copyPage.Forms[i].Fields[j]
			if (field.Type == "password" || field.Type == "hidden") && field.Value != "" {
				field.Value = "[redacted]"
			}
		}
	}
	return &copyPage
}
