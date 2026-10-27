package main

import "core:fmt"
import "core:encoding/json"
import vmem "core:mem/virtual"

Browser_Output :: struct {
	url: string,
	status: int,
	title: string,
	description: string,
	text: string,
	headings: []Heading,
	links: []Page_Link,
	buttons: []Page_Button,
	inputs: []Page_Input,
	forms: []Parsed_Form,
	hydration: Hydration_State,
	truncated: bool,
}

page_output :: proc(p: Parsed_Page, status: int) -> Browser_Output {
	return Browser_Output {
		url=p.semantic.url,
		status=status,
		title=p.semantic.title,
		description=p.semantic.description,
		text=p.semantic.text,
		headings=p.semantic.headings,
		links=p.semantic.links,
		buttons=p.semantic.buttons,
		inputs=p.semantic.inputs,
		forms=p.forms[:],
		hydration=p.hydration,
		truncated=p.truncated,
	}
}

// This writer uses a per-request arena; the JSON and all parsed HTML are
// released once the output is emitted.
export_page_json :: proc(response: Fetch_Result) {
	arena: vmem.Arena
	if vmem.arena_init_growing(&arena) != nil {
		fmt.println("{\"error\":\"arena allocation failed\"}")
		return
	}
	defer vmem.arena_destroy(&arena)
	context.allocator = vmem.arena_allocator(&arena)
	page := parse_html(response.body, response.effective_url)
	payload := page_output(page, response.status)
	data, err := json.marshal(payload)
	if err != nil {
		fmt.println("{\"error\":\"JSON marshal failed\"}")
		return
	}
	fmt.println(string(data))
}
