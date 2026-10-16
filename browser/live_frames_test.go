package browser

import (
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func frameAPICommand(t *testing.T, b *Browser, command Command) Result {
	t.Helper()
	result := b.Execute(command)
	if !result.OK {
		t.Fatalf("command %+v: %s", command, result.Error)
	}
	return result
}

func frameAPIList(t *testing.T, b *Browser) []Frame {
	t.Helper()
	return frameAPICommand(t, b, Command{Cmd: "frames"}).Frames
}

func frameAPINamed(t *testing.T, frames []Frame, name string) Frame {
	t.Helper()
	for _, frame := range frames {
		if frame.Name == name {
			return frame
		}
	}
	t.Fatalf("frame %q missing from %+v", name, frames)
	return Frame{}
}

func frameAPIElement(t *testing.T, b *Browser, selector string) Element {
	t.Helper()
	result := frameAPICommand(t, b, Command{Cmd: "query", Selector: selector})
	if len(result.Matches) != 1 || result.Matches[0].Ref == "" {
		t.Fatalf("query %q: %+v", selector, result.Matches)
	}
	return result.Matches[0]
}

func frameAPISelect(t *testing.T, b *Browser, ref string) *Page {
	t.Helper()
	result := frameAPICommand(t, b, Command{Cmd: "frame", Ref: ref})
	if result.Page == nil || result.Page.FrameRef != ref {
		t.Fatalf("selected frame %q: %+v", ref, result.Page)
	}
	return result.Page
}

func frameAPIAssertSelected(t *testing.T, frames []Frame, ref string) {
	t.Helper()
	count := 0
	for _, frame := range frames {
		if frame.Selected {
			count++
			if frame.Ref != ref {
				t.Fatalf("selected %q instead of %q: %+v", frame.Ref, ref, frames)
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected one selected frame %q: %+v", ref, frames)
	}
}

func TestLiveFrameAPIScopeStatusRefsAndHistory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/previous":
			fmt.Fprint(w, `<title>Previous top page</title><p>Previous document</p>`)
		case "/child":
			fmt.Fprint(w, `<title>Child page</title><p>Child only</p><input id="q" value="child"><button id="send" type="button">Child action</button>`)
		case "/missing":
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `<title>Missing child</title><p>Child status 404</p>`)
		default:
			fmt.Fprint(w, `<title>Main page</title><p>Main only</p><input id="q" value="main"><button id="send" type="button">Main action</button><iframe name="child" src="/child"></iframe><iframe name="missing" src="/missing"></iframe>`)
		}
	}))
	t.Cleanup(server.Close)
	b := newLiveTestBrowser(t)
	frameAPICommand(t, b, Command{Cmd: "open", URL: server.URL + "/previous", RenderJS: true})
	opened := frameAPICommand(t, b, Command{Cmd: "open", URL: server.URL + "/main", RenderJS: true})
	if opened.Page == nil || opened.Page.FrameRef != "fr1" || opened.Page.Status != http.StatusOK {
		t.Fatalf("main observation=%+v", opened.Page)
	}
	mainInput := frameAPIElement(t, b, "#q")
	mainButton := frameAPIElement(t, b, "#send")
	beforeList := b.Current
	frames := frameAPIList(t, b)
	if b.Current != beforeList {
		t.Fatal("listing frames replaced the current observation")
	}
	frameAPIAssertSelected(t, frames, "fr1")
	child, missing := frameAPINamed(t, frames, "child"), frameAPINamed(t, frames, "missing")
	history := append([]string(nil), b.History...)
	page := frameAPISelect(t, b, child.Ref)
	if page.Title != "Child page" || page.Status != http.StatusOK || page.URL != server.URL+"/child" || strings.Contains(page.Text, "Main only") {
		t.Fatalf("child observation=%+v", page)
	}
	frameAPIAssertSelected(t, frameAPIList(t, b), child.Ref)
	childInput, childButton := frameAPIElement(t, b, "#q"), frameAPIElement(t, b, "#send")
	if childInput.Ref == mainInput.Ref || childButton.Ref == mainButton.Ref {
		t.Fatalf("refs crossed scopes: main=%+v/%+v child=%+v/%+v", mainInput, mainButton, childInput, childButton)
	}
	if result := b.Execute(Command{Cmd: "fill", Ref: mainInput.Ref, Value: "wrong scope"}); result.OK || result.Error == "" {
		t.Fatalf("old main reference accepted in child: %+v", result)
	}
	frameAPICommand(t, b, Command{Cmd: "fill", Ref: childInput.Ref, Value: "edited child"})
	if got := frameAPIElement(t, b, "#q").Value; got != "edited child" {
		t.Fatalf("child value=%q", got)
	}
	page = frameAPISelect(t, b, missing.Ref)
	if page.Status != http.StatusNotFound || page.Title != "Missing child" {
		t.Fatalf("child status used the main status: %+v", page)
	}
	page = frameAPISelect(t, b, "fr1")
	if page.URL != server.URL+"/main" || page.Status != http.StatusOK || frameAPIElement(t, b, "#q").Value != "main" {
		t.Fatalf("main state changed through child: %+v", page)
	}
	if result := b.Execute(Command{Cmd: "fill", Ref: childInput.Ref, Value: "wrong parent"}); result.OK {
		t.Fatal("old child reference accepted in main")
	}
	if !reflect.DeepEqual(b.History, history) {
		t.Fatalf("frame selection changed history: got=%v want=%v", b.History, history)
	}
	back := frameAPICommand(t, b, Command{Cmd: "back"})
	if back.Page == nil || back.Page.URL != server.URL+"/previous" || back.Page.FrameRef != "fr1" {
		t.Fatalf("back used child URL: %+v", back)
	}
}

