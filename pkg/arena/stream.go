package arena

// Tolerant SSE/stream parser for the create-evaluation endpoint.
// The exact event schema can shift between deploys, so extraction is
// heuristic: known text-bearing keys are pulled into typed deltas and
// everything else is surfaced as raw. Works with standard SSE framing,
// [DONE] terminators, and bare NDJSON fallback.

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// StreamEvent is one unit extracted from the evaluation stream.
// Kind: "delta" | "reasoning" | "finish" | "done" | "error" | "raw".
// Model: "a" | "b" | "" when attribution is unknown.
type StreamEvent struct {
	Kind  string
	Model string
	Text  string
	Raw   json.RawMessage
}

const streamBufferMax = 4 << 20

// ParseStream reads the response body and invokes fn for every event.
//
// The live create-evaluation wire format (recovered from the arena.ai
// frontend) is NOT SSE: it is the raw Vercel AI SDK data-stream protocol,
// newline-delimited, where each line is <model><code>:<json> — first
// character is the literal model prefix "a" or "b", second is a part code:
//
//	0 text | g reasoning | 3 error | d finish_message | e finish_step
//	f start_step | 2 data | 9 tool_call | a tool_result | c tool_call_delta
//
// Example: a0:"A capital é Par|ad:{"finishReason":"stop"}
//
// Legacy SSE/NDJSON shapes are still accepted as a fallback so schema
// drift in either direction keeps the bridge alive.
func ParseStream(ctx context.Context, r io.Reader, fn func(StreamEvent) error) error {
	if fn == nil {
		return errors.New("nil stream handler")
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), streamBufferMax)

	var dataLines []string
	flush := func() error {
		if len(dataLines) == 0 {
			return nil
		}
		payload := strings.Join(dataLines, "\n")
		dataLines = dataLines[:0]
		trimmed := strings.TrimSpace(payload)
		if trimmed == "" {
			return nil
		}
		if trimmed == "[DONE]" {
			return fn(StreamEvent{Kind: "done"})
		}
		return fn(classifyStreamData([]byte(trimmed)))
	}

	for sc.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := strings.TrimRight(sc.Text(), "\r")
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		// Primary protocol: raw AI SDK line, no SSE framing.
		if ev, ok := parseAIStreamLine(line); ok {
			if err := fn(ev); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			continue
		}
		if strings.HasPrefix(line, "event:") || strings.HasPrefix(line, "id:") || strings.HasPrefix(line, "retry:") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			if err := flush(); err != nil {
				return err
			}
			if err := fn(classifyStreamData([]byte(trimmed))); err != nil {
				return err
			}
		}
	}
	if err := flush(); err != nil {
		return err
	}
	return sc.Err()
}

// parseAIStreamLine decodes one raw AI SDK protocol line:
// <'a'|'b'><code>':'<json payload>.
func parseAIStreamLine(line string) (StreamEvent, bool) {
	if len(line) < 3 {
		return StreamEvent{}, false
	}
	model := line[0]
	if model != 'a' && model != 'b' {
		return StreamEvent{}, false
	}
	if line[2] != ':' {
		return StreamEvent{}, false
	}
	code := string(line[1])
	payload := line[3:]
	raw := append(json.RawMessage(nil), line...)
	m := string(model)

	switch code {
	case "0": // text delta
		var s string
		if err := json.Unmarshal([]byte(payload), &s); err != nil {
			return StreamEvent{Kind: "raw", Model: m, Raw: raw}, true
		}
		return StreamEvent{Kind: "delta", Model: m, Text: s, Raw: raw}, true
	case "g": // reasoning delta
		var s string
		if err := json.Unmarshal([]byte(payload), &s); err != nil {
			return StreamEvent{Kind: "raw", Model: m, Raw: raw}, true
		}
		return StreamEvent{Kind: "reasoning", Model: m, Text: s, Raw: raw}, true
	case "3": // error
		var s string
		if err := json.Unmarshal([]byte(payload), &s); err != nil {
			s = payload
		}
		return StreamEvent{Kind: "error", Model: m, Text: s, Raw: raw}, true
	case "d": // finish_message
		var fm struct {
			FinishReason string `json:"finishReason"`
		}
		_ = json.Unmarshal([]byte(payload), &fm)
		if fm.FinishReason == "error" {
			return StreamEvent{Kind: "error", Model: m, Text: "finishReason=error", Raw: raw}, true
		}
		return StreamEvent{Kind: "finish", Model: m, Raw: raw}, true
	case "e", "f", "2", "9", "a", "b", "c", "8", "h", "i", "j", "k":
		return StreamEvent{Kind: "raw", Model: m, Raw: raw}, true
	}
	return StreamEvent{}, false
}

