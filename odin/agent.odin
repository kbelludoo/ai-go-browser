package main

import "base:runtime"
import "core:bufio"
import "core:encoding/json"
import "core:fmt"
import "core:os"
import "core:strconv"
import "core:strings"
import vmem "core:mem/virtual"

// Newline-delimited JSON interface for long-running agents. Commands are
// processed sequentially in a single process. Cookies live in memory, and the
// current parsed page is recreated from the latest bounded HTML snapshot.
//
// Input (one JSON object per line):
//   {"cmd":"open","url":"https://example.com"}
//   {"cmd":"follow","ref":"l1"} (or {"cmd":"click","ref":"l1"})
//   {"cmd":"submit","ref":"f1","fields":["q=test"]}  (GET and POST supported)
//   {"cmd":"query","selector":"main a"}
//   {"cmd":"back"}
//   {"cmd":"snapshot"}
//   {"cmd":"quit"}
// Responses are *one JSON object* per line; no banners or incidental logs.

Agent_Command :: struct {
	cmd: string,
	url: string,
	ref: string,
	fields: []string,
	selector: string,
}

Agent_Reply :: struct {
	ok: bool,
	error: string,
	page: Browser_Output,
}

Agent_Query_Reply :: struct {
	ok: bool,
	error: string,
	matches: []Query_Element,
}

agent_emit_query :: proc(matches: []Query_Element, error_message: string) {
	reply := Agent_Query_Reply{ok=error_message=="",error=error_message,matches=matches}
	data,err := json.marshal(reply)
	if err != nil {
		fmt.println("{\"ok\":false,\"error\":\"query serialization failed\"}")
		return
	}
	fmt.println(string(data))
}

agent_free_saved :: proc(saved: ^Fetch_Result) {
	heap := runtime.default_context().allocator
	if saved.body != "" {delete(saved.body, heap)}
	if saved.effective_url != "" {delete(saved.effective_url, heap)}
	saved^ = {}
}

agent_emit :: proc(ok: bool, message: string, saved: ^Fetch_Result) {
	reply := Agent_Reply{ok=ok, error=message}
	if ok && saved != nil && saved.effective_url != "" {
		p := parse_html(saved.body, saved.effective_url)
		reply.page = page_output(p, saved.status)
	}
	data, err := json.marshal(reply)
	if err != nil {
		fmt.println("{\"ok\":false,\"error\":\"marshal failed\"}")
		return
	}
	fmt.println(string(data))
}

agent_parse_ref :: proc(value: string, prefix: byte) -> int {
	if len(value) < 2 || value[0] != prefix {return 0}
	n, ok := strconv.parse_int(value[1:], 10)
	if !ok || n < 1 {return 0}
	return n
}

