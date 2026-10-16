package browser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func jsServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/proxima" {
			fmt.Fprint(w, `<title>Destino</title><h2>Funcionou navegação dinâmica</h2>`)
			return
		}
		fmt.Fprint(w, `<!doctype html><html><head><title>Inicial</title></head><body><h1 id="titulo">Antes</h1><div id="estado">Aguardando</div><script>
  document.getElementById("titulo").textContent = "Depois";
  document.querySelector("#estado").innerText = "Carregado via JS";
  document.title = "Título alterado";
  var a = document.createElement('a');
  a.href='/proxima';a.textContent='Link via appendChild';
  document.getElementById('estado').appendChild(a);
  document.write('<a href="/proxima">Link via document.write</a>');
  </script></body></html>`)
	}))
}
func TestJSEnabledAndDisabled(t *testing.T) {
	worker := os.Getenv("AI_JS_WORKER")
	if worker == "" {
		t.Skip("set AI_JS_WORKER to binary for integration test")
	}
	srv := jsServer()
	defer srv.Close()
	b := New([]string{"127.0.0.1"}, false)
	no := b.Execute(Command{Cmd: "open", URL: srv.URL})
	if !no.OK || no.Page.Title != "Inicial" || len(no.Page.Links) != 0 {
		t.Fatalf("static baseline: %+v", no)
	}
	b.JSWorker = worker
	b.JSTimeout = 2 * time.Second
	got := b.Execute(Command{Cmd: "open", URL: srv.URL})
	if !got.OK || got.JS == nil || got.JS.Executed != 1 || got.JS.Changed != 4 {
		t.Fatalf("JS status: %+v", got)
	}
	if got.Page.Title != "Título alterado" || len(got.Page.Links) != 2 || !strings.Contains(got.Page.Text, "Carregado via JS") || got.Page.Headings[0].Text != "Depois" {
		t.Fatalf("DOM/snapshot: %+v", got.Page)
	}
	if got.JS.WorkerPeakRSSMiB <= 0 {
		t.Fatalf("missing worker memory metric: %+v", got.JS)
	}
	next := b.Execute(Command{Cmd: "click", Ref: "l1"})
	if !next.OK || next.Page.Title != "Destino" {
		t.Fatalf("follow dynamically created link: %+v", next)
	}
}
func TestUntrustedInfiniteLoopTimesOut(t *testing.T) {
	worker := os.Getenv("AI_JS_WORKER")
	if worker == "" {
		t.Skip("set AI_JS_WORKER")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<title>Static</title><p id="x">Safe</p><script>while(true) {}</script>`)
	}))
	defer srv.Close()
	b := New([]string{"127.0.0.1"}, false)
	b.JSWorker = worker
	b.JSTimeout = 250 * time.Millisecond
	start := time.Now()
	got := b.Execute(Command{Cmd: "open", URL: srv.URL})
	if time.Since(start) > 2*time.Second {
		t.Fatal("timeout did not protect parent")
	}
	if !got.OK || got.JS == nil || !got.JS.TimedOut || got.Page.Title != "Static" || !strings.Contains(got.Page.Text, "Safe") {
		t.Fatalf("must fail JS closed but keep static HTML: %+v", got)
	}
}
func TestOverlappingNodeMutation(t *testing.T) {
	raw := `<div id="outer"><p id="inner">Child</p>Before</div>`
	scripts, nodes := scanScriptsAndNodes(raw)
	if len(scripts) != 0 || nodes["inner"].Text != "Child" {
		t.Fatalf("nodes: %+v", nodes)
	}
	patched, count := patchHTML(raw, jsJob{Nodes: map[string]jsNode{}, Title: ""}, nodes, jsReply{Nodes: map[string]jsNode{"outer": {TextContent: "New"}, "inner": {TextContent: "other"}}, Title: ""})
	if count != 1 || patched != `<div id="outer">New</div>` {
		t.Fatalf("patch=%q count=%d", patched, count)
	}
}

func TestAppendOnlyPreservesMarkup(t *testing.T) {
	raw := `<div id="host"><strong>Keep</strong></div>`
	_, nodes := scanScriptsAndNodes(raw)
	job := jsJob{Nodes: map[string]jsNode{"host": {TextContent: "Keep"}}}
	got := jsReply{Nodes: map[string]jsNode{"host": {TextContent: "Keep"}}, Appends: map[string][]string{"host": {`<a href="/added">Add</a>`}}}
	out, n := patchHTML(raw, job, nodes, got)
	if n != 1 || out != `<div id="host"><strong>Keep</strong><a href="/added">Add</a></div>` {
		t.Fatalf("append-only changed original: %q", out)
	}
}
func TestScriptErrorDoesNotDiscardPage(t *testing.T) {
	worker := os.Getenv("AI_JS_WORKER")
	if worker == "" {
		t.Skip("set AI_JS_WORKER")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<title>Static</title><h1 id="x">Safe</h1><script>throw new Error('JS test error');</script>`)
	}))
	defer srv.Close()
	b := New([]string{"127.0.0.1"}, false)
	b.JSWorker = worker
	got := b.Execute(Command{Cmd: "open", URL: srv.URL})
	if !got.OK || got.Page.Title != "Static" || len(got.JS.Errors) == 0 || got.JS.Executed != 0 || got.Page.Headings[0].Text != "Safe" {
		t.Fatalf("must report script failure without losing page: %+v", got)
	}
}
