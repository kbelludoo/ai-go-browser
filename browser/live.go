package browser

// A live session keeps Chromium's DOM, event handlers, storage and native
// navigation history across commands. Go snapshots remain bounded observations.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const liveMarker = "data-aigobrowser"

type liveSession struct {
	session  *StealthSession
	cancel   context.CancelFunc
	done     chan struct{}
	mu       sync.Mutex
	changed  chan struct{}
	frameID  string
	loading  bool
	version  uint64
	status   int
	pageURL  string
	request  string
	failure  error
	policy   *Browser
	statuses map[string]int
	frames   *liveFrameController
}

func (b *Browser) liveContext() (context.Context, context.CancelFunc) {
	timeout := b.OperationTimeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return context.WithTimeout(context.Background(), timeout)
}

func (b *Browser) startLive(ctx context.Context) error {
	s, err := LaunchStealth(ctx, StealthConfig{BrowserPath: b.SystemBrowser, Headless: true})
	if err != nil {
		return err
	}
	life, cancel := context.WithCancel(context.Background())
	l := &liveSession{session: s, cancel: cancel, done: make(chan struct{}), changed: make(chan struct{}), statuses: map[string]int{}}
	l.frames = newLiveFrameController()
	b.live = l
	l.setPolicy(b)
	// The initial target is about:blank. Identify its main frame before listening.
	raw, err := s.pageCDP.send(ctx, "Page.getFrameTree", nil)
	if err == nil {
		var tree struct {
			FrameTree struct {
				Frame struct {
					ID string `json:"id"`
				} `json:"frame"`
			} `json:"frameTree"`
		}
		err = json.Unmarshal(raw, &tree)
		l.frameID = tree.FrameTree.Frame.ID
	}
	if err != nil || l.frameID == "" {
		cancel()
		close(l.done)
		b.closeLive()
		return fmt.Errorf("identify live main frame: %v", err)
	}
	go l.events(life)
	// These enables stay ordered. Target.setAutoAttach emits attachedToTarget
	// events that this session's event stream must already be draining, so
	// they are not issued in parallel like the silent enables in LaunchStealth.
	for _, setting := range []struct {
		method string
		params map[string]any
	}{
		{method: "Network.enable", params: map[string]any{"maxTotalBufferSize": 1 << 20, "maxResourceBufferSize": 256 << 10}},
		{method: "Target.setAutoAttach", params: frameAutoAttachParams()},
		// Document interception covers navigation and redirects, including frames.
		// Page scripts' POST/XHR traffic retains normal browser semantics.
		{method: "Fetch.enable", params: map[string]any{"patterns": []map[string]any{{"urlPattern": "*", "resourceType": "Document", "requestStage": "Request"}}}},
	} {
		if _, err := s.pageCDP.send(ctx, setting.method, setting.params); err != nil {
			b.closeLive()
			return fmt.Errorf("enable live %s: %w", setting.method, err)
		}
	}
	if b.UserAgent != "" {
		if _, err := s.pageCDP.send(ctx, "Network.setUserAgentOverride", map[string]any{"userAgent": b.UserAgent}); err != nil {
			b.closeLive()
			return err
		}
	}
	return nil
}

func (l *liveSession) setPolicy(b *Browser) {
	policy := &Browser{AllowAll: b.AllowAll, Allowed: make(map[string]bool, len(b.Allowed))}
	for host, allow := range b.Allowed {
		policy.Allowed[host] = allow
	}
	l.mu.Lock()
	l.policy = policy
	l.failure = nil
	l.mu.Unlock()
}

func (l *liveSession) notifyLocked() {
	close(l.changed)
	l.changed = make(chan struct{})
}