func TestLiveFrameAPINativeClickGeometryAndParentOverlay(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/outer":
			fmt.Fprint(w, `<!doctype html><style>body{margin:0}button{margin:25px 35px;width:140px;height:40px}iframe{display:block;margin:25px 35px;border:5px solid green;width:290px;height:150px}</style><button id="outer" onclick="document.querySelector('#outer-state').textContent='Outer trusted: '+event.isTrusted">Outer</button><p id="outer-state">Outer initial</p><iframe name="nested" src="/nested"></iframe>`)
		case "/nested":
			fmt.Fprint(w, `<!doctype html><style>body{margin:0}button{margin:15px 20px;width:140px;height:45px}</style><button id="nested" onclick="window.clicks=(window.clicks||0)+1;document.querySelector('#nested-state').textContent='Nested trusted: '+event.isTrusted+' count: '+window.clicks">Nested</button><p id="nested-state">Nested initial</p>`)
		default:
			fmt.Fprint(w, `<!doctype html><style>body{margin:0}#embed{position:absolute;left:135px;top:110px;width:500px;height:430px;border:8px solid navy}#veil{display:none;position:absolute;left:130px;top:105px;width:530px;height:460px;z-index:9;background:#ddd}#cover,#uncover{position:relative;z-index:10}</style><button id="cover" onclick="document.querySelector('#veil').style.display='block'">Cover</button><button id="uncover" onclick="document.querySelector('#veil').style.display='none'">Uncover</button><iframe id="embed" name="outer" src="/outer"></iframe><div id="veil">Parent overlay</div>`)
		}
	}))
	t.Cleanup(server.Close)
	b := newLiveTestBrowser(t)
	frameAPICommand(t, b, Command{Cmd: "open", URL: server.URL, RenderJS: true})
	frames := frameAPIList(t, b)
	outer, nested := frameAPINamed(t, frames, "outer"), frameAPINamed(t, frames, "nested")
	if nested.ParentRef != outer.Ref {
		t.Fatalf("nested ancestry=%+v", frames)
	}
	frameAPISelect(t, b, outer.Ref)
	frameAPICommand(t, b, Command{Cmd: "press", Ref: frameAPIElement(t, b, "#outer").Ref})
	if !strings.Contains(b.Current.Text, "Outer trusted: true") {
		t.Fatalf("offset/border click missed outer button: %q", b.Current.Text)
	}
	frameAPISelect(t, b, nested.Ref)
	frameAPICommand(t, b, Command{Cmd: "press", Ref: frameAPIElement(t, b, "#nested").Ref})
	if !strings.Contains(b.Current.Text, "Nested trusted: true count: 1") {
		t.Fatalf("nested click missed button: %q", b.Current.Text)
	}
	frameAPISelect(t, b, "fr1")
	frameAPICommand(t, b, Command{Cmd: "press", Ref: frameAPIElement(t, b, "#cover").Ref})
	frameAPISelect(t, b, nested.Ref)
	blocked := b.Execute(Command{Cmd: "press", Ref: frameAPIElement(t, b, "#nested").Ref})
	if blocked.OK || blocked.Error == "" {
		t.Fatalf("parent overlay did not block frame click: %+v", blocked)
	}
	state := frameAPICommand(t, b, Command{Cmd: "query", Selector: "#nested-state"})
	if len(state.Matches) != 1 || state.Matches[0].Text != "Nested trusted: true count: 1" {
		t.Fatalf("covered child received click: %+v", state.Matches)
	}
	frameAPISelect(t, b, "fr1")
	frameAPICommand(t, b, Command{Cmd: "press", Ref: frameAPIElement(t, b, "#uncover").Ref})
	frameAPISelect(t, b, nested.Ref)
	frameAPICommand(t, b, Command{Cmd: "press", Ref: frameAPIElement(t, b, "#nested").Ref})
	if !strings.Contains(b.Current.Text, "Nested trusted: true count: 2") {
		t.Fatalf("uncovered child click failed: %q", b.Current.Text)
	}
}

