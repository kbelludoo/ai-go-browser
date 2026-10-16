package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type liveFrameState struct {
	url, parent, request string
	requestEndpoint      string
	loading, detached    bool
	status               int
	failure              error
	revision             uint64
}

type liveFrameWorld struct {
	frame, unique string
	contextID     int64
	endpoint      *cdpSession
	revision      uint64
}

type liveFrameCreation struct {
	endpoint *cdpSession
	revision uint64
}

type liveFrameScope struct {
	id, ref, prefix, unique string
	contextID               int64
	endpoint                *cdpSession
	invalid                 bool
}

// All controller data is protected by liveSession.mu. Frame refs identify a
// frame, while document refs are additionally namespaced by scope generation.
type liveFrameController struct {
	selected      *liveFrameScope
	endpoints     map[string]*cdpSession
	states        map[string]*liveFrameState
	refs          map[string]string
	refByID       map[string]string
	worlds        map[string]*liveFrameWorld
	knownWorlds   map[string]*liveFrameWorld
	worldByFrame  map[string]*liveFrameWorld
	creating      map[string]*liveFrameCreation
	observed      *liveFrameWorld
	observedScope *liveFrameScope
	nextRef       int
	generation    uint64
}

func newLiveFrameController() *liveFrameController {
	return &liveFrameController{endpoints: map[string]*cdpSession{}, states: map[string]*liveFrameState{}, refs: map[string]string{}, refByID: map[string]string{}, worlds: map[string]*liveFrameWorld{}, knownWorlds: map[string]*liveFrameWorld{}, worldByFrame: map[string]*liveFrameWorld{}, creating: map[string]*liveFrameCreation{}, nextRef: 2}
}

func frameAutoAttachParams() map[string]any {
	return map[string]any{"autoAttach": true, "waitForDebuggerOnStart": true, "flatten": true, "filter": []map[string]any{{"type": "iframe", "exclude": false}, {"exclude": true}}}
}

func (l *liveSession) frameStateLocked(id string) *liveFrameState {
	state := l.frames.states[id]
	if state == nil {
		if id == "" || len(l.frames.states) >= maxLiveFrames+1 {
			if id != "" {
				l.failure = fmt.Errorf("frame state exceeds %d frames; refresh the frame list", maxLiveFrames)
			}
			return &liveFrameState{detached: true}
		}
		state = &liveFrameState{}
		l.frames.states[id] = state
	}
	return state
}

func sameFrameEndpoint(a, b *cdpSession) bool {
	return a != nil && b != nil && a.conn == b.conn && a.id == b.id
}

func endpointKey(endpoint *cdpSession) string {
	return fmt.Sprintf("%p:%s", endpoint.conn, endpoint.id)
}

func worldKey(endpoint *cdpSession, id int64) string {
	return fmt.Sprintf("%p:%s:%d", endpoint.conn, endpoint.id, id)
}

func (l *liveSession) dropFrameWorldLocked(id string) {
	for key, world := range l.frames.worlds {
		if world.frame == id {
			delete(l.frames.worlds, key)
		}
	}
	delete(l.frames.worldByFrame, id)
	if l.frames.observed != nil && l.frames.observed.frame == id {
		l.frames.observed, l.frames.observedScope = nil, nil
	}
}

func (l *liveSession) dropEndpointWorldsLocked(endpoint *cdpSession, contextID int64) {
	for key, world := range l.frames.knownWorlds {
		if sameFrameEndpoint(world.endpoint, endpoint) && (contextID == 0 || contextID == world.contextID) {
			delete(l.frames.knownWorlds, key)
		}
	}
	for key, world := range l.frames.worlds {
		if sameFrameEndpoint(world.endpoint, endpoint) && (contextID == 0 || contextID == world.contextID) {
			delete(l.frames.worlds, key)
			if l.frames.worldByFrame[world.frame] == world {
				delete(l.frames.worldByFrame, world.frame)
				if l.frames.observed == world {
					l.frames.observed, l.frames.observedScope = nil, nil
				}
			}
		}
	}
	if scope := l.frames.selected; scope != nil && sameFrameEndpoint(scope.endpoint, endpoint) && (contextID == 0 || contextID == scope.contextID) {
		scope.invalid = true
	}
}

