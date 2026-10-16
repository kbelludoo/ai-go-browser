package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func newFrameContextTestSession(t *testing.T, conn *cdpConn, ctx context.Context) *liveSession {
	t.Helper()
	l := &liveSession{session: &StealthSession{pageCDP: conn}, frameID: "main", frames: newLiveFrameController(), changed: make(chan struct{}), done: make(chan struct{})}
	l.frames.states["main"] = &liveFrameState{}
	l.frames.states["child"] = &liveFrameState{parent: "main"}
	life, cancel := context.WithCancel(ctx)
	go func() {
		defer close(l.done)
		for {
			select {
			case <-life.Done():
				return
			case msg := <-conn.events:
				l.frameEvent(life, conn, msg)
			}
		}
	}()
	t.Cleanup(func() { cancel(); <-l.done })
	return l
}

func frameContextEvent(session, frame string, id int64, unique string) cdpMessage {
	return cdpMessage{Method: "Runtime.executionContextCreated", Session: session, Params: json.RawMessage(fmt.Sprintf(`{"context":{"id":%d,"uniqueId":%q,"name":"ai-go-browser-frame","auxData":{"frameId":%q}}}`, id, unique, frame))}
}

func TestLiveFrameContextWaitsForUniqueAndPreservesSelection(t *testing.T) {
	responded, release := make(chan struct{}), make(chan struct{})
	conn, ctx := newCDPSessionTestConn(t, func(ctx context.Context, ws *websocket.Conn) {
		request, ok := readCDPSessionTestMessage(t, ctx, ws)
		if !ok {
			return
		}
		if request.Method != "Page.createIsolatedWorld" || request.Session != "remote" || strings.Contains(string(request.Params), "grantUniveralAccess") {
			t.Errorf("unsafe/misrouted world request: %+v", request)
		}
		writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: request.ID, Session: request.Session, Result: json.RawMessage(`{"executionContextId":42}`)})
		close(responded)
		select {
		case <-release:
		case <-ctx.Done():
			return
		}
		writeCDPSessionTestMessage(t, ctx, ws, frameContextEvent("remote", "nested", 42, "unique-nested"))
		request, ok = readCDPSessionTestMessage(t, ctx, ws)
		if !ok {
			return
		}
		var params map[string]any
		_ = json.Unmarshal(request.Params, &params)
		if request.Method != "Runtime.evaluate" || params["uniqueContextId"] != "unique-nested" || params["contextId"] != nil {
			t.Errorf("evaluation did not pin unique context: %+v", request)
		}
		writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: request.ID, Session: request.Session, Result: json.RawMessage(`{"result":{"value":true}}`)})
	})
	l := newFrameContextTestSession(t, conn, ctx)
	l.mu.Lock()
	l.frames.states["remote-root"] = &liveFrameState{parent: "main"}
	l.frames.states["nested"] = &liveFrameState{parent: "remote-root"}
	l.frames.endpoints["remote-root"] = conn.session("remote")
	selected := &liveFrameScope{id: "child", ref: "fr2", prefix: "fr2.d1:"}
	l.frames.selected = selected
	l.mu.Unlock()
	type outcome struct {
		endpoint *cdpSession
		id       int64
		unique   string
		err      error
	}
	done := make(chan outcome, 1)
	go func() { ep, id, unique, err := l.frameWorld(ctx, "nested"); done <- outcome{ep, id, unique, err} }()
	<-responded
	select {
	case result := <-done:
		t.Fatalf("published reusable numeric ID without unique event: %+v", result)
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	result := <-done
	if result.err != nil || result.id != 42 || result.unique != "unique-nested" || result.endpoint.id != "remote" {
		t.Fatalf("world result: %+v", result)
	}
	l.mu.Lock()
	if l.frames.selected != selected || selected.prefix != "fr2.d1:" || len(l.frames.worldByFrame) != 1 || len(l.frames.creating) != 0 {
		t.Error("nonselecting world helper changed scope or retained pending creation")
	}
	l.mu.Unlock()
	if _, err := result.endpoint.EvaluateWorld(ctx, "true", result.id, result.unique); err != nil {
		t.Fatal(err)
	}
	ep, id, unique, err := l.frameWorld(ctx, "nested")
	if err != nil || id != result.id || unique != result.unique || !sameFrameEndpoint(ep, result.endpoint) {
		t.Fatalf("cached world: %v", err)
	}
}

