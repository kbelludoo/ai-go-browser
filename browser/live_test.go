package browser

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newLiveTestBrowser(t *testing.T) *Browser {
	t.Helper()
	path, err := FindSystemBrowser()
	if err != nil {
		t.Skip("live integration requires installed Chromium/Edge")
	}
	b := New([]string{"127.0.0.1"}, false)
	b.SystemBrowser = path
	b.OperationTimeout = 10 * time.Second
	t.Cleanup(b.Close)
	return b
}

func liveTestField(t *testing.T, b *Browser, name string) string {
	t.Helper()
	for ref, node := range b.Document.fields {
		if node.prop("name") == name {
			return ref
		}
	}
	t.Fatalf("field %q not found", name)
	return ""
}

func liveTestButton(t *testing.T, b *Browser, label string) string {
	t.Helper()
	for _, button := range b.Current.Buttons {
		if button.Label == label {
			return button.Ref
		}
	}
	t.Fatalf("button %q not found", label)
	return ""
}

func liveTestLink(t *testing.T, b *Browser, label string) string {
	t.Helper()
	for _, link := range b.Current.Links {
		if link.Label == label {
			return link.Ref
		}
	}
	t.Fatalf("link %q not found", label)
	return ""
}

func runLiveTestCommand(t *testing.T, b *Browser, c Command) {
	t.Helper()
	handled, err := b.executeLive(c)
	if !handled || err != nil {
		t.Fatalf("live command %+v: handled=%v error=%v", c, handled, err)
	}
}

func TestLivePersistentSPAControlsAndHydration(t *testing.T) {
	hydrate := make(chan struct{})
	var hydrated atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hydrate" {
			<-hydrate
			hydrated.Store(true)
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "Hydrated asynchronously")
			return
		}
		cookie, _ := r.Cookie("go_session")
		if cookie == nil || cookie.Value != "from_go" {
			t.Error("Go session cookie did not reach Chromium")
		}
		http.SetCookie(w, &http.Cookie{Name: "chrome_session", Value: "from_chrome", Path: "/", HttpOnly: true})
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `<!doctype html><title>Live SPA</title><p data-state>Initial</p>
			<button type="button" onclick="window.counter=(window.counter||0)+1;document.querySelector('[data-state]').textContent='Count: '+window.counter">Increase</button>
			<form onsubmit="event.preventDefault();document.querySelector('[data-state]').textContent='Submitted: '+this.elements.q.value">
				<input name="q"><textarea name="notes"></textarea>
				<select name="kind"><option value="a">A</option><option value="b">B</option></select>
				<input name="agree" type="checkbox" value="yes"><button type="submit">Send</button>
			</form><input name="standalone"><input name="immutable" readonly value="locked">
			<script>fetch('/hydrate').then(r=>r.text()).then(text=>{let e=document.createElement('h2');e.className='hydrated';e.textContent=text;document.body.append(e)})</script>`)
	}))
	defer srv.Close()
	b := newLiveTestBrowser(t)
	u, _ := url.Parse(srv.URL)
	b.client.Jar.SetCookies(u, []*http.Cookie{{Name: "go_session", Value: "from_go"}})
	if err := b.openLive(srv.URL, true); err != nil {
		close(hydrate)
		t.Fatal(err)
	}
	if b.Current.Status != http.StatusCreated {
		close(hydrate)
		t.Fatalf("HTTP status = %d, want 201", b.Current.Status)
	}
	pid := b.RenderProcessID()
	if pid <= 0 {
		close(hydrate)
		t.Fatal("missing live process ID")
	}
	for _, cookie := range b.client.Jar.Cookies(u) {
		if cookie.Name == "chrome_session" && cookie.Value == "from_chrome" {
			goto cookiesOK
		}
	}
	close(hydrate)
	t.Fatal("Chromium cookie did not reach Go jar")
