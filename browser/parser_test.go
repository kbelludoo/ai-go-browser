package browser

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
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

func TestPageTextNormalizationAndLimits(t *testing.T) {
	raw := []byte("<p>\t ação \u2003世界 </p><p hidden>segredo</p><p>\u00a0🙂\n fim \r</p>")
	want := "ação 世界 🙂 fim"
	engines := map[string]func(int) Page{
		"Parser": func(limit int) Page {
			return Parse(raw, "https://example.org", 200, len(raw), limit)
		},
		"DOM": func(limit int) Page {
			doc, err := ParseDocument(raw, "https://example.org", 200, len(raw))
			if err != nil {
				t.Fatal(err)
			}
			return doc.Page(limit)
		},
	}
	for name, parse := range engines {
		t.Run(name, func(t *testing.T) {
			for _, limit := range []int{-1, 0, 1, 4, 5, 8, utf8.RuneCountInString(want), 100} {
				t.Run(fmt.Sprint(limit), func(t *testing.T) {
					text := want
					truncated := limit > 0 && utf8.RuneCountInString(want) > limit
					if truncated {
						text = string([]rune(want)[:limit])
					}
					page := parse(limit)
					if page.Text != text || page.Truncated != truncated || !utf8.ValidString(page.Text) {
						t.Fatalf("text=%q truncated=%v; want %q, %v", page.Text, page.Truncated, text, truncated)
					}
				})
			}
		})
	}
}

func TestTextLimitKeepsReferences(t *testing.T) {
	raw := []byte(`<p>Primeiro texto</p><h1>Último título</h1><a href="/next">Próxima</a><form action="/search"><input name="q"><button>Buscar</button></form>`)
	doc, err := ParseDocument(raw, "https://example.org", 200, len(raw))
	if err != nil {
		t.Fatal(err)
	}
	for name, page := range map[string]Page{
		"Parser": Parse(raw, "https://example.org", 200, len(raw), 1),
		"DOM":    doc.Page(1),
	} {
		t.Run(name, func(t *testing.T) {
			if page.Text != "P" || !page.Truncated {
				t.Fatalf("text limit=%+v", page)
			}
			if len(page.Headings) != 1 || page.Headings[0].Text != "Último título" {
				t.Fatalf("headings beyond limit=%+v", page.Headings)
			}
			if len(page.Links) != 1 || page.Links[0].Ref != "l1" || page.Links[0].Label != "Próxima" {
				t.Fatalf("links beyond limit=%+v", page.Links)
			}
			if len(page.Forms) != 1 || len(page.Forms[0].Fields) != 1 || page.Forms[0].Fields[0].Ref != "i1" || page.Forms[0].SubmitLabel != "Buscar" {
				t.Fatalf("forms beyond limit=%+v", page.Forms)
			}
		})
	}
}
