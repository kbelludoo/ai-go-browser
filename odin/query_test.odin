package main

import "core:testing"

@(test)
test_static_query_selector_subset :: proc(t: ^testing.T) {
	html := "<html><head><script>evil</script></head><body><main id='app'><article class='product featured'><a id='read' href='/item/1'>Ver <em>detalhes</em></a><input name='q' value='odin'></article></main><div hidden><a href='/hidden'>Secret</a></div></body></html>"
	link, err := query_static(html, "https://example.com", "main a")
	testing.expect(t, err == "")
	testing.expect(t, len(link)==1)
	if len(link)>0 {
		testing.expect(t, link[0].text=="Ver detalhes")
		testing.expect(t, link[0].href=="https://example.com/item/1")
	}
	by_id, _ := query_static(html, "https://example.com", "#read")
	testing.expect(t, len(by_id)==1)
	by_class, _ := query_static(html, "https://example.com", ".featured")
	testing.expect(t, len(by_class)==1)
	by_attr, _ := query_static(html, "https://example.com", "input[name=q]")
	testing.expect(t, len(by_attr)==1)
	if len(by_attr)>0 {
		testing.expect(t, by_attr[0].value=="odin")
	}
	hidden, _ := query_static(html, "https://example.com", "a")
	testing.expect(t, len(hidden)==1)
}

@(test)
test_query_rejects_unsupported_css_syntax :: proc(t: ^testing.T) {
	_, err := query_static("<button>Hello</button>", "https://example.com", "button:hover")
	testing.expect(t, err!="")
	_, err = query_static("", "https://example.com", "")
	testing.expect(t, err!="")
	_, err = query_static("", "https://example.com", "a > span")
	testing.expect(t, err!="")
}
