package main

import "core:testing"

@(test)
test_page_to_cso :: proc(t: ^testing.T) {
	headings := []Heading{{level=1, text="Principal"}, {level=3, text="Detalhes"}}
	links := []Page_Link{{label="Ver artigo", url="https://www.example.com/artigo?utm_source=x"}, {label="Home", url="/"}}
	fields := []Page_Input{{type="hidden", name="csrf"}, {type="text", name="q"}}
	buttons := []Page_Button{{label="Pesquisar"}}
	p := Semantic_Page{
		url="https://www.example.com/index.html",
		title="Exemplo",
		description="Um portal",
		text="Texto principal da pagina para agentes\nSkip to content",
		headings=headings,
		links=links,
		inputs=fields,
		buttons=buttons,
	}
	got := page_to_cso(p)
	want := "@site: Exemplo [example.com]\n@desc: Um portal\n@seções:\n  H1: Principal\n    - Detalhes\n@conteúdo:\nTexto principal da pagina para agentes\n@links:\n  * Ver artigo -> /artigo\n@ações: in[text:q] | btn[Pesquisar]"
	testing.expectf(t, got == want, "Page CSO: obtido %q; esperado %q", got, want)
}

@(test)
test_page_cso_fallback_and_limits :: proc(t: ^testing.T) {
	p := Semantic_Page{url="https://www.example.com/a"}
	testing.expect(t, page_to_cso(p) == "@site: example.com [example.com]")
	testing.expect(t, len(page_to_cso(p, 5)) == 5)
}
