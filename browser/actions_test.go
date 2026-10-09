package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestDiscoverAndRunAPIBelowButton(t *testing.T) {
	var apiHits, postHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			http.SetCookie(w, &http.Cookie{Name: "auth", Value: "session", Path: "/"})
			fmt.Fprint(w, `<title>Shop</title><a href="/help">Help</a><button id="view">Ver produtos</button><button id="save" hx-post="/api/save">Salvar</button><form action="/search" method="GET"><input name="q" value="initial"></form><section id="extra" hidden>Segredo do catálogo público: 3 categorias</section><button aria-controls="extra">Detalhes</button><script>
        document.getElementById('view').addEventListener('click', function () {
          fetch('/api/products').then(r => r.json());
        });
        fetch('/api/unrelated');
      </script>`)
		case "/api/products":
			apiHits++
			if _, err := r.Cookie("auth"); err != nil {
				http.Error(w, "missing session", 401)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"products":[{"id":1,"name":"Caderno"}],"api_token":"not-for-agent"}`)
		case "/api/save":
			postHits++
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"ok":true}`)
		case "/search":
			fmt.Fprint(w, `<title>Search</title><p>Query `+r.URL.Query().Get("q")+`</p>`)
		default:
			fmt.Fprint(w, `<title>Help</title>`)
		}
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	b := New([]string{u.Hostname()}, false)
	defer b.Close()
	if r := b.Execute(Command{Cmd: "open", URL: srv.URL}); !r.OK {
		t.Fatal(r.Error)
	}
	discovered := b.Execute(Command{Cmd: "discover"})
	if !discovered.OK {
		t.Fatal(discovered.Error)
	}
	if discovered.Page != nil {
		t.Fatal("discover should be compact and omit page")
	}
	if apiHits != 0 || postHits != 0 {
		t.Fatal("discovery must be read-only")
	}
	var products, form, saved, embedded, unassociated Action
	for _, a := range discovered.Actions {
		switch {
		case a.Kind == "api" && a.Label == "Ver produtos":
			products = a
		case a.Kind == "form":
			form = a
		case a.Kind == "button" && a.Label == "Salvar":
			saved = a
		case a.Kind == "embedded_data":
			embedded = a
		case a.Label == "Unassociated API call":
			unassociated = a
		}
	}
	if products.Ref == "" || products.Method != "GET" || products.Confidence != "inferred" || !products.RequiresConfirmation {
		t.Fatalf("missing API for button: %+v", discovered.Actions)
	}
	if !strings.HasSuffix(products.URL, "/api/products") {
		t.Fatal(products)
	}
	if embedded.Preview != "Segredo do catálogo público: 3 categorias" {
		t.Fatalf("embedded: %+v", embedded)
	}
	if unassociated.Ref == "" || !strings.HasSuffix(unassociated.URL, "/api/unrelated") {
		t.Fatalf("unassociated: %+v", discovered.Actions)
	}
	if form.Kind != "form" || form.Method != "GET" {
		t.Fatalf("form: %+v", form)
	}
	if saved.Executable || saved.Method != "POST" {
		t.Fatalf("POST unexpectedly executable: %+v", saved)
	}
	if b.Execute(Command{Cmd: "run", Ref: products.Ref}).OK {
		t.Fatal("inferred API must require confirmation")
	}
	if apiHits != 0 {
		t.Fatal("no fetch should run without confirmation")
	}
	inspected := b.Execute(Command{Cmd: "inspect", Ref: products.Ref})
	if !inspected.OK || inspected.Action == nil || inspected.Action.Ref != products.Ref {
		t.Fatal("inspect failed")
	}
	fetched := b.Execute(Command{Cmd: "run", Ref: products.Ref, Confirm: true})
	if !fetched.OK || fetched.Resource == nil || fetched.Resource.Status != 200 {
		t.Fatalf("run: %+v", fetched)
	}
	raw, _ := json.Marshal(fetched.Resource.Data)
	if !strings.Contains(string(raw), "Caderno") || strings.Contains(string(raw), "not-for-agent") || !strings.Contains(string(raw), "[redacted]") {
		t.Fatalf("filtered JSON expected: %s", raw)
	}
	if apiHits != 1 {
		t.Fatalf("API hits %d", apiHits)
	}
	if b.Current.Title != "Shop" {
		t.Fatal("API fetch must not replace current page")
	}
	if b.Execute(Command{Cmd: "run", Ref: saved.Ref, Confirm: true}).OK {
		t.Fatal("unsafe POST without known body should be refused")
	}
	if postHits != 0 {
		t.Fatal("POST unexpectedly dispatched")
	}
	field := b.Execute(Command{Cmd: "fill", Ref: "i1", Value: "caneta"})
	if !field.OK {
		t.Fatal(field.Error)
	}
	result := b.Execute(Command{Cmd: "run", Ref: form.Ref})
	if !result.OK || result.Page.Title != "Search" || !strings.Contains(result.Page.Text, "caneta") {
		t.Fatalf("submit GET: %+v", result)
	}
}