func classifyStreamData(data []byte) StreamEvent {
	rawCopy := append(json.RawMessage(nil), data...)
	if len(data) == 0 {
		return StreamEvent{Kind: "raw", Raw: rawCopy}
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err == nil {
			return StreamEvent{Kind: "delta", Text: s, Raw: rawCopy}
		}
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return StreamEvent{Kind: "raw", Raw: rawCopy}
	}
	if e, ok := m["error"]; ok {
		switch v := e.(type) {
		case string:
			if v != "" {
				return StreamEvent{Kind: "error", Text: v, Raw: rawCopy}
			}
		case map[string]any:
			if msg, ok := v["message"].(string); ok && msg != "" {
				return StreamEvent{Kind: "error", Text: msg, Raw: rawCopy}
			}
		}
	}
	if done, ok := m["done"].(bool); ok && done {
		return StreamEvent{Kind: "done", Raw: rawCopy}
	}
	model := normalizeModel(m)
	if text, ok := digText(m); ok {
		if text == "" {
			return StreamEvent{Kind: "raw", Raw: rawCopy}
		}
		return StreamEvent{Kind: "delta", Model: model, Text: text, Raw: rawCopy}
	}
	return StreamEvent{Kind: "raw", Raw: rawCopy}
}

func normalizeModel(m map[string]any) string {
	for _, key := range []string{"model", "modelId", "model_id", "arena", "winner", "side", "role"} {
		if v, ok := m[key]; ok {
			if s := modelLabel(v); s != "" {
				return s
			}
		}
	}
	if v, ok := m["index"]; ok {
		if f, isNum := v.(float64); isNum {
			if f == 0 {
				return "a"
			}
			if f == 1 {
				return "b"
			}
		}
	}
	return ""
}

func modelLabel(v any) string {
	s, ok := v.(string)
	if !ok {
		if f, isNum := v.(float64); isNum {
			if f == 0 {
				return "a"
			}
			if f == 1 {
				return "b"
			}
		}
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "a", "model_a", "modela", "left", "0":
		return "a"
	case "b", "model_b", "modelb", "right", "1":
		return "b"
	}
	return ""
}

func digText(m map[string]any) (string, bool) {
	for _, key := range []string{"delta", "content", "text", "response", "output", "token", "message"} {
		v, ok := m[key]
		if !ok {
			continue
		}
		if s, ok := extractString(v); ok {
			return s, true
		}
	}
	return "", false
}

func extractString(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case map[string]any:
		for _, key := range []string{"text", "content", "value", "delta"} {
			if inner, ok := t[key]; ok {
				if s, ok := extractString(inner); ok {
					return s, true
				}
			}
		}
		return "", false
	case []any:
		var sb strings.Builder
		for _, part := range t {
			if s, ok := extractString(part); ok {
				sb.WriteString(s)
			}
		}
		return sb.String(), sb.Len() > 0
	}
	return "", false
}

// ErrStreamServer surfaces a server-side error event as a Go error.
func ErrStreamServer(text string) error {
	return fmt.Errorf("arena stream error: %s", text)
}
