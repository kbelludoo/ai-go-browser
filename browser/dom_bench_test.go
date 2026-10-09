package browser

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkDOM1000Links(b *testing.B) {
	var source strings.Builder
	source.WriteString("<html><head><title>Load test</title></head><body>")
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&source, `<article><h2>Headline %d</h2><a href="/article/%d" class="item">Read item %d</a></article>`, i, i, i)
	}
	source.WriteString("</body></html>")
	raw := []byte(source.String())
	b.ReportAllocs()
	b.SetBytes(int64(len(raw)))
	for i := 0; i < b.N; i++ {
		doc, err := ParseDocument(raw, "https://example.org", 200, len(raw))
		if err != nil {
			b.Fatal(err)
		}
		page := doc.Page(5000)
		if len(page.Links) != 1000 {
			b.Fatalf("links=%d", len(page.Links))
		}
	}
}
