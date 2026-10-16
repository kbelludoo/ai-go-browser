package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"
)

type frameCoordinateHop struct {
	child, parent string
}

// framePointToMain converts CSS viewport coordinates in the selected document
// into the main viewport used by Input.dispatchMouseEvent. Each owner is located
// by FrameID in its parent's renderer, so this works across origin boundaries.
// No input is dispatched when an ancestor clips or covers the requested point.
func (l *liveSession) framePointToMain(ctx context.Context, x, y float64) (float64, float64, error) {
	return l.convertFramePoint(ctx, x, y, true)
}

// verifyFramePointToMain rechecks every parent without scrolling or waiting for
// animation frames. It catches owners that moved or acquired an overlay after
// the first conversion, immediately before dispatching native input.
func (l *liveSession) verifyFramePointToMain(ctx context.Context, localX, localY, expectedMainX, expectedMainY float64) error {
	if math.IsNaN(expectedMainX) || math.IsNaN(expectedMainY) || math.IsInf(expectedMainX, 0) || math.IsInf(expectedMainY, 0) {
		return errors.New("invalid expected frame click coordinates")
	}
	x, y, err := l.convertFramePoint(ctx, localX, localY, false)
	if err != nil {
		return err
	}
	if math.Abs(x-expectedMainX) > .5 || math.Abs(y-expectedMainY) > .5 {
		return errors.New("frame moved while preparing click; take a new snapshot")
	}
	return nil
}

