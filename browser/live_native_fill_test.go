package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// Record DOM state, rather than isolated-world globals, so the same assertions
// can inspect controls inside a selected cross-origin frame or an open shadow.
const nativeFillRecorder = `<script>
function recordNativeFill(root) {
	for (const element of root.querySelectorAll('input,textarea,[contenteditable]')) {
		const events = [];
		for (const type of ['beforeinput','input']) element.addEventListener(type, event => {
			events.push({type:event.type,trusted:event.isTrusted,value:'value' in element ? element.value : element.textContent,
				rendered:'value' in element ? element.value : element.innerText});
			element.setAttribute('data-native-events', JSON.stringify(events));
			if (event.type === 'beforeinput' && element.hasAttribute('data-cancel')) event.preventDefault();
		});
	}
}
recordNativeFill(document);
</script>`

type nativeFillEvent struct {
	Type     string `json:"type"`
	Trusted  bool   `json:"trusted"`
	Value    string `json:"value"`
	Rendered string `json:"rendered"`
}

type nativeFillState struct {
	Value       string            `json:"value"`
	Rendered    string            `json:"rendered"`
	HTML        string            `json:"html"`
	MarkupNodes int               `json:"markup_nodes"`
	Paragraphs  []string          `json:"paragraphs"`
	BadInput    bool              `json:"bad_input"`
	Events      []nativeFillEvent `json:"events"`
}

func nativeFillElement(t *testing.T, b *Browser, selector string) Element {
	t.Helper()
	result := b.Execute(Command{Cmd: "query", Selector: selector})
	if !result.OK || len(result.Matches) != 1 || result.Matches[0].Ref == "" {
		t.Fatalf("query %q: matches=%+v error=%s", selector, result.Matches, result.Error)
	}
	return result.Matches[0]
}

func nativeFillRead(t *testing.T, b *Browser, selector string) nativeFillState {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	raw, err := b.live.evaluate(ctx, `(() => {
		`+liveDOMHelpers+`
		const elements = liveDOM.query(`+jsLiteral(selector)+`, 2);
		if (elements.length !== 1) throw new Error('expected one native fill target');
		const element = elements[0];
		return {value:'value' in element ? element.value : element.textContent,
			rendered:'value' in element ? element.value : element.innerText,
			html:element.innerHTML,
			markup_nodes:element.querySelectorAll('b,img,script').length,
			paragraphs:Array.from(element.querySelectorAll('p'), paragraph => paragraph.textContent),
			bad_input:!!(element.validity && element.validity.badInput),
			events:JSON.parse(element.getAttribute('data-native-events') || '[]')};
	})()`)
	if err != nil {
		t.Fatalf("observe %q: %v", selector, err)
	}
	var state nativeFillState
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("decode native fill state: %v", err)
	}
	return state
}

func nativeFillAssertTrusted(t *testing.T, events []nativeFillEvent, value string) {
	t.Helper()
	before, inputs := 0, 0
	lastInput := ""
	for _, event := range events {
		if !event.Trusted {
			t.Fatalf("synthetic editing event: %+v", events)
		}
		switch event.Type {
		case "beforeinput":
			before++
		case "input":
			inputs++
			lastInput = event.Value
		}
	}
	if before == 0 || inputs == 0 || lastInput != value {
		t.Fatalf("expected trusted beforeinput/input ending with %q: %+v", value, events)
	}
}