func (l *liveSession) events(ctx context.Context) {
	defer close(l.done)
	for {
		select {
		case <-ctx.Done():
			return
		case <-l.session.pageCDP.closed:
			l.mu.Lock()
			l.failure = fmt.Errorf("live page connection closed: %v", l.session.pageCDP.readErr)
			l.notifyLocked()
			l.mu.Unlock()
			return
		case <-l.session.browserCDP.closed:
			l.mu.Lock()
			l.failure = fmt.Errorf("live browser connection closed: %v", l.session.browserCDP.readErr)
			l.notifyLocked()
			l.mu.Unlock()
			return
		case msg := <-l.session.pageCDP.events:
			if l.frameEvent(ctx, l.session.pageCDP, msg) {
				continue
			}
			if msg.Method == "Fetch.requestPaused" {
				l.interceptSession(ctx, l.session.pageCDP.session(msg.Session), msg.Params)
				continue
			}
			if msg.Session != "" {
				continue
			}
			l.mu.Lock()
			switch msg.Method {
			case "Page.frameStartedLoading", "Page.frameStoppedLoading":
				var event struct {
					FrameID string `json:"frameId"`
				}
				if json.Unmarshal(msg.Params, &event) == nil && event.FrameID == l.frameID {
					l.loading = msg.Method == "Page.frameStartedLoading"
				}
			case "Network.requestWillBeSent":
				var event struct {
					FrameID   string `json:"frameId"`
					RequestID string `json:"requestId"`
					Type      string `json:"type"`
				}
				if json.Unmarshal(msg.Params, &event) == nil && event.FrameID == l.frameID && event.Type == "Document" {
					l.request = event.RequestID
					l.loading = true
					l.version++
				}
			case "Network.responseReceived":
				var event struct {
					FrameID  string `json:"frameId"`
					Type     string `json:"type"`
					Response struct {
						URL    string  `json:"url"`
						Status float64 `json:"status"`
					} `json:"response"`
					LoaderID string `json:"loaderId"`
				}
				if json.Unmarshal(msg.Params, &event) == nil && event.FrameID == l.frameID && event.Type == "Document" {
					l.status = int(event.Response.Status)
					l.pageURL = event.Response.URL
					if len(l.statuses) >= 100 {
						for key := range l.statuses {
							delete(l.statuses, key)
							break
						}
					}
					// A URL can have different statuses on different visits.
					// Loader IDs distinguish cached documents during native back.
					if event.LoaderID != "" {
						l.statuses[event.LoaderID] = l.status
					}
				}
			case "Network.loadingFailed":
				var event struct {
					RequestID string `json:"requestId"`
					ErrorText string `json:"errorText"`
					Canceled  bool   `json:"canceled"`
				}
				if json.Unmarshal(msg.Params, &event) == nil && event.RequestID == l.request && !event.Canceled && l.failure == nil {
					l.failure = fmt.Errorf("live navigation failed: %s", event.ErrorText)
					l.loading = false
				}
			case "Page.frameNavigated":
				var event struct {
					Frame struct {
						ID       string `json:"id"`
						ParentID string `json:"parentId"`
						URL      string `json:"url"`
						LoaderID string `json:"loaderId"`
					} `json:"frame"`
				}
				if json.Unmarshal(msg.Params, &event) == nil && event.Frame.ParentID == "" {
					l.frameID = event.Frame.ID
					l.pageURL = event.Frame.URL
					l.version++
					l.status = l.statuses[event.Frame.LoaderID]
				}
			case "Page.navigatedWithinDocument":
				var event struct {
					FrameID string `json:"frameId"`
					URL     string `json:"url"`
				}
				if json.Unmarshal(msg.Params, &event) == nil && event.FrameID == l.frameID {
					l.pageURL = event.URL
					l.version++
				}
			case "Page.javascriptDialogOpening":
				// An unattended modal would stall all later commands.
				go func() {
					dialogCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
					defer cancel()
					_, _ = l.session.pageCDP.send(dialogCtx, "Page.handleJavaScriptDialog", map[string]any{"accept": false})
				}()
			}
			l.notifyLocked()
			l.mu.Unlock()
		case msg := <-l.session.browserCDP.events:
			if !l.frameEvent(ctx, l.session.browserCDP, msg) && msg.Method == "Fetch.requestPaused" {
				l.interceptSession(ctx, l.session.browserCDP.session(msg.Session), msg.Params)
			}
		}
	}
}

func (l *liveSession) intercept(ctx context.Context, raw json.RawMessage) {
	l.interceptSession(ctx, l.session.pageCDP.session(""), raw)
}

func (l *liveSession) interceptSession(ctx context.Context, endpoint *cdpSession, raw json.RawMessage) {
	var event struct {
		RequestID string `json:"requestId"`
		FrameID   string `json:"frameId"`
		Request   struct {
			URL string `json:"url"`
		} `json:"request"`
	}
	if json.Unmarshal(raw, &event) != nil {
		return
	}
	l.mu.Lock()
	policy := l.policy
	l.mu.Unlock()
	u, err := url.Parse(event.Request.URL)
	if err == nil {
		err = policy.validate(u)
	}
	method := "Fetch.continueRequest"
	params := map[string]any{"requestId": event.RequestID}
	if err != nil {
		method = "Fetch.failRequest"
		params["errorReason"] = "BlockedByClient"
		l.mu.Lock()
		if event.FrameID == l.frameID {
			l.failure = fmt.Errorf("live navigation blocked: %w", err)
			l.loading = false
		} else if event.FrameID != "" {
			state := l.frameStateLocked(event.FrameID)
			state.failure = fmt.Errorf("frame navigation blocked: %w", err)
			state.loading = false
		}
		l.notifyLocked()
		l.mu.Unlock()
	}
	callCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, callErr := endpoint.send(callCtx, method, params)
	if callErr != nil {
		l.mu.Lock()
		if event.FrameID == l.frameID {
			l.failure = fmt.Errorf("live navigation interception: %w", callErr)
		} else if event.FrameID != "" {
			state := l.frameStateLocked(event.FrameID)
			if !state.detached {
				state.failure = fmt.Errorf("frame navigation interception: %w", callErr)
			}
		}
		l.notifyLocked()
		l.mu.Unlock()
	}
}

func (l *liveSession) waitNavigation(ctx context.Context, after uint64) error {
	for {
		l.mu.Lock()
		failure, loading, version, changed := l.failure, l.loading, l.version, l.changed
		l.mu.Unlock()
		if failure != nil {
			return failure
		}
		if version > after && !loading {
			return nil
		}
		select {
		case <-changed:
		case <-ctx.Done():
			return fmt.Errorf("live navigation: %w", ctx.Err())
		case <-l.done:
			return errors.New("live browser closed")
		}
	}
}

