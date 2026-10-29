package main

import "core:strings"
import "core:unicode"

// Port of browser/cso.go. This is the first isolated Odin component, not
// a replacement for the existing Go HTTP/DOM/CDP/JavaScript browser.

// estimate_tokens matches the word/punctuation weighting in Go's EstimateTokens.
// It is an estimate, not an actual BPE tokenizer.
estimate_tokens :: proc(text: string) -> int {
	if len(text) == 0 {
		return 0
	}
	words := 0
	punctuation := 0
	in_word := false
	for r in text {
		if unicode.is_letter(r) || unicode.is_digit(r) {
			if !in_word {
				words += 1
				in_word = true
			}
		} else {
			in_word = false
			switch r {
			case '[', ']', '(', ')', '{', '}', '|', '#', '*', '_', 96, '>':
				punctuation += 1
			}
		}
	}
	// Exact integer equivalent of int(float64(words)*1.25) +
	// int(float64(punctuation)*0.75) for practical input lengths.
	result := (words*5)/4 + (punctuation*3)/4
	if result < 1 {
		return 1
	}
	return result
}

is_boilerplate :: proc(source: string) -> bool {
	s := strings.to_lower(strings.trim_space(source))
	if len(s) == 0 {
		return true
	}
	prefixes := []string {
		"pular para", "skip to", "menu principal", "toggle navigation",
		"política de privacidade", "politica de privacidade",
		"privacy policy", "termos de uso", "terms of use",
		"terms of service", "todos os direitos reservados",
		"all rights reserved", "ir para o conteúdo", "ir para o conteudo",
		"cookie", "fale conosco", "voltar ao topo", "back to top",
		"compartilhar", "share on",
	}
	for prefix in prefixes {
		if strings.has_prefix(s, prefix) {
			return true
		}
	}
	return strings.contains(s, "cookie") &&
		(strings.contains(s, "concordo") ||
		strings.contains(s, "aceitar") ||
		strings.contains(s, "aceito"))
}

// The Go CSO path projection discards the query and fragment of links with
// a nonempty path. This deliberately does not implement full URL resolution.
clean_url_path :: proc(raw: string) -> string {
	u := strings.trim_space(raw)
	if strings.has_prefix(u, "javascript:") || strings.has_prefix(u, "mailto:") {
		return ""
	}
	if strings.has_prefix(u, "https://") || strings.has_prefix(u, "http://") {
		protocol_sep := strings.index(u, "://")
		host_offset := protocol_sep+3
		relative_slash := strings.index(u[host_offset:], "/")
		if relative_slash < 0 {
			return ""
		}
		u = u[host_offset+relative_slash:]
	}
	if i := strings.index(u, "?"); i >= 0 {
		u = u[:i]
	}
	if i := strings.index(u, "#"); i >= 0 {
		u = u[:i]
	}
	return u
}

// Replace Markdown links in one line. Nested bracket links are outside the
// scope of the original regex-based implementation as well.
rewrite_links :: proc(line: string) -> string {
	out := strings.builder_make()
	defer strings.builder_destroy(&out)
	i := 0
	for i < len(line) {
		if line[i] != '[' {
			strings.write_byte(&out, line[i])
			i += 1
			continue
		}
		after_open := i+1
		end_label := strings.index(line[after_open:], "](")
		if end_label < 0 {
			strings.write_byte(&out, line[i])
			i += 1
			continue
		}
		label_end := after_open+end_label
		target_begin := label_end+2
		target_end_rel := strings.index(line[target_begin:], ")")
		if target_end_rel < 0 {
			strings.write_byte(&out, line[i])
			i += 1
			continue
		}
		target_end := target_begin+target_end_rel
		label := strings.trim_space(line[after_open:label_end])
		if !is_boilerplate(label) {
			strings.write_string(&out, label)
			strings.write_string(&out, " -> ")
			strings.write_string(&out, clean_url_path(line[target_begin:target_end]))
		}
		i = target_end+1
	}
	return strings.clone(strings.to_string(out))
}

strip_markdown :: proc(line: string) -> string {
	out := strings.builder_make()
	defer strings.builder_destroy(&out)
	i := 0
	for i < len(line) {
		if i+1 < len(line) && ((line[i] == '*' && line[i+1] == '*') ||
			(line[i] == '_' && line[i+1] == '_')) {
			i += 2
			continue
		}
		if line[i] != 96 {
			strings.write_byte(&out, line[i])
		}
		i += 1
	}
	return strings.clone(strings.to_string(out))
}

normalize_spaces :: proc(line: string) -> string {
	out := strings.builder_make()
	defer strings.builder_destroy(&out)
	prev_space := false
	for b in line {
		if b == ' ' || b == '\t' {
			if !prev_space {
				strings.write_byte(&out, ' ')
			}
			prev_space = true
		} else {
			strings.write_rune(&out, b)
			prev_space = false
		}
	}
	return strings.clone(strings.trim_space(strings.to_string(out)))
}

strip_table_markup :: proc(line: string) -> string {
	if !strings.has_prefix(line, "|") || !strings.has_suffix(line, "|") {
		return line
	}
	out := strings.builder_make()
	defer strings.builder_destroy(&out)
	parts := strings.split(line, "|")
	defer delete(parts)
	for part in parts {
		cell := strings.trim_space(part)
		if cell == "" || strings.contains(cell, "---") {
			continue
		}
		if len(out.buf) > 0 {
			strings.write_string(&out, " | ")
		}
		strings.write_string(&out, cell)
	}
	return strings.clone(strings.to_string(out))
}

// markdown_to_cso is the Odin counterpart of browser.MarkdownToCSO.
// The limit is measured in bytes for compatibility with the Go code.
markdown_to_cso :: proc(markdown: string, max_chars: int = 10000) -> string {
	limit := max_chars
	if limit <= 0 {
		limit = 10000
	}
	out := strings.builder_make()
	defer strings.builder_destroy(&out)
	lines := strings.split(markdown, "\n")
	defer delete(lines)
	for original_line in lines {
		line := strings.trim_space(original_line)
		if line == "" || is_boilerplate(line) {
			continue
		}
		if strings.has_prefix(line, "#") {
			level := 0
			for level < len(line) && line[level] == '#' {
				level += 1
			}
			heading := strings.trim_space(line[level:])
			if is_boilerplate(heading) {
				continue
			}
			if len(out.buf) > 0 {
				strings.write_byte(&out, '\n')
			}
			strings.write_byte(&out, 'H')
			strings.write_int(&out, level)
			strings.write_string(&out, ": ")
			strings.write_string(&out, heading)
			continue
		}
		linked := rewrite_links(line)
		cleaned := strip_markdown(linked)
		normalized := normalize_spaces(cleaned)
		normalized = strip_table_markup(normalized)
		normalized = strings.trim_space(normalized)
		if normalized == "" || is_boilerplate(normalized) {
			continue
		}
		if len(out.buf) > 0 {
			strings.write_byte(&out, '\n')
		}
		strings.write_string(&out, normalized)
	}
	res := strings.trim_space(strings.to_string(out))
	if len(res) > limit {
		res = res[:limit]
	}
	return strings.clone(strings.trim_space(res))
}