func TestLiveNativeFillTextControlsAndShadow(t *testing.T) {
	b, _ := liveDOMTestPage(t, `<!doctype html><meta charset="utf-8"><title>Native fill</title>
		<input id="text" value="old café 🙂">
		<textarea id="textarea">old café 🙂
second line</textarea>
		<input id="email" type="email" value="old@example.test">
		<input id="number" type="number" step="any" value="17">
		<input id="password" type="password" value="fake old password">
		<input id="search" type="search" value="old search">
		<input id="tel" type="tel" value="000">
		<input id="url" type="url" value="https://example.test/old">
		<x-fields></x-fields>`+nativeFillRecorder+`<script>
		const root = document.querySelector('x-fields').attachShadow({mode:'open'});
		root.innerHTML = '<input id="shadow-text" value="shadow old café 🙂"><textarea id="shadow-textarea">shadow old</textarea>';
		recordNativeFill(root);
		</script>`)
	cases := []struct {
		id, value string
	}{
		{"text", "Olá café e\u0301 🙂 <b>plain</b>"},
		{"textarea", "Olá café 🙂\nsegunda linha e\u0301"},
		{"email", "café+🙂@example.test"},
		{"number", "42.5"},
		{"password", "fake secret café 🙂"},
		{"search", "search café 🙂"},
		{"tel", "+55 11 1234-5678"},
		{"url", "https://example.test/café?q=🙂"},
		{"shadow-text", "shadow café 🙂"},
		{"shadow-textarea", "shadow café 🙂\nnew line"},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			selector := "#" + tc.id
			ref := nativeFillElement(t, b, selector).Ref
			result := b.Execute(Command{Cmd: "fill", Ref: ref, Value: tc.value})
			if !result.OK {
				t.Fatalf("native replacement: %s", result.Error)
			}
			state := nativeFillRead(t, b, selector)
			if state.Value != tc.value {
				t.Fatalf("replacement value=%q, want %q", state.Value, tc.value)
			}
			nativeFillAssertTrusted(t, state.Events, tc.value)
			snapshotValue := tc.value
			if tc.id == "password" {
				snapshotValue = "[redacted]"
			}
			if got := nativeFillElement(t, b, selector).Value; got != snapshotValue {
				t.Fatalf("snapshot value=%q, want %q", got, snapshotValue)
			}

			previousEvents := len(state.Events)
			ref = nativeFillElement(t, b, selector).Ref
			result = b.Execute(Command{Cmd: "fill", Ref: ref, Value: ""})
			if !result.OK {
				t.Fatalf("native clear: %s", result.Error)
			}
			state = nativeFillRead(t, b, selector)
			if state.Value != "" {
				t.Fatalf("clear left %q", state.Value)
			}
			nativeFillAssertTrusted(t, state.Events[previousEvents:], "")
			// Live observations redact nonempty passwords; an empty value stays
			// empty. The real value and trusted events above are test-only reads.
			if got := nativeFillElement(t, b, selector).Value; got != "" {
				t.Fatalf("snapshot retained cleared value %q", got)
			}
		})
	}
}

