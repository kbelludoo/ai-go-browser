package main

import "core:testing"

@(test)
test_session_redirect_urls :: proc(t: ^testing.T) {
	testing.expect(t, fetch_is_redirect(301))
	testing.expect(t, fetch_is_redirect(308))
	testing.expect(t, !fetch_is_redirect(200))
	testing.expect(t, !fetch_is_redirect(304))
	testing.expect(t, fetch_redirect_url("https://example.com/a?old=1", "?new=2") ==
		"https://example.com/a?new=2")
	testing.expect(t, fetch_redirect_url("https://example.com/a/b", "/target") ==
		"https://example.com/target")
	testing.expect(t, fetch_redirect_url("https://example.com/a/b", "next") ==
		"https://example.com/a/next")
	testing.expect(t, !fetch_validate_url(fetch_redirect_url("https://example.com", "file:///etc/passwd")))
	testing.expect(t, !fetch_validate_url(fetch_redirect_url("https://example.com", "https://user:pass@127.0.0.1")))
}