func TestLiveFrameContextRejectsLateWorldAfterNavigation(t *testing.T) {
	conn, ctx := newCDPSessionTestConn(t, func(ctx context.Context, ws *websocket.Conn) {
		request, ok := readCDPSessionTestMessage(t, ctx, ws)
		if !ok {
			return
		}
		writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: request.ID, Result: json.RawMessage(`{"executionContextId":10}`)})
		writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{Method: "Page.frameNavigated", Params: json.RawMessage(`{"frame":{"id":"child","parentId":"main","url":"http://local/new"}}`)})
		writeCDPSessionTestMessage(t, ctx, ws, frameContextEvent("", "child", 10, "old-document"))
		request, ok = readCDPSessionTestMessage(t, ctx, ws)
		if !ok {
			return
		}
		if request.Method != "Page.createIsolatedWorld" {
			t.Errorf("expected replacement world, got %+v", request)
		}
		// Another late event can arrive during the next request. Only the
		// numeric ID in that request's reply identifies its candidate world.
		writeCDPSessionTestMessage(t, ctx, ws, frameContextEvent("", "child", 10, "old-document"))
		writeCDPSessionTestMessage(t, ctx, ws, frameContextEvent("", "child", 11, "new-document"))
		writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: request.ID, Result: json.RawMessage(`{"executionContextId":11}`)})
	})
	l := newFrameContextTestSession(t, conn, ctx)
	_, id, unique, err := l.frameWorld(ctx, "child")
	if err != nil || id != 11 || unique != "new-document" {
		t.Fatalf("used old context: id=%d unique=%q err=%v", id, unique, err)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.frames.worlds) != 1 || len(l.frames.creating) != 0 {
		t.Fatalf("stale candidates retained: %+v", l.frames)
	}
}

func TestLiveFrameContextMissingUniqueHonestTimeout(t *testing.T) {
	conn, ctx := newCDPSessionTestConn(t, func(ctx context.Context, ws *websocket.Conn) {
		request, ok := readCDPSessionTestMessage(t, ctx, ws)
		if !ok {
			return
		}
		writeCDPSessionTestMessage(t, ctx, ws, frameContextEvent("", "child", 42, ""))
		writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: request.ID, Result: json.RawMessage(`{"executionContextId":42}`)})
	})
	l := newFrameContextTestSession(t, conn, ctx)
	deadline, cancel := context.WithTimeout(ctx, 60*time.Millisecond)
	defer cancel()
	_, _, _, err := l.frameWorld(deadline, "child")
	if err == nil || !strings.Contains(err.Error(), "unique context") {
		t.Fatalf("missing unique ID silently fell back: %v", err)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.frames.worlds) != 0 || len(l.frames.worldByFrame) != 0 || len(l.frames.creating) != 0 {
		t.Fatal("timed-out creation retained context")
	}
}

func TestLiveFrameContextReusesWorldAnnouncedBeforeCreation(t *testing.T) {
	conn, ctx := newCDPSessionTestConn(t, func(ctx context.Context, ws *websocket.Conn) {
		request, ok := readCDPSessionTestMessage(t, ctx, ws)
		if !ok {
			return
		}
		// Existing BFCache worlds do not emit another creation event here.
		writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: request.ID, Result: json.RawMessage(`{"executionContextId":42}`)})
	})
	l := newFrameContextTestSession(t, conn, ctx)
	l.frameEvent(ctx, conn, frameContextEvent("", "child", 42, "restored-document"))
	l.frameEvent(ctx, conn, cdpMessage{Method: "Page.frameNavigated", Params: json.RawMessage(`{"frame":{"id":"child","parentId":"main","url":"http://local/restored"}}`)})
	_, id, unique, err := l.frameWorld(ctx, "child")
	if err != nil || id != 42 || unique != "restored-document" {
		t.Fatalf("ignored existing BFCache identity: %d %q %v", id, unique, err)
	}
	l.frameEvent(ctx, conn, cdpMessage{Method: "Runtime.executionContextsCleared"})
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.frames.knownWorlds) != 0 || len(l.frames.worlds) != 0 || len(l.frames.worldByFrame) != 0 {
		t.Fatal("cleared endpoint retained restored identities")
	}
}

