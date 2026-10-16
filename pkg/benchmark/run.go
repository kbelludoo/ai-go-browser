package benchmark

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"ai-go-browser/browser"
)

const (
	EngineAI       = "ai_go"
	EngineChromium = "chromium_cdp"
)

type Options struct {
	Engine      string
	BrowserPath string
	Run         int
	Timeout     time.Duration
	PublicURLs  []string
}

type Sample struct {
	Engine                string  `json:"engine"`
	Run                   int     `json:"run"`
	Case                  string  `json:"case"`
	URL                   string  `json:"url"`
	FinalURL              string  `json:"final_url,omitempty"`
	ObservedTitle         string  `json:"observed_title,omitempty"`
	ObservedTextExcerpt   string  `json:"observed_text_excerpt,omitempty"`
	HTTPStatus            int     `json:"http_status,omitempty"`
	Phase                 string  `json:"phase"`
	RequiresJS            bool    `json:"requires_js"`
	ChromiumStartedInTask bool    `json:"chromium_started_in_task"`
	Success               bool    `json:"success"`
	Error                 string  `json:"error,omitempty"`
	SetupMS               float64 `json:"setup_ms"`
	TaskMS                float64 `json:"task_ms"`
	TotalMS               float64 `json:"total_ms"`
	SnapshotBytes         int     `json:"snapshot_bytes"`
	DocumentBytes         int     `json:"document_bytes"`
	GoHeapAllocBytes      uint64  `json:"go_heap_alloc_bytes"`
	GoSysBytes            uint64  `json:"go_sys_bytes"`
	HarnessRSSBytes       *uint64 `json:"harness_rss_bytes"`
	HarnessRSSSource      string  `json:"harness_rss_source"`
	ChromiumTreeRSSBytes  *uint64 `json:"chromium_tree_rss_bytes"`
	ChromiumRSSSource     string  `json:"chromium_rss_source"`
}

type driver interface {
	open(context.Context, string, bool) error
	action(context.Context, fixture) error
	snapshot(context.Context) (snapshot, error)
	pid() int
	close()
}

type snapshot struct {
	text, title, url       string
	size, document, status int
}

// Run runs one engine in one fresh worker process. The command launches separate
// workers to isolate Go heaps; Chromium memory is measured separately.
func Run(ctx context.Context, opts Options) []Sample {
	if opts.Timeout <= 0 {
		opts.Timeout = 12 * time.Second
	}
	if opts.Run == 0 {
		opts.Run = 1
	}
	var cases []fixture
	base := ""
	if len(opts.PublicURLs) == 0 {
		server, localCases := localFixtures()
		defer server.Close()
		base, cases = server.URL, localCases
	} else {
		for i, u := range opts.PublicURLs {
			cases = append(cases, fixture{name: fmt.Sprintf("public_%d", i+1), path: u})
		}
	}
	start := time.Now()
	d, setupErr := newDriver(ctx, opts, base)
	setupMS := milliseconds(time.Since(start))
	if d != nil {
		defer d.close()
	}
	out := make([]Sample, 0, len(cases))
	for i, f := range cases {
		row := Sample{Engine: opts.Engine, Run: opts.Run, Case: f.name, URL: base + f.path, Phase: "warm", RequiresJS: f.javascript}
		if i == 0 {
			row.Phase = "cold"
			row.SetupMS = setupMS
		}
		caseStart := time.Now()
		priorPID := 0
		if d != nil {
			priorPID = d.pid()
		}
		err := setupErr
		if err == nil {
			caseCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
			err = d.open(caseCtx, row.URL, f.javascript)
			if err == nil {
				err = d.action(caseCtx, f)
			}
			if err == nil {
				for {
					observed, snapshotErr := d.snapshot(caseCtx)
					if snapshotErr == nil {
						row.SnapshotBytes, row.DocumentBytes = observed.size, observed.document
						row.ObservedTitle, row.FinalURL, row.HTTPStatus = observed.title, observed.url, observed.status
						excerpt := []rune(observed.text)
						if len(excerpt) > 400 {
							excerpt = excerpt[:400]
						}
						row.ObservedTextExcerpt = string(excerpt)
						if f.expected != "" && strings.Contains(observed.text, f.expected) {
							break
						}
						if f.expected == "" && strings.TrimSpace(observed.text) != "" {
							err = publicSmokeProblem(observed)
							break
						}
					}
					select {
					case <-caseCtx.Done():
						err = fmt.Errorf("assertion failed: visible text must contain %q (last snapshot error: %v): %w", f.expected, snapshotErr, caseCtx.Err())
					case <-time.After(50 * time.Millisecond):
					}
					if err != nil {
						break
					}
				}
			}
			cancel()
		}
		row.TaskMS = milliseconds(time.Since(caseStart))
		row.TotalMS = row.TaskMS + row.SetupMS
		row.Success = err == nil
		if err != nil {
			row.Error = err.Error()
		}
		var memory runtime.MemStats
		runtime.ReadMemStats(&memory)
		row.GoHeapAllocBytes, row.GoSysBytes = memory.HeapAlloc, memory.Sys
		row.HarnessRSSBytes, row.HarnessRSSSource = sampleRSS(os.Getpid(), false)
		if d != nil && d.pid() > 0 {
			row.ChromiumStartedInTask = priorPID == 0
			row.ChromiumTreeRSSBytes, row.ChromiumRSSSource = sampleRSS(d.pid(), true)
		} else {
			row.ChromiumRSSSource = "no Chromium process reported"
		}
		out = append(out, row)
	}
	return out
}

