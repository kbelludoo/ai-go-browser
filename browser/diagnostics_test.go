package browser

import (
	"strings"
	"testing"
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