cookiesOK:
	// Explicitly release delayed hydration after the initial navigation completes.
	close(hydrate)
	if err := b.liveWait("h2.hydrated", 3*time.Second); err != nil {
		t.Fatal(err)
	}
	if !hydrated.Load() || !strings.Contains(b.Current.Text, "Hydrated asynchronously") {
		t.Fatalf("hydrated snapshot: %q", b.Current.Text)
	}
	for count := 1; count <= 2; count++ {
		runLiveTestCommand(t, b, Command{Cmd: "press", Ref: liveTestButton(t, b, "Increase")})
		if !strings.Contains(b.Current.Text, fmt.Sprintf("Count: %d", count)) {
			t.Fatalf("persistent event state lost: %q", b.Current.Text)
		}
	}
	// Retain a reference from the last snapshot while the SPA inserts a button
	// before it. The command must still address the original real element.
	oldRef := liveTestButton(t, b, "Increase")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	_, err := b.live.session.Evaluate(ctx, `(() => {
		const element=document.createElement('button');element.type='button';element.textContent='Inserted';
		element.onclick=()=>document.querySelector('[data-state]').textContent='Wrong element';
		document.body.prepend(element);
	})()`)
	cancel()
	if err != nil {
		t.Fatal(err)
	}
	runLiveTestCommand(t, b, Command{Cmd: "press", Ref: oldRef})
	if !strings.Contains(b.Current.Text, "Count: 3") {
		t.Fatalf("stale numbering selected the wrong element: %q", b.Current.Text)
	}
	runLiveTestCommand(t, b, Command{Cmd: "fill", Ref: liveTestField(t, b, "q"), Value: "IA em Go"})
	runLiveTestCommand(t, b, Command{Cmd: "fill", Ref: liveTestField(t, b, "notes"), Value: "Atualizado"})
	runLiveTestCommand(t, b, Command{Cmd: "fill", Ref: liveTestField(t, b, "kind"), Value: "b"})
	runLiveTestCommand(t, b, Command{Cmd: "fill", Ref: liveTestField(t, b, "standalone"), Value: "Sem formulário"})
	checked := true
	runLiveTestCommand(t, b, Command{Cmd: "check", Ref: liveTestField(t, b, "agree"), Checked: &checked})
	matches, err := b.liveQuery("select[name=kind], input[name=standalone], input[name=agree]:checked")
	if err != nil || len(matches) != 3 {
		t.Fatalf("full CSS/control query = %+v, error %v", matches, err)
	}
	if matches[0].Value != "b" || matches[2].Value != "Sem formulário" {
		t.Fatalf("control values not reflected: %+v", matches)
	}
	if matches[0].Ref == "" || matches[0].Type != "select" || len(matches[0].Options) != 2 || matches[1].Checked == nil || !*matches[1].Checked {
		t.Fatalf("query omitted actionable control metadata: %+v", matches)
	}
	if _, err := b.executeLive(Command{Cmd: "fill", Ref: liveTestField(t, b, "agree"), Value: "yes"}); err == nil {
		t.Fatal("fill must not edit checkbox state")
	}
	if _, err := b.executeLive(Command{Cmd: "fill", Ref: liveTestField(t, b, "kind"), Value: "not-an-option"}); err == nil {
		t.Fatal("select accepted an unknown option")
	}
	if _, err := b.executeLive(Command{Cmd: "fill", Ref: liveTestField(t, b, "immutable"), Value: "changed"}); err == nil {
		t.Fatal("readonly live control was edited")
	}
	runLiveTestCommand(t, b, Command{Cmd: "submit", Ref: "f1"})
	if !strings.Contains(b.Current.Text, "Submitted: IA em Go") {
		t.Fatalf("real submit handler did not run: %q", b.Current.Text)
	}
	if b.RenderProcessID() != pid {
		t.Fatal("Chromium was recreated between commands")
	}
	b.closeLive()
	b.closeLive()
	if b.RenderProcessID() != 0 || b.Current == nil || b.Document == nil {
		t.Fatal("release must stop the renderer and preserve the last observation")
	}
}

func TestLiveNamedExternalSubmitterAndMethodOverride(t *testing.T) {
	var posts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/save" {
			if r.Method == http.MethodPost {
				posts.Add(1)
			}
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprintf(w, "<title>Saved</title><p>%s %s %s</p>", r.Method, r.Form.Get("q"), r.Form.Get("operation"))
			return
		}
		fmt.Fprint(w, `<title>External submitter</title>
			<form id="entry" method="get"><input name="q"></form>
			<button form="entry" name="operation" value="publish" formmethod="post" formaction="/save">Publish</button>`)
	}))
	defer srv.Close()
	b := newLiveTestBrowser(t)
	if err := b.openLive(srv.URL, true); err != nil {
		t.Fatal(err)
	}
	runLiveTestCommand(t, b, Command{Cmd: "fill", Ref: liveTestField(t, b, "q"), Value: "article"})
	for _, command := range []string{"submit", "press"} {
		ref := "f1"
		if command == "press" {
			ref = liveTestButton(t, b, "Publish")
		}
		if _, err := b.executeLive(Command{Cmd: command, Ref: ref, Confirm: true}); err == nil {
			t.Fatalf("%s bypassed POST formmethod override", command)
		}
	}
	if posts.Load() != 0 {
		t.Fatal("external POST sent before authorization")
	}
	b.AllowPost = true
	runLiveTestCommand(t, b, Command{Cmd: "submit", Ref: "f1", Confirm: true})
	if !strings.Contains(b.Current.Text, "POST article publish") || b.Current.Status != http.StatusAccepted {
		t.Fatalf("named external submitter/status = %+v", b.Current)
	}
	runLiveTestCommand(t, b, Command{Cmd: "back"})
	matches, err := b.liveQuery("input[name=q]")
	if err != nil || len(matches) != 1 || matches[0].Value != "article" {
		t.Fatalf("native back lost input state: %+v, %v", matches, err)
	}
	// Inspect actual properties when deciding permission, even if the DOM changed
	// after the observation used to choose this reference.
	buttonRef := liveTestButton(t, b, "Publish")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	_, err = b.live.session.Evaluate(ctx, `document.querySelector('button').formMethod='get'`)
	cancel()
	if err != nil {
		t.Fatal(err)
	}
	b.AllowPost = false
	runLiveTestCommand(t, b, Command{Cmd: "press", Ref: buttonRef})
	if !strings.Contains(b.Current.Text, "GET article publish") || posts.Load() != 1 {
		t.Fatalf("GET override ignored: %+v, posts %d", b.Current, posts.Load())
	}
}