func liveTransient(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "execution context was destroyed") ||
		strings.Contains(message, "cannot find context") ||
		strings.Contains(message, "uniquecontextid not found") ||
		strings.Contains(message, "inspected target navigated")
}

// settle waits for document readiness and rendering work rather than sleeping
// a fixed interval. Later asynchronous application work can be awaited with wait.
func (l *liveSession) settle(ctx context.Context) error {
	for {
		l.mu.Lock()
		failure, loading, changed := l.scopeReadinessLocked()
		l.mu.Unlock()
		if failure != nil {
			return failure
		}
		if loading {
			select {
			case <-changed:
				continue
			case <-ctx.Done():
				return fmt.Errorf("live page readiness: %w", ctx.Err())
			}
		}
		_, err := l.evaluate(ctx, `new Promise(resolve => {
			const ready = () => requestAnimationFrame(() => requestAnimationFrame(() => resolve(true)));
			if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', ready, {once:true});
			else ready();
		})`)
		if !liveTransient(err) {
			l.mu.Lock()
			failure, loading, _ = l.scopeReadinessLocked()
			l.mu.Unlock()
			if failure != nil {
				return failure
			}
			if err != nil {
				return err
			}
			if loading {
				continue
			}
			return nil
		}
		select {
		case <-changed:
		case <-ctx.Done():
			return fmt.Errorf("live page readiness: %w", ctx.Err())
		}
	}
}

func (b *Browser) openLive(target string, saveHistory bool) error {
	u, err := url.Parse(target)
	if err != nil {
		return err
	}
	if err := b.validate(u); err != nil {
		return err
	}
	ctx, cancel := b.liveContext()
	defer cancel()
	fresh := b.live == nil
	if fresh {
		if err := b.startLive(ctx); err != nil {
			return err
		}
	}
	l := b.live
	previousURL := ""
	if b.Current != nil {
		previousURL = b.Current.URL
	}
	l.mu.Lock()
	if !fresh {
		previousURL = l.pageURL
	}
	l.frames.selected = nil
	l.mu.Unlock()
	l.setPolicy(b)
	if err := b.cookiesToLive(ctx, u); err != nil {
		if fresh {
			b.closeLive()
		}
		return err
	}
	l.mu.Lock()
	before := l.version
	l.mu.Unlock()
	raw, err := l.session.pageCDP.send(ctx, "Page.navigate", map[string]any{"url": u.String()})
	if err == nil {
		var result struct {
			ErrorText string `json:"errorText"`
			LoaderID  string `json:"loaderId"`
		}
		err = json.Unmarshal(raw, &result)
		if result.ErrorText != "" {
			err = fmt.Errorf("live navigation: %s", result.ErrorText)
		}
		if err == nil {
			err = l.waitNavigation(ctx, before)
		}
	}
	if err == nil {
		err = b.refreshLiveContext(ctx)
	}
	if err != nil {
		if fresh {
			b.closeLive()
		} else if errors.Is(err, context.DeadlineExceeded) {
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			_, _ = l.session.pageCDP.send(stopCtx, "Page.stopLoading", nil)
			stopCancel()
		}
		return err
	}
	b.closeJSSession()
	b.rawHTML = nil
	b.lastJS = nil
	if saveHistory {
		b.rememberLiveURL(previousURL)
	}
	return nil
}

func (b *Browser) refreshLive() error {
	if b.live == nil {
		return errors.New("no live browser session")
	}
	ctx, cancel := b.liveContext()
	defer cancel()
	return b.refreshLiveContext(ctx)
}

func (b *Browser) refreshLiveContext(ctx context.Context) error {
	l := b.live
	if err := l.settle(ctx); err != nil {
		return err
	}
	raw, err := l.evaluate(ctx, liveSnapshotScript)
	if err != nil {
		return err
	}
	var snapshot struct {
		HTML string `json:"html"`
		URL  string `json:"url"`
		Base string `json:"base"`
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return err
	}
	l.mu.Lock()
	failure, _, _ := l.scopeReadinessLocked()
	l.mu.Unlock()
	if failure != nil {
		return failure
	}
	u, err := url.Parse(snapshot.URL)
	if err != nil {
		return err
	}
	if err := b.validateLiveScope(ctx, snapshot.URL); err != nil {
		return err
	}
	if len(snapshot.HTML) > MaxBody {
		return errHeadlessDOMTooLarge
	}
	l.mu.Lock()
	status := l.scopeStatusLocked()
	l.mu.Unlock()
	base := snapshot.Base
	if base == "" {
		base = snapshot.URL
	}
	d, err := ParseDocument([]byte(snapshot.HTML), base, status, len(snapshot.HTML))
	if err != nil {
		return err
	}
	controlWalk(d.Root, func(node *Node) {
		if node.Tag == "aigobrowser-form" {
			node.Tag = "form"
		}
	})
	p := d.PageWithMarkdown(b.MaxText, !b.CompactSnapshots)
	p.URL = snapshot.URL
	l.namespaceScope(d, &p)
	p.Stack = AnalyzeTechStack(snapshot.HTML, utf8.RuneCountInString(p.Text), "utf-8")
	p.Stack.RenderMode = "Chromium Live (JS Executed)"
	p.Stack.Notice = ""
	for key, value := range ExtractStructuredData(snapshot.HTML) {
		p.Metadata[key] = value
	}
	if u.Scheme == "http" || u.Scheme == "https" {
		if err := b.cookiesFromLive(ctx, u); err != nil {
			return err
		}
	}
	if err := l.pinActionObservation(); err != nil {
		return err
	}
	b.Document, b.Current = d, &p
	b.resource = nil
	b.scannedScripts, b.scannedScriptText = false, ""
	return nil
}