agent_execute_line :: proc(line: string, session: ^Fetch_Session, saved: ^Fetch_Result, history: ^[dynamic]string) -> bool {
	arena: vmem.Arena
	if vmem.arena_init_growing(&arena) != nil {
		fmt.println("{\"ok\":false,\"error\":\"arena unavailable\"}")
		return true
	}
	defer vmem.arena_destroy(&arena)
	context.allocator = vmem.arena_allocator(&arena)
	cmd: Agent_Command
	if json.unmarshal_string(line, &cmd) != nil {
		agent_emit(false, "invalid JSON", saved)
		return true
	}
	if cmd.cmd == "quit" {
		agent_emit(true, "", nil)
		return false
	}
	if cmd.cmd == "query" {
		if saved.effective_url == "" {
			agent_emit_query(nil,"no page opened")
			return true
		}
		matches, query_err := query_static(saved.body,saved.effective_url,cmd.selector)
		agent_emit_query(matches,query_err)
		return true
	}
	if cmd.cmd == "back" {
		if len(history^) == 0 {
			agent_emit(false, "no history", saved)
			return true
		}
		prev_url := pop(history)
		defer delete(prev_url, runtime.default_context().allocator)
		arena_allocator := context.allocator
		context.allocator = runtime.default_context().allocator
		next := fetch_session_get(session, prev_url)
		context.allocator = arena_allocator
		if next.error != "" {
			agent_emit(false, next.error, saved)
			agent_free_saved(&next)
			return true
		}
		agent_free_saved(saved)
		saved^ = next
		agent_emit(true, "", saved)
		return true
	}
	if cmd.cmd == "snapshot" {
		if saved.effective_url == "" {
			agent_emit(false, "no page opened", saved)
		} else {
			agent_emit(true, "", saved)
		}
		return true
	}
	target := ""
	post_body := ""
	is_post := false
	switch cmd.cmd {
	case "open":
		target = cmd.url
	case "follow", "click":
		if saved.effective_url == "" {
			agent_emit(false, "no page opened", saved)
			return true
		}
		index := agent_parse_ref(cmd.ref, 'l')
		if index == 0 {
			agent_emit(false, "invalid link ref", saved)
			return true
		}
		p := parse_html(saved.body, saved.effective_url)
		if index > len(p.links) {
			agent_emit(false, "link not found", saved)
			return true
		}
		target = p.links[index-1].url
	case "submit":
		if saved.effective_url == "" {
			agent_emit(false, "no page opened", saved)
			return true
		}
		index := agent_parse_ref(cmd.ref, 'f')
		if index == 0 {
			agent_emit(false, "invalid form ref", saved)
			return true
		}
		p := parse_html(saved.body, saved.effective_url)
		if index > len(p.forms) {
			agent_emit(false, "form not found", saved)
			return true
		}
		form := p.forms[index-1]
		if form.method == "post" {
			body, ok := form_post_body(form, cmd.fields)
			if !ok {
				agent_emit(false, "form POST validation failed", saved)
				return true
			}
			target = form.action
			post_body = body
			is_post = true
		} else {
			encoded, ok := form_get_url(form, cmd.fields)
			if !ok {
				agent_emit(false, "form is not valid, or field not found", saved)
				return true
			}
			target = encoded
		}
	case:
		agent_emit(false, "unknown command", saved)
		return true
	}
	if !fetch_validate_url(target) {
		agent_emit(false, "invalid URL", saved)
		return true
	}
	// Keep session HTML and URLs on the default heap across JSONL turns.
	// All command parsing, snapshot extraction and JSON output use the arena.
	arena_allocator := context.allocator
	context.allocator = runtime.default_context().allocator
	next: Fetch_Result
	if is_post {
		next = fetch_session_post(session, target, post_body)
	} else {
		next = fetch_session_get(session, target)
	}
	context.allocator = arena_allocator
	if next.error != "" {
		agent_emit(false, next.error, saved)
		agent_free_saved(&next)
		return true
	}
	if saved.effective_url != "" && saved.effective_url != next.effective_url {
		if len(history^) >= 32 {
			oldest := history^[0]
			delete(oldest, runtime.default_context().allocator)
			ordered_remove(history, 0)
		}
		url_copy := strings.clone(saved.effective_url, runtime.default_context().allocator)
		append(history, url_copy)
	}
	agent_free_saved(saved)
	saved^ = next
	agent_emit(true, "", saved)
	return true
}

agent_run :: proc(allow_private: bool) {
	session: Fetch_Session
	if !fetch_session_init(&session, allow_private) {
		fmt.println("{\"ok\":false,\"error\":\"session unavailable\"}")
		return
	}
	defer fetch_session_close(&session)
	current: Fetch_Result
	defer agent_free_saved(&current)
	history := make([dynamic]string, runtime.default_context().allocator)
	defer {
		for h in history {
			delete(h, runtime.default_context().allocator)
		}
		delete(history)
	}

	scanner: bufio.Scanner
	bufio.scanner_init(&scanner, os.to_reader(os.stdin))
	defer bufio.scanner_destroy(&scanner)
	for bufio.scan(&scanner) {
		line := strings.trim_space(bufio.scanner_text(&scanner))
		if line == "" {continue}
		if !agent_execute_line(line, &session, &current, &history) {
			break
		}
	}
}
