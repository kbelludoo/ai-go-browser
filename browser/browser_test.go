package browser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNavigationCookieAndForm(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			http.SetCookie(w, &http.Cookie{Name: "tok", Value: "42"})
			fmt.Fprint(w, `<title>Home</title><a href="/next">Next</a><form action="/search"><input name="q"></form><form action="/save" method="POST"><input name="note"></form>`)
		case "/next":
			if _, e := r.Cookie("tok"); e != nil {
				http.Error(w, "no session", 401)
				return
			}
			fmt.Fprint(w, "<title>Session ok</title><h2>Hello</h2>")
		case "/search":
			fmt.Fprintf(w, "<title>Search</title>Result=%s", r.URL.Query().Get("q"))
		case "/save":
			if r.Method != "POST" {
				http.Error(w, "bad method", 400)
				return
			}
			fmt.Fprintf(w, "<title>Saved</title>Value=%s", r.FormValue("note"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	b := New([]string{u.Hostname()}, false)
	run := func(c Command) Result {
		t.Helper()
		r := b.Execute(c)
		if !r.OK {
			t.Fatalf("%s: %s", c.Cmd, r.Error)
		}
		return r
	}
	a := run(Command{Cmd: "open", URL: srv.URL})
	if len(a.Page.Links) != 1 || len(a.Page.Forms) != 2 {
		t.Fatalf("initial: %+v", a.Page)
	}
	next := run(Command{Cmd: "click", Ref: "l1"})
	if next.Page.Title != "Session ok" {
		t.Fatalf("cookie? %+v", next)
	}
	run(Command{Cmd: "back"})
	run(Command{Cmd: "fill", Ref: "i1", Value: "um teste"})
	found := run(Command{Cmd: "submit", Ref: "f1"})
	if !strings.Contains(found.Page.Text, "Result=um teste") {
		t.Fatalf("search: %+v", found.Page)
	}
	run(Command{Cmd: "back"})
	blocked := b.Execute(Command{Cmd: "submit", Ref: "f2", Confirm: true})
	if blocked.OK {
		t.Fatal("POST must be blocked without allowPost")
	}
	b.AllowPost = true
	run(Command{Cmd: "fill", Ref: "i2", Value: "nota"})
	blocked = b.Execute(Command{Cmd: "submit", Ref: "f2"})
	if blocked.OK {
		t.Fatal("POST must require explicit confirmation")
	}
	save := run(Command{Cmd: "submit", Ref: "f2", Confirm: true})
	if !strings.Contains(save.Page.Text, "Value=nota") {
		t.Fatalf("save: %+v", save.Page)
	}
}
func TestDisallowedSchemeAndHosts(t *testing.T) {
	b := New([]string{"example.org"}, false)
	for _, s := range []string{"file:///etc/passwd", "javascript:alert(1)", "http://127.0.0.1/", "http://evil.com"} {
		if b.Execute(Command{Cmd: "open", URL: s}).OK {
			t.Fatalf("should block %q", s)
		}
	}
}
func TestRedirectBlockedByAllowlist(t *testing.T) {
	from := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://untrusted.invalid/secret", http.StatusFound)
	}))
	defer from.Close()
	source, _ := url.Parse(from.URL)
	b := New([]string{source.Hostname()}, false)
	r := b.Execute(Command{Cmd: "open", URL: from.URL})
	if r.OK || !strings.Contains(r.Error, "allowlist") {
		t.Fatalf("redirect allowlist: %+v", r)
	}
}

func TestPasswordRedaction(t *testing.T) {
	src := `<form method="POST"><input type="hidden" name="csrf" value="secret_token"><input name="pass" type="password"></form>`
	page := Parse([]byte(src), "http://example.org/", 200, len(src), 5000)
	b := New(nil, true)
	b.Current = &page
	r := b.Execute(Command{Cmd: "fill", Ref: "i2", Value: "my_password"})
	if !r.OK {
		t.Fatal(r.Error)
	}
	if got := r.Page.Forms[0].Fields[0].Value; got != "[redacted]" {
		t.Fatalf("hidden token exposed: %q", got)
	}
	if got := r.Page.Forms[0].Fields[1].Value; got != "[redacted]" {
		t.Fatalf("password exposed: %q", got)
	}
	if b.Current.Forms[0].Fields[1].Value != "my_password" {
		t.Fatal("password in internal session changed")
	}
}