func TestLiveFrameActionPinsObservationAndRejectsChangedDocument(t *testing.T) {
	for _, selected := range []bool{false, true} {
		t.Run(fmt.Sprintf("selected-%t", selected), func(t *testing.T) {
			conn, ctx := newCDPSessionTestConn(t, func(ctx context.Context, ws *websocket.Conn) {
				request, ok := readCDPSessionTestMessage(t, ctx, ws)
				if !ok {
					return
				}
				var params map[string]any
				_ = json.Unmarshal(request.Params, &params)
				if request.Method != "Runtime.evaluate" || params["uniqueContextId"] != "observed-document" || params["contextId"] != nil {
					t.Errorf("action changed/recreated context: %+v", request)
				}
				writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: request.ID, Result: json.RawMessage(`{"result":{"value":true}}`)})
			})
			l := newFrameContextTestSession(t, conn, ctx)
			id := "main"
			l.mu.Lock()
			if selected {
				id = "child"
				l.frames.selected = &liveFrameScope{id: id, ref: "fr2", unique: "observed-document", endpoint: conn.session(""), contextID: 42}
			}
			world := &liveFrameWorld{frame: id, unique: "observed-document", contextID: 42, endpoint: conn.session("")}
			l.frames.worlds[worldKey(world.endpoint, 42)] = world
			l.frames.worldByFrame[id] = world
			l.mu.Unlock()
			if err := l.pinActionObservation(); err != nil {
				t.Fatal(err)
			}
			if _, err := l.evaluateAction(ctx, "true"); err != nil {
				t.Fatal(err)
			}
			l.frameEvent(ctx, conn, cdpMessage{Method: "Page.frameNavigated", Params: json.RawMessage(fmt.Sprintf(`{"frame":{"id":%q,"url":"http://local/changed"}}`, id))})
			if _, err := l.evaluateAction(ctx, "true"); err == nil || !strings.Contains(err.Error(), "new snapshot") {
				t.Fatalf("action reinterpreted old reference: %v", err)
			}
			if err := l.pinActionObservation(); err == nil {
				t.Fatal("observation pinned a disappeared context")
			}
			conn.mu.Lock()
			calls := conn.nextID
			conn.mu.Unlock()
			if calls != 1 {
				t.Fatalf("stale action issued %d requests", calls)
			}
		})
	}
}

func TestLiveFrameContextCacheInvalidationAndSessionIsolation(t *testing.T) {
	conn := &cdpConn{}
	l := &liveSession{session: &StealthSession{pageCDP: conn}, frameID: "main", frames: newLiveFrameController(), changed: make(chan struct{})}
	l.frames.states["main"] = &liveFrameState{}
	l.frames.states["child"] = &liveFrameState{parent: "main"}
	l.frames.states["other"] = &liveFrameState{parent: "main"}
	remote := conn.session("remote")
	l.frames.endpoints["child"] = remote
	l.frames.creating["child"] = &liveFrameCreation{endpoint: remote}
	l.frames.creating["other"] = &liveFrameCreation{endpoint: conn.session("")}
	l.frameEvent(context.Background(), conn, frameContextEvent("remote", "child", 42, "remote-unique"))
	l.frameEvent(context.Background(), conn, frameContextEvent("", "other", 42, "local-unique"))
	for id, world := range l.frames.worlds {
		l.frames.worldByFrame[world.frame] = world
		_ = id
	}
	l.frames.selected = &liveFrameScope{id: "child", endpoint: remote, contextID: 42, unique: "remote-unique"}
	l.frameEvent(context.Background(), conn, cdpMessage{Method: "Runtime.executionContextsCleared", Session: "remote"})
	if !l.frames.selected.invalid || len(l.frames.worlds) != 1 || l.frames.worldByFrame["other"] == nil {
		t.Fatal("clear missed selected scope or removed another session's context")
	}
	l.frameEvent(context.Background(), conn, cdpMessage{Method: "Runtime.executionContextDestroyed", Params: json.RawMessage(`{"executionContextId":42}`)})
	if len(l.frames.worlds) != 0 || len(l.frames.worldByFrame) != 0 {
		t.Fatal("destroyed context remained cached")
	}
	l.frameEvent(context.Background(), conn, cdpMessage{Method: "Page.frameNavigated", Session: "remote", Params: json.RawMessage(`{"frame":{"id":"child","url":"http://local/new"}}`)})
	if l.frames.states["child"].parent != "main" {
		t.Fatal("OOPIF root erased its parent")
	}
	l.frames.selected.invalid = false
	revision := l.frames.states["child"].revision
	l.frameEvent(context.Background(), conn, cdpMessage{Method: "Page.navigatedWithinDocument", Session: "remote", Params: json.RawMessage(`{"frameId":"child","url":"http://local/new#section"}`)})
	if l.frames.selected.invalid || l.frames.states["child"].revision != revision || l.frames.states["child"].url != "http://local/new#section" {
		t.Fatal("same-document navigation invalidated controls")
	}
	l.frameEvent(context.Background(), conn, cdpMessage{Method: "Target.detachedFromTarget", Params: json.RawMessage(`{"sessionId":"remote"}`)})
	if l.frames.endpoints["child"] != nil || !l.frames.selected.invalid {
		t.Fatal("detached renderer left active scope")
	}
}