func TestLiveNavigationRedirectStatusHistoryAndPolicy(t *testing.T) {
	var blockedVisits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/gone":
			w.WriteHeader(http.StatusGone)
			fmt.Fprint(w, "<title>Gone</title><p>Resource removed</p>")
		case "/redirect":
			http.Redirect(w, r, "/gone", http.StatusFound)
		case "/blocked":
			blockedVisits.Add(1)
			fmt.Fprint(w, "Should not be requested")
		case "/escape":
			http.Redirect(w, r, "http://localhost:"+strings.Split(r.Host, ":")[1]+"/blocked", http.StatusFound)
		default:
			fmt.Fprint(w, `<title>Navigation</title><p>First page</p><a href="/redirect">Redirect</a><a href="/escape">Escape</a>`)
		}
	}))
	defer srv.Close()
	b := newLiveTestBrowser(t)
	if err := b.openLive(srv.URL, true); err != nil {
		t.Fatal(err)
	}
	runLiveTestCommand(t, b, Command{Cmd: "click", Ref: liveTestLink(t, b, "Redirect")})
	if b.Current.URL != srv.URL+"/gone" || b.Current.Status != http.StatusGone {
		t.Fatalf("final URL/status = %s/%d", b.Current.URL, b.Current.Status)
	}
	runLiveTestCommand(t, b, Command{Cmd: "back"})
	if b.Current.URL != srv.URL+"/" || b.Current.Status != http.StatusOK {
		t.Fatalf("back URL/status = %s/%d", b.Current.URL, b.Current.Status)
	}
	_, err := b.executeLive(Command{Cmd: "click", Ref: liveTestLink(t, b, "Escape")})
	if err == nil || !strings.Contains(err.Error(), "not on allowlist") {
		b.live.mu.Lock()
		failure, currentURL := b.live.failure, b.live.pageURL
		b.live.mu.Unlock()
		t.Fatalf("redirect policy error = %v (event failure %v; URL %s)", err, failure, currentURL)
	}
	if blockedVisits.Load() != 0 {
		t.Fatal("unauthorized redirect destination was requested")
	}
}

func TestLivePOSTConfirmationAndSessionContext(t *testing.T) {
	var submits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			submits.Add(1)
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			fmt.Fprintf(w, "<title>Saved</title><p>Saved %s</p>", r.Form.Get("q"))
			return
		}
		fmt.Fprint(w, `<title>Post</title><form method="post"><input name="q"><button>Save</button></form>`)
	}))
	defer srv.Close()
	b := newLiveTestBrowser(t)
	if err := b.openLive(srv.URL, true); err != nil {
		t.Fatal(err)
	}
	// openLive's operation context is already canceled here. The session survives.
	runLiveTestCommand(t, b, Command{Cmd: "fill", Ref: liveTestField(t, b, "q"), Value: "with confirmation"})
	for _, command := range []string{"submit", "press"} {
		ref := "f1"
		if command == "press" {
			ref = liveTestButton(t, b, "Save")
		}
		if _, err := b.executeLive(Command{Cmd: command, Ref: ref}); err == nil {
			t.Fatalf("%s bypassed POST confirmation", command)
		}
	}
	if submits.Load() != 0 {
		t.Fatal("POST sent before confirmation")
	}
	b.AllowPost = true
	if _, err := b.executeLive(Command{Cmd: "submit", Ref: "f1"}); err == nil {
		t.Fatal("allow-post bypassed confirm")
	}
	runLiveTestCommand(t, b, Command{Cmd: "submit", Ref: "f1", Confirm: true})
	if submits.Load() != 1 || !strings.Contains(b.Current.Text, "Saved with confirmation") {
		t.Fatalf("confirmed POST snapshot = %q; requests = %d", b.Current.Text, submits.Load())
	}
}