func (l *liveSession) frameDescendsFromLocked(id, ancestor string) bool {
	for count := 0; id != "" && count <= maxLiveFrames; count++ {
		if id == ancestor {
			return true
		}
		state := l.frames.states[id]
		if state == nil {
			break
		}
		id = state.parent
	}
	return false
}

func (l *liveSession) invalidateFrameLocked(id string, detached bool) {
	// Determine descendants before deleting any ancestry metadata.
	var ids []string
	for candidate := range l.frames.states {
		if l.frameDescendsFromLocked(candidate, id) {
			ids = append(ids, candidate)
		}
	}
	if scope := l.frames.selected; scope != nil && l.frameDescendsFromLocked(scope.id, id) {
		scope.invalid = true
	}
	for _, candidate := range ids {
		state := l.frames.states[candidate]
		state.revision++
		if detached {
			for key, world := range l.frames.knownWorlds {
				if world.frame == candidate {
					delete(l.frames.knownWorlds, key)
				}
			}
			state.detached = true
			state.loading = false
		}
		l.dropFrameWorldLocked(candidate)
		if detached {
			delete(l.frames.endpoints, candidate)
			delete(l.frames.refs, l.frames.refByID[candidate])
			delete(l.frames.refByID, candidate)
			if l.frames.selected == nil || l.frames.selected.id != candidate {
				delete(l.frames.states, candidate)
			}
		}
	}
}

// Only call this after collecting a complete, valid frame tree. One removed
// selected frame may remain as a tombstone, so old commands fail explicitly.
func (l *liveSession) pruneFrameStatesLocked(active map[string]bool) {
	for id := range l.frames.states {
		if !active[id] {
			l.invalidateFrameLocked(id, true)
		}
	}
	for id, endpoint := range l.frames.endpoints {
		if !active[id] {
			l.dropEndpointWorldsLocked(endpoint, 0)
			delete(l.frames.endpoints, id)
		}
	}
	for key, world := range l.frames.knownWorlds {
		if !active[world.frame] {
			delete(l.frames.knownWorlds, key)
		}
	}
}

func (l *liveSession) failFrameLocked(id string, err error) {
	if id == l.frameID {
		l.failure, l.loading = err, false
	} else {
		state := l.frameStateLocked(id)
		state.failure, state.loading, state.status = err, false, 0
	}
	l.notifyLocked()
}

