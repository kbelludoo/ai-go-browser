package main

import "core:strings"
import "core:encoding/entity"
import "core:encoding/json"

// This is a bounded static HTML extraction layer, NOT a fully conforming HTML5 DOM.
// It deliberately does not evaluate scripts, perform layout or process CSS selectors.
HTML_MAX_BYTES :: 2*1024*1024
HTML_MAX_TAGS  :: 50000
HTML_MAX_ITEMS :: 2048

Hydration_State :: struct {
	json_ld: [dynamic]string,
	next_data: string,
	nuxt_data: string,
}

Parsed_Form :: struct {
	method: string,
	action: string,
	fields: [dynamic]Page_Input,
}

Parsed_Page :: struct {
	semantic: Semantic_Page,
	headings: [dynamic]Heading,
	links:    [dynamic]Page_Link,
	inputs:   [dynamic]Page_Input,
	buttons:  [dynamic]Page_Button,
	forms:    [dynamic]Parsed_Form,
	hydration: Hydration_State,
	tags_read: int,
	truncated: bool,
}

HTML_Tag :: struct {
	name: string,
	attrs: string,
	closing: bool,
	self_closed: bool,
}

ascii_lower :: proc(s: string) -> string {
	b := strings.builder_make()
	defer strings.builder_destroy(&b)
	for v in s {
		if v >= 'A' && v <= 'Z' {
			strings.write_rune(&b, v+32)
		} else {
			strings.write_rune(&b, v)
		}
	}
	return strings.clone(strings.to_string(b))
}

html_space :: proc(b: byte) -> bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n' || b == '\f'
}

html_name :: proc(b: byte) -> bool {
	return !html_space(b) && b != '=' && b != '/' && b != '>'
}

html_tag :: proc(raw: string) -> HTML_Tag {
	part := strings.trim_space(raw)
	if len(part) == 0 || part[0] == '!' || part[0] == '?' {
		return {}
	}
	closing := false
	if part[0] == '/' {
		closing = true
		part = strings.trim_space(part[1:])
	}
	i := 0
	for i < len(part) && html_name(part[i]) {
		i += 1
	}
	if i == 0 {
		return {}
	}
	return HTML_Tag {
		name=ascii_lower(part[:i]),
		attrs=part[i:],
		closing=closing,
		self_closed=strings.has_suffix(part, "/"),
	}
}

html_attr :: proc(t: HTML_Tag, key: string) -> string {
	raw := t.attrs
	i := 0
	for i < len(raw) {
		for i < len(raw) && (html_space(raw[i]) || raw[i] == '/') {
			i += 1
		}
		start := i
		for i < len(raw) && html_name(raw[i]) {
			i += 1
		}
		if start == i {
			if i < len(raw) {i += 1}
			continue
		}
		name := raw[start:i]
		for i < len(raw) && html_space(raw[i]) {
			i += 1
		}
		value := ""
		if i < len(raw) && raw[i] == '=' {
			i += 1
			for i < len(raw) && html_space(raw[i]) {i += 1}
			if i < len(raw) && (raw[i] == '"' || raw[i] == '\'') {
				quote := raw[i]
				i += 1
				value_start := i
				for i < len(raw) && raw[i] != quote {i += 1}
				value = raw[value_start:i]
				if i < len(raw) {i += 1}
			} else {
				value_start := i
				for i < len(raw) && !html_space(raw[i]) && raw[i] != '>' {i += 1}
				value = raw[value_start:i]
			}
		}
		if ascii_lower(name) == key {
			decoded, err := entity.decode_xml(value)
			if err == .None {
				return decoded
			}
			return value
		}
	}
	return ""
}


html_tag_has_attr :: proc(t: HTML_Tag, target: string) -> bool {
	raw := t.attrs
	i := 0
	for i < len(raw) {
		for i < len(raw) && (html_space(raw[i]) || raw[i] == '/') {i += 1}
		start := i
		for i < len(raw) && html_name(raw[i]) {i += 1}
		if start == i {
			if i < len(raw) {i += 1}
			continue
		}
		name := raw[start:i]
		if ascii_lower(name) == target {return true}
		for i < len(raw) && html_space(raw[i]) {i += 1}
		if i < len(raw) && raw[i] == '=' {
			i += 1
			for i < len(raw) && html_space(raw[i]) {i += 1}
			if i < len(raw) && (raw[i] == '"' || raw[i] == '\'') {
				q := raw[i]
				i += 1
				for i < len(raw) && raw[i] != q {i += 1}
				if i < len(raw) {i += 1}
			} else {
				for i < len(raw) && !html_space(raw[i]) {i += 1}
			}
		}
	}
	return false
}

html_origin :: proc(base: string) -> string {
	i := strings.index(base, "://")
	if i < 0 {
		return ""
	}
	authority := base[i+3:]
	j := strings.index(authority, "/")
	if j >= 0 {
		return base[:i+3+j]
	}
	if j := strings.index(authority, "?"); j >= 0 {
		return base[:i+3+j]
	}
	return base
}

