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
)

func liveDOMTestPage(t *testing.T, html string) (*Browser, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, html)
	}))
	t.Cleanup(srv.Close)
	b := newLiveTestBrowser(t)
	if err := b.openLive(srv.URL, false); err != nil {
		t.Fatal(err)
	}
	return b, srv.URL
}

func TestLiveOpenShadowComposedTreeAndNativeForm(t *testing.T) {
	b, _ := liveDOMTestPage(t, `<!doctype html><title>Shadow forms</title>
		<form id="duplicate"><input name="outer"><x-chat><span slot="heading">Projected title</span><p slot="unused">Unrendered light text</p></x-chat></form>
		<p id="state">Initial</p><script>
		const root = document.querySelector('x-chat').attachShadow({mode:'open'});
		root.innerHTML = '<slot name="heading"></slot><input name="orphan"><x-inner></x-inner>';
		const form = document.createElement('form'); form.id='duplicate';
		form.innerHTML='<input name="inner"><button>Send shadow</button>';
		root.insertBefore(form,root.querySelector('input'));
		form.onsubmit = event => {event.preventDefault();document.querySelector('#state').textContent='Sent: '+event.target.elements.inner.value};
		const nested = root.querySelector('x-inner').attachShadow({mode:'open'});
		nested.innerHTML = '<div role="button" aria-label="Nested action" style="padding:10px;display:block">Nested</div>';
		nested.querySelector('div').onclick = event => {document.querySelector('#state').textContent='Trusted: '+event.isTrusted};
		</script>`)
	if strings.Count(b.Current.Text, "Projected title") != 1 || strings.Contains(b.Current.Text, "Unrendered light text") {
		t.Fatalf("composed text: %q", b.Current.Text)
	}
	if len(b.Current.Forms) != 2 || len(b.Current.Forms[0].Fields) != 1 || len(b.Current.Forms[1].Fields) != 1 || len(b.Current.Inputs) != 1 {
		t.Fatalf("shadow form association: forms=%+v inputs=%+v", b.Current.Forms, b.Current.Inputs)
	}
	if b.Current.Forms[0].Fields[0].Name != "outer" || b.Current.Forms[1].Fields[0].Name != "inner" || b.Current.Inputs[0].Name != "orphan" {
		t.Fatalf("controls crossed root boundaries: %+v", b.Current.Forms)
	}
	matches, err := b.liveQuery("x-inner [role=button]")
	if err != nil || len(matches) != 0 {
		t.Fatalf("CSS combinator crossed shadow boundary: %+v %v", matches, err)
	}
	matches, err = b.liveQuery("[role=button]")
	if err != nil || len(matches) != 1 || matches[0].Ref == "" || matches[0].Type != "button" || matches[0].Label != "Nested action" {
		t.Fatalf("nested query: %+v %v", matches, err)
	}
	runLiveTestCommand(t, b, Command{Cmd: "press", Ref: matches[0].Ref})
	if !strings.Contains(b.Current.Text, "Trusted: true") {
		t.Fatalf("native shadow click: %s", b.Current.Text)
	}
	runLiveTestCommand(t, b, Command{Cmd: "fill", Ref: liveTestField(t, b, "inner"), Value: "Olá shadow"})
	runLiveTestCommand(t, b, Command{Cmd: "press", Ref: liveTestButton(t, b, "Send shadow")})
	if !strings.Contains(b.Current.Text, "Sent: Olá shadow") {
		t.Fatalf("shadow submit: %s", b.Current.Text)
	}
}

func TestLiveEditableTrustedInputAndARIA(t *testing.T) {
	b, _ := liveDOMTestPage(t, `<!doctype html><title>Editor</title>
		<x-editor></x-editor><div role="textbox" aria-label="Readonly widget">Static widget</div>
		<div aria-disabled="true"><a aria-label="Disabled action">Disabled</a></div>
		<script>
		const root = document.querySelector('x-editor').attachShadow({mode:'open'});
		root.innerHTML = '<div contenteditable="plaintext-only" role="textbox" aria-label="Chat message" style="min-height:40px;border:1px solid black">Old text</div><a aria-label="Send message" style="display:block;padding:10px">Send</a>';
		window.inputs=[];
		root.querySelector('div').addEventListener('input',event=>window.inputs.push({trusted:event.isTrusted,text:event.target.innerText}));
		root.querySelector('a').onclick=event=>window.sent=event.isTrusted;
		</script>`)
	q, err := b.liveQuery("[aria-label='Chat message']")
	if err != nil || len(q) != 1 || q[0].Ref == "" || q[0].Type != "contenteditable" || q[0].ReadOnly {
		t.Fatalf("editable query: %+v %v", q, err)
	}
	value := "Olá <img src=x> 🙂"
	runLiveTestCommand(t, b, Command{Cmd: "fill", Ref: q[0].Ref, Value: value})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	raw, err := b.live.session.Evaluate(ctx, `({inputs:window.inputs,images:document.querySelector('x-editor').shadowRoot.querySelectorAll('img').length})`)
	if err != nil {
		t.Fatal(err)
	}
	var state struct {
		Inputs []struct {
			Trusted bool   `json:"trusted"`
			Text    string `json:"text"`
		} `json:"inputs"`
		Images int `json:"images"`
	}
	if err := json.Unmarshal(raw, &state); err != nil || len(state.Inputs) != 1 || !state.Inputs[0].Trusted || state.Inputs[0].Text != value || state.Images != 0 {
		t.Fatalf("native editor result: %s %v", raw, err)
	}
	runLiveTestCommand(t, b, Command{Cmd: "fill", Ref: q[0].Ref, Value: ""})
	q, err = b.liveQuery("[aria-label='Chat message']")
	if err != nil || len(q) != 1 || q[0].Value != "" {
		t.Fatalf("clear editable: %+v %v", q, err)
	}
	runLiveTestCommand(t, b, Command{Cmd: "press", Ref: liveTestButton(t, b, "Send message")})
	raw, err = b.live.session.Evaluate(ctx, "window.sent")
	if err != nil || string(raw) != "true" {
		t.Fatalf("anchor button click: %s %v", raw, err)
	}
	q, err = b.liveQuery("[aria-label='Readonly widget']")
	if err != nil || len(q) != 1 || !q[0].ReadOnly {
		t.Fatalf("readonly widget: %+v %v", q, err)
	}
	if _, err := b.executeLive(Command{Cmd: "fill", Ref: q[0].Ref, Value: "forbidden"}); err == nil {
		t.Fatal("noneditable ARIA textbox accepted fill")
	}
	if _, err := b.executeLive(Command{Cmd: "press", Ref: liveTestButton(t, b, "Disabled action")}); err == nil {
		t.Fatal("ARIA disabled action was pressed")
	}
}

