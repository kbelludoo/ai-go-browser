package browser

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type benchmarkTransport struct{ html string }

func (tr benchmarkTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"text/html; charset=utf-8"}},
		Body:       io.NopCloser(strings.NewReader(tr.html)),
		Request:    req,
	}, nil
}

func BenchmarkNavigateLargeHTML(b *testing.B) {
	raw := `<html><head><title>Large page</title></head><body><h1>Article</h1><script>` + strings.Repeat("x", 1<<20) + `</script><p>Readable content.</p></body></html>`
	br := New([]string{"example.org"}, false)
	br.client.Transport = benchmarkTransport{html: raw}
	defer br.Close()
	b.ReportAllocs()
	b.SetBytes(int64(len(raw)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := br.request("GET", "https://example.org", nil, false); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryStats(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = mem()
	}
}