func TestLiveFrameAPICrossOriginNavigationAndDetach(t *testing.T) {
	var mainURL, otherOrigin string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/cross":
			fmt.Fprintf(w, `<title>Cross child</title><input id="q" value="remote"><a id="navigate" href="%s/next-frame">Navigate child</a>`, mainURL)
		case "/next-frame":
			fmt.Fprint(w, `<title>Navigated child</title><p>Child changed origin</p><input id="q" value="new document"><button id="detach" onclick="parent.postMessage('detach-child','*')">Detach selected child</button>`)
		default:
			fmt.Fprintf(w, `<title>Cross main</title><p>Main document remains</p><input id="q" value="main"><iframe id="embed" name="cross" src="%s/cross" style="margin:80px;border:6px solid navy;width:400px;height:250px"></iframe><script>addEventListener('message',event=>{if(event.data==='detach-child')document.querySelector('#embed').remove()})</script>`, otherOrigin)
		}
	}))
	t.Cleanup(server.Close)
	mainURL = server.URL
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	u.Host = "localhost:" + u.Port()
	otherOrigin = u.String()
	b := newLiveTestBrowser(t)
	b.AllowAll = true
	frameAPICommand(t, b, Command{Cmd: "open", URL: mainURL, RenderJS: true})
	mainInput := frameAPIElement(t, b, "#q")
	child := frameAPINamed(t, frameAPIList(t, b), "cross")
	if !child.Allowed || child.URL != otherOrigin+"/cross" {
		t.Fatalf("cross-site child=%+v", child)
	}
	frameAPISelect(t, b, child.Ref)
	oldChildInput := frameAPIElement(t, b, "#q")
	if oldChildInput.Ref == mainInput.Ref {
		t.Fatal("cross-origin field reused parent reference")
	}
	frameAPICommand(t, b, Command{Cmd: "fill", Ref: oldChildInput.Ref, Value: "remote edited"})
	if frameAPIElement(t, b, "#q").Value != "remote edited" {
		t.Fatal("cross-origin fill did not reach selected document")
	}
	frameAPICommand(t, b, Command{Cmd: "click", Ref: frameAPIElement(t, b, "#navigate").Ref})
	frames := frameAPIList(t, b)
	navigated := frameAPINamed(t, frames, "cross")
	if navigated.ID != child.ID || navigated.Ref != child.Ref || navigated.URL != mainURL+"/next-frame" {
		t.Fatalf("frame identity changed across process swap: before=%+v after=%+v", child, navigated)
	}
	page := frameAPISelect(t, b, child.Ref)
	if page.Title != "Navigated child" || page.URL != mainURL+"/next-frame" {
		t.Fatalf("new frame document=%+v", page)
	}
	if result := b.Execute(Command{Cmd: "fill", Ref: oldChildInput.Ref, Value: "stale"}); result.OK {
		t.Fatal("old document reference accepted after child navigation")
	}
	snapshot := frameAPICommand(t, b, Command{Cmd: "snapshot"})
	if snapshot.Page == nil || snapshot.Page.FrameRef != child.Ref || snapshot.Page.Title != "Navigated child" {
		t.Fatalf("snapshot escaped frame scope: %+v", snapshot)
	}
	detached := b.Execute(Command{Cmd: "press", Ref: frameAPIElement(t, b, "#detach").Ref})
	if detached.OK && detached.Page != nil && detached.Page.FrameRef == "fr1" {
		t.Fatal("detaching selected frame silently returned main")
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		frames = frameAPIList(t, b)
		present := false
		for _, frame := range frames {
			present = present || frame.ID == child.ID
		}
		if !present {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("selected frame was not removed: %+v", frames)
		}
		time.Sleep(10 * time.Millisecond)
	}
	stale := b.Execute(Command{Cmd: "snapshot"})
	if stale.OK || stale.Error == "" || (stale.Page != nil && stale.Page.FrameRef == "fr1") {
		t.Fatalf("detached frame snapshot silently fell back: %+v", stale)
	}
	if result := b.Execute(Command{Cmd: "frame", Ref: child.Ref}); result.OK {
		t.Fatal("detached frame could be selected")
	}
	page = frameAPISelect(t, b, "fr1")
	if page.URL != mainURL+"/" || page.Title != "Cross main" || frameAPIElement(t, b, "#q").Value != "main" || len(b.History) != 0 {
		t.Fatalf("child navigation altered top page/history: page=%+v history=%v", page, b.History)
	}
}