// refreshLiveRead observes the selected document's visible text without
// transferring or re-parsing its DOM, which is the whole cost the full
// refresh pays on every observation. It produces no references, so it can
// never invalidate the snapshot an agent is acting on; the reference-bearing
// observation stays in Document/Current until the next full refresh.
func (b *Browser) refreshLiveRead(ctx context.Context) (*Page, error) {
	l := b.live
	limit := b.MaxText
	if limit <= 0 {
		limit = 5000
	}
	raw, err := l.evaluate(ctx, liveReadScript(limit))
	if err != nil {
		return nil, err
	}
	var observed struct {
		URL       string `json:"url"`
		Title     string `json:"title"`
		Text      string `json:"text"`
		Truncated bool   `json:"truncated"`
		Status    int    `json:"status"`
	}
	if err := json.Unmarshal(raw, &observed); err != nil {
		return nil, err
	}
	if err := b.validateLiveScope(ctx, observed.URL); err != nil {
		return nil, err
	}
	p := &Page{
		URL:       observed.URL,
		Title:     observed.Title,
		Text:      observed.Text,
		Status:    observed.Status,
		Truncated: observed.Truncated,
		Headings:  []Heading{},
		Links:     []Link{},
		Forms:     []Form{},
	}
	p.Stack.RenderMode = "Chromium Live (read)"
	l.mu.Lock()
	scope := l.frames.selected
	l.mu.Unlock()
	if scope == nil {
		p.FrameRef = "fr1"
	} else {
		p.FrameRef = scope.ref
	}
	return p, nil
}

// executeLiveRead handles snapshot/text with view:"read". A live read is
// published through readPage so Execute returns it instead of Current, and it
// is cleared in the same call so no later command inherits a ref-free page.
func (b *Browser) executeLiveRead(c Command) (bool, error) {
	if b.live == nil {
		return false, nil
	}
	ctx, cancel := b.liveContext()
	defer cancel()
	p, err := b.refreshLiveRead(ctx)
	if err != nil {
		return true, err
	}
	b.readPage = p
	return true, nil
}

func (b *Browser) cookiesToLive(ctx context.Context, u *url.URL) error {
	if b.client == nil || b.client.Jar == nil {
		return nil
	}
	cookies := b.client.Jar.Cookies(u)
	if len(cookies) == 0 {
		return nil
	}
	items := make([]map[string]any, 0, len(cookies))
	for _, cookie := range cookies {
		// CookieJar exposes cookies applicable to this URL, without their original
		// domain/path metadata. Bind this conservative transfer to this host.
		items = append(items, map[string]any{"name": cookie.Name, "value": cookie.Value, "url": u.String()})
	}
	_, err := b.live.session.pageCDP.send(ctx, "Network.setCookies", map[string]any{"cookies": items})
	return err
}

