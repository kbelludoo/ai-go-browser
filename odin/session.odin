package main

import "base:runtime"
import "core:c/libc"
import "core:strings"
import "vendor:curl"

// One in-process libcurl cookie jar per CLI invocation. It is never saved to
// disk and never shared across distinct invocations. All HTTP requests are GET.
//
// Redirects are followed manually so fetch_prereq can validate the resolved
// address on *each* new connection. No automatic redirects or proxies.
FETCH_REDIRECT_LIMIT :: 5
FETCH_LOCATION_LIMIT :: 2048

Fetch_Session :: struct {
	handle: ^curl.CURL,
	policy: Fetch_Policy,
}

Fetch_Headers :: struct {
	location: [dynamic]byte,
}

fetch_headers_callback :: proc "c" (
	incoming: [^]byte, item_size, count: int, userdata: rawptr,
) -> int {
	context = runtime.default_context()
	h := (^Fetch_Headers)(userdata)
	if h == nil || item_size <= 0 || count <= 0 {return 0}
	if item_size > 8192/count {return 0}
	n := item_size*count
	if n > 8192 {return 0} // Reject overlong headers rather than risk overflow.
	line := incoming[:n]
	// Clear any previous Location when libcurl starts a new response,
	// e.g. after an informational 100 Continue response.
	if n >= 5 && line[0] == 'H' && line[1] == 'T' && line[2] == 'T' &&
		line[3] == 'P' && line[4] == '/' {
		clear(&h.location)
		return n
	}
	prefix := "location:"
	if n < len(prefix) {return n}
	for i in 0..<len(prefix) {
		b := line[i]
		if b >= 'A' && b <= 'Z' {b += 32}
		if b != prefix[i] {return n}
	}
	first := len(prefix)
	for first < n && (line[first] == ' ' || line[first] == '\t') {first += 1}
	last := n
	for last > first && (line[last-1] == '\r' || line[last-1] == '\n' ||
		line[last-1] == ' ' || line[last-1] == '\t') {
		last -= 1
	}
	if last-first > FETCH_LOCATION_LIMIT || last <= first {return n}
	clear(&h.location)
	append(&h.location, ..line[first:last])
	return n
}

fetch_session_init :: proc(s: ^Fetch_Session, allow_private: bool = false) -> bool {
	if curl.global_init(curl.GLOBAL_ALL) != .E_OK {return false}
	s.handle = curl.easy_init()
	if s.handle == nil {return false}
	s.policy.allow_private = allow_private
	h := s.handle
	ok := true
	// Reject older/unsupported libcurl builds where any critical policy
	// option is ignored; otherwise SSRF/proxy/TLS checks could fail open.
	if curl.easy_setopt(h, .USERAGENT, cstring("ai-odin-browser/0.2 (semantic browser)")) != .E_OK {ok = false}
	if curl.easy_setopt(h, .ACCEPT_ENCODING, cstring("")) != .E_OK {ok = false}
	if curl.easy_setopt(h, .PROTOCOLS_STR, cstring("http,https")) != .E_OK {ok = false}
	if curl.easy_setopt(h, .REDIR_PROTOCOLS_STR, cstring("http,https")) != .E_OK {ok = false}
	if curl.easy_setopt(h, .FOLLOWLOCATION, libc.long(0)) != .E_OK {ok = false}
	if curl.easy_setopt(h, .CONNECTTIMEOUT_MS, libc.long(4000)) != .E_OK {ok = false}
	if curl.easy_setopt(h, .TIMEOUT_MS, libc.long(12000)) != .E_OK {ok = false}
	if curl.easy_setopt(h, .NOSIGNAL, libc.long(1)) != .E_OK {ok = false}
	if curl.easy_setopt(h, .SSL_VERIFYPEER, libc.long(1)) != .E_OK {ok = false}
	if curl.easy_setopt(h, .SSL_VERIFYHOST, libc.long(2)) != .E_OK {ok = false}
	if curl.easy_setopt(h, .NOPROXY, cstring("*")) != .E_OK {ok = false}
	if curl.easy_setopt(h, .PROXY, cstring("")) != .E_OK {ok = false}
	if curl.easy_setopt(h, .HTTPGET, libc.long(1)) != .E_OK {ok = false}
	if curl.easy_setopt(h, .WRITEFUNCTION, fetch_write) != .E_OK {ok = false}
	if curl.easy_setopt(h, .HEADERFUNCTION, fetch_headers_callback) != .E_OK {ok = false}
	if curl.easy_setopt(h, .PREREQFUNCTION, fetch_prereq) != .E_OK {ok = false}
	if curl.easy_setopt(h, .PREREQDATA, &s.policy) != .E_OK {ok = false}
	if curl.easy_setopt(h, .COOKIEFILE, cstring("")) != .E_OK {ok = false}
	if !allow_private {
		if curl.easy_setopt(h, .IPRESOLVE, libc.long(curl.IPRESOLVE_V4)) != .E_OK {
			ok = false
		}
	}
	if !ok {
		curl.easy_cleanup(h)
		s.handle = nil
		return false
	}
	return true
}

