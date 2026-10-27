package main

import "core:strings"

// An explicitly bounded static selector scanner. Not a full CSS engine.
// Supported: tag, #id, .class, tag.class, tag#id, [attr], [attr=value],
// and one descendant combinator ("main a"). No JavaScript, event loop or
// live DOM; the caller must treat matches as observations, not executable API.
QUERY_MAX_RESULTS :: 32

Query_Element :: struct {
	tag: string,
	id: string,
	role: string,
	label: string,
	text: string,
	href: string,
	name: string,
	value: string,
	type: string,
	disabled: bool,
}

Query_Frame :: struct {
	tag: string,
	hidden: bool,
	ancestor_match: bool,
	result_index: int,
}

query_simple_matches :: proc(tag: HTML_Tag, selector: string) -> bool {
	s := strings.trim_space(selector)
	if s == "" {return false}
	attr_key := ""
	attr_value := ""
	attr_present := false
	if start := strings.index(s, "["); start >= 0 {
		if !strings.has_suffix(s, "]") {return false}
		filter := s[start+1:len(s)-1]
		s = s[:start]
		if eq := strings.index(filter, "="); eq >= 0 {
			attr_key = ascii_lower(strings.trim_space(filter[:eq]))
			attr_value = strings.trim_space(filter[eq+1:])
			if len(attr_value)>=2 &&
				((attr_value[0]=='"' && attr_value[len(attr_value)-1]=='"') ||
				 (attr_value[0]=='\'' && attr_value[len(attr_value)-1]=='\'')) {
				attr_value = attr_value[1:len(attr_value)-1]
			}
		} else {
			attr_key = ascii_lower(strings.trim_space(filter))
			attr_present = true
		}
		if attr_key == "" {return false}
	}
	id := ""
	class := ""
	core := s
	if start := strings.index(core, "#"); start >= 0 {
		id = core[start+1:]
		core = core[:start]
	}
	if start := strings.index(core, "."); start >= 0 {
		class = core[start+1:]
		core = core[:start]
	}
	if core != "" && core != "*" && ascii_lower(core) != tag.name {return false}
	if id != "" && html_attr(tag, "id") != id {return false}
	if class != "" {
		found := false
		for part in strings.split(html_attr(tag, "class"), " ") {
			if part == class {found = true}
		}
		if !found {return false}
	}
	if attr_key != "" {
		if attr_present {
			if !html_tag_has_attr(tag, attr_key) {return false}
		} else if html_attr(tag, attr_key) != attr_value {
			return false
		}
	}
	return true
}

query_inert_tag :: proc(tag: string) -> bool {
	switch tag {
	case "head", "script", "style", "noscript", "template", "svg":
		return true
	}
	return false
}

query_void_tag :: proc(tag: string) -> bool {
	switch tag {
	case "area", "base", "br", "col", "embed", "hr", "img", "input",
		"link", "meta", "param", "source", "track", "wbr":
		return true
	}
	return false
}

query_static :: proc(html: string, url: string, selector: string) -> ([]Query_Element, string) {
	if selector == "" || len(selector)>200 ||
		strings.contains(selector, ":") || strings.contains(selector, ">") ||
		strings.contains(selector, ",") || strings.contains(selector, "~") ||
		strings.contains(selector, "+") {
		return nil, "selector missing, too long or unsupported"
	}
	parts := strings.split(selector, " ")
	defer delete(parts)
	if len(parts)<1 || len(parts)>2 {
		return nil, "at most one descendant combinator supported"
	}
	for p in parts {
		if p == "" {return nil, "empty selector component"}
	}
	out := make([dynamic]Query_Element)
	stack := make([dynamic]Query_Frame)
	src := html
	if len(src)>HTML_MAX_BYTES {src=src[:HTML_MAX_BYTES]}
	i := 0
	tag_count := 0
	for i<len(src) && tag_count<HTML_MAX_TAGS {
		if src[i]!='<' {
			k := strings.index(src[i:], "<")
			if k<0 {k=len(src)-i}
			if k>0 {
				for frame in stack {
					if frame.result_index >= 0 {
						idx := frame.result_index
						if len(out[idx].text) < 700 {
							word := html_decode(src[i:i+k])
							out[idx].text = strings.concatenate([]string{out[idx].text, word})
						}
					}
				}
			}
			i+=k
			continue
		}
		if strings.has_prefix(src[i:], "<!--") {
			k := strings.index(src[i+4:], "-->")
			if k<0 {break}
			i+=4+k+3
			continue
		}
		end := i+1
		quote: byte=0
		for end<len(src) {
			c:=src[end]
			if quote!=0 {
				if c==quote {quote=0}
			} else if c=='\'' || c=='"' {
				quote=c
			} else if c=='>' {break}
			end+=1
		}
		if end>=len(src) {break}
		tag:=html_tag(src[i+1:end])
		i=end+1
		tag_count+=1
		if tag.name=="" {continue}
		if tag.closing {
			for j:=len(stack)-1;j>=0;j-=1 {
				if stack[j].tag==tag.name {
					resize(&stack,j)
					break
				}
			}
			continue
		}
		hidden := query_inert_tag(tag.name) || html_tag_has_attr(tag, "hidden") ||
			ascii_lower(html_attr(tag, "aria-hidden")) == "true"
		if len(stack)>0 && stack[len(stack)-1].hidden {hidden=true}
		is_ancestor := !hidden && query_simple_matches(tag,parts[0])
		match := !hidden && query_simple_matches(tag,parts[len(parts)-1])
		if match && len(parts)==2 {
			match=false
			for fr in stack {
				if fr.ancestor_match {match=true;break}
			}
		}
		idx: int=-1
		if match && len(out)<QUERY_MAX_RESULTS {
			idx=len(out)
			append(&out,Query_Element{
				tag=tag.name,
				id=html_attr(tag,"id"),
				role=html_attr(tag,"role"),
				label=html_attr(tag,"aria-label"),
				href=html_resolve(url,html_attr(tag,"href")),
				name=html_attr(tag,"name"),
				value=html_attr(tag,"value"),
				type=html_attr(tag,"type"),
				disabled=html_tag_has_attr(tag,"disabled"),
			})
		}
		if !tag.self_closed && !query_void_tag(tag.name) {
			append(&stack,Query_Frame{
				tag=tag.name,
				hidden=hidden,
				ancestor_match=is_ancestor,
				result_index=idx,
			})
		}
	}
	for i in 0..<len(out) {
		out[i].text=html_normalize_visible_text(out[i].text)
		if len(out[i].text)>700 {out[i].text=out[i].text[:700]}
	}
	return out[:], ""
}