html_resolve :: proc(base, ref: string) -> string {
	href := strings.trim_space(ref)
	if href == "" || href[0] == '#' {
		return ""
	}
	lower := ascii_lower(href)
	if strings.has_prefix(lower, "https://") || strings.has_prefix(lower, "http://") {
		return href
	}
	if strings.contains(lower, "://") || strings.has_prefix(lower, "javascript:") ||
		strings.has_prefix(lower, "mailto:") || strings.has_prefix(lower, "data:") {
		return ""
	}
	origin := html_origin(base)
	if origin == "" {
		return ""
	}
	if strings.has_prefix(href, "//") {
		if strings.has_prefix(base, "https://") {
			return strings.concatenate([]string{"https:", href})
		}
		return strings.concatenate([]string{"http:", href})
	}
	if strings.has_prefix(href, "/") {
		return strings.concatenate([]string{origin, href})
	}
	path := base[len(origin):]
	if i := strings.index(path, "?"); i >= 0 { path = path[:i] }
	if i := strings.index(path, "#"); i >= 0 { path = path[:i] }
	slash := strings.last_index(path, "/")
	if slash < 0 {
		return strings.concatenate([]string{origin, "/", href})
	}
	return strings.concatenate([]string{origin, path[:slash+1], href})
}

html_decode :: proc(src: string) -> string {
	value, err := entity.decode_xml(src)
	if err != .None {
		return strings.clone(src)
	}
	return value
}

is_block :: proc(tag: string) -> bool {
	switch tag {
	case "div", "p", "li", "br", "tr", "section", "article", "header", "footer",
		"h1", "h2", "h3", "h4", "h5", "h6":
		return true
	}
	return false
}

heading_level :: proc(tag: string) -> int {
	if len(tag) == 2 && tag[0] == 'h' && tag[1] >= '1' && tag[1] <= '6' {
		return int(tag[1]-'0')
	}
	return 0
}

