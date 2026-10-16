package browser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompactSnapshotPreservesActionsAndExplicitMarkdown(t *testing.T) {
	var content strings.Builder
	content.WriteString(`<title>Many links</title><form><input name="q"><button>Search</button></form>`)
	for i := 1; i <= 250; i++ {
		fmt.Fprintf(&content, `<h2>Section %d</h2><a id="link%d" href="/next">Link %d</a><p>%s</p>`, i, i, i, strings.Repeat("More content. ", 20))
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/next" {
			fmt.Fprint(w, `<title>Navigation completed</title><p>Destination</p>`)
			return
		}
		fmt.Fprint(w, content.String())
	}))
	defer server.Close()
	b := New([]string{"127.0.0.1"}, false)
	defer b.Close()
	b.CompactSnapshots = true
	result := b.Execute(Command{Cmd: "open", URL: server.URL})
	if !result.OK || result.Page.Markdown != "" || b.Current.Markdown != "" || len(result.Page.Links) != 200 || result.Page.OmittedLinks != 50 || len(result.Page.Headings) != 100 || result.Page.OmittedHeadings != 150 {
		t.Fatalf("compact snapshot: %+v", result)
	}
	if len(b.Current.Links) != 250 || len(b.Current.Headings) != 250 {
		t.Fatal("observation limits discarded live document state")
	}
	markdown := b.Execute(Command{Cmd: "markdown"})
	if !markdown.OK || !strings.Contains(markdown.Page.Markdown, "Section 250") || b.Current.Markdown != "" {
		t.Fatal("explicit Markdown missing or permanently retained")
	}
	query := b.Execute(Command{Cmd: "query", Selector: "#link250"})
	if !query.OK || len(query.Matches) != 1 || query.Matches[0].Ref != "l250" {
		t.Fatalf("query omitted link: %+v", query)
	}
	if result := b.Execute(Command{Cmd: "click", Ref: query.Matches[0].Ref}); !result.OK || result.Page.Title != "Navigation completed" {
		t.Fatalf("navigation beyond observation cap: %+v", result)
	}
}

func BenchmarkNavigateSnapshotModes(b *testing.B) {
	var raw strings.Builder
	raw.WriteString(`<title>Article</title>`)
	for i := 0; i < 250; i++ {
		fmt.Fprintf(&raw, `<h2>Section %d</h2><p>%s</p><a href="/read">Read more</a>`, i, strings.Repeat("ação 🌎 ", 100))
	}
	for _, compact := range []bool{false, true} {
		b.Run(fmt.Sprintf("compact=%t", compact), func(b *testing.B) {
			browser := New([]string{"example.org"}, false)
			browser.CompactSnapshots = compact
			browser.MaxHistory = 0
			browser.client.Transport = benchmarkTransport{html: raw.String()}
			defer browser.Close()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if result := browser.Execute(Command{Cmd: "open", URL: "https://example.org"}); !result.OK {
					b.Fatal(result.Error)
				}
			}
		})
	}
}
