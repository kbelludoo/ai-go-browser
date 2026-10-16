package browser

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestCDPEventQueueOverflowReleasesPendingAndKeepsDeliveredResponse(t *testing.T) {
	peerClosed := make(chan struct{})
	conn, ctx := newCDPSessionTestConn(t, func(ctx context.Context, ws *websocket.Conn) {
		first, ok := readCDPSessionTestMessage(t, ctx, ws)
		if !ok {
			return
		}
		if _, ok = readCDPSessionTestMessage(t, ctx, ws); !ok {
			return
		}
		for range 256 {
			if !writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{Method: "Page.frameStartedLoading", Params: json.RawMessage(`{"frameId":"main"}`)}) {
				return
			}
		}
		if !writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: first.ID, Result: json.RawMessage(`{"delivered":true}`)}) {
			return
		}
		if !writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{Method: "Target.attachedToTarget", Params: json.RawMessage(`{"sessionId":"paused-child"}`)}) {
			return
		}
		_, _, _ = ws.Read(ctx)
		close(peerClosed)
	})
	type outcome struct {
		data json.RawMessage
		err  error
	}
	results := make(chan outcome, 2)
	for range 2 {
		go func() { data, err := conn.send(ctx, "Test.pending", nil); results <- outcome{data, err} }()
	}
	var successes, failures int
	for range 2 {
		select {
		case result := <-results:
			if result.err == nil && string(result.data) == `{"delivered":true}` {
				successes++
			} else if result.err != nil && strings.Contains(result.err.Error(), "event queue overflow") {
				failures++
			} else {
				t.Fatalf("unexpected response: %+v", result)
			}
		case <-ctx.Done():
			t.Fatal("overflow did not release pending requests")
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("successes=%d failures=%d", successes, failures)
	}
	assertNoCDPPending(t, conn)
	if _, err := conn.send(ctx, "Test.afterOverflow", nil); err == nil || !strings.Contains(err.Error(), "event queue overflow") {
		t.Fatalf("new request error: %v", err)
	}
	select {
	case <-peerClosed:
	case <-ctx.Done():
		t.Fatal("overflow did not close peer transport")
	}
}

func TestCDPFullEventQueueStillDeliversResponses(t *testing.T) {
	conn, ctx := newCDPSessionTestConn(t, func(ctx context.Context, ws *websocket.Conn) {
		request, ok := readCDPSessionTestMessage(t, ctx, ws)
		if !ok {
			return
		}
		for range 256 {
			if !writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{Method: "Page.frameStartedLoading"}) {
				return
			}
		}
		writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: request.ID, Result: json.RawMessage(`{"ok":true}`)})
	})
	data, err := conn.send(ctx, "Test.response", nil)
	if err != nil || string(data) != `{"ok":true}` {
		t.Fatalf("response: %s %v", data, err)
	}
	if len(conn.events) != cap(conn.events) {
		t.Fatalf("queue has %d events, want %d", len(conn.events), cap(conn.events))
	}
	select {
	case <-conn.closed:
		t.Fatalf("full queue without another event closed: %v", conn.readErr)
	default:
	}
}

func TestCDPEventSizeLimitExcludesLargeResponses(t *testing.T) {
	for _, response := range []bool{false, true} {
		t.Run(map[bool]string{false: "event", true: "response"}[response], func(t *testing.T) {
			payload, _ := json.Marshal(map[string]string{"text": strings.Repeat("x", maxCDPEventBytes)})
			conn, ctx := newCDPSessionTestConn(t, func(ctx context.Context, ws *websocket.Conn) {
				request, ok := readCDPSessionTestMessage(t, ctx, ws)
				if !ok {
					return
				}
				message := cdpMessage{Method: "Runtime.consoleAPICalled", Params: payload}
				if response {
					message = cdpMessage{ID: request.ID, Result: payload}
				}
				writeCDPSessionTestMessage(t, ctx, ws, message)
			})
			data, err := conn.send(ctx, "Test.large", nil)
			if response {
				if err != nil || len(data) <= maxCDPEventBytes {
					t.Fatalf("large response length=%d err=%v", len(data), err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "cdp event exceeds") {
				t.Fatalf("large event error: %v", err)
			}
			assertNoCDPPending(t, conn)
		})
	}
}

func TestLiveSessionSurfacesEitherConnectionFailure(t *testing.T) {
	for _, pageFails := range []bool{false, true} {
		t.Run(map[bool]string{false: "browser", true: "page"}[pageFails], func(t *testing.T) {
			failed, ctx := newCDPSessionTestConn(t, func(ctx context.Context, ws *websocket.Conn) {
				if _, ok := readCDPSessionTestMessage(t, ctx, ws); !ok {
					return
				}
				payload, _ := json.Marshal(map[string]string{"text": strings.Repeat("x", maxCDPEventBytes)})
				writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{Method: "Runtime.consoleAPICalled", Params: payload})
			})
			other, _ := newCDPSessionTestConn(t, func(ctx context.Context, ws *websocket.Conn) {})
			session := &StealthSession{pageCDP: other, browserCDP: failed}
			if pageFails {
				session.pageCDP, session.browserCDP = failed, other
			}
			l := &liveSession{session: session, frames: newLiveFrameController(), done: make(chan struct{}), changed: make(chan struct{})}
			go l.events(ctx)
			if _, err := failed.send(ctx, "Test.trigger", nil); err == nil {
				t.Fatal("oversized event did not fail")
			}
			select {
			case <-l.done:
			case <-time.After(time.Second):
				t.Fatal("live session ignored connection failure")
			}
			l.mu.Lock()
			failure := l.failure
			l.mu.Unlock()
			if failure == nil || !strings.Contains(failure.Error(), "cdp event exceeds") {
				t.Fatalf("lost failure cause: %v", failure)
			}
		})
	}
}
