package main

import "core:strings"

// Minimal structured page representation for the CSO layer.
// Transport, parsing and interactive state are intentionally separate modules.
Heading :: struct {
	level: int,
	text:  string,
}

Page_Link :: struct {
	label: string,
	url:   string,
}

Page_Input :: struct {
	type: string,
	name: string,
	value: string,
	disabled: bool,
	checked: bool,
}

Page_Button :: struct {
	label: string,
}

Semantic_Page :: struct {
	url:         string,
	title:       string,
	description: string,
	text:        string,
	headings:    []Heading,
	links:       []Page_Link,
	inputs:      []Page_Input,
	buttons:     []Page_Button,
}

site_domain :: proc(raw_url: string) -> string {
	url := strings.trim_space(raw_url)
	if i := strings.index(url, "://"); i >= 0 {
		url = url[i+3:]
	}
	if i := strings.index(url, "/"); i >= 0 {
		url = url[:i]
	}
	if i := strings.index(url, "?"); i >= 0 {
		url = url[:i]
	}
	if i := strings.index(url, ":"); i >= 0 {
		url = url[:i]
	}
	if strings.has_prefix(url, "www.") {
		return url[4:]
	}
	return url
}

// page_to_cso ports the structured-page ToCSO pathway from browser/cso.go.
// It is intentionally independent of how the page was obtained.
page_to_cso :: proc(p: Semantic_Page, max_chars: int = 15000) -> string {
	limit := max_chars
	if limit <= 0 {
		limit = 15000
	}

	out := strings.builder_make()
	defer strings.builder_destroy(&out)
	domain := site_domain(p.url)
	title := strings.trim_space(p.title)
	if title == "" {
		title = domain
	}
	strings.write_string(&out, "@site: ")
	strings.write_string(&out, title)
	strings.write_string(&out, " [")
	strings.write_string(&out, domain)
	strings.write_string(&out, "]\n")

	if p.description != "" {
		desc := strings.trim_space(p.description)
		if len(desc) > 200 {
			desc = desc[:200]
			strings.write_string(&out, "@desc: ")
			strings.write_string(&out, desc)
			strings.write_string(&out, "...\n")
		} else {
			strings.write_string(&out, "@desc: ")
			strings.write_string(&out, desc)
			strings.write_byte(&out, '\n')
		}
	}

	if len(p.headings) > 0 {
		strings.write_string(&out, "@seções:\n")
		for heading in p.headings {
			h := strings.trim_space(heading.text)
			if h == "" || is_boilerplate(h) {
				continue
			}
			if heading.level <= 2 {
				strings.write_string(&out, "  H")
				strings.write_int(&out, heading.level)
				strings.write_string(&out, ": ")
			} else {
				strings.write_string(&out, "    - ")
			}
			strings.write_string(&out, h)
			strings.write_byte(&out, '\n')
		}
	}

	if p.text != "" {
		lines := strings.split(p.text, "\n")
		defer delete(lines)
		content := strings.builder_make()
		defer strings.builder_destroy(&content)
		count := 0
		for original_line in lines {
			line := normalize_spaces(original_line)
			if len(line) <= 10 || is_boilerplate(line) {
				continue
			}
			strings.write_string(&content, line)
			strings.write_byte(&content, '\n')
			count += 1
			if count >= 35 {
				break
			}
		}
		if len(content.buf) > 0 {
			strings.write_string(&out, "@conteúdo:\n")
			strings.write_string(&out, strings.to_string(content))
		}
	}

	if len(p.links) > 0 {
		link_list := strings.builder_make()
		defer strings.builder_destroy(&link_list)
		count := 0
		for link in p.links {
			label := strings.trim_space(link.label)
			if len(label) < 3 || is_boilerplate(label) {
				continue
			}
			target := clean_url_path(link.url)
			if target == "" || target == "/" {
				continue
			}
			strings.write_string(&link_list, "  * ")
			strings.write_string(&link_list, label)
			strings.write_string(&link_list, " -> ")
			strings.write_string(&link_list, target)
			strings.write_byte(&link_list, '\n')
			count += 1
			if count >= 8 {
				break
			}
		}
		if count > 0 {
			strings.write_string(&out, "@links:\n")
			strings.write_string(&out, strings.to_string(link_list))
		}
	}

	if len(p.inputs) > 0 || len(p.buttons) > 0 {
		actions := strings.builder_make()
		defer strings.builder_destroy(&actions)
		count := 0
		for field in p.inputs {
			if field.type == "hidden" {
				continue
			}
			if count > 0 {
				strings.write_string(&actions, " | ")
			}
			strings.write_string(&actions, "in[")
			strings.write_string(&actions, field.type)
			strings.write_byte(&actions, ':')
			strings.write_string(&actions, field.name)
			strings.write_byte(&actions, ']')
			count += 1
			if count >= 4 {
				break
			}
		}
		for button in p.buttons {
			label := strings.trim_space(button.label)
			if label == "" {
				continue
			}
			if count > 0 {
				strings.write_string(&actions, " | ")
			}
			strings.write_string(&actions, "btn[")
			strings.write_string(&actions, label)
			strings.write_byte(&actions, ']')
			count += 1
			if count >= 6 {
				break
			}
		}
		if count > 0 {
			strings.write_string(&out, "@ações: ")
			strings.write_string(&out, strings.to_string(actions))
			strings.write_byte(&out, '\n')
		}
	}

	res := strings.trim_space(strings.to_string(out))
	if len(res) > limit {
		res = res[:limit]
	}
	return strings.clone(strings.trim_space(res))
}
