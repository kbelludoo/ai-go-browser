package main

import "core:testing"

@(test)
test_empty_and_basic_tokens :: proc(t: ^testing.T) {
	testing.expect(t, estimate_tokens("") == 0)
	testing.expect(t, estimate_tokens("hello world") == 2)
	testing.expect(t, estimate_tokens("#") == 1)
	testing.expect(t, estimate_tokens("123 hello") == 2)
}

@(test)
test_markdown_to_cso :: proc(t: ^testing.T) {
	input := "# Guia\n\n**Odin** e [documentação](https://odin-lang.org/docs/overview/?utm_source=x)\n`codigo`
	want := "H1: Guia\nOdin e documentação -> /docs/overview/\ncodigo"
	got := markdown_to_cso(input)
	testing.expectf(t, got == want, "CSO divergente: obtido %q, esperado %q", got, want)
}

@(test)
test_filter_noise :: proc(t: ^testing.T) {
	input := "Skip to content\n# Conteúdo\nPrivacy policy\nTexto principal"
	got := markdown_to_cso(input)
	testing.expect(t, got == "H1: Conteúdo\nTexto principal")
}

@(test)
test_table_and_limit :: proc(t: ^testing.T) {
	got := markdown_to_cso("| Coluna | Valor |\n| --- | --- |\n| A | 2 |")
	testing.expect(t, got == "Coluna | Valor\nA | 2")
	testing.expect(t, markdown_to_cso("abcdef", 3) == "abc")
}