func (b *Browser) cookiesFromLive(ctx context.Context, u *url.URL) error {
	if b.client == nil || b.client.Jar == nil {
		return nil
	}
	raw, err := b.live.session.GetCookies(ctx, u.String())
	if err != nil {
		return err
	}
	var response struct {
		Cookies []struct {
			Name     string  `json:"name"`
			Value    string  `json:"value"`
			Domain   string  `json:"domain"`
			Path     string  `json:"path"`
			Secure   bool    `json:"secure"`
			HTTPOnly bool    `json:"httpOnly"`
			Session  bool    `json:"session"`
			Expires  float64 `json:"expires"`
		} `json:"cookies"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return err
	}
	cookies := make([]*http.Cookie, 0, len(response.Cookies))
	for _, item := range response.Cookies {
		cookie := &http.Cookie{Name: item.Name, Value: item.Value, Domain: item.Domain, Path: item.Path, Secure: item.Secure, HttpOnly: item.HTTPOnly}
		if !item.Session && item.Expires > 0 {
			cookie.Expires = time.Unix(int64(item.Expires), 0)
		}
		cookies = append(cookies, cookie)
	}
	b.client.Jar.SetCookies(u, cookies)
	return nil
}

func jsLiteral(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func liveElementExpression(marker string) string {
	return "(() => {" + liveDOMHelpers + "return liveDOM.query(" + jsLiteral("["+liveMarker+"="+strconv.Quote(marker)+"]") + ", 1)[0] || null;})()"
}

func (b *Browser) rememberLiveURL(previousURL string) {
	if previousURL == "" {
		return
	}
	current := b.Current
	b.Current = &Page{URL: previousURL}
	b.rememberCurrent()
	b.Current = current
}

func (b *Browser) executeLive(c Command) (bool, error) {
	if b.live == nil {
		return false, nil
	}
	command := strings.ToLower(strings.TrimSpace(c.Cmd))
	switch command {
	case "snapshot", "text", "markdown":
		return true, b.refreshLive()
	case "discover", "inspect", "run":
		return false, b.refreshLive()
	case "fill", "check", "click", "press", "submit", "back":
	default:
		return false, nil
	}
	ctx, cancel := b.liveContext()
	defer cancel()
	b.live.setPolicy(b)
	if command == "back" {
		return true, b.backLive(ctx)
	}
	if b.Document == nil || b.Current == nil {
		return true, errors.New("no page open")
	}
	b.live.mu.Lock()
	beforeVersion := b.live.version
	previousURL := b.live.pageURL
	scope := b.live.frames.selected
	invalidScope := scope != nil && scope.invalid
	b.live.mu.Unlock()
	if invalidScope {
		if err := b.refreshLiveContext(ctx); err != nil {
			return true, err
		}
		return true, errors.New("frame document changed; use references from a new snapshot")
	}
	var node *Node
	var submitter *Node
	switch command {
	case "fill", "check":
		node = b.Document.fields[c.Ref]
	case "click":
		node = b.Document.links[c.Ref]
	case "press":
		node = b.Document.buttons[c.Ref]
	case "submit":
		for index, form := range b.Current.Forms {
			if form.Ref != c.Ref {
				continue
			}
			formIndex := 0
			controlWalk(b.Document.Root, func(n *Node) {
				if n.Tag == "form" {
					if formIndex == index {
						node = n
					}
					formIndex++
				}
			})
			if form.SubmitRef != "" {
				submitter = b.Document.buttons[form.SubmitRef]
			}
			break
		}
	}
	if node == nil || node.prop(liveMarker) == "" {
		return true, fmt.Errorf("live %s reference %q not found", command, c.Ref)
	}
	element := liveElementExpression(node.prop(liveMarker))
	var err error
	if command == "check" {
		if c.Checked == nil {
			checked := true
			c.Checked = &checked
		}
		typ := node.controlType()
		if typ != "checkbox" && typ != "radio" {
			return true, errors.New("check requires a checkbox or radio field")
		}
		var current bool
		raw, checkErr := b.live.evaluateAction(ctx, `(() => {
			`+liveDOMHelpers+`
			const element = `+element+`;
			if (!element || !element.isConnected) throw new Error('stale live field reference; take a new snapshot');
			if (liveDOM.disabled(element)) throw new Error('field cannot be edited');
			return element.checked;
		})()`)
		if checkErr != nil {
			return true, checkErr
		}
		if checkErr := json.Unmarshal(raw, &current); checkErr != nil {
			return true, checkErr
		}
		if current != *c.Checked {
			if typ == "radio" && !*c.Checked {
				return true, errors.New("select another radio field to clear this choice")
			}
			err = b.clickLiveElement(ctx, element)
		}
	} else if command == "fill" {
		if node.disabled() || node.readOnly() || node.controlType() == "hidden" || node.controlType() == "file" || node.controlType() == "submit" {
			return true, errors.New("field cannot be edited")
		}
		if node.controlType() == "checkbox" || node.controlType() == "radio" {
			return true, errors.New("use check for checkbox or radio fields")
		}
		if node.controlType() == "contenteditable" {
			err = b.fillLiveEditable(ctx, element, c.Value)
		} else if liveNativeTextField(node.controlType()) {
			err = b.fillLiveText(ctx, element, c.Value, false)
		} else {
			_, err = b.live.evaluateAction(ctx, `(() => {
			`+liveDOMHelpers+`
			const element = `+element+`;
			if (!element || !element.isConnected) throw new Error('stale live field reference; take a new snapshot');
			if (liveDOM.disabled(element) || liveDOM.readonly(element)) throw new Error('field cannot be edited');
			const value = `+jsLiteral(c.Value)+`;
			const prototype = element instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype :
				element instanceof HTMLSelectElement ? HTMLSelectElement.prototype : HTMLInputElement.prototype;
			if (element instanceof HTMLSelectElement) {
				const option = Array.from(element.options).find(option => option.value === value);
				if (!option || option.disabled || (option.parentElement instanceof HTMLOptGroupElement && option.parentElement.disabled)) throw new Error('select value is missing or disabled');
			}
			const setter = Object.getOwnPropertyDescriptor(prototype, 'value').set;
			setter.call(element, value);
			element.dispatchEvent(new Event('input', {bubbles:true, composed:true}));
			element.dispatchEvent(new Event('change', {bubbles:true}));
			return true;
		})()`)
		}
	} else if command == "submit" {
		submitterExpression := "null"
		if submitter != nil {
			submitterExpression = liveElementExpression(submitter.prop(liveMarker))
		}
		if err := b.validateLiveSubmission(ctx, element, submitterExpression, c.Confirm, true); err != nil {
			return true, err
		}
		_, err = b.live.evaluateAction(ctx, `(() => {
			const element = `+element+`;
			if (!element || !element.isConnected) throw new Error('stale live form reference; take a new snapshot');
			const submitter = `+submitterExpression+`;
			if (submitter && submitter.form !== element) throw new Error('submitter does not belong to this form');
			if (submitter) HTMLFormElement.prototype.requestSubmit.call(element, submitter);
			else HTMLFormElement.prototype.requestSubmit.call(element);
			return true;
		})()`)
	} else {
		if command == "click" {
			target := resolveURL(b.Current.URL, node.prop("href"))
			for _, link := range b.Current.Links {
				if link.Ref == c.Ref {
					target = link.URL
					break
				}
			}
			u, parseErr := url.Parse(target)
			if parseErr != nil {
				return true, parseErr
			}
			if err := b.validate(u); err != nil {
				return true, err
			}
		}
		if command == "press" {
			for _, action := range b.Discover() {
				if action.Method == "POST" && action.Source == "javascript:button#"+c.Ref && (!b.AllowPost || !c.Confirm) {
					return true, errors.New("POST requires -allow-post and confirm:true")
				}
			}
			if node.controlType() == "submit" || node.controlType() == "image" {
				if err := b.validateLiveSubmission(ctx, "null", element, c.Confirm, false); err != nil {
					return true, err
				}
			}
		}
		err = b.clickLiveElement(ctx, element)
	}
	if liveActionTransient(err, command, node.controlType()) {
		err = nil
	}
	if err == nil {
		err = b.refreshLiveContext(ctx)
	}
	b.live.mu.Lock()
	afterVersion := b.live.version
	afterURL := b.live.pageURL
	b.live.mu.Unlock()
	if err == nil && (previousURL != afterURL || afterVersion > beforeVersion) {
		b.rememberLiveURL(previousURL)
	}
	return true, err
}

func liveActionTransient(err error, command, controlType string) bool {
	// Even a preparation failure must remain a failed native fill. A joined
	// observation/navigation error cannot turn a rejected edit into success.
	if command == "fill" && (controlType == "contenteditable" || liveNativeTextField(controlType)) {
		return false
	}
	return liveTransient(err) && !strings.Contains(strings.ToLower(err.Error()), "cannot find context") && !strings.Contains(strings.ToLower(err.Error()), "uniquecontextid not found")
}

func (b *Browser) validateLiveSubmission(ctx context.Context, formExpression, submitterExpression string, confirm, checkConstraints bool) error {
	raw, err := b.live.evaluateAction(ctx, `(() => {
		const submitter = `+submitterExpression+`;
		const form = `+formExpression+` || (submitter && submitter.form);
		if (!form) return null;
		if (!form.isConnected || (submitter && (!submitter.isConnected || submitter.form !== form))) throw new Error('stale live form reference; take a new snapshot');
		if (submitter && submitter.matches(':disabled')) throw new Error('submitter is disabled');
		const property = key => Object.getOwnPropertyDescriptor(HTMLFormElement.prototype, key).get.call(form);
		const invalid = `+strconv.FormatBool(checkConstraints)+` && !property('noValidate') && !(submitter && submitter.formNoValidate) && !HTMLFormElement.prototype.checkValidity.call(form);
		return {
			method: (submitter && submitter.hasAttribute('formmethod') ? submitter.formMethod : property('method')).toUpperCase(),
			url: submitter && submitter.hasAttribute('formaction') ? submitter.formAction : property('action'),
			invalid: invalid ? Array.from(property('elements')).filter(element => element.willValidate && !element.validity.valid).slice(0,10).map(element => ({name:element.name||element.id||element.tagName,message:element.validationMessage})) : []
		};
	})()`)
	if err != nil {
		return err
	}
	if string(raw) == "null" {
		return nil
	}
	var submission struct {
		Method  string `json:"method"`
		URL     string `json:"url"`
		Invalid []struct {
			Name    string `json:"name"`
			Message string `json:"message"`
		} `json:"invalid"`
	}
	if err := json.Unmarshal(raw, &submission); err != nil {
		return err
	}
	if len(submission.Invalid) > 0 {
		return fmt.Errorf("form validation failed for %s: %s", submission.Invalid[0].Name, submission.Invalid[0].Message)
	}
	if submission.Method != "GET" && submission.Method != "POST" {
		if submission.Method == "DIALOG" {
			return nil
		}
		return errors.New("unsupported form method")
	}
	if submission.Method == "POST" && (!b.AllowPost || !confirm) {
		return errors.New("POST requires -allow-post and confirm:true")
	}
	u, err := url.Parse(submission.URL)
	if err != nil {
		return err
	}
	return b.validate(u)
}

func resolveURL(base, target string) string {
	u, _ := url.Parse(base)
	return resolve(u, target)
}

func (b *Browser) fillLiveEditable(ctx context.Context, expression, value string) error {
	return b.fillLiveText(ctx, expression, value, true)
}

func (b *Browser) clickLiveElement(ctx context.Context, expression string) error {
	raw, err := b.live.evaluateAction(ctx, `(() => {
		`+liveDOMHelpers+`
		const element = `+expression+`;
		if (!element || !element.isConnected) throw new Error('stale live element reference; take a new snapshot');
		if (liveDOM.disabled(element)) throw new Error('element is disabled');
		element.scrollIntoView({block:'center', inline:'center'});
		const rect = element.getBoundingClientRect();
		if (!liveDOM.visible(element)) throw new Error('element is not visible');
		const x = rect.left + rect.width/2, y = rect.top + rect.height/2;
		const hit = liveDOM.hit(x, y);
		if (!hit || !liveDOM.contains(element, hit)) throw new Error('element is covered by another element');
		return {x, y};
	})()`)
	if err != nil {
		return err
	}
	var point struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	if err := json.Unmarshal(raw, &point); err != nil {
		return err
	}
	localX, localY := point.X, point.Y
	point.X, point.Y, err = b.live.framePointToMain(ctx, point.X, point.Y)
	if err != nil {
		return err
	}
	_, err = b.live.evaluateAction(ctx, `(() => {
		`+liveDOMHelpers+`
		const element = `+expression+`;
		if (!element || !element.isConnected || liveDOM.disabled(element) || !liveDOM.visible(element)) throw new Error('element changed while preparing click; take a new snapshot');
		const x = `+strconv.FormatFloat(localX, 'g', -1, 64)+`, y = `+strconv.FormatFloat(localY, 'g', -1, 64)+`;
		const rect = element.getBoundingClientRect(), hit = liveDOM.hit(x, y);
		if (Math.abs(rect.left+rect.width/2-x) > .5 || Math.abs(rect.top+rect.height/2-y) > .5 || !hit || !liveDOM.contains(element, hit)) throw new Error('element changed while preparing click; take a new snapshot');
		return true;
	})()`)
	if err != nil {
		return err
	}
	if err = b.live.verifyFramePointToMain(ctx, localX, localY, point.X, point.Y); err != nil {
		return err
	}
	for _, event := range []string{"mousePressed", "mouseReleased"} {
		_, err = b.live.session.pageCDP.send(ctx, "Input.dispatchMouseEvent", map[string]any{"type": event, "x": point.X, "y": point.Y, "button": "left", "clickCount": 1})
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Browser) backLive(ctx context.Context) error {
	if len(b.History) == 0 {
		return errors.New("no history")
	}
	b.live.mu.Lock()
	b.live.frames.selected = nil
	b.live.mu.Unlock()
	raw, err := b.live.session.pageCDP.send(ctx, "Page.getNavigationHistory", nil)
	if err != nil {
		return err
	}
	var history struct {
		CurrentIndex int `json:"currentIndex"`
		Entries      []struct {
			ID  int    `json:"id"`
			URL string `json:"url"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &history); err != nil {
		return err
	}
	if history.CurrentIndex <= 0 || history.CurrentIndex >= len(history.Entries) {
		err = b.openLive(b.History[len(b.History)-1], false)
	} else {
		entry := history.Entries[history.CurrentIndex-1]
		u, parseErr := url.Parse(entry.URL)
		if parseErr != nil {
			return parseErr
		}
		if err := b.validate(u); err != nil {
			return err
		}
		b.live.mu.Lock()
		before := b.live.version
		b.live.mu.Unlock()
		_, err = b.live.session.pageCDP.send(ctx, "Page.navigateToHistoryEntry", map[string]any{"entryId": entry.ID})
		if err == nil {
			err = b.live.waitNavigation(ctx, before)
		}
		if err == nil {
			err = b.refreshLiveContext(ctx)
		}
	}
	if err == nil {
		b.History[len(b.History)-1] = ""
		b.History = b.History[:len(b.History)-1]
	}
	return err
}

