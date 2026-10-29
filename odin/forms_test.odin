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
test_form_build_post_request :: proc(t: ^testing.T) {
	html := "<form action='/login' method='POST'><input name='user' value='alice'><input name='pass' type='password'><input type='hidden' name='csrf' value='secret123'></form>"
	p := parse_html(html, "https://example.com/login_page")
	testing.expect(t, len(p.forms) == 1)
	body, ok := form_post_body(p.forms[0], []string{"pass=mypassword"})
	testing.expect(t, ok)
	testing.expect(t, body == "user=alice&pass=mypassword&csrf=secret123")
	// Must fail if attempting to override a nonexistent field
	_, bad := form_post_body(p.forms[0], []string{"role=admin"})
	testing.expect(t, !bad)
}

@(test)
test_form_rejects_mismatched_method :: proc(t: ^testing.T) {
	f_post := Parsed_Form{action="https://example.com/action",method="post"}
	_, ok_get := form_get_url(f_post, nil)
	testing.expect(t, !ok_get)

	f_get := Parsed_Form{action="https://example.com/action",method="get"}
	_, ok_post := form_post_body(f_get, nil)
	testing.expect(t, !ok_post)
}
