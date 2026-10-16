package browser

import (
	"strings"
	"testing"
)

var (
	benchmarkTechStackResult      TechStack
	benchmarkStructuredDataResult map[string]string
)

func BenchmarkAnalyzeTechStackLargeHTML(b *testing.B) {
	const size = 8 << 20
	for _, tc := range []struct {
		name   string
		body   string
		marker string
	}{
		{"StaticLowercase", `<p>article text without framework markers.</p>`, ""},
		{"StaticMixedCase", `<P>Article TEXT without framework markers.</P>`, ""},
		{"NextMixedCaseAtEnd", `<P>Article TEXT without framework markers.</P>`, `<script id="__NEXT_DATA__">{}</script>`},
	} {
		b.Run(tc.name, func(b *testing.B) {
			html := strings.Repeat(tc.body, size/len(tc.body)) + tc.marker
			b.SetBytes(int64(len(html)))
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				benchmarkTechStackResult = AnalyzeTechStack(html, len(html)/2, "utf-8")
			}
		})
	}
}

func BenchmarkExtractStructuredDataLargeHTML(b *testing.B) {
	const size = 8 << 20
	html := `<script id="__NEXT_DATA__"> {"props":{"title":"Example"}} </script>` +
		`<script type="application/ld+json"> {"@type":"Article","name":"Example"} </script>` +
		strings.Repeat(`<p>Article text.</p>`, size/len(`<p>Article text.</p>`))
	b.SetBytes(int64(len(html)))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchmarkStructuredDataResult = ExtractStructuredData(html)
	}
}