func TestLiveWaitFindsLateShadowAndVisibleMatch(t *testing.T) {
	b, _ := liveDOMTestPage(t, `<!doctype html><title>Late shadow</title><x-late></x-late>
		<div class="ready" style="display:none">Hidden first match</div>
		<button onclick="setTimeout(()=>{let root=document.querySelector('x-late').attachShadow({mode:'open'});root.innerHTML='<p class=ready>Ready in shadow</p>';},120)">Attach</button>`)
	runLiveTestCommand(t, b, Command{Cmd: "press", Ref: liveTestButton(t, b, "Attach")})
	if err := b.liveWait(".ready", 3*time.Second); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.Current.Text, "Ready in shadow") || strings.Contains(b.Current.Text, "Hidden first match") {
		t.Fatalf("shadow readiness: %s", b.Current.Text)
	}
	result := b.Execute(Command{Cmd: "frames"})
	if !result.OK || len(result.Frames) != 1 || !result.Frames[0].Main || result.Page != nil {
		t.Fatalf("frames observation: %+v", result)
	}
	if err := b.liveWait("[", 300*time.Millisecond); err == nil {
		t.Fatal("invalid wait selector accepted")
	}
}

func TestLiveQueryNativeARIAAndLinkButtonReferences(t *testing.T) {
	b, _ := liveDOMTestPage(t, `<!doctype html><title>ARIA refs</title>
		<input id="native" role="textbox" value="abc"><a id="link" role="button" href="#next">Go</a>
		<div id="unicode"></div><script>document.querySelector('#unicode').textContent='🙂漢'.repeat(2000)</script>`)
	matches, err := b.liveQuery("#native")
	if err != nil || len(matches) != 1 || matches[0].Type != "text" || matches[0].ReadOnly || matches[0].Value != "abc" || !strings.HasPrefix(matches[0].Ref, "i") {
		t.Fatalf("native ARIA field metadata: %+v %v", matches, err)
	}
	matches, err = b.liveQuery("#unicode")
	if err != nil || len(matches) != 1 || matches[0].Text != strings.Repeat("🙂漢", 600) {
		t.Fatalf("bounded Unicode query: %+v %v", matches, err)
	}
	matches, err = b.liveQuery("#link")
	if err != nil || len(matches) != 1 || matches[0].Type != "button" || !strings.HasPrefix(matches[0].Ref, "b") {
		t.Fatalf("link button ref: %+v %v", matches, err)
	}
	runLiveTestCommand(t, b, Command{Cmd: "press", Ref: matches[0].Ref})
	if !strings.HasSuffix(b.Current.URL, "#next") {
		t.Fatalf("link button action: %s", b.Current.URL)
	}
}

func TestLiveProjectedClickAndScopedLabels(t *testing.T) {
	b, _ := liveDOMTestPage(t, `<!doctype html><title>Slots and labels</title>
		<label for="q">Light label</label><input id="q"><x-action><span slot="icon" style="display:block;width:100px;height:30px">Projected click</span></x-action>
		<p id="state">Initial</p><script>
		const root = document.querySelector('x-action').attachShadow({mode:'open'});
		root.innerHTML='<label for="q">Shadow label</label><input id="q"><button type="button" aria-label="Projected action"><slot name="icon"></slot></button>';
		root.querySelector('button').onclick=event=>document.querySelector('#state').textContent='Trusted projected: '+event.isTrusted;
		</script>`)
	if len(b.Current.Inputs) != 2 || b.Current.Inputs[0].Label != "Light label" || b.Current.Inputs[1].Label != "Shadow label" {
		t.Fatalf("cross-root labels: %+v", b.Current.Inputs)
	}
	q, err := b.liveQuery("input")
	if err != nil || len(q) != 2 || q[0].Label != "Light label" || q[1].Label != "Shadow label" {
		t.Fatalf("query scoped labels: %+v %v", q, err)
	}
	runLiveTestCommand(t, b, Command{Cmd: "press", Ref: liveTestButton(t, b, "Projected action")})
	if !strings.Contains(b.Current.Text, "Trusted projected: true") {
		t.Fatalf("projected click: %s", b.Current.Text)
	}
}
