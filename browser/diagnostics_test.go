package browser

import (
	"strings"
	"testing"
	"unsafe"
)

func TestWindows1252AndISO88591Normalization(t *testing.T) {
	// "Ação de Navegação: Café, Pão & R$ 15,00" encoded in ISO-8859-1 / Windows-1252
	// 'ç' = 0xE7, 'ã' = 0xE3, 'é' = 0xE9, 'ã' = 0xE3
	rawBytes := []byte{
		'A', 0xE7, 0xE3, 'o', ' ', 'd', 'e', ' ', 'N', 'a', 'v', 'e', 'g', 'a', 0xE7, 0xE3, 'o',
		':', ' ', 'C', 'a', 'f', 0xE9, ',', ' ', 'P', 0xE3, 'o',
	}

	enc, normalized := DetectAndNormalizeEncoding("text/html; charset=iso-8859-1", rawBytes)
	if enc != "iso-8859-1" {
		t.Fatalf("expected iso-8859-1, got %s", enc)
	}

	text := string(normalized)
	expected := "Ação de Navegação: Café, Pão"
	if text != expected {
		t.Fatalf("expected %q, got %q", expected, text)
	}
}

func TestMetaCharsetDetection(t *testing.T) {
	htmlContent := []byte(`<!DOCTYPE html><html><head><meta charset="windows-1252"></head><body>Teste</body></html>`)
	enc, _ := DetectAndNormalizeEncoding("text/html", htmlContent)
	if enc != "windows-1252" {
		t.Fatalf("expected windows-1252, got %s", enc)
	}
}

func TestTechStackDiagnostics(t *testing.T) {
	// Test WordPress
	wpHTML := `<html><head><link rel="stylesheet" href="/wp-content/themes/twentytwenty/style.css"></head><body>` + strings.Repeat("Texto de artigo no blog. ", 50) + `</body></html>`
	stackWP := AnalyzeTechStack(wpHTML, 1000, "utf-8")
	if stackWP.Framework != "WordPress" || stackWP.RenderMode != "SSR (PHP/Server)" {
		t.Fatalf("unexpected WP stack: %+v", stackWP)
	}

	// Test React CSR (Empty body shell)
	reactCSR := `<html><head><script src="/static/js/main.chunk.js"></script></head><body><div id="root"></div><script>/* React bundle */</script></body></html>`
	stackReact := AnalyzeTechStack(reactCSR, 20, "utf-8")
	if stackReact.RenderMode != "CSR (Client-Side Rendering)" || stackReact.Notice == "" {
		t.Fatalf("expected CSR detection and warning, got: %+v", stackReact)
	}

	// Test Next.js SSR
	nextHTML := `<html><head></head><body><div id="__next"><h1>Next Page</h1><p>` + strings.Repeat("Conteúdo rico do servidor. ", 30) + `</p></div><script id="__NEXT_DATA__">{}</script></body></html>`
	stackNext := AnalyzeTechStack(nextHTML, 800, "utf-8")
	if stackNext.Framework != "Next.js" || stackNext.RenderMode != "SSR (Server Rendered)" {
		t.Fatalf("unexpected Next stack: %+v", stackNext)
	}
}

func TestExtractStructuredDataDoesNotRetainHTML(t *testing.T) {
	for _, tc := range []struct {
		name, key, script, want string
	}{
		{
			name:   "NextData",
			key:    "next_data",
			script: `<script id="__NEXT_DATA__"> {"props":{"title":"Example"}} </script>`,
			want:   `{"props":{"title":"Example"}}`,
		},
		{
			name:   "SingleJSONLD",
			key:    "json_ld",
			script: `<script type="application/ld+json"> {"@type":"Article","name":"Example"} </script>`,
			want:   `{"@type":"Article","name":"Example"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			html := tc.script + strings.Repeat("x", 2<<20)
			got := ExtractStructuredData(html)[tc.key]
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
			// Equal values must have independent storage, so keeping metadata
			// does not keep this multi-megabyte source document alive.
			sourceValue := html[strings.Index(html, tc.want):]
			if unsafe.StringData(got) == unsafe.StringData(sourceValue) {
				t.Fatal("structured data still aliases the HTML document")
			}
		})
	}
}

func TestExtractStructuredDataLimitsAndJoining(t *testing.T) {
	largeNext := strings.Repeat("n", 8193)
	html := `<script id="__NEXT_DATA__"> ` + largeNext + ` </script>` +
		`<script type="application/ld+json"> </script>` +
		`<script type="application/ld+json"> {"name":"first"} </script>` +
		`<script type="application/ld+json"> {"name":"second"} </script>` +
		`<script type="application/ld+json"> {"name":"ignored fourth"} </script>`
	got := ExtractStructuredData(html)
	if want := largeNext[:8192] + "... [truncated]"; got["next_data"] != want {
		t.Fatalf("unexpected Next.js truncation: length %d", len(got["next_data"]))
	}
	if want := `{"name":"first"}` + "\n---\n" + `{"name":"second"}`; got["json_ld"] != want {
		t.Fatalf("expected first three JSON-LD matches only, got %q", got["json_ld"])
	}
	tooLarge := `<script type="application/ld+json">` + strings.Repeat("x", 4096) + `</script>`
	if data := ExtractStructuredData(tooLarge); len(data) != 0 {
		t.Fatalf("expected oversized JSON-LD to be omitted, got %+v", data)
	}
}