fetch_session_close :: proc(s: ^Fetch_Session) {
	if s.handle != nil {
		curl.easy_cleanup(s.handle)
		s.handle = nil
	}
}

// Always free both .body and .effective_url of each returned Fetch_Result.
fetch_session_once :: proc(s: ^Fetch_Session, target: string) -> (Fetch_Result, string) {
	out := Fetch_Result{effective_url=strings.clone(target)}
	if s.handle == nil {
		out.error = "session not initialized"
		return out, ""
	}
	if !fetch_validate_url(target) {
		out.error = "rejected invalid or unsupported URL"
		return out, ""
	}
	received := Fetch_Buffer{}
	defer delete(received.bytes)
	headers := Fetch_Headers{}
	defer delete(headers.location)

	url_c := strings.clone_to_cstring(target)
	defer delete(url_c)
	h := s.handle
	if curl.easy_setopt(h, .URL, url_c) != .E_OK ||
		curl.easy_setopt(h, .HTTPGET, libc.long(1)) != .E_OK ||
		curl.easy_setopt(h, .WRITEDATA, &received) != .E_OK ||
		curl.easy_setopt(h, .HEADERDATA, &headers) != .E_OK {
		out.error = "libcurl failed to apply request policy"
		return out, ""
	}
	// NOTE: each callback data pointer is valid only for this synchronous call.
	err := curl.easy_perform(h)
	if received.exceeded {
		out.error = "response exceeds 2 MiB limit"
		return out, ""
	}
	if err != .E_OK {
		out.error = string(curl.easy_strerror(err))
		return out, ""
	}
	status: libc.long
	curl.easy_getinfo(h, .RESPONSE_CODE, &status)
	out.status = int(status)
	effective: cstring
	if curl.easy_getinfo(h, .EFFECTIVE_URL, &effective) == .E_OK && effective != nil {
		delete(out.effective_url)
		out.effective_url = strings.clone(string(effective))
	}
	out.body = strings.clone(string(received.bytes[:]))
	return out, strings.clone(string(headers.location[:]))
}

fetch_is_redirect :: proc(status: int) -> bool {
	switch status {
	case 301, 302, 303, 307, 308:
		return true
	}
	return false
}

// Resolve only HTTP(S) URLs; the final result must still pass
// fetch_validate_url, and libcurl verifies each connected IPv4 address.
// Location values containing fragments alone are not navigations.
fetch_redirect_url :: proc(base, location: string) -> string {
	loc := strings.trim_space(location)
	if loc == "" || strings.has_prefix(loc, "#") {
		return ""
	}
	if strings.has_prefix(loc, "?") {
		prefix := base
		if i := strings.index(prefix, "#"); i >= 0 {prefix = prefix[:i]}
		if i := strings.index(prefix, "?"); i >= 0 {prefix = prefix[:i]}
		return strings.concatenate([]string{prefix, loc})
	}
	// html_resolve returns a borrowed slice for an absolute URL, but owns
	// its result for relative URLs. Return an owned copy for every branch.
	// Otherwise freeing the Location header invalidates the next URL and
	// can cause a use-after-free / double-free during redirect validation.
	if strings.has_prefix(loc, "http://") || strings.has_prefix(loc, "https://") {
		return strings.clone(loc)
	}
	return html_resolve(base, loc)
}

fetch_session_get :: proc(s: ^Fetch_Session, target: string) -> Fetch_Result {
	current := strings.clone(target)
	defer delete(current)
	if !fetch_validate_url(current) {
		return Fetch_Result {error="invalid initial URL", effective_url=strings.clone(current)}
	}
	for hop in 0..=FETCH_REDIRECT_LIMIT {
		result, location := fetch_session_once(s, current)
		if result.error != "" {
			delete(location)
			return result
		}
		if !fetch_is_redirect(result.status) || location == "" {
			delete(location)
			return result
		}
		if hop >= FETCH_REDIRECT_LIMIT {
			delete(location)
			delete(result.body)
			result.body = ""
			result.error = "redirect limit exceeded (5 hops)"
			return result
		}
		next := fetch_redirect_url(result.effective_url, location)
		delete(location)
		if !fetch_validate_url(next) {
			if next != "" {delete(next)}
			delete(result.body)
			result.body = ""
			result.error = "redirect rejected: unsupported or unsafe URL"
			return result
		}
		delete(result.body)
		delete(result.effective_url)
		delete(current)
		current = next
	}
	return Fetch_Result {error="redirect loop"}
}