func milliseconds(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

func newDriver(ctx context.Context, opts Options, localURL string) (driver, error) {
	switch opts.Engine {
	case EngineAI:
		allowed := []string{"*"}
		if localURL != "" {
			u, _ := url.Parse(localURL)
			allowed = []string{u.Hostname()}
		}
		b := browser.New(allowed, false)
		b.CompactSnapshots = true
		b.AutoRenderJS = true
		b.SystemBrowser = opts.BrowserPath
		return &aiDriver{b: b}, nil
	case EngineChromium:
		s, err := browser.LaunchStealth(ctx, browser.StealthConfig{BrowserPath: opts.BrowserPath, Headless: true})
		if err != nil {
			return nil, err
		}
		return &chromiumDriver{s: s}, nil
	default:
		return nil, fmt.Errorf("unknown engine %q", opts.Engine)
	}
}

type aiDriver struct{ b *browser.Browser }

func (d *aiDriver) command(c browser.Command) error {
	r := d.b.Execute(c)
	if !r.OK {
		return fmt.Errorf("%s: %s", c.Cmd, r.Error)
	}
	return nil
}
func (d *aiDriver) open(ctx context.Context, u string, js bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return d.command(browser.Command{Cmd: "open", URL: u, RenderJS: js})
}
func (d *aiDriver) action(ctx context.Context, f fixture) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch f.action {
	case "form", "spa":
		if d.b.Current == nil {
			return fmt.Errorf("no current page")
		}
		name := "q"
		if f.action == "spa" {
			name = "name"
		}
		formRef, fieldRef := "", ""
		for _, form := range d.b.Current.Forms {
			for _, field := range form.Fields {
				if field.Name == name {
					formRef, fieldRef = form.Ref, field.Ref
				}
			}
		}
		if fieldRef == "" {
			return fmt.Errorf("input %q was not discovered", name)
		}
		if err := d.command(browser.Command{Cmd: "fill", Ref: fieldRef, Value: f.value}); err != nil {
			return err
		}
		if f.action == "form" {
			return d.command(browser.Command{Cmd: "submit", Ref: formRef})
		}
		for _, button := range d.b.Current.Buttons {
			if button.ID == "go" {
				return d.command(browser.Command{Cmd: "press", Ref: button.Ref})
			}
		}
		return fmt.Errorf("SPA action button was not discovered")
	case "link":
		for _, link := range d.b.Current.Links {
			if strings.Contains(link.URL, "/cookie-result") {
				return d.command(browser.Command{Cmd: "click", Ref: link.Ref})
			}
		}
		return fmt.Errorf("cookie navigation link was not discovered")
	}
	return nil
}
func (d *aiDriver) snapshot(ctx context.Context) (snapshot, error) {
	if err := ctx.Err(); err != nil {
		return snapshot{}, err
	}
	r := d.b.Execute(browser.Command{Cmd: "snapshot"})
	if !r.OK || r.Page == nil {
		return snapshot{}, fmt.Errorf("snapshot: %s", r.Error)
	}
	b, err := json.Marshal(r.Page)
	return snapshot{text: r.Page.Text, title: r.Page.Title, url: r.Page.URL, size: len(b), document: r.Page.Bytes, status: r.Page.Status}, err
}
func (d *aiDriver) pid() int {
	if p, ok := any(d.b).(interface{ RenderProcessID() int }); ok {
		return p.RenderProcessID()
	}
	return 0
}
func (d *aiDriver) close() { d.b.Close() }

