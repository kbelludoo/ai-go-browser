package main

import "base:runtime"
import "core:c/libc"
import "core:strconv"
import "core:strings"

// Native libcurl HTTP/HTTPS transport, designed to replace Go's fetch layer
// incrementally. GET only; redirects are intentionally NOT followed because
// a redirect may change the IP / host. TLS peer and hostname checks remain on.
FETCH_MAX_BYTES :: 2*1024*1024

Fetch_Policy :: struct {
	allow_private: bool,
}

Fetch_Buffer :: struct {
	bytes: [dynamic]byte,
	exceeded: bool,
}

Fetch_Result :: struct {
	body: string,
	status: int,
	effective_url: string,
	error: string,
}

fetch_is_public_ipv4 :: proc(ip: string) -> bool {
	parts := strings.split(ip, ".")
	defer delete(parts)
	if len(parts) != 4 {return false}
	octets: [4]int
	for part, idx in parts {
		n, ok := strconv.parse_int(part, 10)
		if !ok || n < 0 || n > 255 || len(part) == 0 {return false}
		octets[idx] = n
	}
	a, b, c := octets[0], octets[1], octets[2]
	// Non-global IPv4; deny before *sending* HTTP data.
	if a == 0 || a == 10 || a == 127 || a >= 224 ||
		(a == 100 && b >= 64 && b <= 127) ||
		(a == 169 && b == 254) ||
		(a == 172 && b >= 16 && b <= 31) ||
		(a == 192 && b == 168) ||
		(a == 192 && b == 0) ||
		(a == 192 && b == 88 && c == 99) ||
		(a == 198 && (b == 18 || b == 19)) ||
		(a == 192 && b == 0 && c == 2) ||
		(a == 198 && b == 51 && c == 100) ||
		(a == 203 && b == 0 && c == 113) {
		return false
	}
	return true
}

fetch_prereq :: proc "c" (
	clientp: rawptr,
	remote: cstring,
	local_ip: cstring,
	remote_port: libc.int,
	local_port: libc.int,
) -> libc.int {
	context = runtime.default_context()
	policy := (^Fetch_Policy)(clientp)
	if policy == nil {return 1}
	if policy.allow_private {return 0}
	// IPv6 is rejected in this initial restricted mode until its public-range
	// classification is ported; this prevents private IPv6 / IPv4-mapped bypass.
	if fetch_is_public_ipv4(string(remote)) {
		return 0
	}
	return 1
}

fetch_write :: proc "c" (incoming: [^]byte, item_size: int, item_count: int, userdata: rawptr) -> int {
	context = runtime.default_context()
	buf := (^Fetch_Buffer)(userdata)
	if buf == nil || item_size <= 0 || item_count <= 0 {return 0}
	if item_count > FETCH_MAX_BYTES/item_size {
		buf.exceeded = true
		return 0
	}
	n := item_size*item_count
	if len(buf.bytes) > FETCH_MAX_BYTES-n {
		buf.exceeded = true
		return 0
	}
	before := len(buf.bytes)
	resize(&buf.bytes, before+n)
	copy(buf.bytes[before:], incoming[:n])
	return n
}

fetch_validate_url :: proc(target: string) -> bool {
	if !(strings.has_prefix(target, "https://") || strings.has_prefix(target, "http://")) {
		return false
	}
	origin := html_origin(target)
	scheme_i := strings.index(origin, "://")
	if scheme_i < 0 || len(origin) <= scheme_i+3 {return false}
	host := origin[scheme_i+3:]
	if strings.contains(host, "@") {return false}
	if strings.contains(target, "\n") || strings.contains(target, "\r") ||
		strings.contains(target, " ") || strings.contains(target, "\t") {
		return false
	}
	return true
}

// All callers now use the same cookie-capable, fail-closed transport policy.
// This helper preserves the former one-shot API while avoiding two diverging
// sets of curl options or separate SSRF implementations.
fetch_get :: proc(target: string, allow_private: bool = false) -> Fetch_Result {
	session: Fetch_Session
	if !fetch_session_init(&session, allow_private) {
		return Fetch_Result {
			effective_url=strings.clone(target),
			error="secure libcurl initialization failed",
		}
	}
	defer fetch_session_close(&session)
	return fetch_session_get(&session, target)
}