func (l *liveSession) frameEvent(ctx context.Context, conn *cdpConn, msg cdpMessage) bool {
	endpoint := conn.session(msg.Session)
	if msg.Method == "Target.attachedToTarget" {
		var event struct {
			SessionID  string `json:"sessionId"`
			TargetInfo struct {
				Type string `json:"type"`
			} `json:"targetInfo"`
		}
		if json.Unmarshal(msg.Params, &event) != nil || event.SessionID == "" {
			return true
		}
		child := conn.session(event.SessionID)
		if event.TargetInfo.Type != "iframe" {
			_, _ = child.send(ctx, "Runtime.runIfWaitingForDebugger", nil)
			return true
		}
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		var err error
		for _, method := range []string{"Page.enable", "Runtime.enable", "Network.enable", "Fetch.enable", "Target.setAutoAttach"} {
			var params map[string]any
			switch method {
			case "Network.enable":
				params = map[string]any{"maxTotalBufferSize": 1 << 20, "maxResourceBufferSize": 256 << 10}
			case "Fetch.enable":
				params = map[string]any{"patterns": []map[string]any{{"urlPattern": "*", "resourceType": "Document", "requestStage": "Request"}}}
			case "Target.setAutoAttach":
				params = frameAutoAttachParams()
			}
			if _, err = child.send(callCtx, method, params); err != nil {
				break
			}
		}
		if err == nil {
			var raw json.RawMessage
			raw, err = child.send(callCtx, "Page.getFrameTree", nil)
			var tree struct {
				Tree liveFrameTree `json:"frameTree"`
			}
			if err == nil && len(raw) > maxFrameMetadataBytes {
				err = errors.New("frame metadata exceeds budget")
			}
			if err == nil {
				err = json.Unmarshal(raw, &tree)
			}
			if err == nil && tree.Tree.Frame.ID == "" {
				err = errors.New("attached frame has no frame ID")
			}
			if err == nil && tree.Tree.Frame.ID != "" {
				l.mu.Lock()
				if l.frames.endpoints[tree.Tree.Frame.ID] == nil && len(l.frames.endpoints) >= maxLiveFrames {
					err = errors.New("too many Chromium frame sessions")
				} else {
					if previous := l.frames.endpoints[tree.Tree.Frame.ID]; previous != nil && !sameFrameEndpoint(previous, child) {
						l.dropEndpointWorldsLocked(previous, 0)
					}
					l.frames.endpoints[tree.Tree.Frame.ID] = child
					state := l.frameStateLocked(tree.Tree.Frame.ID)
					state.url, state.detached = tree.Tree.Frame.URL, false
					if tree.Tree.Frame.ParentID != "" {
						state.parent = tree.Tree.Frame.ParentID
					}
				}
				l.notifyLocked()
				l.mu.Unlock()
			}
		}
		// A setup timeout must not strand an auto-attached renderer in its paused
		// initial state. Resume has a fresh deadline, independent of setup.
		resumeCtx, resumeCancel := context.WithTimeout(ctx, 2*time.Second)
		_, resumeErr := child.send(resumeCtx, "Runtime.runIfWaitingForDebugger", nil)
		resumeCancel()
		if err == nil {
			err = resumeErr
		}
		if err != nil && ctx.Err() == nil {
			l.mu.Lock()
			l.failure = fmt.Errorf("configure Chromium frame: %w", err)
			l.notifyLocked()
			l.mu.Unlock()
		}
		return true
	}
	if msg.Method == "Target.detachedFromTarget" {
		var event struct {
			SessionID string `json:"sessionId"`
		}
		if json.Unmarshal(msg.Params, &event) == nil {
			l.mu.Lock()
			for id, child := range l.frames.endpoints {
				if child.conn == conn && child.id == event.SessionID {
					l.dropEndpointWorldsLocked(child, 0)
					delete(l.frames.endpoints, id)
					if scope := l.frames.selected; scope != nil && l.frameDescendsFromLocked(scope.id, id) {
						scope.invalid = true
					}
				}
			}
			l.notifyLocked()
			l.mu.Unlock()
		}
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	switch msg.Method {
	case "Runtime.executionContextCreated":
		var event struct {
			Context struct {
				ID       int64  `json:"id"`
				UniqueID string `json:"uniqueId"`
				Name     string `json:"name"`
				Aux      struct {
					FrameID string `json:"frameId"`
				} `json:"auxData"`
			} `json:"context"`
		}
		if json.Unmarshal(msg.Params, &event) == nil && event.Context.Name == "ai-go-browser-frame" && event.Context.ID > 0 && event.Context.UniqueID != "" && event.Context.Aux.FrameID != "" {
			id := event.Context.Aux.FrameID
			key := worldKey(endpoint, event.Context.ID)
			// A BFCache restoration may announce its existing named world before
			// Page.createIsolatedWorld. Retain bounded identities independently
			// from the currently published document and in-flight creation.
			if l.frames.knownWorlds[key] == nil && len(l.frames.knownWorlds) >= 2*maxLiveFrames {
				for previousKey, previous := range l.frames.knownWorlds {
					if l.frames.worldByFrame[previous.frame] == nil || l.frames.worldByFrame[previous.frame].unique != previous.unique {
						delete(l.frames.knownWorlds, previousKey)
						break
					}
				}
			}
			if len(l.frames.knownWorlds) < 2*maxLiveFrames || l.frames.knownWorlds[key] != nil {
				l.frames.knownWorlds[key] = &liveFrameWorld{frame: id, contextID: event.Context.ID, unique: event.Context.UniqueID, endpoint: endpoint}
			}
			state := l.frames.states[id]
			if state == nil && id == l.frameID {
				state = l.frameStateLocked(id)
			}
			creation := l.frames.creating[id]
			if state != nil && !state.detached && creation != nil && creation.revision == state.revision && sameFrameEndpoint(creation.endpoint, endpoint) {
				key := worldKey(endpoint, event.Context.ID)
				if l.frames.worlds[key] == nil && len(l.frames.worlds) >= maxLiveFrames {
					l.failure = errors.New("frame execution contexts exceed budget")
					break
				}
				world := &liveFrameWorld{frame: id, contextID: event.Context.ID, unique: event.Context.UniqueID, endpoint: endpoint, revision: creation.revision}
				l.frames.worlds[worldKey(endpoint, world.contextID)] = world
			}
		}
	case "Runtime.executionContextDestroyed", "Runtime.executionContextsCleared":
		var event struct {
			ID int64 `json:"executionContextId"`
		}
		_ = json.Unmarshal(msg.Params, &event)
		if msg.Method == "Runtime.executionContextsCleared" {
			l.dropEndpointWorldsLocked(endpoint, 0)
		} else if event.ID > 0 {
			l.dropEndpointWorldsLocked(endpoint, event.ID)
		}
	case "Page.frameAttached":
		var event struct {
			ID     string `json:"frameId"`
			Parent string `json:"parentFrameId"`
		}
		if json.Unmarshal(msg.Params, &event) == nil {
			state := l.frameStateLocked(event.ID)
			state.parent, state.detached = event.Parent, false
		}
	case "Page.frameDetached":
		var event struct {
			ID     string `json:"frameId"`
			Reason string `json:"reason"`
		}
		if json.Unmarshal(msg.Params, &event) == nil {
			l.invalidateFrameLocked(event.ID, event.Reason != "swap")
		}
	case "Page.frameNavigated":
		var event struct {
			Frame struct {
				ID     string `json:"id"`
				Parent string `json:"parentId"`
				URL    string `json:"url"`
			} `json:"frame"`
		}
		if json.Unmarshal(msg.Params, &event) == nil {
			state := l.frameStateLocked(event.Frame.ID)
			l.invalidateFrameLocked(event.Frame.ID, false)
			state.url, state.detached = event.Frame.URL, false
			// An OOPIF root omits parentId in its own renderer's events.
			if event.Frame.Parent != "" || event.Frame.ID == l.frameID {
				state.parent = event.Frame.Parent
			}
		}
	case "Page.navigatedWithinDocument":
		var event struct {
			ID  string `json:"frameId"`
			URL string `json:"url"`
		}
		if json.Unmarshal(msg.Params, &event) == nil {
			l.frameStateLocked(event.ID).url = event.URL
		}
	case "Page.frameStartedLoading", "Page.frameStoppedLoading":
		var event struct {
			ID string `json:"frameId"`
		}
		if json.Unmarshal(msg.Params, &event) == nil {
			l.frameStateLocked(event.ID).loading = msg.Method == "Page.frameStartedLoading"
		}
	case "Network.requestWillBeSent":
		var event struct {
			ID      string `json:"frameId"`
			Request string `json:"requestId"`
			Type    string `json:"type"`
		}
		if json.Unmarshal(msg.Params, &event) == nil && event.Type == "Document" {
			state := l.frameStateLocked(event.ID)
			state.request = event.Request
			state.requestEndpoint = endpointKey(endpoint)
			state.loading = true
			state.failure = nil
			state.status = 0
		}
	case "Network.responseReceived":
		var event struct {
			ID       string `json:"frameId"`
			Type     string `json:"type"`
			Response struct {
				URL    string  `json:"url"`
				Status float64 `json:"status"`
			} `json:"response"`
		}
		if json.Unmarshal(msg.Params, &event) == nil && event.Type == "Document" {
			state := l.frameStateLocked(event.ID)
			state.url = event.Response.URL
			state.status = int(event.Response.Status)
		}
	case "Network.loadingFailed":
		var event struct {
			Request  string `json:"requestId"`
			Error    string `json:"errorText"`
			Canceled bool   `json:"canceled"`
		}
		if json.Unmarshal(msg.Params, &event) == nil && !event.Canceled {
			for id, state := range l.frames.states {
				if id != l.frameID && state.request == event.Request && state.requestEndpoint == endpointKey(endpoint) {
					state.failure = fmt.Errorf("frame navigation failed: %s", event.Error)
					state.loading = false
				}
			}
		}
	}
	l.notifyLocked()
	return false
}

func (l *liveSession) scopeReadinessLocked() (error, bool, <-chan struct{}) {
	if l.failure != nil {
		return l.failure, false, l.changed
	}
	if scope := l.frames.selected; scope != nil && scope.id != l.frameID {
		state := l.frameStateLocked(scope.id)
		if state.detached {
			return errors.New("selected frame was removed; select another frame"), false, l.changed
		}
		return state.failure, state.loading, l.changed
	}
	return nil, l.loading, l.changed
}

func (l *liveSession) scopeStatusLocked() int {
	if scope := l.frames.selected; scope != nil && scope.id != l.frameID {
		return l.frameStateLocked(scope.id).status
	}
	return l.status
}

func (l *liveSession) evaluate(ctx context.Context, expression string) (json.RawMessage, error) {
	if err := l.ensureFrameWorld(ctx); err != nil {
		return nil, err
	}
	l.mu.Lock()
	var scope *liveFrameScope
	mainID := l.frameID
	if l.frames.selected != nil {
		copy := *l.frames.selected
		scope = &copy
	}
	l.mu.Unlock()
	if scope == nil {
		endpoint, contextID, unique, err := l.frameWorld(ctx, mainID)
		if err != nil {
			return nil, err
		}
		return endpoint.EvaluateWorld(ctx, expression, contextID, unique)
	}
	return scope.endpoint.EvaluateWorld(ctx, expression, scope.contextID, scope.unique)
}

// Pin only after a successfully decoded observation, immediately before it is
// published. Actions must remain in that document even if navigation starts
// between looking up a reference and sending the evaluation command.
func (l *liveSession) pinActionObservation() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	id := l.frameID
	scope := l.frames.selected
	if scope != nil {
		if scope.invalid {
			return errors.New("frame document changed; take a new snapshot")
		}
		id = scope.id
	}
	world, state := l.frames.worldByFrame[id], l.frames.states[id]
	if world == nil || state == nil || state.detached || world.revision != state.revision || (scope != nil && world.unique != scope.unique) {
		return errors.New("live document changed while observing; take a new snapshot")
	}
	l.frames.observed, l.frames.observedScope = world, scope
	return nil
}

