package main

import "base:runtime"
import "core:c/libc"
import "core:strconv"
import "core:strings"
import "vendor:curl"

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
		(a == 192 && b == 0 && c == 0) ||
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
	n := item_size*item_count
	if n > FETCH_MAX_BYTES || len(buf.bytes) > FETCH_MAX_BYTES-n {
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

fetch_get :: proc(target: string, allow_private: bool = false) -> Fetch_Result {
	out := Fetch_Result{effective_url=strings.clone(target)}
	if !fetch_validate_url(target) {
		out.error = "URL must be http/https without credentials, whitespace or controls"
		return out
	}
	curl.global_init(curl.GLOBAL_ALL)
	handle := curl.easy_init()
	if handle == nil {
		out.error = "libcurl initialization failed"
		return out
	}
	defer curl.easy_cleanup(handle)
	received := Fetch_Buffer{}
	defer delete(received.bytes)
	policy := Fetch_Policy{allow_private=allow_private}

	url_c := strings.clone_to_cstring(target)
	defer delete(url_c)
	// libcurl copies CURLOPT_URL internally.
	curl.easy_setopt(handle, .URL, url_c)
	curl.easy_setopt(handle, .USERAGENT, cstring("ai-odin-browser/0.1 (semantic browser)"))
	curl.easy_setopt(handle, .ACCEPT_ENCODING, cstring(""))
	curl.easy_setopt(handle, .PROTOCOLS_STR, cstring("http,https"))
	curl.easy_setopt(handle, .REDIR_PROTOCOLS_STR, cstring("http,https"))
	curl.easy_setopt(handle, .FOLLOWLOCATION, libc.long(0))
	curl.easy_setopt(handle, .CONNECTTIMEOUT_MS, libc.long(4000))
	curl.easy_setopt(handle, .TIMEOUT_MS, libc.long(12000))
	curl.easy_setopt(handle, .NOSIGNAL, libc.long(1))
	curl.easy_setopt(handle, .SSL_VERIFYPEER, libc.long(1))
	curl.easy_setopt(handle, .SSL_VERIFYHOST, libc.long(2))
	curl.easy_setopt(handle, .NOPROXY, cstring("*"))
	curl.easy_setopt(handle, .PROXY, cstring(""))
	curl.easy_setopt(handle, .WRITEFUNCTION, fetch_write)
	curl.easy_setopt(handle, .WRITEDATA, &received)
	curl.easy_setopt(handle, .PREREQFUNCTION, fetch_prereq)
	curl.easy_setopt(handle, .PREREQDATA, &policy)
	if !allow_private {
		// Restricted mode currently classifies IPv4 only; avoid IPv6
		// handshake failure on dual-stack hosts, and deny private IPv4.
		curl.easy_setopt(handle, .IPRESOLVE, libc.long(curl.IPRESOLVE_V4))
	}

	err := curl.easy_perform(handle)
	if received.exceeded {
		out.error = "response exceeds 2 MiB limit"
		return out
	}
	if err != .E_OK {
		out.error = string(curl.easy_strerror(err))
		return out
	}
	status: libc.long
	curl.easy_getinfo(handle, .RESPONSE_CODE, &status)
	out.status = int(status)
	effective: cstring
	if curl.easy_getinfo(handle, .EFFECTIVE_URL, &effective) == .E_OK && effective != nil {
		delete(out.effective_url)
		out.effective_url = strings.clone(string(effective))
	}
	out.body = strings.clone(string(received.bytes[:]))
	return out
}
