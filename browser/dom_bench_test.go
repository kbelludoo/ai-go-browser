package browser

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
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

func BenchmarkTextLimit(b *testing.B) {
	raw := []byte("<p>" + strings.Repeat("ação 世界 🙂 ", 25000) + "</p>")
	doc, err := ParseDocument(raw, "https://example.org", 200, len(raw))
	if err != nil {
		b.Fatal(err)
	}
	b.Run("DOM", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(raw)))
		for i := 0; i < b.N; i++ {
			page := doc.Page(256)
			if !page.Truncated || utf8.RuneCountInString(page.Text) != 256 {
				b.Fatal("text limit not enforced")
			}
		}
	})
	b.Run("Parser", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(raw)))
		for i := 0; i < b.N; i++ {
			page := Parse(raw, "https://example.org", 200, len(raw), 256)
			if !page.Truncated || utf8.RuneCountInString(page.Text) != 256 {
				b.Fatal("text limit not enforced")
			}
		}
	})
}

func BenchmarkDOMQueryTextLimit(b *testing.B) {
	raw := []byte(`<a id="large" href="/next">` + strings.Repeat("ação 世界 🙂 ", 25000) + "</a>")
	doc, err := ParseDocument(raw, "https://example.org", 200, len(raw))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches, err := doc.Query("#large")
		if err != nil || len(matches) != 1 || utf8.RuneCountInString(matches[0].Text) != 700 {
			b.Fatalf("query limit: %+v, %v", matches, err)
		}
	}
}