// Actions never create or replace a context. A system-unique ID prevents the
// protocol from applying an old marker in a different document or renderer.
func (l *liveSession) evaluateAction(ctx context.Context, expression string) (json.RawMessage, error) {
	l.mu.Lock()
	world, scope := l.frames.observed, l.frames.selected
	valid := world != nil && scope == l.frames.observedScope && (scope == nil || !scope.invalid)
	if valid {
		state := l.frames.states[world.frame]
		valid = state != nil && !state.detached && state.revision == world.revision && l.frames.worldByFrame[world.frame] == world && (scope != nil || world.frame == l.frameID)
	}
	l.mu.Unlock()
	if !valid {
		return nil, errors.New("live document changed; use references from a new snapshot")
	}
	return world.endpoint.EvaluateWorld(ctx, expression, world.contextID, world.unique)
}

func (l *liveSession) ensureFrameWorld(ctx context.Context) error {
	l.mu.Lock()
	scope := l.frames.selected
	invalid := scope != nil && scope.invalid
	var id, ref string
	if invalid {
		id, ref = scope.id, scope.ref
	}
	l.mu.Unlock()
	if !invalid {
		return nil
	}
	return l.createFrameWorld(ctx, id, ref)
}

// A same-process child inside an OOPIF belongs to that ancestor's endpoint,
// rather than the main renderer. Incomplete ancestry is an explicit error.
func (l *liveSession) frameEndpointLocked(id string) (*cdpSession, error) {
	if l.session == nil || l.session.pageCDP == nil || id == "" {
		return nil, errors.New("frame context unavailable")
	}
	seen := make(map[string]bool)
	for count := 0; count <= maxLiveFrames; count++ {
		if id == l.frameID {
			return l.session.pageCDP.session(""), nil
		}
		if seen[id] {
			return nil, errors.New("invalid frame ancestry")
		}
		seen[id] = true
		state := l.frames.states[id]
		if state == nil || state.detached {
			return nil, errors.New("selected frame was removed; select another frame")
		}
		if endpoint := l.frames.endpoints[id]; endpoint != nil {
			return endpoint, nil
		}
		if state.parent == "" {
			return nil, errors.New("frame ancestry unavailable; run frames again")
		}
		id = state.parent
	}
	return nil, errors.New("frame ancestry exceeds budget")
}

