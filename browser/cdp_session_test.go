package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func newCDPSessionTestConn(t *testing.T, handler func(context.Context, *websocket.Conn)) (*cdpConn, context.Context) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept test websocket: %v", err)
			return
		}
		defer ws.CloseNow()
		handler(ctx, ws)
		// Keep reading control frames until the client closes. Avoid an early
		// EOF racing a completed reply or session event in transport tests.
		_, _, _ = ws.Read(ctx)
	}))
	conn, err := dialCDP(ctx, "ws"+strings.TrimPrefix(server.URL, "http"))
	if err != nil {
		cancel()
		server.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = conn.ws.CloseNow()
		cancel()
		server.Close()
		<-conn.closed
	})
	return conn, ctx
}

func readCDPSessionTestMessage(t *testing.T, ctx context.Context, ws *websocket.Conn) (cdpMessage, bool) {
	t.Helper()
	_, raw, err := ws.Read(ctx)
	if err != nil {
		t.Errorf("read test CDP request: %v", err)
		return cdpMessage{}, false
	}
	var request cdpMessage
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Errorf("decode test CDP request: %v", err)
		return cdpMessage{}, false
	}
	return request, true
}

func writeCDPSessionTestMessage(t *testing.T, ctx context.Context, ws *websocket.Conn, message cdpMessage) bool {
	t.Helper()
	raw, err := json.Marshal(message)
	if err == nil {
		err = ws.Write(ctx, websocket.MessageText, raw)
	}
	if err != nil {
		t.Errorf("write test CDP message: %v", err)
		return false
	}
	return true
}

func assertNoCDPPending(t *testing.T, conn *cdpConn) {
	t.Helper()
	conn.mu.Lock()
	count := len(conn.pending)
	conn.mu.Unlock()
	if count != 0 {
		t.Fatalf("retained %d pending CDP requests", count)
	}
}

func TestCDPSessionConcurrentResponsesAndEvents(t *testing.T) {
	const count = 24
	conn, ctx := newCDPSessionTestConn(t, func(ctx context.Context, ws *websocket.Conn) {
		requests := make([]cdpMessage, 0, count)
		seen := make(map[int]bool)
		for range count {
			request, ok := readCDPSessionTestMessage(t, ctx, ws)
			if !ok {
				return
			}
			if request.ID == 0 || seen[request.ID] {
				t.Errorf("global request ID collision: %+v", request)
			}
			seen[request.ID] = true
			var params map[string]any
			if err := json.Unmarshal(request.Params, &params); err != nil || params["sessionId"] != nil {
				t.Errorf("session ID must belong to envelope, not params: %s", request.Params)
			}
			requests = append(requests, request)
		}
		for _, session := range []string{"frame-a", "frame-b", ""} {
			if !writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{Method: "Runtime.executionContextCreated", Session: session, Params: json.RawMessage(`{"context":{"id":42}}`)}) {
				return
			}
		}
		for i := len(requests) - 1; i >= 0; i-- {
			request := requests[i]
			if !writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: request.ID, Session: request.Session, Result: request.Params}) {
				return
			}
		}
	})
	type outcome struct {
		ordinal int
		value   json.RawMessage
		err     error
	}
	results := make(chan outcome, count)
	for ordinal := range count {
		go func() {
			session := []string{"", "frame-a", "frame-b"}[ordinal%3]
			var raw json.RawMessage
			var err error
			if session == "" {
				raw, err = conn.send(ctx, "Test.echo", map[string]any{"ordinal": ordinal})
			} else {
				raw, err = conn.session(session).send(ctx, "Test.echo", map[string]any{"ordinal": ordinal})
			}
			results <- outcome{ordinal, raw, err}
		}()
	}
	for range count {
		result := <-results
		var decoded struct {
			Ordinal int `json:"ordinal"`
		}
		if result.err != nil || json.Unmarshal(result.value, &decoded) != nil || decoded.Ordinal != result.ordinal {
			t.Fatalf("misrouted response: ordinal=%d body=%s error=%v", result.ordinal, result.value, result.err)
		}
	}
	for _, session := range []string{"frame-a", "frame-b", ""} {
		select {
		case event := <-conn.events:
			if event.Session != session || event.Method != "Runtime.executionContextCreated" {
				t.Fatalf("misrouted session event: %+v", event)
			}
		case <-ctx.Done():
			t.Fatal("session event was lost")
		}
	}
	assertNoCDPPending(t, conn)
}

