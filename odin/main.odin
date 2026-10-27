package main

import "core:fmt"
import "core:os"

// Local, deterministic first milestone: transform an input Markdown file into
// the compact semantic representation consumed by AI agents.
main :: proc() {
	if len(os.args) != 3 {
		fmt.println("Uso: ai-odin-browser cso|tokens arquivo.md")
		return
	}
	operation := os.args[1]
	if operation != "cso" && operation != "tokens" {
		fmt.println("Operacao invalida. Use cso ou tokens.")
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
		return
	}
	fmt.println(markdown_to_cso(string(data)))
}