func TestLiveFrameContextBudgetPruningAndNetworkIsolation(t *testing.T) {
	conn := &cdpConn{}
	l := &liveSession{session: &StealthSession{pageCDP: conn}, frameID: "main", frames: newLiveFrameController(), changed: make(chan struct{})}
	for i := range 1000 {
		l.frameEvent(context.Background(), conn, cdpMessage{Method: "Runtime.executionContextCreated", Params: json.RawMessage(fmt.Sprintf(`{"context":{"id":%d,"uniqueId":"default","name":"","auxData":{"frameId":"child"}}}`, i+1))})
	}
	if len(l.frames.worlds) != 0 {
		t.Fatal("unrelated site contexts consumed browser cache")
	}
	for i := range 3 * maxLiveFrames {
		l.frameEvent(context.Background(), conn, frameContextEvent("", "cached-frame", int64(i+1), fmt.Sprintf("cached-%d", i)))
	}
	if len(l.frames.knownWorlds) > 2*maxLiveFrames {
		t.Fatal("BFCache identity registry exceeded budget")
	}
	l.frames.states["main"] = &liveFrameState{}
	l.frames.states["child"] = &liveFrameState{parent: "main"}
	l.frames.states["other"] = &liveFrameState{parent: "main"}
	l.frames.selected = &liveFrameScope{id: "child", endpoint: conn.session("remote")}
	for _, session := range []string{"remote", ""} {
		frame := "other"
		if session == "remote" {
			frame = "child"
		}
		l.frameEvent(context.Background(), conn, cdpMessage{Method: "Network.requestWillBeSent", Session: session, Params: json.RawMessage(fmt.Sprintf(`{"frameId":%q,"requestId":"same-id","type":"Document"}`, frame))})
	}
	l.frameEvent(context.Background(), conn, cdpMessage{Method: "Network.loadingFailed", Session: "remote", Params: json.RawMessage(`{"requestId":"same-id","errorText":"local failure"}`)})
	if l.frames.states["child"].failure == nil || l.frames.states["other"].failure != nil {
		t.Fatal("request IDs collided across sessions")
	}
	l.frames.endpoints["child"] = conn.session("remote")
	l.pruneFrameStatesLocked(map[string]bool{"main": true, "other": true})
	if len(l.frames.knownWorlds) != 0 {
		t.Fatal("orphan frame identity survived complete metadata pruning")
	}
	if len(l.frames.states) != 3 || !l.frames.states["child"].detached || !l.frames.selected.invalid || len(l.frames.endpoints) != 0 {
		t.Fatal("removed selected frame did not leave a bounded tombstone")
	}
	if _, err := l.frameEndpointLocked("child"); err == nil {
		t.Fatal("removed selected frame fell back to main document")
	}
	l.frames.selected = nil
	l.pruneFrameStatesLocked(map[string]bool{"main": true, "other": true})
	if len(l.frames.states) != 2 {
		t.Fatal("unselected tombstone retained")
	}
	for i := range maxLiveFrames + 10 {
		l.frameStateLocked(fmt.Sprintf("overflow-%d", i))
	}
	if len(l.frames.states) > maxLiveFrames+1 || l.failure == nil {
		t.Fatal("frame state budget is not enforced")
	}
}