func TestLiveWaitAndQueryValidation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<title>Queries</title><form><input type="password" name="secret" value="sensitive"><input type="hidden" name="token" value="token-value"></form>`)
	}))
	defer srv.Close()
	b := newLiveTestBrowser(t)
	if err := b.openLive(srv.URL, true); err != nil {
		t.Fatal(err)
	}
	matches, err := b.liveQuery("input")
	if err != nil || len(matches) != 2 {
		t.Fatalf("query = %+v, %v", matches, err)
	}
	for _, match := range matches {
		if match.Value != "[redacted]" {
			t.Fatalf("live query exposed control value: %+v", match)
		}
	}
	if _, err := b.liveQuery("["); err == nil {
		t.Fatal("invalid CSS selector accepted")
	}
	if err := b.liveWait(".missing", 100*time.Millisecond); err == nil {
		t.Fatal("missing selector did not time out")
	}
	if err := b.refreshLive(); err != nil {
		t.Fatalf("wait timeout poisoned live session: %v", err)
	}
}

func TestLiveSettleCanceled(t *testing.T) {
	// A navigation wait remains bounded even when no browser events arrive.
	l := &liveSession{changed: make(chan struct{}), done: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := l.waitNavigation(ctx, 0); err == nil {
		t.Fatal("canceled wait succeeded")
	}
}

func TestLiveNavigationTimeoutRecoveryAndRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			<-r.Context().Done()
			return
		}
		fmt.Fprintf(w, "<title>%s</title><p>Ready</p>", r.URL.Path)
	}))
	defer srv.Close()
	b := newLiveTestBrowser(t)
	if err := b.openLive(srv.URL+"/first", true); err != nil {
		t.Fatal(err)
	}
	previous, session := b.Current, b.live
	pid := b.RenderProcessID()
	b.OperationTimeout = 150 * time.Millisecond
	if err := b.openLive(srv.URL+"/slow", true); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("slow navigation error = %v", err)
	}
	if b.Current != previous || b.RenderProcessID() != pid {
		t.Fatal("failed navigation replaced observation or renderer")
	}
	b.OperationTimeout = 5 * time.Second
	b.resource = &Resource{URL: "previous typed resource"}
	if err := b.openLive(srv.URL+"/recovered", true); err != nil {
		t.Fatalf("recover navigation: %v", err)
	}
	if b.Current.Title != "/recovered" || b.RenderProcessID() != pid || b.resource != nil {
		t.Fatal("navigation did not recover in the same session")
	}
	previous, document := b.Current, b.Document
	if result := b.Execute(Command{Cmd: "release"}); !result.OK {
		t.Fatal(result.Error)
	}
	if b.RenderProcessID() != 0 || b.Current != previous || b.Document != document {
		t.Fatal("release lost the last observation")
	}
	if err := b.openLive(srv.URL+"/new", true); err != nil {
		t.Fatal(err)
	}
	if b.live == session || b.RenderProcessID() <= 0 || b.Current.Title != "/new" {
		t.Fatal("render after release did not create a fresh session")
	}
}

func TestLiveDisabledFieldsetAndLegend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<title>Fieldset</title><form><fieldset disabled>
			<legend><input name="legend" value="editable"></legend><input name="blocked" value="original">
			<button type="button">Disabled button</button></fieldset></form>`)
	}))
	defer srv.Close()
	b := newLiveTestBrowser(t)
	if err := b.openLive(srv.URL, true); err != nil {
		t.Fatal(err)
	}
	runLiveTestCommand(t, b, Command{Cmd: "fill", Ref: liveTestField(t, b, "legend"), Value: "updated"})
	if _, err := b.executeLive(Command{Cmd: "fill", Ref: liveTestField(t, b, "blocked"), Value: "changed"}); err == nil {
		t.Fatal("disabled fieldset control was edited")
	}
	if _, err := b.executeLive(Command{Cmd: "press", Ref: liveTestButton(t, b, "Disabled button")}); err == nil {
		t.Fatal("disabled fieldset button was pressed")
	}
	matches, err := b.liveQuery("input[name=blocked]")
	if err != nil || len(matches) != 1 || !matches[0].Disabled || matches[0].Value != "original" {
		t.Fatalf("disabled live query = %+v, %v", matches, err)
	}
}
