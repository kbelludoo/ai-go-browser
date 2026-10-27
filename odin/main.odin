package main

import "core:fmt"
import "core:os"
import "core:strconv"
import "core:strings"
import vmem "core:mem/virtual"
import "base:runtime"

// Minimal native Odin command-line browser. This is a static semantic client,
// not a JavaScript/Chromium replacement.
print_usage :: proc() {
	fmt.println("AI Odin Browser - portable semantic browser prototype")
	fmt.println("Uso:")
	fmt.println("  ai-odin-browser cso arquivo.md")
	fmt.println("  ai-odin-browser tokens arquivo.md")
	fmt.println("  ai-odin-browser open https://example.com")
	fmt.println("  ai-odin-browser follow https://example.com l1")
	fmt.println("  Adicione --allow-private no fim para testar um servidor local.")
}

print_page :: proc(response: Fetch_Result, page: Parsed_Page) {
	fmt.printf("[%d] %s\n", response.status, page.semantic.title)
	fmt.printf("URL: %s\n", response.effective_url)
	if page.truncated {
		fmt.println("AVISO: limite de HTML ou tags atingido")
	}
	fmt.println(page_to_cso(page.semantic))
	if len(page.links) > 0 {
		fmt.println("@references:")
		for link, i in page.links {
			fmt.printf("  l%d: %s -> %s\n", i+1, link.label, link.url)
		}
	}
	if len(page.forms) > 0 {
		fmt.println("@forms:")
		for form, i in page.forms {
			fmt.printf("  f%d: %s %s\n", i+1, form.method, form.action)
		}
	}
}

// Each semantic operation uses a temporary arena. All intermediate HTML,
// Markdown, entities and Page data are reclaimed in one operation.
print_fetched_page :: proc(response: Fetch_Result) {
	arena: vmem.Arena
	if vmem.arena_init_growing(&arena) != nil {
		fmt.println("Erro: não foi possível alocar arena")
		return
	}
	defer vmem.arena_destroy(&arena)
	context.allocator = vmem.arena_allocator(&arena)
	print_page(response, parse_html(response.body, response.effective_url))
}

get_link_target :: proc(response: Fetch_Result, index: int) -> (string, bool) {
	arena: vmem.Arena
	if vmem.arena_init_growing(&arena) != nil {
		return "", false
	}
	defer vmem.arena_destroy(&arena)
	context.allocator = vmem.arena_allocator(&arena)
	page := parse_html(response.body, response.effective_url)
	if index < 1 || index > len(page.links) {
		return "", false
	}
	// Duplicate into the general heap before releasing the per-page arena.
	return strings.clone(page.links[index-1].url, runtime.default_context().allocator), true
}

main :: proc() {
	if len(os.args) < 3 {
		print_usage()
		return
	}
	operation := os.args[1]
	if operation == "cso" || operation == "tokens" {
		if len(os.args) != 3 {
			print_usage()
			return
		}
		data, err := os.read_entire_file(os.args[2], context.allocator)
		if err != nil {
			fmt.printf("Erro ao ler arquivo: %v\n", err)
			return
		}
		defer delete(data)
		if operation == "tokens" {
			fmt.println(estimate_tokens(string(data)))
		} else {
			fmt.println(markdown_to_cso(string(data)))
		}
		return
	}
	if operation != "open" && operation != "follow" {
		print_usage()
		return
	}
	allow_private := false
	for arg in os.args[3:] {
		if arg == "--allow-private" {allow_private = true}
	}
	if operation == "open" && len(os.args) > 4 ||
		operation == "follow" && (len(os.args) < 4 || len(os.args) > 5) {
		print_usage()
		return
	}
	target := os.args[2]
	first := fetch_get(target, allow_private)
	defer delete(first.body)
	defer delete(first.effective_url)
	if first.error != "" {
		fmt.printf("Erro HTTP: %s\n", first.error)
		return
	}
	if operation == "open" {
		print_fetched_page(first)
		return
	}
	if !strings.has_prefix(os.args[3], "l") {
		fmt.println("Referência inválida. Use l1, l2...")
		return
	}
	index, ok := strconv.parse_int(os.args[3][1:], 10)
	if !ok || index < 1 {
		fmt.println("Referência inválida. Use l1, l2...")
		return
	}
	next_url, found := get_link_target(first, index)
	if !found {
		fmt.printf("Link l%d não encontrado\n", index)
		return
	}
	defer delete(next_url)
	next := fetch_get(next_url, allow_private)
	defer delete(next.body)
	defer delete(next.effective_url)
	if next.error != "" {
		fmt.printf("Erro ao seguir link: %s\n", next.error)
		return
	}
	print_fetched_page(next)
}
