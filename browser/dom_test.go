package browser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestDOMHTMLFragmentsAndRawText(t *testing.T) {
	src := `<!doctype html><head><title>Nome &amp; marca</title><style>.foo:after {content:'<b>fake</b>'}</style></head><body><script>var html = '<a href="/evil">wrong</a>'; var z = 2 < 10;</script><ul><li>First<li>Second</ul><a href="/good" class="action" id="go"><em>Go</em> next</a><p>Visible</p><div hidden>Secret</div></body>`
	d, e := ParseDocument([]byte(src), "https://example.org/root", 200, len(src))
	if e != nil {
		t.Fatal(e)
	}
	p := d.Page(5000)
	if p.Title != "Nome & marca" {
		t.Fatalf("title=%q", p.Title)
	}
	if len(p.Links) != 1 || p.Links[0].URL != "https://example.org/good" || p.Links[0].Label != "Go next" {
		t.Fatalf("links=%+v", p.Links)
	}
	for _, forbidden := range []string{"wrong", "Secret", "fake", "var html"} {
		if strings.Contains(p.Text, forbidden) {
			t.Errorf("hidden content leaked: %q", p.Text)
		}
	}
	if !strings.Contains(p.Text, "First Second") || !strings.Contains(p.Text, "Visible") {
		t.Errorf("visible text missing: %q", p.Text)
	}
	for _, sel := range []string{"a#go", "a.action", "#go", "body a[href='/good']"} {
		matches, err := d.Query(sel)
		if err != nil || len(matches) != 1 || matches[0].Href != "https://example.org/good" {
			t.Fatalf("query %q => %+v %v", sel, matches, err)
		}
	}
}

func TestDOMFormFieldsAndSessionPersistence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/search" {
			fmt.Fprintf(w, `<title>Search</title><h1>Found %s</h1>`, r.URL.Query().Get("q"))
			return
		}
		fmt.Fprint(w, `<title>Form</title><form action="/search"><input id="query" name="q" placeholder="Termo" value="start"><input type="password" name="passwd" value="private"><textarea name="note">One &amp; two</textarea><select name="color"><option value="red">Red</option><option value="blue" selected>Blue</option></select><button type="submit">Find</button></form>`)
	}))
	defer srv.Close()
	b := New([]string{"127.0.0.1"}, false)
	result := b.Execute(Command{Cmd: "open", URL: srv.URL})
	if !result.OK {
		t.Fatal(result.Error)
	}
	fields := result.Page.Forms[0].Fields
	if len(fields) != 4 || fields[0].Value != "start" || fields[1].Value != "[redacted]" || fields[2].Value != "One & two" || fields[3].Value != "blue" {
		t.Fatalf("fields=%+v", fields)
	}
	q := b.Execute(Command{Cmd: "query", Selector: "form input#query"})
	if !q.OK || len(q.Matches) != 1 || q.Matches[0].Value != "start" {
		t.Fatalf("query=%+v", q)
	}
	fill := b.Execute(Command{Cmd: "fill", Ref: "i1", Value: "go ai"})
	if !fill.OK {
		t.Fatal(fill.Error)
	}
	snap := b.Execute(Command{Cmd: "snapshot"})
	if snap.Page.Forms[0].Fields[0].Value != "go ai" {
		t.Fatalf("lost field in snapshot=%+v", snap.Page.Forms)
	}
	q = b.Execute(Command{Cmd: "query", Selector: "#query"})
	if len(q.Matches) != 1 || q.Matches[0].Value != "go ai" {
		t.Fatalf("lost field in DOM=%+v", q)
	}
	pw := b.Execute(Command{Cmd: "query", Selector: "input[type=password]"})
	if len(pw.Matches) != 1 || pw.Matches[0].Value != "[redacted]" {
		t.Fatalf("secret leak: %+v", pw)
	}
	submitted := b.Execute(Command{Cmd: "submit", Ref: "f1"})
	if !submitted.OK || !strings.Contains(submitted.Page.Headings[0].Text, "go ai") {
		t.Fatalf("submit=%+v", submitted)
	}
}

func TestDOMLimitsAndUnsupportedSelectors(t *testing.T) {
	nested := strings.Repeat("<div>", 140) + "hello" + strings.Repeat("</div>", 140)
	if _, e := ParseDocument([]byte(nested), "https://example.org", 200, len(nested)); e == nil {
		t.Fatal("nesting limit not enforced")
	}
	many := strings.Repeat("<span>x</span>", 8000)
	if _, e := ParseDocument([]byte(many), "https://example.org", 200, len(many)); e == nil {
		t.Fatal("node limit not enforced")
	}
	d, e := ParseDocument([]byte(`<div class="yes">OK</div><div hidden class="yes">secret</div>`), "https://example.org", 200, 54)
	if e != nil {
		t.Fatal(e)
	}
	matches, e := d.Query(".yes")
	if e != nil || len(matches) != 1 || matches[0].Text != "OK" {
		t.Fatalf("hidden=%+v %v", matches, e)
	}
	if _, e := d.Query("div:hover"); e == nil {
		t.Fatal("unsupported pseudo-class silently accepted")
	}
}

func TestDOMJSIntegration(t *testing.T) {
	worker := os.Getenv("AI_JS_WORKER")
	if worker == "" {
		t.Skip("set AI_JS_WORKER")
	}
	srv := jsServer()
	defer srv.Close()
	b := New([]string{"127.0.0.1"}, false)
	b.JSWorker = worker
	got := b.Execute(Command{Cmd: "open", URL: srv.URL})
	if !got.OK || got.Page.Title != "Título alterado" || len(got.Page.Links) != 2 {
		t.Fatalf("JS DOM integration %+v", got)
	}
	q := b.Execute(Command{Cmd: "query", Selector: "#estado"})
	if !q.OK || len(q.Matches) != 1 || !strings.Contains(q.Matches[0].Text, "Carregado via JS") {
		t.Fatalf("query JS=%+v", q)
	}
}

func TestQueryDoesNotRepeatPageSnapshot(t *testing.T) {
	b := New(nil, false)
	d, e := ParseDocument([]byte(`<main><h1>Big page</h1><p>Many words</p><a href="/a" id="btn">Go</a></main>`), "https://example.org/", 200, 80)
	if e != nil {
		t.Fatal(e)
	}
	b.Document = d
	p := d.Page(5000)
	b.Current = &p
	r := b.Execute(Command{Cmd: "query", Selector: "#btn"})
	if !r.OK || len(r.Matches) != 1 || r.Page != nil {
		t.Fatalf("query unexpectedly repeats entire page: %+v", r)
	}
	snap := b.Execute(Command{Cmd: "snapshot"})
	if snap.Page == nil || snap.Page.Title != "" {
		t.Fatalf("snapshot lost: %+v", snap)
	}
}

func TestFailClosedWithoutHostAllowlist(t *testing.T) {
	b := New(nil, false)
	r := b.Execute(Command{Cmd: "open", URL: "https://example.org/"})
	if r.OK || !strings.Contains(r.Error, "network disabled") {
		t.Fatalf("expected default network deny: %+v", r)
	}
}
