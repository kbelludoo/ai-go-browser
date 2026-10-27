package main

import "core:testing"
import "core:encoding/json"
import "core:strings"

@(test)
test_browser_structured_json_output :: proc(t: ^testing.T) {
	page := parse_html("<html><head><title>API</title></head><body><h1>OK</h1><a href='/x'>Link</a></body></html>", "https://example.com")
	output := page_output(page, 200)
	data, err := json.marshal(output)
	testing.expect(t, err == nil)
	testing.expect(t, json.is_valid(data))
	testing.expect(t, strings.contains(string(data), "\"status\":200"))
	testing.expect(t, strings.contains(string(data), "Link"))
}
