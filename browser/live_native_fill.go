package browser

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

func liveNativeTextField(kind string) bool {
	switch kind {
	case "text", "textarea", "search", "tel", "url", "email", "password", "number":
		return true
	}
	return false
}

// Fill text through Chromium's editing pipeline. A veto or a formatter can
// change the outcome; never replace the native result with a forced setter.
func (b *Browser) fillLiveText(ctx context.Context, expression, value string, editable bool) error {
	check := liveDOMHelpers + `
		const element = ` + expression + `;
		const editable = ` + strconv.FormatBool(editable) + `;
		if (!element || !element.isConnected) throw new Error('stale live field reference; take a new snapshot');
		if (liveDOM.disabled(element) || liveDOM.readonly(element)) throw new Error('field cannot be edited');
		if (editable ? !element.isContentEditable : !(element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement)) throw new Error('field type changed; take a new snapshot');
		if (!editable && element instanceof HTMLInputElement && !['text','search','tel','url','email','password','number'].includes(element.type)) throw new Error('field type changed; take a new snapshot');
		const focused = () => {
			let active = document.activeElement;
			while (active && active.shadowRoot && active.shadowRoot.activeElement) active = active.shadowRoot.activeElement;
			return document.hasFocus() && active === element;
		};
	`
	_, err := b.live.evaluateAction(ctx, `(() => {`+check+`
		element.scrollIntoView({block:'center', inline:'center'});
		if (!liveDOM.visible(element)) throw new Error('field is not visible');
		element.focus();
		if (!focused()) throw new Error('field did not receive focus');
		if (editable) {
			const selection = window.getSelection(), range = document.createRange();
			range.selectNodeContents(element);
			selection.removeAllRanges();
			selection.addRange(range);
		} else element.select();
		return true;
	})()`)
	if err != nil {
		return err
	}
	// Focus/selection listeners can schedule microtasks after preparation.
	// Recheck the pinned document before input targets the focused renderer.
	_, err = b.live.evaluateAction(ctx, `(() => {`+check+`
		if (!focused() || !liveDOM.visible(element)) throw new Error('field lost focus before native input; take a new snapshot');
		if (editable) {
			const selection = window.getSelection();
			if (!selection || selection.rangeCount !== 1) throw new Error('field selection changed before native input');
			const range = selection.getRangeAt(0);
			if (range.startContainer !== element || range.startOffset !== 0 || range.endContainer !== element || range.endOffset !== element.childNodes.length) throw new Error('field selection changed before native input');
		} else if (element.selectionStart !== null && (element.selectionStart !== 0 || element.selectionEnd !== element.value.length)) throw new Error('field selection changed before native input');
		return true;
	})()`)
	if err != nil {
		return err
	}
	if value == "" && !editable {
		for _, kind := range []string{"rawKeyDown", "keyUp"} {
			_, err = b.live.session.pageCDP.send(ctx, "Input.dispatchKeyEvent", map[string]any{"type": kind, "key": "Delete", "code": "Delete", "windowsVirtualKeyCode": 46, "nativeVirtualKeyCode": 46})
			if err != nil {
				return b.observeNativeFillFailure(ctx, err)
			}
		}
	} else {
		_, err = b.live.session.pageCDP.send(ctx, "Input.insertText", map[string]any{"text": value})
		if err != nil {
			return b.observeNativeFillFailure(ctx, err)
		}
	}
	// Rich editors retain their paragraph model: rendered line breaks can differ
	// from the requested text. Observe that native result without forcing it.
	// For controls, a detached clone obtains native value normalization; errors
	// never include requested or actual password values.
	_, err = b.live.evaluateAction(ctx, `(() => {`+check+`
		const requested = `+jsLiteral(value)+`;
		if (!editable) {
			const normalized = element.cloneNode(false);
			normalized.value = requested;
			if (element.type === 'number' && element.validity.badInput) throw new Error('number field did not accept a complete numeric value; observe the native result');
			if (element.value !== normalized.value) throw new Error('field did not accept the requested text; observe the native result');
		}
		return true;
	})()`)
	if err != nil {
		return b.observeNativeFillFailure(ctx, err)
	}
	return err
}

// An input handler may format or veto the value after a real edit. Preserve
// that outcome in the response even when fill reports that it was rejected.
func (b *Browser) observeNativeFillFailure(ctx context.Context, original error) error {
	if err := b.refreshLiveContext(ctx); err != nil {
		return errors.Join(original, fmt.Errorf("could not observe native fill outcome: %w", err))
	}
	return original
}
