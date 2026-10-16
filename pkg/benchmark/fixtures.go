package benchmark

import (
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
)

type fixture struct {
	name, path, expected, action, value string
	javascript                          bool
}

func localFixtures() (*httptest.Server, []fixture) {
	const unicodeText = "Ação e café — 日本語 😀"
	cases := []fixture{
		{name: "static", path: "/static", expected: "BENCH_STATIC_OK"},
		{name: "large_html", path: "/large", expected: "BENCH_LARGE_OK"},
		{name: "unicode", path: "/unicode", expected: unicodeText},
		{name: "get_form", path: "/form", expected: "FORM_OK:café teste", action: "form", value: "café teste"},
		{name: "link_cookie", path: "/cookie", expected: "COOKIE_OK:present", action: "link"},
		{name: "spa_click_fill", path: "/spa", expected: "SPA_OK:Ana café", action: "spa", value: "Ana café", javascript: true},
		{name: "async_dom", path: "/async", expected: "ASYNC_DOM_OK", javascript: true},
	}
	mux := http.NewServeMux()
	write := func(w http.ResponseWriter, title, body string) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><title>%s</title></head><body>%s</body></html>`, title, body)
	}
	mux.HandleFunc("/static", func(w http.ResponseWriter, r *http.Request) {
		write(w, "Static", `<main><h1>BENCH_STATIC_OK</h1><p>Article with useful visible content.</p><a href="/unicode">Unicode</a></main>`)
	})
	large := `<h1>BENCH_LARGE_OK</h1><script type="application/json">{"payload":"` + strings.Repeat("x", 2<<20) + `"}</script><p>Visible article.</p>`
	mux.HandleFunc("/large", func(w http.ResponseWriter, r *http.Request) { write(w, "Large", large) })
	mux.HandleFunc("/unicode", func(w http.ResponseWriter, r *http.Request) { write(w, "Unicode", "<h1>"+unicodeText+"</h1>") })
	mux.HandleFunc("/form", func(w http.ResponseWriter, r *http.Request) {
		write(w, "Form", `<form id="search" action="/form-result" method="get"><label for="q">Search</label><input id="q" name="q"><button type="submit">Search</button></form>`)
	})
	mux.HandleFunc("/form-result", func(w http.ResponseWriter, r *http.Request) {
		write(w, "Form result", "<h1>FORM_OK:"+html.EscapeString(r.URL.Query().Get("q"))+"</h1>")
	})
	mux.HandleFunc("/cookie", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "bench", Value: "present", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
		write(w, "Cookie", `<a id="continue" href="/cookie-result">Continue with cookie</a>`)
	})
	mux.HandleFunc("/cookie-result", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("bench")
		value := "missing"
		if err == nil {
			value = cookie.Value
		}
		write(w, "Cookie result", "<h1>COOKIE_OK:"+html.EscapeString(value)+"</h1>")
	})
	mux.HandleFunc("/spa", func(w http.ResponseWriter, r *http.Request) {
		write(w, "SPA", `<div id="root"><form onsubmit="return false"><label for="name">Name</label><input id="name" name="name"><button id="go" type="button">Generate greeting</button></form><p id="result">Waiting for action</p></div><script>document.getElementById('go').addEventListener('click',function(){setTimeout(function(){document.getElementById('result').textContent='SPA_OK:'+document.getElementById('name').value},120)})</script>`)
	})
	mux.HandleFunc("/async", func(w http.ResponseWriter, r *http.Request) {
		write(w, "Asynchronous DOM", `<div id="root"><p>Waiting for asynchronous content</p></div><script>setTimeout(function(){const p=document.createElement('p');p.textContent='ASYNC_DOM_OK';document.getElementById('root').appendChild(p)},800)</script>`)
	})
	return httptest.NewServer(mux), cases
}
