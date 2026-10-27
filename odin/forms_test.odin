package main

import "core:testing"
import "core:strings"

@(test)
test_form_build_get_request :: proc(t: ^testing.T) {
	html := "<form action='/search' method='GET'><input name='q'><input type='hidden' name='csrf' value='token &amp; key'><input type='checkbox' name='promo'><input disabled name='internal' value='bad'><button>Search</button></form>"
	p := parse_html(html, "https://example.com/start")
	testing.expect(t, len(p.forms) == 1)
	testing.expect(t, len(p.forms[0].fields) == 4)
	target, ok := form_get_url(p.forms[0], []string{"q=Odin Browser"})
	testing.expect(t, ok)
	testing.expectf(t, target == "https://example.com/search?q=Odin%20Browser&csrf=token%20%26%20key",
		"GET URL %q", target)
	_, ok = form_get_url(p.forms[0], []string{"admin=yes"})
	testing.expect(t, !ok)
}

@(test)
test_form_rejects_post_and_invalid_fields :: proc(t: ^testing.T) {
	f := Parsed_Form{action="https://example.com/action",method="post"}
	_, ok := form_get_url(f, nil)
	testing.expect(t, !ok)
	p := parse_html("<form action='/search'><input name='q'></form>", "https://example.com/page")
	url, good := form_get_url(p.forms[0], nil)
	testing.expect(t, good)
	testing.expect(t, strings.contains(url, "https://example.com/search?q="))
}
