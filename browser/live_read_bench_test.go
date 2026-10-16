package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// wideReadFixture serves a page whose full observation is expensive to
// serialize in the browser, transfer and re-parse in Go. That is exactly what
// polling pays on a real site, and it is the cost view:"read" removes.
func wideReadFixture(tb testing.TB) (*Browser, func()) {
	tb.Helper()
	var page strings.Builder
	page.WriteString(`<title>Wide page</title><p>WIDE_MARKER</p>`)
	for i := 0; i < 2000; i++ {
		fmt.Fprintf(&page, `<article><h2>Headline %d</h2><a href="/item/%d">Item %d</a><p>Body text for item %d with words to read</p></article>`, i, i, i, i)
	}
	html := page.String()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, html)
	}))
	path, err := FindSystemBrowser()
	if err != nil {
		server.Close()
		tb.Skip("observation benchmark requires installed Chromium/Edge")
	}
	b := New([]string{"127.0.0.1"}, false)
	b.SystemBrowser = path
	b.OperationTimeout = 30 * time.Second
	if result := b.Execute(Command{Cmd: "open", URL: server.URL, RenderJS: true}); !result.OK {
		b.Close()
		server.Close()
		tb.Fatalf("open: %s", result.Error)
	}
	return b, func() { b.Close(); server.Close() }
}

// BenchmarkLiveObservation measures the cost of one poll on a wide page: the
// full observation transfers and re-parses the document, the read view only
// transfers visible text. Payload bytes are reported so the token saving is
// visible alongside the latency saving.
func BenchmarkLiveObservation(b *testing.B) {
	session, cleanup := wideReadFixture(b)
	defer cleanup()

	for _, view := range []struct {
		name string
		cmd  Command
	}{
		{"Full", Command{Cmd: "snapshot"}},
		{"Read", Command{Cmd: "snapshot", View: "read"}},
	} {
		result := session.Execute(view.cmd)
		if !result.OK || result.Page == nil {
			b.Fatalf("%s: %s", view.name, result.Error)
		}
		if !strings.Contains(result.Page.Text, "WIDE_MARKER") {
			b.Fatalf("%s observation lost the marker: %q", view.name, result.Page.Text)
		}
		payload, err := json.Marshal(result.Page)
		if err != nil {
			b.Fatal(err)
		}
		if len(payload) == 0 {
			b.Fatalf("%s produced an empty payload", view.name)
		}
		b.Run(view.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			b.ReportMetric(float64(len(payload)), "payload-B/op")
			for i := 0; i < b.N; i++ {
				got := session.Execute(view.cmd)
				if !got.OK || got.Page == nil {
					b.Fatalf("%s: %s", view.name, got.Error)
				}
			}
		})
	}
}
