package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"runtime/metrics"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxBody = 5 << 20

type Command struct {
	Cmd       string `json:"cmd"`
	URL       string `json:"url,omitempty"`
	Ref       string `json:"ref,omitempty"`
	Selector  string `json:"selector,omitempty"`
	Value     string `json:"value,omitempty"`
	Confirm   bool   `json:"confirm,omitempty"`
	RenderJS  bool   `json:"js,omitempty"`
	Checked   *bool  `json:"checked,omitempty"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
	// View selects an observation shape for snapshot/text. "read" returns only
	// url, title, visible text, status and engine metadata, omitting every
	// reference list. In a Chromium session that skips transferring and
	// re-parsing the DOM, which dominates polling while a page settles. An
	// empty view keeps the full observation with references.
	View string `json:"view,omitempty"`
}
type Memory struct {
	RSSMiB    float64 `json:"rss_mib"`
	HeapMiB   float64 `json:"heap_mib"`
	SysMiB    float64 `json:"go_sys_mib"`
	RSSSource string  `json:"rss_source"`
}
type Result struct {
	OK         bool         `json:"ok"`
	Error      string       `json:"error,omitempty"`
	Help       []string     `json:"help,omitempty"`
	Page       *Page        `json:"page,omitempty"`
	Matches    []Element    `json:"matches,omitempty"`
	Frames     []Frame      `json:"frames,omitempty"`
	Actions    []Action     `json:"actions,omitempty"`
	Action     *Action      `json:"action,omitempty"`
	Resource   *Resource    `json:"resource,omitempty"`
	Memory     Memory       `json:"memory"`
	DurationMs int64        `json:"duration_ms"`
	JS         *JSStats     `json:"js,omitempty"`
	Engine     string       `json:"engine,omitempty"`
	Warnings   []string     `json:"warnings,omitempty"`
	Ticket     *TicketGrant `json:"ticket,omitempty"`
}

type Browser struct {
	client    *http.Client
	Current   *Page
	Document  *Document
	History   []string
	Allowed   map[string]bool
	AllowAll  bool
	AllowPost bool
	// NativeOnly enforces strict 100% Go execution: never spawns external Chromium/Edge/Firefox processes.
	NativeOnly bool
	// BlockPrivateNetworks prevents requests to internal RFC1918 IPs, loopback, and cloud metadata (SSRF defense).
	BlockPrivateNetworks bool
	MaxText   int
	// MaxHistory bounds retained URLs. Zero disables history; a negative value is unlimited.
	MaxHistory int
	// CompactSnapshots omits automatic Markdown generation and caps observation
	// lists. The full DOM and explicit markdown command remain available.
	CompactSnapshots bool
	// Engine selects auto (HTTP first), http, or chromium. An active live
	// session remains in Chromium until release or Close is called.
	Engine            string
	OperationTimeout  time.Duration
	UserAgent         string
	AutoRenderJS      bool
	SystemBrowser     string
	ScanScripts       bool
	scannedScripts    bool
	scannedScriptText string
	JSWorker          string
	JSTimeout         time.Duration
	lastJS            *JSStats
	jsSession         *jsSession
	rawHTML           []byte
	live              *liveSession
	resource          *Resource
	// readPage carries a view:"read" observation for the current command. It
	// is published instead of Current and cleared immediately, so a read never
	// replaces the reference-bearing snapshot the agent acts on.
	readPage *Page
	// Relay is the ticket-relay subsystem (default on when configured): the
	// /api/ticket receiver, the workflow dispatch, and the captcha gate loop.
	Relay *TicketRelay
}

// Close releases page data, idle HTTP connections and the JavaScript worker.
// It is safe to call more than once. The browser can be used again after Close.
func (b *Browser) Close() {
	b.closeLive()
	b.closeJSSession()
	if b.client != nil {
		b.client.CloseIdleConnections()
	}
	b.Current = nil
	b.Document = nil
	b.History = nil
	b.rawHTML = nil
	b.scannedScripts = false
	b.scannedScriptText = ""
	b.lastJS = nil
	b.resource = nil
	b.readPage = nil
}

func (b *Browser) rememberCurrent() {
	if b.MaxHistory == 0 {
		b.History = nil
		return
	}
	if b.Current == nil {
		return
	}
	if b.MaxHistory > 0 && len(b.History) >= b.MaxHistory {
		// Reuse the backing array and clear discarded URLs, including when the
		// caller lowers the limit after the browser has already been used.
		keep := b.MaxHistory - 1
		copy(b.History, b.History[len(b.History)-keep:])
		clear(b.History[keep:])
		b.History = b.History[:keep]
	}
	b.History = append(b.History, b.Current.URL)
}

func New(allowed []string, allowPost bool) *Browser {
	jar, _ := cookiejar.New(nil)
	b := &Browser{
		Allowed:          map[string]bool{},
		AllowPost:        allowPost,
		MaxText:          5000,
		MaxHistory:       100,
		Engine:           "auto",
		OperationTimeout: 20 * time.Second,
		NativeOnly:       os.Getenv("AI_GO_BROWSER_NATIVE_ONLY") == "1",
	}
	for _, h := range allowed {
		if h = strings.ToLower(strings.TrimSpace(h)); h != "" {
			b.Allowed[h] = true
			if h == "*" {
				b.AllowAll = true
			}
		}
	}
	tr := NewSafeTransport(false)
	b.client = &http.Client{Timeout: 20 * time.Second, Jar: jar, Transport: tr}
	b.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 15 {
			return errors.New("many redirects")
		}
		return b.validate(req.URL)
	}
	return b
}

// NewNative creates a Browser configured for strict 100% Go execution (no external browsers) with SSRF defense.
func NewNative(allowed []string, allowPost bool) *Browser {
	b := New(allowed, allowPost)
	b.NativeOnly = true
	b.SetBlockPrivateNetworks(true)
	return b
}

// SetBlockPrivateNetworks toggles SSRF defense to block RFC1918 / loopback / cloud metadata.
func (b *Browser) SetBlockPrivateNetworks(block bool) {
	b.BlockPrivateNetworks = block
	jar := b.client.Jar
	b.client.Transport = NewSafeTransport(block)
	b.client.Jar = jar
}

func (b *Browser) validate(u *url.URL) error {
	if u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return errors.New("only http(s) URLs are allowed")
	}
	if u.User != nil {
		return errors.New("credentials in URL not allowed")
	}
	if b.BlockPrivateNetworks {
		host := strings.ToLower(u.Hostname())
		if host == "localhost" || strings.HasSuffix(host, ".localhost") || host == "metadata.google.internal" {
			return fmt.Errorf("%w: %s", ErrSSRFBlocked, host)
		}
		if ip := net.ParseIP(host); ip != nil {
			if isPrivateIP(ip) {
				return fmt.Errorf("%w: %s", ErrSSRFBlocked, host)
			}
		}
	}
	if b.AllowAll || b.Allowed["*"] {
		return nil
	}
	// Fail closed by default: an AI agent must explicitly opt in to network hosts.
	// This is NOT a complete SSRF defense (DNS rebinding/private address
	// resolution require controls at the network or container boundary).
	if len(b.Allowed) == 0 {
		return errors.New("network disabled: set -allow-hosts to authorize destinations")
	}
	host := strings.ToLower(u.Hostname())
	if b.Allowed[host] {
		return nil
	}
	for allowed := range b.Allowed {
		if strings.HasPrefix(allowed, "*.") && strings.HasSuffix(host, allowed[1:]) {
			return nil
		}
		if strings.HasSuffix(host, "."+allowed) {
			return nil
		}
	}
	return fmt.Errorf("host %s not on allowlist", u.Hostname())
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
		// HTML GET submission replaces the action URL's query component.
		u.RawQuery = params.Encode()
	}
	if method == "POST" {
		body = strings.NewReader(params.Encode())
	}
	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return err
	}
	ua := b.UserAgent
	if ua == "" {
		ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,pt-BR;q=0.8,pt;q=0.7")
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	if method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	res, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	contentType := res.Header.Get("Content-Type")
	media := responseMediaType(contentType)
	limit := MaxBody
	if media != "" && !htmlMediaType(media) {
		limit = maxResourceBytes
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, int64(limit)+1))
	if err != nil {
		return err
	}
	if media == "" {
		media = sniffResponseMediaType(data)
	}
	if !htmlMediaType(media) {
		truncated := len(data) > maxResourceBytes
		count := len(data)
		if truncated {
			data = data[:maxResourceBytes]
		}
		out := resourceSnapshot(data, res.Request.URL.String(), res.StatusCode, contentType, truncated)
		b.installResource(out, count, saveHistory)
		return nil
	}
	if len(data) > MaxBody {
		return fmt.Errorf("response exceeds %d bytes", MaxBody)
	}
	// Keep wire bytes separate from the bounded UTF-8 representation.
	originalLen := len(data)
	encoding, data, err := DecodeHTTPBody(contentType, data)
	if err != nil {
		return err
	}

	// Execute explicitly enabled inline JavaScript before extracting the agent snapshot.
	data, b.lastJS = b.runInlineJS(data, res.Request.URL.String())
	d, err := ParseDocument(data, res.Request.URL.String(), res.StatusCode, originalLen)
	if err != nil {
		return err
	}
	p := d.PageWithMarkdown(b.MaxText, !b.CompactSnapshots)
	raw := string(data)
	p.Stack = AnalyzeTechStack(raw, utf8.RuneCountInString(p.Text), encoding)
	structData := ExtractStructuredData(raw)
	for k, v := range structData {
		p.Metadata[k] = v
	}
	// Only an active JS worker needs source HTML for subsequent event patches.
	// The response buffer is owned by this navigation, so no clone is needed.
	b.rawHTML = nil
	if b.jsSession != nil {
		b.rawHTML = data
	}
	b.scannedScripts = false
	b.scannedScriptText = ""
	b.closeLive()
	b.resource = nil
	b.Document = d
	if saveHistory {
		b.rememberCurrent()
	}
	b.Current = &p
	return nil
}

// RenderPageWithJS opens a persistent Chromium page, including later JS interactions.
func (b *Browser) RenderPageWithJS(targetURL string) error {
	if b.NativeOnly {
		return ErrExternalBrowserDisabled
	}
	return b.openLive(targetURL, true)
}

// relayContext budgets one full relay loop: workflow dispatch, runner push
// (TokenWait) and the two completion round trips. The caller cancels.
func (b *Browser) relayContext() (context.Context, context.CancelFunc) {
	budget := b.Relay.Config().TokenWait + 2*b.Relay.Config().HTTPTimeout + 30*time.Second
	return context.WithTimeout(context.Background(), budget)
}

func (b *Browser) Execute(c Command) Result {
	start := time.Now()
	r := Result{}
	var err error
	cmd := strings.ToLower(strings.TrimSpace(c.Cmd))
	c.Cmd = cmd
	handled := false
	if b.live != nil && cmd != "open" && cmd != "render" && cmd != "query" && cmd != "wait" && cmd != "release" && cmd != "quit" && cmd != "help" {
		if (cmd == "snapshot" || cmd == "text") && strings.EqualFold(c.View, "read") {
			handled, err = b.executeLiveRead(c)
		} else {
			handled, err = b.executeLive(c)
		}
	}
	if !handled && err == nil {
		switch cmd {
		case "open":
			if c.RenderJS || b.Engine == "chromium" || b.live != nil {
				if b.NativeOnly {
					err = ErrExternalBrowserDisabled
				} else {
					err = b.RenderPageWithJS(c.URL)
				}
			} else {
				err = b.request("GET", c.URL, nil, true)
				if err == nil && !b.NativeOnly && b.Engine != "http" && b.AutoRenderJS && b.Current != nil && strings.HasPrefix(b.Current.Stack.RenderMode, "CSR") && utf8.RuneCountInString(b.Current.Text) < 100 {
					// HTTP navigation already recorded the old page. Rendering the
					// same destination must not duplicate it in history.
					if renderErr := b.openLive(b.Current.URL, false); renderErr != nil {
						r.Warnings = append(r.Warnings, "JavaScript fallback failed: "+renderErr.Error())
					}
				}
			}
		case "render":
			target := c.URL
			if target == "" && b.Current != nil {
				target = b.Current.URL
			}
			if target == "" {
				err = errors.New("no URL to render")
			} else {
				err = b.RenderPageWithJS(target)
			}
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
			err = b.pressControl(c)
		case "fill":
			err = b.fillControl(c.Ref, c.Value)
		case "check":
			checked := true
			if c.Checked != nil {
				checked = *c.Checked
			}
			err = b.checkControl(c.Ref, checked)
		case "query":
			if b.live != nil {
				r.Matches, err = b.liveQuery(c.Selector)
				break
			}
			if b.Document == nil {
				err = errors.New("no page open")
				break
			}
			r.Matches, err = b.Document.Query(c.Selector)
		case "frames":
			r.Frames, err = b.liveFrames()
		case "frame":
			err = b.selectLiveFrame(c.Ref)
		case "submit":
			err = b.submitForm(c.Ref, "", c.Confirm)
		case "back":
			if len(b.History) == 0 {
				err = errors.New("no history")
				break
			}
			last := b.History[len(b.History)-1]
			err = b.request("GET", last, nil, false)
			if err == nil {
				b.History[len(b.History)-1] = ""
				b.History = b.History[:len(b.History)-1]
			}
		case "wait":
			if c.Selector == "" {
				err = errors.New("wait requires selector")
				break
			}
			if b.live != nil {
				timeout := b.OperationTimeout
				if c.TimeoutMs > 0 {
					timeout = time.Duration(c.TimeoutMs) * time.Millisecond
				}
				err = b.liveWait(c.Selector, timeout)
			} else if b.Document == nil {
				err = errors.New("no page open")
			} else {
				var matches []Element
				matches, err = b.Document.Query(c.Selector)
				if err == nil && len(matches) == 0 {
					err = errors.New("selector not found in static page; use render for dynamic DOM")
				}
			}
		case "release":
			b.closeLive()
		case "markdown":
			if b.Current == nil {
				err = errors.New("no page open")
			}
		case "text":
			if b.Current == nil {
				err = errors.New("no page open")
			}
		case "help":
			r.Help = []string{`{"cmd":"open","url":"http://..."}`, `{"cmd":"snapshot"}`, `{"cmd":"markdown"}`, `{"cmd":"text"}`, `{"cmd":"discover"}`, `{"cmd":"inspect","ref":"a1"}`, `{"cmd":"run","ref":"a1","confirm":true}`, `{"cmd":"query","selector":"form input[name=q]"}`, `{"cmd":"click","ref":"l1"}`, `{"cmd":"press","ref":"b1"}`, `{"cmd":"fill","ref":"i1","value":"value"}`, `{"cmd":"submit","ref":"f1","confirm":true}`, `{"cmd":"back"}`, `{"cmd":"quit"}`}
			r.Help = append(r.Help, `{"cmd":"open","url":"http://...","js":true}`, `{"cmd":"check","ref":"i1","checked":true}`, `{"cmd":"wait","selector":"#ready","timeout_ms":5000}`, `{"cmd":"frames"}`, `{"cmd":"frame","ref":"fr2"}`, `{"cmd":"release"}`)
			// A read is for waiting on visible text: it skips the document
			// transfer and returns no references, so it cannot be acted on.
			r.Help = append(r.Help, `{"cmd":"snapshot","view":"read"}`, `{"cmd":"text","view":"read"}`)
			// The relay runs without configuration until TICKET_RELAY_* env is
			// set; ticket then dispatches the runner and returns the grant.
			r.Help = append(r.Help, `{"cmd":"ticket"}`)
		case "quit":
			b.Close()
		case "ticket":
			if b.Relay == nil || !b.Relay.Configured() {
				err = errors.New("ticket relay não configurado (TICKET_RELAY_GAME, TIBIDLE_SESSION, TICKET_RELAY_TUNNEL, TICKET_RELAY_REPO)")
				break
			}
			tctx, cancel := b.relayContext()
			grant, gerr := b.Relay.Resolve(tctx)
			cancel()
			if gerr != nil {
				err = gerr
			} else {
				r.Ticket = grant
			}
		default:
			err = fmt.Errorf("unknown command %q", c.Cmd)
		}
	}
	r.OK = err == nil
	if err != nil {
		r.Error = err.Error()
	}
	if b.resource != nil && r.Resource == nil && cmd != "query" && cmd != "discover" && cmd != "inspect" {
		r.Resource = b.resource
	}
	// Query is deliberately compact: avoid repeating the entire page snapshot
	// for every agent lookup, which can dominate LLM input tokens.
	if b.readPage != nil {
		// A live view:"read" observation replaces the reference-bearing page
		// for this command only; it is cleared so the next command sees the
		// snapshot the agent still holds references into.
		r.Page = b.readPage
		b.readPage = nil
	} else if b.Current != nil && b.resource == nil && cmd != "query" && cmd != "frames" && cmd != "discover" && cmd != "inspect" && !(cmd == "run" && r.Resource != nil) {
		r.Page = safePage(b.Current)
		if (cmd == "snapshot" || cmd == "text") && strings.EqualFold(c.View, "read") {
			r.Page = readProjection(r.Page)
		} else if b.CompactSnapshots {
			r.Page.Markdown = ""
			if len(r.Page.Links) > 200 {
				r.Page.OmittedLinks = len(r.Page.Links) - 200
				r.Page.Links = r.Page.Links[:200]
			}
			if len(r.Page.Headings) > 100 {
				r.Page.OmittedHeadings = len(r.Page.Headings) - 100
				r.Page.Headings = r.Page.Headings[:100]
			}
		}
		if cmd == "markdown" && b.Document != nil && r.Page.Markdown == "" {
			r.Page.Markdown = b.Document.Markdown()
		}
	}
	// Autonomy box: a wall response trips the relay once per interval —
	// dispatch, wait for the runner's push, finish the ticket here.
	if err == nil && b.Relay != nil && b.Relay.Configured() &&
		cmd != "ticket" && cmd != "help" && cmd != "quit" && resultIsGate(r) {
		gctx, cancel := b.relayContext()
		grant, gerr, _ := b.Relay.TryResolve(gctx)
		cancel()
		if gerr != nil {
			r.Warnings = append(r.Warnings, "ticket-relay: "+gerr.Error())
		} else {
			r.Ticket = grant
		}
	}
	r.JS = b.lastJS
	r.Engine = "http"
	if b.live != nil {
		r.Engine = "chromium"
	} else if b.jsSession != nil {
		r.Engine = "duktape"
	}
	r.Memory = mem()
	r.DurationMs = time.Since(start).Milliseconds()
	return r
}
func mem() Memory {
	// runtime/metrics avoids the stop-the-world statistics read on each command.
	samples := [2]metrics.Sample{
		{Name: "/memory/classes/heap/objects:bytes"},
		{Name: "/memory/classes/total:bytes"},
	}
	metrics.Read(samples[:])
	sys := float64(samples[1].Value.Uint64()) / 1048576
	out := Memory{
		HeapMiB:   float64(samples[0].Value.Uint64()) / 1048576,
		SysMiB:    sys,
		RSSMiB:    sys,
		RSSSource: "go_sys_estimate",
	}
	if m := ReadProcessMemory(os.Getpid()); m.Available {
		out.RSSMiB = m.RSSMiB
		out.RSSSource = m.Source
	}
	return out
}
func Decode(line []byte) (Command, error) {
	var c Command
	err := json.Unmarshal(line, &c)
	return c, err
}

// readProjection drops every reference list from an observation. A read is for
// consuming visible content; keeping the lists would re-serialize the very
// data the read path exists to avoid, and returning them would invite agents to
// treat read output as action-ready references.
func readProjection(p *Page) *Page {
	if p == nil {
		return nil
	}
	out := *p
	out.Markdown = ""
	out.Metadata = nil
	out.Headings = []Heading{}
	out.Links = []Link{}
	out.Forms = []Form{}
	out.Buttons = nil
	out.Inputs = nil
	out.OmittedLinks, out.OmittedHeadings = 0, 0
	return &out
}

// safePage prevents browser-agent logs from displaying credentials and hidden tokens.
// Actual values remain in the browser session and are used when submitting forms.
func safePage(p *Page) *Page {
	if p == nil {
		return nil
	}
	copyPage := *p
	copyPage.Inputs = append([]Field(nil), p.Inputs...)
	for i := range copyPage.Inputs {
		field := &copyPage.Inputs[i]
		if (field.Type == "password" || field.Type == "hidden") && field.Value != "" {
			field.Value = "[redacted]"
		}
	}
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