func (b *Browser) liveQuery(selector string) ([]Element, error) {
	if b.live == nil {
		return nil, errors.New("no live browser session")
	}
	if selector == "" || len(selector) > 4096 {
		return nil, errors.New("selector must contain 1 to 4096 bytes")
	}
	ctx, cancel := b.liveContext()
	defer cancel()
	if err := b.refreshLiveContext(ctx); err != nil {
		return nil, err
	}
	raw, err := b.live.evaluate(ctx, `(() => {
		`+liveDOMHelpers+`
		const results = liveDOM.query(`+jsLiteral(selector)+`, 32).map(element => {
			const control = element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement || element instanceof HTMLSelectElement;
			const selected = element instanceof HTMLSelectElement;
			const editable = !control && !(element instanceof HTMLButtonElement) && (liveDOM.editable(element) || liveDOM.role(element, 'textbox'));
			const disabledOption = option => option.disabled || (option.parentElement instanceof HTMLOptGroupElement && option.parentElement.disabled);
			return {
				marker: element.getAttribute('data-aigobrowser') || '',
				tag: element.tagName.toLowerCase(), id: element.id, role: element.getAttribute('role') || '',
				label: liveDOM.label(element),
				text: liveDOM.prefix(element.innerText, 1200), href: element.hasAttribute('href') ? element.href || '' : '', name: element.name || '',
				type: selected ? 'select' : editable ? 'contenteditable' : liveDOM.genericButton(element) ? 'button' : control || element instanceof HTMLButtonElement ? (element.type || element.tagName.toLowerCase()) : '',
				disabled: liveDOM.disabled(element),
				readonly: editable ? !element.isContentEditable || liveDOM.readonly(element) : control ? liveDOM.readonly(element) : false,
				required: control ? !!element.required : false,
				pattern: control ? element.getAttribute('pattern') || '' : '',
				min: control ? element.getAttribute('min') || '' : '',
				max: control ? element.getAttribute('max') || '' : '',
				step: control ? element.getAttribute('step') || '' : '',
				...(element.type === 'checkbox' || element.type === 'radio' ? {checked: element.checked} : {}),
				multiple: selected ? element.multiple : false,
				...(selected ? {
					values: Array.from(element.selectedOptions).filter(option => !disabledOption(option)).map(option => option.value),
					options: Array.from(element.options).map(option => ({value:option.value, label:option.label, selected:option.selected, disabled:disabledOption(option)}))
				} : {}),
				value: element.type === 'password' || element.type === 'hidden' ? (element.value ? '[redacted]' : '') : editable ? element.innerText || '' : (element.value || '')
			};
		});
		if (new Blob([JSON.stringify(results)]).size > 5242880) throw new Error('live query exceeds 5242880 bytes');
		return results;
	})()`)
	if err != nil {
		return nil, err
	}
	var matches []struct {
		Element
		Marker string `json:"marker"`
	}
	if err := json.Unmarshal(raw, &matches); err != nil {
		return nil, err
	}
	refs := b.liveRefs()
	out := make([]Element, 0, len(matches))
	for _, match := range matches {
		match.Ref = refs[match.Marker]
		out = append(out, match.Element)
	}
	return out, nil
}