// The caller keeps the page and its backing HTML source alive while reading
// the result. This implementation is optimized for single-shot CLI use.
parse_html :: proc(source: string, page_url: string) -> Parsed_Page {
	result := Parsed_Page{}
	result.semantic.url = strings.clone(page_url)

	html := source
	if len(html) > HTML_MAX_BYTES {
		html = html[:HTML_MAX_BYTES]
		result.truncated = true
	}
	body := strings.builder_make()
	defer strings.builder_destroy(&body)
	captured := strings.builder_make()
	defer strings.builder_destroy(&captured)
	active: string = ""
	active_href := ""
	pending_button := Page_Button{}
	hidden := ""
	script_kind := ""
	script_raw := strings.builder_make()
	defer strings.builder_destroy(&script_raw)
	in_head := false
	active_form := -1

	i := 0
	for i < len(html) && result.tags_read < HTML_MAX_TAGS {
		if html[i] != '<' {
			j := strings.index(html[i:], "<")
			if j < 0 {j = len(html)-i}
			chunk := html_decode(html[i:i+j])
			if hidden == "script" && script_kind != "" {
				if len(script_raw.buf)+j <= 256*1024 {
					strings.write_string(&script_raw, html[i:i+j])
				}
			} else if hidden == "" {
				if active != "" {
					strings.write_string(&captured, chunk)
				}
				if !in_head {
					strings.write_string(&body, chunk)
					strings.write_byte(&body, ' ')
				}
			}
			i += j
			continue
		}
		if strings.has_prefix(html[i:], "<!--") {
			j := strings.index(html[i+4:], "-->")
			if j < 0 {break}
			i += 4+j+3
			continue
		}
		j := i+1
		quote: byte = 0
		for j < len(html) {
			c := html[j]
			if quote != 0 {
				if c == quote {quote = 0}
			} else if c == '\'' || c == '"' {
				quote = c
			} else if c == '>' {
				break
			}
			j += 1
		}
		if j >= len(html) {break}
		tag := html_tag(html[i+1:j])
		i = j+1
		result.tags_read += 1
		if tag.name == "" {continue}
		if hidden != "" {
			if tag.closing && tag.name == hidden {
				if hidden == "script" && script_kind != "" {
					value := strings.trim_space(strings.to_string(script_raw))
					if value != "" && json.is_valid(transmute([]byte)value) {
						switch script_kind {
						case "ld":
							if len(result.hydration.json_ld) < 64 {
								append(&result.hydration.json_ld, strings.clone(value))
							}
						case "next":
							result.hydration.next_data = strings.clone(value)
						case "nuxt":
							result.hydration.nuxt_data = strings.clone(value)
						}
					}
				}
				hidden = ""
				script_kind = ""
			}
			continue
		}
		if !tag.closing {
			switch tag.name {
			case "script", "style", "noscript", "template", "svg":
				hidden = tag.name
				if tag.name == "script" {
					script_kind = ""
					strings.builder_reset(&script_raw)
					if ascii_lower(html_attr(tag, "type")) == "application/ld+json" {
						script_kind = "ld"
					} else if html_attr(tag, "id") == "__NEXT_DATA__" {
						script_kind = "next"
					} else if html_attr(tag, "id") == "__NUXT_DATA__" {
						script_kind = "nuxt"
					}
				}
				continue
			case "head":
				in_head = true
			case "meta":
				if ascii_lower(html_attr(tag, "name")) == "description" {
					result.semantic.description = html_attr(tag, "content")
				}
			case "input":
				if len(result.inputs) < HTML_MAX_ITEMS {
					field_type := ascii_lower(html_attr(tag, "type"))
					if field_type == "" {field_type = "text"}
					field := Page_Input {
						type=field_type,
						name=html_attr(tag, "name"),
						value=html_attr(tag, "value"),
						disabled=html_tag_has_attr(tag, "disabled"),
						checked=html_tag_has_attr(tag, "checked"),
					}
					append(&result.inputs, field)
					if active_form >= 0 && len(result.forms[active_form].fields) < HTML_MAX_ITEMS {
						append(&result.forms[active_form].fields, field)
					}
				}
			case "form":
				if len(result.forms) < HTML_MAX_ITEMS {
					method := ascii_lower(html_attr(tag, "method"))
					if method == "" {method = "get"}
					action := html_resolve(page_url, html_attr(tag, "action"))
					if action == "" {action = page_url}
					append(&result.forms, Parsed_Form {method=method, action=action})
					active_form = len(result.forms)-1
				}
			}
			if active == "" {
				if tag.name == "title" || tag.name == "a" || tag.name == "button" ||
					heading_level(tag.name) > 0 {
					active = tag.name
					active_href = ""
					strings.builder_reset(&captured)
					if tag.name == "a" {active_href = html_resolve(page_url, html_attr(tag, "href"))}
					if tag.name == "button" {
						pending_button = Page_Button {
							disabled=html_tag_has_attr(tag, "disabled"),
						}
						// Prefer explicit HTML overrides, then htmx, then form
						// association. A data-* URL is only a hint, not execution.
						if hint := html_attr(tag, "formaction"); hint != "" {
							pending_button.action = html_resolve(page_url, hint)
							pending_button.method = ascii_lower(html_attr(tag, "formmethod"))
							pending_button.source = "formaction"
						} else if hint := html_attr(tag, "hx-get"); hint != "" {
							pending_button.action = html_resolve(page_url, hint)
							pending_button.method = "get"
							pending_button.source = "hx-get"
						} else if hint := html_attr(tag, "hx-post"); hint != "" {
							pending_button.action = html_resolve(page_url, hint)
							pending_button.method = "post"
							pending_button.source = "hx-post"
						} else if hint := html_attr(tag, "data-url"); hint != "" {
							pending_button.action = html_resolve(page_url, hint)
							pending_button.source = "data-url"
						} else if active_form >= 0 {
							pending_button.action = result.forms[active_form].action
							pending_button.method = result.forms[active_form].method
							pending_button.source = "form"
						}
						if pending_button.method == "" && pending_button.source == "formaction" &&
							active_form >= 0 {
							pending_button.method = result.forms[active_form].method
						}
					}
				}
			}
			if tag.name == "br" {strings.write_byte(&body, '\n')}
		} else {
			if tag.name == "head" {in_head = false}
			if tag.name == "form" {active_form = -1}
			if is_block(tag.name) && !in_head {
				strings.write_byte(&body, '\n')
			}
			if active == tag.name {
				value := normalize_spaces(strings.to_string(captured))
				switch active {
				case "title":
					result.semantic.title = strings.clone(value)
				case "a":
					if value != "" && active_href != "" && len(result.links) < HTML_MAX_ITEMS {
						append(&result.links, Page_Link{label=strings.clone(value), url=strings.clone(active_href)})
					}
				case "button":
					if value != "" && len(result.buttons) < HTML_MAX_ITEMS {
						pending_button.label = strings.clone(value)
						if pending_button.action == "" {
							pending_button.method = ""
							pending_button.source = ""
						}
						append(&result.buttons, pending_button)
					}
				case:
					level := heading_level(active)
					if level > 0 && value != "" && len(result.headings) < HTML_MAX_ITEMS {
						append(&result.headings, Heading{level=level, text=strings.clone(value)})
					}
				}
				active = ""
				strings.builder_reset(&captured)
			}
		}
	}
	if result.tags_read >= HTML_MAX_TAGS || i < len(html) {result.truncated = true}
	result.semantic.text = strings.clone(strings.to_string(body))
	result.semantic.headings = result.headings[:]
	result.semantic.links = result.links[:]
	result.semantic.inputs = result.inputs[:]
	result.semantic.buttons = result.buttons[:]
	return result
}