// frameWorld does not alter the selected frame or reference namespace. It
// returns only contexts observed with a system-unique ID, so process swaps
// cannot silently reuse a numeric execution context in the wrong document.
func (l *liveSession) frameWorld(ctx context.Context, id string) (*cdpSession, int64, string, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, 0, "", err
		}
		l.mu.Lock()
		endpoint, err := l.frameEndpointLocked(id)
		if err != nil {
			l.mu.Unlock()
			return nil, 0, "", err
		}
		state := l.frameStateLocked(id)
		revision := state.revision
		if world := l.frames.worldByFrame[id]; world != nil && world.revision == revision && sameFrameEndpoint(world.endpoint, endpoint) {
			l.mu.Unlock()
			return world.endpoint, world.contextID, world.unique, nil
		}
		if l.frames.creating[id] != nil {
			changed := l.changed
			l.mu.Unlock()
			if err := l.waitFrameChange(ctx, changed); err != nil {
				return nil, 0, "", err
			}
			continue
		}
		creation := &liveFrameCreation{endpoint: endpoint, revision: revision}
		l.frames.creating[id] = creation
		l.mu.Unlock()
		world, err := l.createFrameContext(ctx, id, creation)
		l.mu.Lock()
		if l.frames.creating[id] == creation {
			delete(l.frames.creating, id)
			if err != nil {
				l.dropFrameWorldLocked(id)
			}
			l.notifyLocked()
		}
		l.mu.Unlock()
		if errors.Is(err, errFrameContextChanged) {
			continue
		}
		if err != nil {
			return nil, 0, "", err
		}
		return world.endpoint, world.contextID, world.unique, nil
	}
}