func TestInlineOnclickFunctionAndUnknownMethods(t *testing.T) {
	src := `<button onclick="loadProducts()">Go</button><button onclick="fetch('/api/write',{method:'POST',body:'x'})">Write</button><button onclick="fetch('/api/dynamic',{method: whatever})">Dynamic</button><script>function loadProducts(){return fetch('/api/products')}</script>`
	d, e := ParseDocument([]byte(src), "https://example.com/", 200, len(src))
	if e != nil {
		t.Fatal(e)
	}
	b := New([]string{"example.com"}, false)
	b.Document = d
	p := d.Page(5000)
	b.Current = &p
	all := b.Discover()
	got := map[string]Action{}
	for _, a := range all {
		if a.Kind == "api" {
			got[a.Label] = a
		}
	}
	if a := got["Go"]; a.Method != "GET" || !strings.HasSuffix(a.URL, "/api/products") {
		t.Fatalf("callback resolver failed: %+v", all)
	}
	if a := got["Write"]; a.Method != "POST" || a.Executable {
		t.Fatalf("must not reconstruct inferred post body: %+v", all)
	}
	if a := got["Dynamic"]; a.Method != "UNKNOWN" || a.Executable {
		t.Fatalf("must not guess dynamic method: %+v", all)
	}
}

func TestReadOnlyDiscoveryDoesNotFetchExternalScriptUnlessEnabled(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/app.js" {
			w.Header().Set("Content-Type", "application/javascript")
			hits++
			fmt.Fprint(w, `document.getElementById('show').addEventListener('click',()=>fetch('/api/result'))`)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<button id="show">Show</button><script src="/app.js"></script>`)
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	b := New([]string{u.Hostname()}, false)
	opened := b.Execute(Command{Cmd: "open", URL: srv.URL})
	if !opened.OK {
		t.Fatal(opened.Error)
	}
	r := b.Execute(Command{Cmd: "discover"})
	if hits != 0 {
		t.Fatal("external scripts should be opt-in")
	}
	for _, a := range r.Actions {
		if a.Kind == "api" {
			t.Fatal("should not discover unloaded script", a)
		}
	}
	b.ScanScripts = true
	r = b.Execute(Command{Cmd: "discover"})
	if hits != 1 {
		t.Fatalf("expected one external fetch, got %d", hits)
	}
	b.Execute(Command{Cmd: "discover"})
	if hits != 1 {
		t.Fatalf("external script should be cached, got %d", hits)
	}
	found := false
	for _, a := range r.Actions {
		if a.Kind == "api" && strings.HasSuffix(a.URL, "/api/result") {
			found = true
		}
	}
	if !found {
		t.Fatalf("external script missing from actions: %+v", r.Actions)
	}
}

func TestDiscoveryRejectsOffAllowlistActionsAndComments(t *testing.T) {
	src := `<button onclick="fetch('https://other.test/private')">Private</button><script>
    // fetch('/fake-from-comment')
    /*fetch('/fake-from-block')*/
    var message="// not a comment";
    fetch('/real');
  </script>`
	d, e := ParseDocument([]byte(src), "https://example.org/", 200, len(src))
	if e != nil {
		t.Fatal(e)
	}
	b := New([]string{"example.org"}, false)
	b.Document = d
	p := d.Page(1000)
	b.Current = &p
	actions := b.Discover()
	for _, a := range actions {
		if strings.Contains(a.URL, "fake-") {
			t.Fatalf("false positive comment: %+v", a)
		}
		if strings.Contains(a.URL, "other.test") {
			r := b.Execute(Command{Cmd: "run", Ref: a.Ref, Confirm: true})
			if r.OK || !strings.Contains(r.Error, "allowlist") {
				t.Fatalf("cross domain must fail: %+v", r)
			}
		}
	}
}

// No dependency on server implementation language: a PHP-style SSR endpoint,
// a Python-style GET form, htmx on a non-button, and an advertised JSON feed.
func TestServerLanguageIndependentActions(t *testing.T) {
	var phpHits, pythonHits, htmxHits, feedHits, postHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			fmt.Fprint(w, `<html><head><title>Catalogo</title><link rel="alternate" type="application/json" title="Feed JSON" href="/api/feed.json"></head><body>
				<a href="/products.php?category=office">Catalogo PHP</a>
				<form method="GET" action="/django/search"><input type="text" name="q" value="caneta"><button>Buscar Python</button></form>
				<section hx-get="/htmx/catalog" aria-label="Consultar catalogo">Ver catalogo HTMX</section>
				<button hx-delete="/admin/delete">Apagar</button>
				<button data-url="/experimental/guess" data-method="GET">Candidato</button>
				<form action="/checkout.php" method="POST"><input name="sku" value="44"><input type="submit" formaction="/checkout-special.php" value="Comprar"></form>
			</body></html>`)
		case "/products.php":
			phpHits++
			fmt.Fprint(w, `<title>PHP catalog</title><h1>Produtos</h1><p>Caderno e caneta</p>`)
		case "/django/search":
			pythonHits++
			fmt.Fprint(w, `<title>Django</title><p>Busca: `+r.URL.Query().Get("q")+`</p>`)
		case "/htmx/catalog":
			htmxHits++
			fmt.Fprint(w, `<p>Item servido como HTML, nao JSON.</p>`)
		case "/api/feed.json":
			feedHits++
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"items":[{"name":"Lapis"}]}`)
		case "/admin/delete", "/checkout-special.php", "/experimental/guess":
			postHits++
			fmt.Fprint(w, `must not execute`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	b := New([]string{u.Hostname()}, false)
	defer b.Close()
	if open := b.Execute(Command{Cmd: "open", URL: srv.URL}); !open.OK {
		t.Fatal(open.Error)
	}
	actions := b.Execute(Command{Cmd: "discover"})
	if !actions.OK {
		t.Fatal(actions.Error)
	}
	if phpHits+pythonHits+htmxHits+feedHits+postHits != 0 {
		t.Fatal("discovery caused requests")
	}
	found := map[string]Action{}
	for _, a := range actions.Actions {
		found[a.Label] = a
	}
	for _, key := range []string{"Catalogo PHP", "Buscar Python", "Consultar catalogo", "Feed JSON"} {
		if found[key].URL == "" {
			t.Fatalf("missing %q: %+v", key, actions.Actions)
		}
	}
	if found["Consultar catalogo"].Source != "html:hx-get" || !found["Consultar catalogo"].RequiresConfirmation {
		t.Fatalf("htmx not properly inferred: %+v", found["Consultar catalogo"])
	}
	for _, key := range []string{"Apagar", "Candidato", "Comprar"} {
		a := found[key]
		if a.URL == "" || a.Executable {
			t.Fatalf("not executable: %q: %+v", key, a)
		}
		if result := b.Execute(Command{Cmd: "run", Ref: a.Ref, Confirm: true}); result.OK {
			t.Fatalf("unsafe action %s allowed", key)
		}
	}
	if result := b.Execute(Command{Cmd: "run", Ref: found["Catalogo PHP"].Ref}); !result.OK || result.Page.Title != "PHP catalog" {
		t.Fatalf("php route: %+v", result)
	}
	if result := b.Execute(Command{Cmd: "back"}); !result.OK {
		t.Fatal(result.Error)
	}
	if result := b.Execute(Command{Cmd: "run", Ref: found["Buscar Python"].Ref}); !result.OK || !strings.Contains(result.Page.Text, "caneta") {
		t.Fatalf("python route: %+v", result)
	}
	if result := b.Execute(Command{Cmd: "back"}); !result.OK {
		t.Fatal(result.Error)
	}
	if result := b.Execute(Command{Cmd: "run", Ref: found["Consultar catalogo"].Ref}); result.OK {
		t.Fatal("htmx needs explicit confirmation")
	}
	if result := b.Execute(Command{Cmd: "run", Ref: found["Consultar catalogo"].Ref, Confirm: true}); !result.OK || result.Resource == nil || !strings.Contains(result.Resource.Body, "Item servido") {
		t.Fatalf("htmx resource: %+v", result)
	}
	if result := b.Execute(Command{Cmd: "run", Ref: found["Feed JSON"].Ref, Confirm: true}); !result.OK || result.Resource == nil || result.Resource.Status != 200 {
		t.Fatalf("json feed: %+v", result)
	}
	if phpHits != 1 || pythonHits != 1 || htmxHits != 1 || feedHits != 1 || postHits != 0 {
		t.Fatalf("unexpected request counts php=%d python=%d htmx=%d feed=%d unsafe=%d", phpHits, pythonHits, htmxHits, feedHits, postHits)
	}
}