func TestLiveNativeFillEditableMultilineObservation(t *testing.T) {
	b, _ := liveDOMTestPage(t, `<!doctype html><meta charset="utf-8"><title>Multiline editors</title>
		<div id="rich-editor" contenteditable="true" style="white-space:pre-wrap;min-height:40px"><p>old <strong>rich</strong> content</p></div>
		<x-plain-editor></x-plain-editor>`+nativeFillRecorder+`<script>
		const root = document.querySelector('x-plain-editor').attachShadow({mode:'open'});
		root.innerHTML = '<div id="plain-shadow-editor" contenteditable="plaintext-only" style="white-space:pre-wrap;min-height:40px">old plain content</div>';
		recordNativeFill(root);
		</script>`)
	cases := []struct {
		name, value string
	}{
		{"blank-lines", "  início café 🙂\n\nsegunda linha e\u0301  "},
		{"crlf", "primeira café 🙂\r\n\r\núltima e\u0301"},
		{"literal-markup", "<b>café 🙂</b>\n\n<img src=x> & texto literal"},
		{"clear", ""},
	}
	for _, id := range []string{"rich-editor", "plain-shadow-editor"} {
		t.Run(id, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					selector := "#" + id
					previous := nativeFillRead(t, b, selector)
					result := b.Execute(Command{Cmd: "fill", Ref: nativeFillElement(t, b, selector).Ref, Value: tc.value})
					if !result.OK || result.Page == nil {
						state := nativeFillRead(t, b, selector)
						t.Fatalf("native editable fill: OK=%v error=%s rendered=%q textContent=%q html=%q events=%+v", result.OK, result.Error, state.Rendered, state.Value, state.HTML, state.Events)
					}
					state := nativeFillRead(t, b, selector)
					expected := strings.ReplaceAll(strings.ReplaceAll(tc.value, "\r\n", "\n"), "\r", "\n")
					rendered := state.Rendered
					if id == "rich-editor" {
						// Chromium preserves this editor's paragraph model. The
						// native paragraphs preserve the inserted lines, while
						// their rendered spacing differs from literal LF counts.
						if tc.value == "" {
							if state.Value != "" {
								t.Fatalf("rich clear retained text: %+v", state)
							}
						} else if len(state.Paragraphs) != 3 || strings.Join(state.Paragraphs, "\n") != expected {
							t.Fatalf("native paragraphs=%q, want three lines preserving %q; html=%q", state.Paragraphs, expected, state.HTML)
						}
					} else if rendered != expected {
						t.Fatalf("plaintext rendered text=%q, want %q", rendered, expected)
					}
					if state.MarkupNodes != 0 {
						t.Fatalf("literal text became DOM markup: %+v", state)
					}
					query := nativeFillElement(t, b, selector)
					if query.Type != "contenteditable" || query.Value != rendered {
						t.Fatalf("query disagrees with rendered text %q: %+v", rendered, query)
					}
					found := false
					for _, field := range result.Page.Inputs {
						if field.ID == id {
							found = true
							if field.Type != "contenteditable" || field.Value != rendered {
								t.Fatalf("fill snapshot disagrees with rendered text %q: %+v", rendered, field)
							}
						}
					}
					if !found {
						t.Fatalf("editor %q missing from fill snapshot: %+v", id, result.Page.Inputs)
					}
					events := append([]nativeFillEvent(nil), state.Events[len(previous.Events):]...)
					for i := range events {
						events[i].Value = events[i].Rendered
					}
					// The event oracle keeps the browser's raw rendered text,
					// independently of the Go observations above.
					nativeFillAssertTrusted(t, events, state.Rendered)
				})
			}
		})
	}
}

func TestLiveNativeFillReportsFormattedAndLimitedResults(t *testing.T) {
	b, _ := liveDOMTestPage(t, `<!doctype html><meta charset="utf-8"><title>Native edit outcomes</title>
		<input id="formatted" value="old">
		<input id="limited" maxlength="4" value="old">
		<input id="number-minus" type="number" value="17">
		<input id="number-exponent" type="number" value="17">
		<script>document.getElementById('formatted').addEventListener('input', event => {
			event.target.value = event.target.value.toUpperCase();
		});</script>`+nativeFillRecorder)
	cases := []struct {
		id, requested, actual string
		badInput              bool
	}{
		{"formatted", "café", "CAFÉ", false},
		{"limited", "abcdefgh", "abcd", false},
		{"number-minus", "-", "", true},
		{"number-exponent", "1e", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			selector := "#" + tc.id
			result := b.Execute(Command{Cmd: "fill", Ref: nativeFillElement(t, b, selector).Ref, Value: tc.requested})
			if result.OK || !strings.Contains(strings.ToLower(result.Error), "did not accept") || result.Page == nil {
				t.Fatalf("altered native result must return an error and observation: OK=%v error=%s page=%+v", result.OK, result.Error, result.Page)
			}
			state := nativeFillRead(t, b, selector)
			if state.Value != tc.actual || state.BadInput != tc.badInput {
				t.Fatalf("native result was overwritten: value=%q badInput=%v, want %q/%v", state.Value, state.BadInput, tc.actual, tc.badInput)
			}
			nativeFillAssertTrusted(t, state.Events, tc.actual)
			// Inspect the returned failure observation before querying again:
			// a query refresh must not hide an outdated Page in the fill result.
			found := false
			for _, field := range result.Page.Inputs {
				if field.ID == tc.id {
					found = true
					if field.Value != tc.actual {
						t.Fatalf("failed fill returned stale value %q, want native result %q", field.Value, tc.actual)
					}
				}
			}
			if !found {
				t.Fatalf("control %q missing from failure observation: %+v", tc.id, result.Page.Inputs)
			}
			if query := nativeFillElement(t, b, selector); query.Value != tc.actual {
				t.Fatalf("query disagrees with altered native value %q: %+v", tc.actual, query)
			}
		})
	}
}