type chromiumDriver struct{ s *browser.StealthSession }

func (d *chromiumDriver) open(ctx context.Context, u string, js bool) error {
	if ready, ok := any(d.s).(interface {
		NavigateReady(context.Context, string) error
	}); ok {
		return ready.NavigateReady(ctx, u)
	}
	return fmt.Errorf("competent baseline requires StealthSession.NavigateReady without a fixed settling delay")
}
func (d *chromiumDriver) action(ctx context.Context, f fixture) error {
	value, _ := json.Marshal(f.value)
	var expr string
	switch f.action {
	case "form":
		expr = `(()=>{const q=document.querySelector('#q');q.value=` + string(value) + `;q.dispatchEvent(new Event('input',{bubbles:true}));document.querySelector('#search').requestSubmit();return true})()`
	case "spa":
		expr = `(()=>{const q=document.querySelector('#name');q.value=` + string(value) + `;q.dispatchEvent(new Event('input',{bubbles:true}));document.querySelector('#go').click();return true})()`
	case "link":
		expr = `(()=>{document.querySelector('#continue').click();return true})()`
	default:
		return nil
	}
	_, err := d.s.Evaluate(ctx, expr)
	return err
}

// Both compact snapshots cap visible text at 5000 Unicode code points and links
// at 200. The baseline uses DOM extraction, not raw HTML as its only option.
const compactDOM = `(()=>({url:location.href,title:document.title,status:performance.getEntriesByType('navigation')[0]?.responseStatus||0,text:Array.from(document.body?.innerText||'').slice(0,5000).join(''),headings:Array.from(document.querySelectorAll('h1,h2,h3,h4,h5,h6')).slice(0,100).map(x=>({level:Number(x.tagName.slice(1)),text:x.innerText})),links:Array.from(document.querySelectorAll('a[href]')).slice(0,200).map(x=>({label:x.innerText,url:x.href})),buttons:Array.from(document.querySelectorAll('button,input[type=button],input[type=submit]')).map(x=>({id:x.id,label:x.innerText||x.value,type:x.type})),forms:Array.from(document.forms).map(x=>({action:x.action,method:x.method,fields:Array.from(x.elements).filter(e=>e.name).map(e=>({name:e.name,type:e.type,value:e.value,label:e.labels?.[0]?.innerText||''}))})),document_bytes:new TextEncoder().encode(document.documentElement.outerHTML).length}))()`

func (d *chromiumDriver) snapshot(ctx context.Context) (snapshot, error) {
	b, err := d.s.Evaluate(ctx, compactDOM)
	if err != nil {
		return snapshot{}, err
	}
	var p struct {
		Text   string `json:"text"`
		Title  string `json:"title"`
		URL    string `json:"url"`
		Status int    `json:"status"`
		Bytes  int    `json:"document_bytes"`
	}
	if err = json.Unmarshal(b, &p); err != nil {
		return snapshot{}, err
	}
	return snapshot{text: p.Text, title: p.Title, url: p.URL, status: p.Status, size: len(b), document: p.Bytes}, nil
}

func publicSmokeProblem(p snapshot) error {
	if p.status >= 400 {
		return fmt.Errorf("public smoke returned HTTP %d", p.status)
	}
	if p.url == "about:blank" || strings.HasPrefix(p.url, "chrome-error:") {
		return fmt.Errorf("public smoke remained on browser error page: %s", p.url)
	}
	text := strings.ToLower(p.title + "\n" + p.text)
	for _, marker := range []string{"just a moment...", "verify you are human", "checking your browser before accessing", "access denied", "403 forbidden", "err_name_not_resolved", "err_connection_refused"} {
		if strings.Contains(text, marker) {
			return fmt.Errorf("public smoke found possible access challenge/error: %q", marker)
		}
	}
	return nil
}
func (d *chromiumDriver) pid() int {
	if p, ok := any(d.s).(interface{ ProcessID() int }); ok {
		return p.ProcessID()
	}
	return 0
}
func (d *chromiumDriver) close() { d.s.Close() }