var errFrameContextChanged = errors.New("frame context changed")

func (l *liveSession) waitFrameChange(ctx context.Context, changed <-chan struct{}) error {
	select {
	case <-changed:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("frame unique context unavailable: %w", ctx.Err())
	case <-l.done:
		return errors.New("live browser closed")
	}
}

func (l *liveSession) createFrameContext(ctx context.Context, id string, creation *liveFrameCreation) (*liveFrameWorld, error) {
	endpoint := creation.endpoint
	raw, err := endpoint.send(ctx, "Page.createIsolatedWorld", map[string]any{"frameId": id, "worldName": "ai-go-browser-frame"})
	if err != nil {
		return nil, fmt.Errorf("frame context unavailable: %w", err)
	}
	var result struct {
		ID int64 `json:"executionContextId"`
	}
	if json.Unmarshal(raw, &result) != nil || result.ID <= 0 {
		return nil, errors.New("frame has no execution context")
	}
	for {
		l.mu.Lock()
		state := l.frames.states[id]
		currentEndpoint, endpointErr := l.frameEndpointLocked(id)
		if state == nil || state.detached || endpointErr != nil {
			l.mu.Unlock()
			return nil, errors.New("frame was removed while creating its context")
		}
		if state.revision != creation.revision || !sameFrameEndpoint(currentEndpoint, endpoint) {
			l.mu.Unlock()
			return nil, errFrameContextChanged
		}
		world := l.frames.worlds[worldKey(endpoint, result.ID)]
		if world == nil {
			if known := l.frames.knownWorlds[worldKey(endpoint, result.ID)]; known != nil && known.frame == id {
				copy := *known
				copy.revision = creation.revision
				world = &copy
			}
		}
		if world != nil && world.frame == id && world.revision == creation.revision {
			// Only the ID in this command's response can be published as a reusable
			// world. Late creation events from an older document stay candidates.
			l.dropFrameWorldLocked(id)
			l.frames.worlds[worldKey(endpoint, world.contextID)] = world
			l.frames.worldByFrame[id] = world
			l.mu.Unlock()
			return world, nil
		}
		changed := l.changed
		l.mu.Unlock()
		if err = l.waitFrameChange(ctx, changed); err != nil {
			return nil, err
		}
	}
}