func TestLiveNativeFillHonorsBeforeInputCancellation(t *testing.T) {
	b, _ := liveDOMTestPage(t, `<!doctype html><title>Cancel native edits</title>
		<input id="cancel-text" data-cancel value="keep café 🙂">
		<textarea id="cancel-textarea" data-cancel>keep café 🙂</textarea>
		<div id="cancel-editor" data-cancel contenteditable="true" style="min-height:30px">keep café 🙂</div>`+nativeFillRecorder)
	for _, selector := range []string{"#cancel-text", "#cancel-textarea", "#cancel-editor"} {
		t.Run(selector, func(t *testing.T) {
			for _, value := range []string{"rejected replacement", ""} {
				before := nativeFillRead(t, b, selector)
				result := b.Execute(Command{Cmd: "fill", Ref: nativeFillElement(t, b, selector).Ref, Value: value})
				if !result.OK && result.Error == "" {
					t.Fatal("canceled fill failed without an error")
				}
				after := nativeFillRead(t, b, selector)
				if after.Value != "keep café 🙂" {
					t.Fatalf("canceled edit changed content: %+v", after)
				}
				events := after.Events[len(before.Events):]
				if len(events) == 0 {
					t.Fatal("native beforeinput was not offered to the canceling handler")
				}
				for _, event := range events {
					if event.Type != "beforeinput" || !event.Trusted {
						t.Fatalf("canceled edit emitted input or synthetic beforeinput: %+v", events)
					}
				}
			}
		})
	}
}

func TestLiveNativeFillRejectsMicrotaskFocusDiversion(t *testing.T) {
	b, _ := liveDOMTestPage(t, `<!doctype html><title>Focus diversion</title>
		<input id="target-text" value="keep text">
		<div id="target-editor" contenteditable="true" style="min-height:30px">keep editor</div>
		<input id="decoy" value="keep decoy">`+nativeFillRecorder+`<script>
		for (const id of ['target-text','target-editor']) document.getElementById(id).addEventListener('focus', () => {
			queueMicrotask(() => document.getElementById('decoy').focus());
		});
		</script>`)
	for _, selector := range []string{"#target-text", "#target-editor"} {
		t.Run(selector, func(t *testing.T) {
			before := nativeFillRead(t, b, selector)
			result := b.Execute(Command{Cmd: "fill", Ref: nativeFillElement(t, b, selector).Ref, Value: "must not reach decoy"})
			if result.OK || !strings.Contains(strings.ToLower(result.Error), "focus") {
				t.Fatalf("focus diversion must fail clearly: OK=%v error=%s", result.OK, result.Error)
			}
			after := nativeFillRead(t, b, selector)
			decoy := nativeFillRead(t, b, "#decoy")
			if after.Value != before.Value || len(after.Events) != 0 || decoy.Value != "keep decoy" || len(decoy.Events) != 0 {
				t.Fatalf("focus diversion edited a control: target=%+v decoy=%+v", after, decoy)
			}
		})
	}
}

