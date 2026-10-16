package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const maxLiveFrames = 100
const maxFrameMetadataBytes = 256 << 10

// Frame identifies an observed document. Allowed reports its destination policy.
type Frame struct {
	Ref       string `json:"ref"`
	ID        string `json:"id"`
	ParentRef string `json:"parent_ref,omitempty"`
	URL       string `json:"url"`
	Name      string `json:"name,omitempty"`
	Main      bool   `json:"main"`
	Allowed   bool   `json:"allowed"`
	Selected  bool   `json:"selected"`
}

type liveFrameTree struct {
	Frame struct {
		ID       string `json:"id"`
		URL      string `json:"url"`
		Name     string `json:"name"`
		ParentID string `json:"parentId"`
	} `json:"frame"`
	Children []liveFrameTree `json:"childFrames"`
}

type frameCaptureState struct {
	state       *liveFrameState
	revision    uint64
	parent, url string
	detached    bool
}

// A capture spans several renderer RPCs. A complete tree may only prune the
// inventory it actually observed, never an attachment from a later instant.
type frameCaptureFence struct {
	mainID    string
	states    map[string]frameCaptureState
	endpoints map[string]*cdpSession
}

func (l *liveSession) frameCaptureFenceLocked() frameCaptureFence {
	fence := frameCaptureFence{mainID: l.frameID, states: make(map[string]frameCaptureState, len(l.frames.states)), endpoints: make(map[string]*cdpSession, len(l.frames.endpoints))}
	for id, state := range l.frames.states {
		fence.states[id] = frameCaptureState{state: state, revision: state.revision, parent: state.parent, url: state.url, detached: state.detached}
	}
	for id, endpoint := range l.frames.endpoints {
		fence.endpoints[id] = endpoint
	}
	return fence
}

func (f frameCaptureFence) matchesLocked(l *liveSession) bool {
	if f.mainID != l.frameID || len(f.states) != len(l.frames.states) || len(f.endpoints) != len(l.frames.endpoints) {
		return false
	}
	for id, previous := range f.states {
		state := l.frames.states[id]
		if state != previous.state || state.revision != previous.revision || state.parent != previous.parent || state.url != previous.url || state.detached != previous.detached {
			return false
		}
	}
	for id, endpoint := range f.endpoints {
		if l.frames.endpoints[id] != endpoint {
			return false
		}
	}
	return true
}

// liveFrames observes the current CDP frame tree without changing the selected
// document, session state or action references. No frame is navigated or read.
func (b *Browser) liveFrames() ([]Frame, error) {
	if b.live == nil {
		return nil, errors.New("frames requires an active Chromium session")
	}
	ctx, cancel := b.liveContext()
	defer cancel()
	return b.liveFramesContext(ctx)
}