func TestLiveFrameRepeatedTopNavigationAndRemoteDescendant(t *testing.T) {
	var crossURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/shadow":
			fmt.Fprint(w, `<div id="host"></div><script>document.querySelector('#host').attachShadow({mode:'open'}).innerHTML='<input id="shadow-input">'</script>`)
		case "/editor":
			fmt.Fprint(w, `<div id="editor" contenteditable="true" role="textbox">draft</div>`)
		case "/form":
			fmt.Fprint(w, `<form action="/receipt"><input name="q" value="accepted"><button id="send">Send</button></form>`)
		case "/receipt":
			fmt.Fprint(w, `<p>Receipt</p>`)
		case "/frames":
			fmt.Fprintf(w, `<title>Main</title><iframe name="same" src="/child"></iframe><iframe name="remote" src="%s/remote"></iframe>`, crossURL)
		case "/remote":
			fmt.Fprint(w, `<title>Remote</title><iframe name="remote-nested" src="/child"></iframe>`)
		case "/child":
			fmt.Fprint(w, `<title>Child</title><input id="note"><button id="check" type="button" onclick="this.textContent='trusted '+event.isTrusted+' '+document.querySelector('#note').value">Check</button>`)
		default:
			fmt.Fprint(w, `<p>Fresh top</p>`)
		}
	}))
	t.Cleanup(server.Close)
	crossURL = strings.Replace(server.URL, "127.0.0.1", "localhost", 1)
	b := newLiveTestBrowser(t)
	b.Allowed["localhost"] = true
	for iteration := range 3 {
		frameAPICommand(t, b, Command{Cmd: "open", URL: server.URL + "/shadow", RenderJS: true})
		frameAPICommand(t, b, Command{Cmd: "fill", Ref: frameAPIElement(t, b, "#shadow-input").Ref, Value: "shadow"})
		frameAPICommand(t, b, Command{Cmd: "open", URL: server.URL + "/editor", RenderJS: true})
		frameAPICommand(t, b, Command{Cmd: "fill", Ref: frameAPIElement(t, b, "#editor").Ref, Value: "editor"})
		frameAPICommand(t, b, Command{Cmd: "open", URL: server.URL + "/form", RenderJS: true})
		frameAPICommand(t, b, Command{Cmd: "press", Ref: frameAPIElement(t, b, "#send").Ref})
		frameAPICommand(t, b, Command{Cmd: "open", URL: server.URL + "/frames", RenderJS: true})
		frames := frameAPIList(t, b)
		for _, name := range []string{"same", "remote-nested"} {
			frame := frameAPINamed(t, frames, name)
			frameAPISelect(t, b, frame.Ref)
			value := fmt.Sprintf("iteration-%d-%s", iteration, name)
			frameAPICommand(t, b, Command{Cmd: "fill", Ref: frameAPIElement(t, b, "#note").Ref, Value: value})
			frameAPICommand(t, b, Command{Cmd: "press", Ref: frameAPIElement(t, b, "#check").Ref})
			if got := frameAPIElement(t, b, "#check").Text; got != "trusted true "+value {
				t.Fatalf("reused session/frame selected wrong document: %q", got)
			}
		}
		b.live.mu.Lock()
		remoteID := frameAPINamed(t, frames, "remote").ID
		if b.live.frames.endpoints[remoteID] == nil {
			t.Error("cross-site fixture was not attached as OOPIF")
		}
		b.live.mu.Unlock()
		frameAPICommand(t, b, Command{Cmd: "open", URL: server.URL + "/fresh", RenderJS: true})
		frameAPIList(t, b)
		b.live.mu.Lock()
		if len(b.live.frames.states) > 1 || len(b.live.frames.worlds) > 1 || len(b.live.frames.worldByFrame) > 1 || len(b.live.frames.creating) != 0 {
			t.Errorf("repeated navigation retained frame state: states=%d worlds=%d pending=%d", len(b.live.frames.states), len(b.live.frames.worlds), len(b.live.frames.creating))
		}
		b.live.mu.Unlock()
	}
}