func (l *liveSession) convertFramePoint(ctx context.Context, x, y float64, scroll bool) (float64, float64, error) {
	if math.IsNaN(x) || math.IsNaN(y) || math.IsInf(x, 0) || math.IsInf(y, 0) {
		return 0, 0, errors.New("invalid frame click coordinates")
	}
	l.mu.Lock()
	scope := l.frames.selected
	hops, err := l.frameCoordinatePathLocked()
	l.mu.Unlock()
	if err != nil {
		return 0, 0, err
	}
	for _, hop := range hops {
		endpoint, contextID, _, err := l.frameWorld(ctx, hop.parent)
		if err != nil {
			return 0, 0, fmt.Errorf("parent frame context unavailable: %w", err)
		}
		x, y, err = frameOwnerPoint(ctx, endpoint, contextID, hop.child, x, y, scroll)
		if err != nil {
			return 0, 0, err
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.frames.selected != scope || (scope != nil && scope.invalid) {
		return 0, 0, errors.New("selected frame changed while preparing click; take a new snapshot")
	}
	for _, hop := range hops {
		state := l.frames.states[hop.child]
		if state == nil || state.detached || state.parent != hop.parent {
			return 0, 0, errors.New("frame ancestry changed while preparing click; take a new snapshot")
		}
	}
	return x, y, nil
}

// Caller holds liveSession.mu. Bound the path and reject incomplete metadata
// rather than accidentally sending child coordinates to the main viewport.
func (l *liveSession) frameCoordinatePathLocked() ([]frameCoordinateHop, error) {
	scope := l.frames.selected
	if scope == nil || scope.id == l.frameID {
		return nil, nil
	}
	if scope.invalid {
		return nil, errors.New("selected frame context changed; take a new snapshot")
	}
	var hops []frameCoordinateHop
	seen := map[string]bool{}
	for id := scope.id; id != l.frameID; {
		if id == "" || seen[id] || len(hops) >= maxLiveFrames {
			return nil, errors.New("invalid frame ancestry")
		}
		seen[id] = true
		state := l.frames.states[id]
		if state == nil || state.detached || state.parent == "" {
			return nil, errors.New("frame ancestry unavailable; run frames again")
		}
		hops = append(hops, frameCoordinateHop{child: id, parent: state.parent})
		id = state.parent
	}
	return hops, nil
}

func frameOwnerPoint(ctx context.Context, endpoint *cdpSession, contextID int64, child string, x, y float64, scroll bool) (float64, float64, error) {
	raw, err := endpoint.send(ctx, "DOM.getFrameOwner", map[string]any{"frameId": child})
	if err != nil {
		return 0, 0, fmt.Errorf("frame owner unavailable: %w", err)
	}
	var owner struct {
		BackendID int64 `json:"backendNodeId"`
	}
	if err = json.Unmarshal(raw, &owner); err != nil || owner.BackendID == 0 {
		return 0, 0, errors.New("frame has no owner element")
	}
	raw, err = endpoint.send(ctx, "DOM.resolveNode", map[string]any{"backendNodeId": owner.BackendID, "executionContextId": contextID})
	if err != nil {
		return 0, 0, fmt.Errorf("frame owner context unavailable: %w", err)
	}
	var resolved struct {
		Object struct {
			ID string `json:"objectId"`
		} `json:"object"`
	}
	if err = json.Unmarshal(raw, &resolved); err != nil || resolved.Object.ID == "" {
		return 0, 0, errors.New("frame owner cannot be resolved")
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_, _ = endpoint.send(releaseCtx, "Runtime.releaseObject", map[string]any{"objectId": resolved.Object.ID})
	}()
	call := func(prepare bool) (float64, float64, error) {
		raw, err := endpoint.send(ctx, "Runtime.callFunctionOn", map[string]any{
			"objectId": resolved.Object.ID, "functionDeclaration": frameOwnerPointFunction,
			"arguments":     []map[string]any{{"value": x}, {"value": y}, {"value": prepare}, {"value": scroll}},
			"returnByValue": true, "awaitPromise": true,
		})
		if err != nil {
			return 0, 0, err
		}
		value, err := cdpEvaluationValue(raw)
		if err != nil {
			return 0, 0, err
		}
		var point struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		}
		if err = json.Unmarshal(value, &point); err != nil || math.IsNaN(point.X) || math.IsNaN(point.Y) || math.IsInf(point.X, 0) || math.IsInf(point.Y, 0) {
			return 0, 0, errors.New("invalid frame owner coordinates")
		}
		return point.X, point.Y, nil
	}
	if scroll {
		borderX, borderY, err := call(true)
		if err != nil {
			return 0, 0, err
		}
		// Scrolling a point rather than the whole iframe also handles owners larger
		// than their parent viewport. The protocol rect is local to the border box.
		_, err = endpoint.send(ctx, "DOM.scrollIntoViewIfNeeded", map[string]any{"backendNodeId": owner.BackendID, "rect": map[string]any{"x": borderX, "y": borderY, "width": 1, "height": 1}})
		if err != nil {
			return 0, 0, fmt.Errorf("frame owner could not be scrolled into view: %w", err)
		}
	}
	return call(false)
}

const frameOwnerPointFunction = `async function(x, y, prepare, settle) {
	` + liveDOMHelpers + `
	const element = this;
	if (!element.isConnected || !['IFRAME','FRAME'].includes(element.tagName)) throw new Error('frame owner was removed');
	if (!prepare && settle) await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
	for (let current = element, depth = 0; current; current = liveDOM.parent(current)) {
		if (++depth > 512) throw new Error('frame owner ancestry exceeds 512 levels');
		const style = getComputedStyle(current);
		if (style.offsetPath && style.offsetPath !== 'none') throw new Error('frame click does not support motion-path transforms');
		if (style.perspective !== 'none') throw new Error('frame click does not support perspective transforms');
		if (style.rotate && !['none','0deg','0'].includes(style.rotate)) throw new Error('frame click does not support rotated transforms');
		if (style.scale && style.scale !== 'none' && style.scale.split(/\s+/).some(value => !(parseFloat(value) > 0))) throw new Error('frame click does not support reflected transforms');
		if (style.transform !== 'none') {
			const matrix = new DOMMatrixReadOnly(style.transform);
			if (!matrix.is2D || Math.abs(matrix.b) > 1e-7 || Math.abs(matrix.c) > 1e-7 || matrix.a <= 0 || matrix.d <= 0) throw new Error('frame click does not support rotated, skewed or reflected transforms');
		}
	}
	const style = getComputedStyle(element), rect = element.getBoundingClientRect();
	if (!liveDOM.visible(element)) throw new Error('frame owner is not visible');
	const number = value => parseFloat(value) || 0;
	const left = number(style.borderLeftWidth) + number(style.paddingLeft);
	const top = number(style.borderTopWidth) + number(style.paddingTop);
	const horizontal = left + number(style.borderRightWidth) + number(style.paddingRight);
	const vertical = top + number(style.borderBottomWidth) + number(style.paddingBottom);
	const width = number(style.width) + (style.boxSizing === 'border-box' ? 0 : horizontal);
	const height = number(style.height) + (style.boxSizing === 'border-box' ? 0 : vertical);
	if (!(width > horizontal && height > vertical) || !Number.isFinite(x) || !Number.isFinite(y) || x < 0 || y < 0 || x >= width-horizontal || y >= height-vertical) throw new Error('click point is outside the frame viewport');
	const offsetX = (left+x)*rect.width/width;
	const offsetY = (top+y)*rect.height/height;
	if (prepare) return {x:offsetX, y:offsetY};
	const parentX = rect.left + offsetX;
	const parentY = rect.top + offsetY;
	const hit = liveDOM.hit(parentX, parentY);
	if (hit !== element) throw new Error('frame is covered by another element in its parent document');
	return {x:parentX, y:parentY};
}`