func (b *Browser) liveRefs() map[string]string {
	refs := make(map[string]string, len(b.Document.fields)+len(b.Document.buttons)+len(b.Document.links)+len(b.Current.Forms))
	// A link with role=button has both observations. Query uses its button
	// type, matching Document.Query, so its ref must accept press as well.
	for ref, node := range b.Document.links {
		refs[node.prop(liveMarker)] = ref
	}
	for ref, node := range b.Document.fields {
		refs[node.prop(liveMarker)] = ref
	}
	for ref, node := range b.Document.buttons {
		refs[node.prop(liveMarker)] = ref
	}
	index := 0
	controlWalk(b.Document.Root, func(node *Node) {
		if node.Tag != "form" {
			return
		}
		if index < len(b.Current.Forms) {
			refs[node.prop(liveMarker)] = b.Current.Forms[index].Ref
		}
		index++
	})
	return refs
}

func (b *Browser) liveWait(selector string, timeout time.Duration) error {
	if b.live == nil {
		return errors.New("wait requires a live browser session")
	}
	if selector == "" || len(selector) > 4096 {
		return errors.New("selector must contain 1 to 4096 bytes")
	}
	if timeout <= 0 {
		timeout = b.OperationTimeout
	}
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err := b.live.evaluate(ctx, `new Promise((resolve, reject) => {
		`+liveDOMHelpers+`
		const selector = `+jsLiteral(selector)+`;
		const observed = new WeakSet(), observers = [];
		let roots = [], lastScan = -Infinity, frame = 0, done = false, timer;
		const finish = error => {
			if (done) return;
			done = true;
			for (const observer of observers) observer.disconnect();
			cancelAnimationFrame(frame);
			clearTimeout(timer);
			if (error) reject(error); else resolve(true);
		};
		const check = (rescan = false) => {
			if (done) return;
			try {
				// MutationObserver does not cross shadow boundaries. Observe each
				// root separately; periodic discovery also finds attachShadow on
				// an existing host, which emits no document mutation.
				if (rescan || performance.now() - lastScan >= 100) {
					roots = Array.from(liveDOM.roots());
					lastScan = performance.now();
					for (const root of roots) if (!observed.has(root)) {
						observed.add(root);
						const observer = new MutationObserver(() => check(true));
						observer.observe(root, {subtree:true, childList:true, attributes:true, characterData:true});
						observers.push(observer);
					}
				}
				for (const root of roots) for (const element of root.querySelectorAll(selector)) {
					if (liveDOM.visible(element)) { finish(); return; }
				}
			} catch (error) { finish(error); }
		};
		const poll = () => { check(); if (!done) frame = requestAnimationFrame(poll); };
		timer = setTimeout(() => finish(new Error('wait selector timed out')), `+strconv.FormatInt(timeout.Milliseconds(), 10)+`);
		check(true);
		if (!done) frame = requestAnimationFrame(poll);
	})`)
	if err != nil {
		return fmt.Errorf("wait %q: %w", selector, err)
	}
	return b.refreshLiveContext(ctx)
}

func (b *Browser) closeLive() {
	l := b.live
	if l == nil {
		return
	}
	b.live = nil
	l.cancel()
	<-l.done
	l.session.Close()
	if l.session.proc != nil {
		_ = l.session.proc.Wait()
	}
}

// RenderProcessID allows diagnostics to account for Chromium separately from Go.
func (b *Browser) RenderProcessID() int {
	if b.live == nil {
		return 0
	}
	return b.live.session.ProcessID()
}
