package browser

import (
	"strings"
	"testing"
)

func TestParseLinksFormsAndHiddenContent(t *testing.T) {
	page := `<!DOCTYPE html><html><head><title>Meu &amp; título</title><style>.a { color: red; }</style><script>var x='<a href="/fake">fake</a>'</script></head><body><h1>Exemplo <em>de agente</em></h1><p>Conteúdo visível</p><div hidden><a href="/secret">segredo</a></div><div aria-hidden="true">omitido</div><a href="/p?x=1&amp;y=2" aria-label="Abrir">página <span>real</span></a><form action="/search" method="get"><input name="q" type="text" aria-label="Pesquisa"><button>Buscar</button></form></body></html>`
	p := Parse([]byte(page), "http://example.test/", 200, len(page), 5000)
	if p.Title != "Meu & título" {
		t.Errorf("title=%q", p.Title)
	}
	if len(p.Headings) != 1 || p.Headings[0].Text != "Exemplo de agente" {
		t.Errorf("headings=%+v", p.Headings)
	}
	if len(p.Links) != 1 || p.Links[0].URL != "http://example.test/p?x=1&y=2" {
		t.Errorf("links=%+v", p.Links)
	}
	if len(p.Forms) != 1 || len(p.Forms[0].Fields) != 1 || p.Forms[0].Fields[0].Ref != "i1" {
		t.Errorf("forms=%+v", p.Forms)
	}
	if strings.Contains(p.Text, "omitido") || strings.Contains(p.Text, "fake") || strings.Contains(p.Text, "red") {
		t.Errorf("leaked hidden text: %q", p.Text)
	}
}
func TestEscapedQuotedTagAndBlockedSchemes(t *testing.T) {
	raw := `<title>Teste</title><a href="/x?text=a>b">Abrir</a><a href="javascript:alert(1)">não</a><a href="file:///tmp/test">não</a><p>ok</p>`
	p := Parse([]byte(raw), "https://example.org/root", 200, len(raw), 5000)
	if len(p.Links) != 1 || p.Links[0].URL != "https://example.org/x?text=a>b" {
		t.Fatalf("links = %+v", p.Links)
	}
}
func TestTextLimit(t *testing.T) {
	p := Parse([]byte("<p>123456789abcdef</p>"), "https://example.org", 200, 22, 8)
	if !p.Truncated || p.Text != "12345678" {
		t.Fatalf("truncation: %+v", p)
	}
}