func (b *Browser) liveFramesContext(ctx context.Context) ([]Frame, error) {
	if b.live == nil {
		return nil, errors.New("frames requires an active Chromium session")
	}
	l := b.live
	l.mu.Lock()
	fence := l.frameCaptureFenceLocked()
	l.mu.Unlock()
	raw, err := l.session.pageCDP.send(ctx, "Page.getFrameTree", nil)
	if err != nil {
		return nil, fmt.Errorf("list live frames: %w", err)
	}
	var response struct {
		Tree liveFrameTree `json:"frameTree"`
	}
	if len(raw) > maxFrameMetadataBytes {
		return nil, errors.New("frame metadata exceeds budget")
	}
	if err = json.Unmarshal(raw, &response); err != nil {
		return nil, err
	}
	// Remote process roots may be omitted, or have incomplete children in the
	// main session's tree. Merge by actual FrameID, never by TargetID.
	var pending []liveFrameTree
	for id, endpoint := range fence.endpoints {
		childRaw, childErr := endpoint.send(ctx, "Page.getFrameTree", nil)
		if childErr != nil {
			l.mu.Lock()
			state := l.frames.states[id]
			absent := state == nil || state.detached || l.frames.endpoints[id] != endpoint
			l.mu.Unlock()
			if absent {
				continue
			}
			return nil, fmt.Errorf("list child frame: %w", childErr)
		} // a detached target is absent in the next tree
		if len(childRaw) > maxFrameMetadataBytes {
			return nil, errors.New("frame metadata exceeds budget")
		}
		var child struct {
			Tree liveFrameTree `json:"frameTree"`
		}
		if err = json.Unmarshal(childRaw, &child); err != nil {
			return nil, err
		}
		if child.Tree.Frame.ParentID == "" {
			l.mu.Lock()
			if state := l.frames.states[child.Tree.Frame.ID]; state != nil {
				child.Tree.Frame.ParentID = state.parent
			}
			l.mu.Unlock()
		}
		pending = append(pending, child.Tree)
	}
	for len(pending) > 0 {
		progress := false
		remaining := pending[:0]
		for _, tree := range pending {
			if mergeFrameTree(&response.Tree, tree) {
				progress = true
			} else {
				remaining = append(remaining, tree)
			}
		}
		pending = remaining
		if !progress {
			return nil, errors.New("remote frame ancestry unavailable; frame list is incomplete")
		}
	}
	raw, err = json.Marshal(response)
	if err != nil {
		return nil, err
	}
	frames, err := b.framesFromMetadata(raw)
	if err != nil {
		return nil, err
	}
	// Metadata validation is pure. Publish parent relationships, references and
	// pruning under one lock only if the renderer inventory is still unchanged.
	parents, capturedIDs := map[string]string{}, map[string]string{}
	for _, frame := range frames {
		capturedIDs[frame.Ref] = frame.ID
	}
	for _, frame := range frames {
		parents[frame.ID] = capturedIDs[frame.ParentRef]
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if !fence.matchesLocked(l) {
		return nil, errors.New("frame topology changed while listing; run frames again")
	}
	refs, byID := map[string]string{}, map[string]string{}
	active := make(map[string]bool, len(frames))
	for i := range frames {
		frame := &frames[i]
		active[frame.ID] = true
		ref := l.frames.refByID[frame.ID]
		if frame.Main {
			ref = "fr1"
		} else if ref == "" {
			ref = "fr" + strconv.Itoa(l.frames.nextRef)
			l.frames.nextRef++
		}
		byID[frame.ID] = ref
		frame.Ref = ref
		if selected := l.frames.selected; selected != nil {
			frame.Selected = selected.id == frame.ID
		} else {
			frame.Selected = frame.Main
		}
		state := l.frameStateLocked(frame.ID)
		state.url, state.parent, state.detached = frame.URL, parents[frame.ID], false
		refs[ref] = frame.ID
	}
	for i := range frames {
		if !frames[i].Main {
			frames[i].ParentRef = byID[l.frameStateLocked(frames[i].ID).parent]
		}
	}
	l.frames.refs, l.frames.refByID = refs, byID
	l.pruneFrameStatesLocked(active)
	return frames, nil
}

func mergeFrameTree(root *liveFrameTree, tree liveFrameTree) bool {
	if root.Frame.ID == tree.Frame.ID {
		if tree.Frame.URL != "" {
			root.Frame.URL = tree.Frame.URL
		}
		if tree.Frame.Name != "" {
			root.Frame.Name = tree.Frame.Name
		}
		if tree.Frame.ParentID != "" {
			root.Frame.ParentID = tree.Frame.ParentID
		}
		for _, child := range tree.Children {
			merged := false
			for i := range root.Children {
				if root.Children[i].Frame.ID == child.Frame.ID {
					mergeFrameTree(&root.Children[i], child)
					merged = true
					break
				}
			}
			if !merged {
				root.Children = append(root.Children, child)
			}
		}
		return true
	}
	for i := range root.Children {
		if mergeFrameTree(&root.Children[i], tree) {
			return true
		}
	}
	if root.Frame.ID == tree.Frame.ParentID {
		root.Children = append(root.Children, tree)
		return true
	}
	return false
}

func (b *Browser) framesFromMetadata(raw json.RawMessage) ([]Frame, error) {
	if len(raw) > maxFrameMetadataBytes {
		return nil, fmt.Errorf("frame metadata exceeds %d bytes", maxFrameMetadataBytes)
	}
	var response struct {
		Tree liveFrameTree `json:"frameTree"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("invalid frame metadata: %w", err)
	}
	var frames []Frame
	seen := make(map[string]bool)
	var visit func(liveFrameTree, string, bool) error
	visit = func(tree liveFrameTree, parent string, parentAllowed bool) error {
		if len(frames) >= maxLiveFrames {
			return fmt.Errorf("frame tree exceeds %d frames; list is incomplete", maxLiveFrames)
		}
		if tree.Frame.ID == "" || seen[tree.Frame.ID] {
			return errors.New("invalid frame metadata: missing or duplicate frame ID")
		}
		seen[tree.Frame.ID] = true
		u, err := url.Parse(tree.Frame.URL)
		allowed := err == nil && b.validate(u) == nil
		if tree.Frame.URL == "about:blank" || tree.Frame.URL == "about:srcdoc" {
			allowed = parent != "" && parentAllowed
		}
		if strings.HasPrefix(tree.Frame.URL, "blob:") {
			origin, err := url.Parse(strings.TrimPrefix(tree.Frame.URL, "blob:"))
			allowed = err == nil && b.validate(origin) == nil
		}
		ref := "fr" + strconv.Itoa(len(frames)+1)
		frames = append(frames, Frame{Ref: ref, ID: tree.Frame.ID, ParentRef: parent, URL: tree.Frame.URL, Name: tree.Frame.Name, Main: parent == "", Allowed: allowed})
		for _, child := range tree.Children {
			if err := visit(child, ref, allowed); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(response.Tree, "", false); err != nil {
		// Never present a silently truncated tree as a complete observation.
		return nil, err
	}
	return frames, nil
}