func (l *liveSession) createFrameWorld(ctx context.Context, id, ref string) error {
	endpoint, contextID, unique, err := l.frameWorld(ctx, id)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	world := l.frames.worldByFrame[id]
	state := l.frames.states[id]
	if state == nil || state.detached || world == nil || world.revision != state.revision || world.unique != unique || world.contextID != contextID || !sameFrameEndpoint(world.endpoint, endpoint) {
		return errors.New("frame context changed while selecting; run frame again")
	}
	l.frames.generation++
	l.frames.selected = &liveFrameScope{id: id, ref: ref, prefix: ref + ".d" + strconv.FormatUint(l.frames.generation, 10) + ":", contextID: contextID, endpoint: endpoint, unique: unique}
	return nil
}

func (l *liveSession) namespaceScope(d *Document, p *Page) {
	l.mu.Lock()
	scope := l.frames.selected
	l.mu.Unlock()
	if scope == nil {
		p.FrameRef = "fr1"
		return
	}
	p.FrameRef = scope.ref
	prefix := scope.prefix
	rewrite := func(ref string) string {
		if ref == "" {
			return ""
		}
		return prefix + ref
	}
	fields, buttons, links := map[string]*Node{}, map[string]*Node{}, map[string]*Node{}
	for ref, n := range d.fields {
		n.ref = rewrite(ref)
		fields[n.ref] = n
	}
	for ref, n := range d.buttons {
		n.ref = rewrite(ref)
		buttons[n.ref] = n
	}
	for ref, n := range d.links {
		links[rewrite(ref)] = n
		if !n.genericButton() {
			n.ref = rewrite(ref)
		}
	}
	d.fields, d.buttons, d.links = fields, buttons, links
	for i := range p.Inputs {
		p.Inputs[i].Ref = rewrite(p.Inputs[i].Ref)
	}
	for i := range p.Buttons {
		p.Buttons[i].Ref = rewrite(p.Buttons[i].Ref)
		p.Buttons[i].FormRef = rewrite(p.Buttons[i].FormRef)
	}
	for i := range p.Links {
		p.Links[i].Ref = rewrite(p.Links[i].Ref)
	}
	for i := range p.Forms {
		form := &p.Forms[i]
		form.Ref = rewrite(form.Ref)
		form.SubmitRef = rewrite(form.SubmitRef)
		for j := range form.Fields {
			form.Fields[j].Ref = rewrite(form.Fields[j].Ref)
		}
	}
	controlWalk(d.Root, func(n *Node) {
		if n.Tag == "form" {
			n.ref = rewrite(n.ref)
		}
	})
}

func (b *Browser) selectLiveFrame(ref string) error {
	if b.live == nil {
		return errors.New("frame requires an active Chromium session")
	}
	l := b.live
	l.mu.Lock()
	id := l.frames.refs[ref]
	l.mu.Unlock()
	if id == "" {
		return errors.New("unknown frame reference; run frames before selecting")
	}
	ctx, cancel := b.liveContext()
	defer cancel()
	frames, err := b.liveFramesContext(ctx)
	if err != nil {
		return err
	}
	found, allowed := false, false
	for _, frame := range frames {
		if frame.ID == id {
			found, allowed = true, frame.Allowed
			break
		}
	}
	if !found {
		return errors.New("frame was removed; run frames again")
	}
	if !allowed {
		return errors.New("frame destination is not allowed")
	}
	l.mu.Lock()
	previous := l.frames.selected
	l.mu.Unlock()
	if err = l.createFrameWorld(ctx, id, ref); err != nil {
		return err
	}
	if err = b.refreshLiveContext(ctx); err != nil {
		l.mu.Lock()
		l.frames.selected = previous
		l.mu.Unlock()
		return err
	}
	return nil
}

func (b *Browser) validateLiveScope(ctx context.Context, target string) error {
	u, err := url.Parse(target)
	if err != nil {
		return err
	}
	if err = b.validate(u); err == nil {
		return nil
	}
	if target != "about:blank" && target != "about:srcdoc" && !strings.HasPrefix(target, "blob:") {
		return err
	}
	frames, listErr := b.liveFramesContext(ctx)
	if listErr != nil {
		return listErr
	}
	b.live.mu.Lock()
	scope := b.live.frames.selected
	b.live.mu.Unlock()
	if scope != nil {
		for _, frame := range frames {
			if frame.ID == scope.id && frame.Allowed {
				return nil
			}
		}
	}
	return err
}
