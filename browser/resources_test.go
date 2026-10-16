package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestOpenResourcesRedactionHistoryAndHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json":
			w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
			w.WriteHeader(http.StatusUnprocessableEntity)
			fmt.Fprint(w, `{"count":9007199254740993,"items":[{"title":"ação","access_token":"do-not-print"}]}`)
		case "/text":
			w.Header().Set("Content-Type", "text/plain; charset=windows-1252")
			_, _ = w.Write([]byte{'o', 'l', 0xe1})
		case "/xml":
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprint(w, `<catalog><item>One</item></catalog>`)
		case "/null":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, "null")
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<title>HTML again</title><a href="/json">Data</a>`)
		}
	}))
	defer server.Close()
	b := New([]string{"127.0.0.1"}, false)
	defer b.Close()
	opened := b.Execute(Command{Cmd: "open", URL: server.URL + "/json"})
	if !opened.OK || opened.Page != nil || opened.Resource == nil || opened.Resource.Status != 422 || b.Document != nil {
		t.Fatalf("JSON open: %+v", opened)
	}
	data, _ := json.Marshal(opened)
	if strings.Contains(string(data), "do-not-print") || !strings.Contains(string(data), "[redacted]") || !strings.Contains(string(data), "9007199254740993") {
		t.Fatalf("redaction or integer precision lost: %s", data)
	}
	for _, path := range []string{"/text", "/xml", "/null"} {
		result := b.Execute(Command{Cmd: "open", URL: server.URL + path})
		if !result.OK || result.Resource == nil {
			t.Fatalf("resource %s: %+v", path, result)
		}
		if path == "/text" && result.Resource.Body != "olá" {
			t.Fatalf("text encoding: %q", result.Resource.Body)
		}
		if path == "/null" && result.Resource.Body != "null" {
			t.Fatalf("JSON null omitted: %+v", result.Resource)
		}
	}
	if result := b.Execute(Command{Cmd: "back"}); !result.OK || result.Resource == nil || result.Resource.URL != server.URL+"/xml" {
		t.Fatalf("resource back: %+v", result)
	}
	if result := b.Execute(Command{Cmd: "snapshot"}); !result.OK || result.Resource == nil {
		t.Fatalf("resource snapshot: %+v", result)
	}
	if result := b.Execute(Command{Cmd: "open", URL: server.URL}); !result.OK || result.Resource != nil || result.Page == nil || result.Page.Title != "HTML again" {
		t.Fatalf("return to HTML: %+v", result)
	}
	b.Close()
	if b.resource != nil {
		t.Fatal("Close retained resource")
	}
}

func TestResourceBudgetAndBinaryResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/large-json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"secret":"sensitive-value","payload":"%s"}`, strings.Repeat("x", maxResourceBytes))
		case "/invalid-json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"secret":"sensitive-value",`)
		case "/large-text":
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprint(w, strings.Repeat("á", maxResourceBytes/2-1)+"🌎"+strings.Repeat("x", maxResourceBytes))
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte{0, 1, 2, 255})
		}
	}))
	defer server.Close()
	b := New([]string{"127.0.0.1"}, false)
	defer b.Close()
	for _, path := range []string{"/large-json", "/invalid-json", "/large-text", "/binary"} {
		result := b.Execute(Command{Cmd: "open", URL: server.URL + path})
		if !result.OK || result.Resource == nil {
			t.Fatalf("%s: %+v", path, result)
		}
		resource := result.Resource
		if len(resource.Body) > maxResourceBytes || !utf8.ValidString(resource.Body) {
			t.Fatalf("resource body outside budget or invalid UTF-8: %s", path)
		}
		if strings.Contains(resource.Body, "sensitive-value") {
			t.Fatal("incomplete JSON leaked raw secret")
		}
		if strings.HasPrefix(path, "/large-") && !resource.Truncated {
			t.Fatal("overflow omitted truncated flag")
		}
		if path == "/binary" && resource.Body != "[binary response omitted]" {
			t.Fatalf("binary body exposed: %q", resource.Body)
		}
	}
}