func TestCDPSessionTimeoutDropsLateResponse(t *testing.T) {
	received := make(chan struct{})
	release := make(chan struct{})
	conn, ctx := newCDPSessionTestConn(t, func(ctx context.Context, ws *websocket.Conn) {
		request, ok := readCDPSessionTestMessage(t, ctx, ws)
		if !ok {
			return
		}
		close(received)
		select {
		case <-release:
		case <-ctx.Done():
			return
		}
		if !writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: request.ID, Session: request.Session, Result: json.RawMessage(`"late"`)}) {
			return
		}
		request, ok = readCDPSessionTestMessage(t, ctx, ws)
		if ok {
			writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: request.ID, Session: request.Session, Result: json.RawMessage(`"fresh"`)})
		}
	})
	op, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := conn.sendSession(op, "frame-a", "Test.slow", nil)
		result <- err
	}()
	select {
	case <-received:
	case <-ctx.Done():
		t.Fatal("server never received request")
	}
	if err := <-result; !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("request timeout = %v", err)
	}
	assertNoCDPPending(t, conn)
	close(release)
	raw, err := conn.sendSession(ctx, "frame-b", "Test.next", nil)
	if err != nil || string(raw) != `"fresh"` {
		t.Fatalf("late response affected another session: %s, error=%v", raw, err)
	}
	assertNoCDPPending(t, conn)
}

func TestCDPSessionQueuedWriteAndCanceledRequest(t *testing.T) {
	conn, ctx := newCDPSessionTestConn(t, func(ctx context.Context, ws *websocket.Conn) {
		request, ok := readCDPSessionTestMessage(t, ctx, ws)
		if !ok {
			return
		}
		if request.Method != "Test.live" {
			t.Errorf("canceled request reached WebSocket: %+v", request)
		}
		writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: request.ID, Session: request.Session, Result: json.RawMessage(`true`)})
	})
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := conn.sendSession(canceled, "frame-a", "Test.canceled", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled request = %v", err)
	}
	if _, err := conn.sendSession(ctx, "frame-a", "Test.unmarshalable", map[string]any{"bad": func() {}}); err == nil {
		t.Fatal("unsupported JSON params accepted")
	}
	conn.writeGate <- struct{}{}
	op, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	_, err := conn.sendSession(op, "frame-a", "Test.queued", nil)
	cancel()
	<-conn.writeGate
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("queued request ignored timeout: %v", err)
	}
	assertNoCDPPending(t, conn)
	if raw, err := conn.sendSession(ctx, "frame-b", "Test.live", nil); err != nil || string(raw) != "true" {
		t.Fatalf("cancellation broke shared transport: %s, error=%v", raw, err)
	}
	assertNoCDPPending(t, conn)
}

func TestCDPSessionMismatchAndProtocolError(t *testing.T) {
	conn, ctx := newCDPSessionTestConn(t, func(ctx context.Context, ws *websocket.Conn) {
		for ordinal := range 3 {
			request, ok := readCDPSessionTestMessage(t, ctx, ws)
			if !ok {
				return
			}
			response := cdpMessage{ID: request.ID, Session: request.Session, Result: json.RawMessage(`true`)}
			switch ordinal {
			case 0:
				response.Session = "other-frame"
			case 1:
				response.Session = ""
				response.Result = nil
				response.Error = &cdpError{Code: -32001, Message: "Session not found"}
			}
			if !writeCDPSessionTestMessage(t, ctx, ws, response) {
				return
			}
		}
	})
	if _, err := conn.sendSession(ctx, "frame-a", "Test.mismatch", nil); err == nil || !strings.Contains(err.Error(), "session mismatch") {
		t.Fatalf("session mismatch accepted: %v", err)
	}
	_, err := conn.sendSession(ctx, "frame-a", "Test.detached", nil)
	var protocolErr *cdpError
	if !errors.As(err, &protocolErr) || protocolErr.Code != -32001 || protocolErr.Message != "Session not found" {
		t.Fatalf("protocol error lost: %v", err)
	}
	if _, err := conn.send(ctx, "Test.default", nil); err != nil {
		t.Fatalf("default endpoint regressed: %v", err)
	}
	assertNoCDPPending(t, conn)
}

func TestCDPSessionClosedConnectionReleasesRequests(t *testing.T) {
	const count = 8
	conn, ctx := newCDPSessionTestConn(t, func(ctx context.Context, ws *websocket.Conn) {
		for range count {
			if _, ok := readCDPSessionTestMessage(t, ctx, ws); !ok {
				return
			}
		}
		_ = ws.CloseNow()
	})
	results := make(chan error, count)
	for ordinal := range count {
		go func() {
			_, err := conn.sendSession(ctx, fmt.Sprintf("frame-%d", ordinal), "Test.pending", nil)
			results <- err
		}()
	}
	for range count {
		if err := <-results; err == nil || !strings.Contains(err.Error(), "connection closed") {
			t.Fatalf("closed transport response = %v", err)
		}
	}
	assertNoCDPPending(t, conn)
	if _, err := conn.send(ctx, "Test.closed", nil); err == nil {
		t.Fatal("closed connection accepted request")
	}
	assertNoCDPPending(t, conn)
}