func TestLiveNativeFillRejectsReadOnlyAndDisabled(t *testing.T) {
	b, _ := liveDOMTestPage(t, `<!doctype html><title>Protected controls</title>
		<input id="readonly" readonly value="locked">
		<textarea id="disabled" disabled>locked</textarea>
		<fieldset disabled><input id="fieldset" value="locked"></fieldset>
		<div aria-readonly="true"><div id="aria-readonly" contenteditable="true" style="min-height:30px">locked</div></div>
		<div aria-disabled="true"><input id="aria-disabled" value="locked"></div>
		<input id="focus-readonly" value="locked" onfocus="this.readOnly=true">
		<input id="focus-disabled" value="locked" onfocus="this.disabled=true">`+nativeFillRecorder)
	for _, selector := range []string{"#readonly", "#disabled", "#fieldset", "#aria-readonly", "#aria-disabled", "#focus-readonly", "#focus-disabled"} {
		t.Run(selector, func(t *testing.T) {
			result := b.Execute(Command{Cmd: "fill", Ref: nativeFillElement(t, b, selector).Ref, Value: "forbidden"})
			if result.OK || result.Error == "" {
				t.Fatalf("protected control accepted fill: OK=%v error=%s", result.OK, result.Error)
			}
			state := nativeFillRead(t, b, selector)
			if state.Value != "locked" || len(state.Events) != 0 {
				t.Fatalf("protected control changed: %+v", state)
			}
		})
	}
}

func TestLiveNativeFillSelectedSameAndCrossOriginFrames(t *testing.T) {
	var mainURL, otherOrigin string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/child" {
			fmt.Fprint(w, `<!doctype html><title>Child fill</title><input id="field" value="child old">
				<div id="editor" contenteditable="true" style="min-height:30px">child editor old</div>`+nativeFillRecorder)
			return
		}
		fmt.Fprintf(w, `<!doctype html><title>Main fill</title><input id="field" value="main unchanged">
			<div id="editor" contenteditable="true" style="min-height:30px">main editor unchanged</div>
			<iframe name="same-fill" src="%s/child"></iframe><iframe name="cross-fill" src="%s/child"></iframe>%s`, mainURL, otherOrigin, nativeFillRecorder)
	}))
	t.Cleanup(server.Close)
	mainURL = server.URL
	u, err := url.Parse(mainURL)
	if err != nil {
		t.Fatal(err)
	}
	u.Host = "localhost:" + u.Port()
	otherOrigin = u.String()
	b := newLiveTestBrowser(t)
	b.AllowAll = true
	frameAPICommand(t, b, Command{Cmd: "open", URL: mainURL, RenderJS: true})
	mainFrame := frameAPINamedMainForFill(t, frameAPIList(t, b))
	for _, name := range []string{"same-fill", "cross-fill"} {
		t.Run(name, func(t *testing.T) {
			frame := frameAPINamed(t, frameAPIList(t, b), name)
			frameAPISelect(t, b, frame.Ref)
			for _, selector := range []string{"#field", "#editor"} {
				value := name + selector + " café 🙂"
				result := b.Execute(Command{Cmd: "fill", Ref: nativeFillElement(t, b, selector).Ref, Value: value})
				if !result.OK || result.Page == nil || result.Page.FrameRef != frame.Ref {
					t.Fatalf("fill escaped selected frame: OK=%v error=%s page=%+v", result.OK, result.Error, result.Page)
				}
				state := nativeFillRead(t, b, selector)
				if state.Value != value {
					t.Fatalf("selected frame value=%q, want %q", state.Value, value)
				}
				nativeFillAssertTrusted(t, state.Events, value)
			}
			frameAPISelect(t, b, mainFrame.Ref)
			mainField := nativeFillRead(t, b, "#field")
			mainEditor := nativeFillRead(t, b, "#editor")
			if b.Current.URL != mainURL+"/" || mainField.Value != "main unchanged" || mainEditor.Value != "main editor unchanged" || len(mainField.Events) != 0 || len(mainEditor.Events) != 0 {
				t.Fatalf("child fill altered main document: URL=%s field=%+v editor=%+v", b.Current.URL, mainField, mainEditor)
			}
		})
	}
}

func frameAPINamedMainForFill(t *testing.T, frames []Frame) Frame {
	t.Helper()
	for _, frame := range frames {
		if frame.Main {
			return frame
		}
	}
	t.Fatalf("main frame missing from %+v", frames)
	return Frame{}
}