func TestLiveFrameAPIDuplicateURLsKeepStableReferences(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/shared" {
			fmt.Fprint(w, `<title>Shared document</title><input id="q"><script>document.querySelector('#q').value=window.name</script>`)
			return
		}
		fmt.Fprint(w, `<title>Duplicate frame URLs</title><button id="insert" onclick="const frame=document.createElement('iframe');frame.name='inserted';frame.src='/shared';document.body.insertBefore(frame,document.querySelector('iframe'))">Insert before</button><iframe name="first" src="/shared"></iframe><iframe name="second" src="/shared"></iframe>`)
	}))
	t.Cleanup(server.Close)
	b := newLiveTestBrowser(t)
	frameAPICommand(t, b, Command{Cmd: "open", URL: server.URL, RenderJS: true})
	frames := frameAPIList(t, b)
	first, second := frameAPINamed(t, frames, "first"), frameAPINamed(t, frames, "second")
	frameAPICommand(t, b, Command{Cmd: "press", Ref: frameAPIElement(t, b, "#insert").Ref})
	frames = frameAPIList(t, b)
	newFirst, newSecond, inserted := frameAPINamed(t, frames, "first"), frameAPINamed(t, frames, "second"), frameAPINamed(t, frames, "inserted")
	if first.ID != newFirst.ID || first.Ref != newFirst.Ref || second.ID != newSecond.ID || second.Ref != newSecond.Ref || inserted.Ref == first.Ref || inserted.Ref == second.Ref {
		t.Fatalf("insertion reassigned frame refs: before=%+v/%+v after=%+v", first, second, frames)
	}
	frameAPISelect(t, b, second.Ref)
	if got := frameAPIElement(t, b, "#q").Value; got != "second" {
		t.Fatalf("same-URL lookup selected wrong frame: %q", got)
	}
	frameAPISelect(t, b, first.Ref)
	if got := frameAPIElement(t, b, "#q").Value; got != "first" {
		t.Fatalf("first frame selected wrong document: %q", got)
	}
}

