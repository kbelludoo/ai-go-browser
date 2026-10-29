package main

import "core:testing"
import "core:strings"

@(test)
test_html_static_semantics :: proc(t: ^testing.T) {
	html := "<!doctype html><html><head><title>Exemplo &amp; Teste</title><meta name='description' content='Um portal'></head><body><h1>Bem <em>vindo</em></h1><p>Texto &amp; detalhes do portal de teste.</p><a href='/artigo'>Ler artigo</a><form action='/busca' method='GET'><input name='q'><input type='hidden' name='csrf'><button>Pesquisar</button></form><script>document.write('SEGREDO')</script></body></html>"
	p := parse_html(html, "https://example.com")
	testing.expect(t, p.semantic.title == "Exemplo & Teste")
	testing.expect(t, p.semantic.description == "Um portal")
	testing.expect(t, len(p.headings) == 1)
	testing.expect(t, p.headings[0].text == "Bem vindo")
	testing.expect(t, len(p.links) == 1)
	testing.expect(t, p.links[0].url == "https://example.com/artigo")
	testing.expect(t, len(p.forms) == 1)
	testing.expect(t, p.forms[0].method == "get")
	testing.expect(t, len(p.inputs) == 2)
	testing.expect(t, len(p.buttons) == 1)
	testing.expect(t, !strings.contains(p.semantic.text, "SEGREDO"))
	testing.expect(t, strings.contains(page_to_cso(p.semantic), "Ler artigo -> /artigo"))
}

@(test)
test_html_attribute_quote_and_security_scheme :: proc(t: ^testing.T) {
	html := "<a href=\"/x?q=1>2\">A</a><a href='javascript:alert(1)'>Bad</a><h2>Segundo</h2>"
	p := parse_html(html, "https://example.org/base")
	testing.expect(t, len(p.links) == 1)
	testing.expect(t, p.links[0].url == "https://example.org/x?q=1>2")
	testing.expect(t, len(p.headings) == 1)
	testing.expect(t, p.headings[0].level == 2)
}