func TestCDPSessionEvaluateContextAndUniqueWorld(t *testing.T) {
	conn, ctx := newCDPSessionTestConn(t, func(ctx context.Context, ws *websocket.Conn) {
		for ordinal := range 4 {
			request, ok := readCDPSessionTestMessage(t, ctx, ws)
			if !ok {
				return
			}
			var params map[string]any
			if err := json.Unmarshal(request.Params, &params); err != nil {
				t.Error(err)
				return
			}
			if request.Method != "Runtime.evaluate" || params["awaitPromise"] != true || params["returnByValue"] != true || params["expression"] != "Promise.resolve({ok:true})" {
				t.Errorf("evaluation flags changed: %+v %s", request, request.Params)
			}
			switch ordinal {
			case 0:
				if request.Session != "child" || params["contextId"] != float64(42) || params["uniqueContextId"] != nil {
					t.Errorf("numeric child context: %s", request.Params)
				}
			case 1:
				if request.Session != "child" || params["uniqueContextId"] != "unique-world" || params["contextId"] != nil {
					t.Errorf("unique child context: %s", request.Params)
				}
			case 2:
				if request.Session != "" || params["contextId"] != nil || params["uniqueContextId"] != nil {
					t.Errorf("default page evaluation changed: %s", request.Params)
				}
			case 3:
				if request.Session != "" || params["contextId"] != float64(17) {
					t.Errorf("page numeric context: %s", request.Params)
				}
			}
			if !writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: request.ID, Session: request.Session, Result: json.RawMessage(`{"result":{"type":"object","value":{"ok":true}}}`)}) {
				return
			}
		}
	})
	page := &StealthSession{pageCDP: conn}
	endpoint := conn.session("child")
	for _, evaluate := range []func() (json.RawMessage, error){
		func() (json.RawMessage, error) {
			return endpoint.EvaluateContext(ctx, "Promise.resolve({ok:true})", 42)
		},
		func() (json.RawMessage, error) {
			return endpoint.EvaluateWorld(ctx, "Promise.resolve({ok:true})", 42, "unique-world")
		},
		func() (json.RawMessage, error) { return page.Evaluate(ctx, "Promise.resolve({ok:true})") },
		func() (json.RawMessage, error) { return page.EvaluateContext(ctx, "Promise.resolve({ok:true})", 17) },
	} {
		if raw, err := evaluate(); err != nil || string(raw) != `{"ok":true}` {
			t.Fatalf("evaluation value lost: %s, error=%v", raw, err)
		}
	}
	if _, err := endpoint.EvaluateContext(ctx, "true", -1); err == nil {
		t.Fatal("negative context accepted")
	}
	var missing *StealthSession
	if _, err := missing.Evaluate(ctx, "true"); err == nil {
		t.Fatal("missing browser session accepted")
	}
	var noEndpoint *cdpSession
	if _, err := noEndpoint.EvaluateContext(ctx, "true", 1); err == nil {
		t.Fatal("missing CDP endpoint accepted")
	}
	if _, err := (&cdpConn{}).session("child").send(ctx, "Test.nil", nil); err == nil {
		t.Fatal("missing CDP connection accepted")
	}
}

func TestCDPEvaluationErrorsAndValues(t *testing.T) {
	for _, tc := range []struct {
		name, result, errorContains, value string
	}{
		{"ExceptionDescription", `{"result":{"value":"do not return"},"exceptionDetails":{"text":"Uncaught","exception":{"description":"TypeError: broken"}}}`, "TypeError: broken", ""},
		{"ThrownValue", `{"exceptionDetails":{"text":"Uncaught","exception":{"value":"custom failure"}}}`, "custom failure", ""},
		{"ExceptionText", `{"exceptionDetails":{"text":"execution context destroyed"}}`, "execution context destroyed", ""},
		{"ReturnedError", `{"result":{"subtype":"error","description":"Error: result"}}`, "Error: result", ""},
		{"InvalidJSON", `{`, "unexpected", ""},
		{"Undefined", `{"result":{"type":"undefined"}}`, "", ""},
		{"Null", `{"result":{"type":"object","subtype":"null","value":null}}`, "", "null"},
		{"Number", `{"result":{"type":"number","value":42}}`, "", "42"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := cdpEvaluationValue(json.RawMessage(tc.result))
			if tc.errorContains != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errorContains) || raw != nil {
					t.Fatalf("evaluation exception lost: raw=%s error=%v", raw, err)
				}
			} else if err != nil || string(raw) != tc.value {
				t.Fatalf("evaluation value changed: raw=%s error=%v", raw, err)
			}
		})
	}
}