func TestLiveFrameAPISrcdocInheritsParentPolicy(t *testing.T) {
	child := `<title>Inline child</title><p>Inline content</p><input id="q" value="inline"><a id="relative" href="/inline-destination">Relative destination</a>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<title>Srcdoc main</title><iframe name="inline" srcdoc="%s"></iframe>`, html.EscapeString(child))
	}))
	t.Cleanup(server.Close)
	b := newLiveTestBrowser(t)
	frameAPICommand(t, b, Command{Cmd: "open", URL: server.URL, RenderJS: true})
	inline := frameAPINamed(t, frameAPIList(t, b), "inline")
	if inline.URL != "about:srcdoc" || !inline.Allowed {
		t.Fatalf("srcdoc did not inherit permitted parent policy: %+v", inline)
	}
	page := frameAPISelect(t, b, inline.Ref)
	if page.Title != "Inline child" || page.URL != "about:srcdoc" {
		t.Fatalf("srcdoc observation=%+v", page)
	}
	if len(page.Links) != 1 || page.Links[0].URL != server.URL+"/inline-destination" {
		t.Fatalf("srcdoc link lost document.baseURI: %+v", page.Links)
	}
	frameAPICommand(t, b, Command{Cmd: "fill", Ref: frameAPIElement(t, b, "#q").Ref, Value: "edited inline"})
	if frameAPIElement(t, b, "#q").Value != "edited inline" {
		t.Fatal("srcdoc fill missed selected document")
	}
	frameAPISelect(t, b, "fr1")
	if b.Current.URL != server.URL+"/" || len(b.History) != 0 {
		t.Fatalf("srcdoc corrupted top URL/history: %+v %v", b.Current, b.History)
	}
	frameAPISelect(t, b, inline.Ref)
	frameAPICommand(t, b, Command{Cmd: "open", URL: server.URL + "/next-top", RenderJS: true})
	if b.Current.FrameRef != "fr1" || !reflect.DeepEqual(b.History, []string{server.URL + "/"}) {
		t.Fatalf("open from child saved child URL or retained scope: page=%+v history=%v", b.Current, b.History)
	}
	frameAPICommand(t, b, Command{Cmd: "back"})
	if b.Current.URL != server.URL+"/" || b.Current.FrameRef != "fr1" || len(b.History) != 0 {
		t.Fatalf("back after open from child lost top page: page=%+v history=%v", b.Current, b.History)
	}
}

func TestLiveFrameAPIRejectsDeniedHost(t *testing.T) {
	var remoteURL string
	var deniedHits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/denied" {
			deniedHits.Add(1)
			fmt.Fprint(w, `<title>Denied child</title><input id="secret" value="must not observe">`)
			return
		}
		fmt.Fprintf(w, `<title>Allowed main</title><p>Main remains usable</p><iframe name="denied" src="%s/denied"></iframe>`, remoteURL)
	}))
	t.Cleanup(server.Close)
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	u.Host = "localhost:" + u.Port()
	remoteURL = u.String()
	b := newLiveTestBrowser(t)
	frameAPICommand(t, b, Command{Cmd: "open", URL: server.URL, RenderJS: true})
	current := b.Current
	denied := frameAPINamed(t, frameAPIList(t, b), "denied")
	if denied.Allowed || deniedHits.Load() != 0 {
		t.Fatalf("forbidden iframe fetched or marked permitted: frame=%+v hits=%d", denied, deniedHits.Load())
	}
	result := b.Execute(Command{Cmd: "frame", Ref: denied.Ref})
	if result.OK || result.Error == "" || b.Current != current || strings.Contains(b.Current.Text, "must not observe") {
		t.Fatalf("forbidden frame selection changed observation: %+v", result)
	}
	frameAPIAssertSelected(t, frameAPIList(t, b), "fr1")
}
