package main

import "core:testing"

@(test)
test_fetch_ip_policy :: proc(t: ^testing.T) {
	bad_ips := []string{"127.0.0.1", "10.1.2.3", "172.16.0.1", "192.168.10.1",
		"169.254.169.254", "100.64.1.1", "0.0.0.0", "8.8.8.8.x",
		"::1", "::ffff:127.0.0.1", "203.0.113.10",
		"192.0.0.9", "192.88.99.1", "198.18.1.1", "255.255.255.255"}
	for ip in bad_ips {
		testing.expectf(t, !fetch_is_public_ipv4(ip), "must reject %q", ip)
	}
	testing.expect(t, fetch_is_public_ipv4("8.8.8.8"))
	testing.expect(t, fetch_is_public_ipv4("1.1.1.1"))
}

@(test)
test_fetch_url_validation :: proc(t: ^testing.T) {
	testing.expect(t, fetch_validate_url("https://example.com/path?q=hi"))
	testing.expect(t, !fetch_validate_url("file:///etc/passwd"))
	testing.expect(t, !fetch_validate_url("https://user:pass@example.com"))
	testing.expect(t, !fetch_validate_url("https://"))
	testing.expect(t, !fetch_validate_url("http://example.com/\nBad"))
}
