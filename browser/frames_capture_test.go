package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/coder/websocket"
)

func frameCaptureTree(main bool, replacement bool, current bool) json.RawMessage {
	childURL := "http://frame.test/a"
	if replacement && current {
		childURL = "http://frame.test/replaced"
	}
	child := fmt.Sprintf(`{"frame":{"id":"child","parentId":"main","url":%q}}`, childURL)
	if !main {
		return json.RawMessage(`{"frameTree":` + child + `}`)
	}
	children := child
	if current && !replacement {
		children += `,{"frame":{"id":"new-child","parentId":"main","url":"http://frame.test/new"}}`
	}
	return json.RawMessage(`{"frameTree":{"frame":{"id":"main","url":"http://frame.test/"},"childFrames":[` + children + `]}}`)
}

func TestFrameCaptureDoesNotPruneConcurrentAttachment(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		t.Run(fmt.Sprintf("replacement-%t", replacement), func(t *testing.T) {
			newID := "new-child"
			if replacement {
				newID = "child"
			}
			conn, ctx := newCDPSessionTestConn(t, func(ctx context.Context, ws *websocket.Conn) {
				request, ok := readCDPSessionTestMessage(t, ctx, ws)
				if !ok {
					return
				}
				if request.Method != "Page.getFrameTree" || request.Session != "" {
					t.Errorf("initial main request: %+v", request)
				}
				writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: request.ID, Result: frameCaptureTree(true, replacement, false)})
				staleRequest, ok := readCDPSessionTestMessage(t, ctx, ws)
				if !ok {
					return
				}
				if staleRequest.Method != "Page.getFrameTree" || staleRequest.Session != "old-renderer" {
					t.Errorf("initial remote request: %+v", staleRequest)
				}
				// Attach a renderer while the old remote tree response is pending.
				// The real event handler configures and resumes this flattened session.
				writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{Method: "Target.attachedToTarget", Params: json.RawMessage(`{"sessionId":"new-renderer","targetInfo":{"type":"iframe"}}`)})
				for {
					request, ok = readCDPSessionTestMessage(t, ctx, ws)
					if !ok {
						return
					}
					if request.Session != "new-renderer" {
						t.Errorf("new renderer setup: %+v", request)
					}
					result := json.RawMessage(`{}`)
					if request.Method == "Page.getFrameTree" {
						result = json.RawMessage(fmt.Sprintf(`{"frameTree":{"frame":{"id":%q,"parentId":"main","url":"http://frame.test/new"}}}`, newID))
					}
					writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: request.ID, Session: request.Session, Result: result})
					if request.Method == "Runtime.runIfWaitingForDebugger" {
						break
					}
				}
				writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: staleRequest.ID, Session: staleRequest.Session, Result: frameCaptureTree(false, replacement, false)})
				// A later stable capture should include the new session and still
				// prune inventory that was genuinely absent from the complete tree.
				request, ok = readCDPSessionTestMessage(t, ctx, ws)
				if !ok {
					return
				}
				writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: request.ID, Result: frameCaptureTree(true, replacement, true)})
				count := 2
				if replacement {
					count = 1
				}
				for range count {
					request, ok = readCDPSessionTestMessage(t, ctx, ws)
					if !ok {
						return
					}
					result := frameCaptureTree(false, false, true)
					if request.Session == "new-renderer" {
						result = json.RawMessage(fmt.Sprintf(`{"frameTree":{"frame":{"id":%q,"parentId":"main","url":%q}}}`, newID, map[bool]string{true: "http://frame.test/replaced", false: "http://frame.test/new"}[replacement]))
					}
					writeCDPSessionTestMessage(t, ctx, ws, cdpMessage{ID: request.ID, Session: request.Session, Result: result})
				}
			})
			l := newFrameContextTestSession(t, conn, ctx)
			current := &Page{URL: "http://frame.test/a"}
			b := &Browser{AllowAll: true, live: l, Current: current}
			l.mu.Lock()
			l.frames.states["main"].url = "http://frame.test/"
			l.frames.states["child"].url = "http://frame.test/a"
			l.frames.states["stale"] = &liveFrameState{parent: "main"}
			oldEndpoint := conn.session("old-renderer")
			l.frames.endpoints["child"] = oldEndpoint
			l.frames.refs = map[string]string{"fr1": "main", "fr2": "child"}
			l.frames.refByID = map[string]string{"main": "fr1", "child": "fr2"}
			l.frames.nextRef = 3
			l.mu.Unlock()
			frames, err := b.liveFramesContext(ctx)
			if err == nil || !strings.Contains(err.Error(), "topology changed") || frames != nil {
				t.Fatalf("stale capture published: frames=%+v err=%v", frames, err)
			}
			l.mu.Lock()
			state, endpoint := l.frames.states[newID], l.frames.endpoints[newID]
			preserved := state != nil && !state.detached && endpoint != nil && endpoint.id == "new-renderer" && l.frames.states["stale"] != nil && l.frames.refs["fr2"] == "child"
			l.mu.Unlock()
			if !preserved || b.Current != current {
				t.Fatal("stale capture erased new endpoint/state or changed observation/refs")
			}
			frames, err = b.liveFramesContext(ctx)
			if err != nil {
				t.Fatal(err)
			}
			l.mu.Lock()
			stale, liveEndpoint, stateCount := l.frames.states["stale"], l.frames.endpoints[newID], len(l.frames.states)
			l.mu.Unlock()
			if stale != nil || liveEndpoint != endpoint || stateCount != len(frames) {
				t.Fatalf("stable capture failed to prune exactly absent states: states=%d frames=%d", stateCount, len(frames))
			}
			found := false
			for _, frame := range frames {
				if frame.ID == newID && frame.ParentRef == "fr1" {
					found = true
				}
			}
			if !found {
				t.Fatalf("new renderer missing from next capture: %+v", frames)
			}
		})
	}
}

func TestFrameCaptureMetadataIsPure(t *testing.T) {
	l := &liveSession{frames: newLiveFrameController()}
	state := &liveFrameState{parent: "original", url: "http://frame.test/original"}
	l.frames.states["child"] = state
	b := &Browser{AllowAll: true, live: l}
	if _, err := b.framesFromMetadata(frameCaptureTree(true, false, false)); err != nil {
		t.Fatal(err)
	}
	if len(l.frames.states) != 1 || l.frames.states["child"] != state || state.parent != "original" || state.url != "http://frame.test/original" {
		t.Fatal("metadata parsing changed controller before fence validation")
	}
}
