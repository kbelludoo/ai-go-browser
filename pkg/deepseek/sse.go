package deepseek

// SSE parsing for DeepSeek's chat completion stream.
//
// The stream sends an initial snapshot frame whose "v" is the full response
// object (with fragments[].content), then a series of append frames:
//
//	{"p":"response/fragments/-1/content","o":"APPEND","v":" what"}  (sets path)
//	{"v":"'s"}                                                       (appends to it)
//
// We track the active append path and emit only RESPONSE-fragment text. Error
// frames ({"type":"error"}) are raised instead of skipped so a failed
// generation is never mistaken for an empty answer.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

type sseResult struct {
	messageID *int64
}

// isJSONObject reports whether raw is a JSON object (not null/string/number).
func isJSONObject(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && trimmed[0] == '{'
}

func jsonInt(raw json.RawMessage) (int64, bool) {
	var f float64
	if err := json.Unmarshal(raw, &f); err != nil {
		return 0, false
	}
	return int64(f), true
}

// parseSSE reads an SSE completion stream, invoking emit for every reply-text
// delta, and returns the assistant message id (when observed).
func parseSSE(r io.Reader, emit func(string) error) (*sseResult, error) {
	br := bufio.NewReaderSize(r, 1<<16)
	res := &sseResult{}
	var activePath string
	havePath := false
	emittedInitial := false

	handleLine := func(line string) error {
		if !strings.HasPrefix(line, "data:") {
			return nil
		}
		payload := strings.TrimSpace(line[len("data:"):])
		if payload == "" || payload == "[DONE]" {
			return nil
		}
		var obj map[string]json.RawMessage
		if json.Unmarshal([]byte(payload), &obj) != nil {
			return nil
		}

		// Error frame: the backend refused to generate. No "v" key, so it would
		// otherwise fall through every branch below and yield nothing.
		if t, ok := obj["type"]; ok {
			var typ string
			if json.Unmarshal(t, &typ) == nil && typ == "error" {
				var content, finish string
				_ = json.Unmarshal(obj["content"], &content)
				_ = json.Unmarshal(obj["finish_reason"], &finish)
				if content == "" {
					content = "the server reported an unspecified error"
				}
				if isAuthFailure(0, content+finish) {
					return &AuthError{Message: content}
				}
				return &BackendError{Message: content, FinishReason: finish}
			}
		}

		v := obj["v"]

		// Snapshot frame: full response object.
		if isJSONObject(v) {
			var snapshot map[string]json.RawMessage
			if json.Unmarshal(v, &snapshot) == nil {
				if _, hasResponse := snapshot["response"]; hasResponse {
					captureMessageID(res, snapshot)
					var response struct {
						Fragments []struct {
							Type    string `json:"type"`
							Content string `json:"content"`
						} `json:"fragments"`
					}
					var respObj map[string]json.RawMessage
					if json.Unmarshal(snapshot["response"], &respObj) == nil {
						_ = json.Unmarshal(respObj["fragments"], &response.Fragments)
					}
					for _, frag := range response.Fragments {
						if frag.Type == "RESPONSE" && frag.Content != "" {
							activePath = "response/fragments/-1/content"
							havePath = true
							if !emittedInitial {
								emittedInitial = true
								if err := emit(frag.Content); err != nil {
									return err
								}
							}
						}
					}
					return nil
				}
			}
		}

		// Path-setting append frame.
		if pRaw, ok := obj["p"]; ok {
			var p string
			if json.Unmarshal(pRaw, &p) == nil {
				activePath = p
				havePath = true
			}
			if havePath && strings.HasSuffix(activePath, "message_id") {
				if id, ok := jsonInt(v); ok {
					res.messageID = &id
				}
			}
			var op string
			_ = json.Unmarshal(obj["o"], &op)
			if op == "APPEND" {
				var s string
				if json.Unmarshal(v, &s) == nil && havePath && strings.HasSuffix(activePath, "content") {
					return emit(s)
				}
			}
			return nil
		}

		// Bare append to the current path.
		var s string
		if json.Unmarshal(v, &s) == nil && havePath && strings.HasSuffix(activePath, "content") {
			return emit(s)
		}
		return nil
	}

	for {
		line, err := br.ReadString('\n')
		if len(line) > 0 {
			if herr := handleLine(strings.TrimRight(line, "\r\n")); herr != nil {
				return res, herr
			}
		}
		if err != nil {
			if err == io.EOF {
				return res, nil
			}
			return res, err
		}
	}
}

// captureMessageID best-effort pulls the assistant message id out of a snapshot
// frame. DeepSeek nests the assistant message under "response"; we check there
// first, then the snapshot root, accepting message_id or id.
func captureMessageID(res *sseResult, snapshot map[string]json.RawMessage) {
	for _, container := range []map[string]json.RawMessage{decodeObject(snapshot["response"]), snapshot} {
		if container == nil {
			continue
		}
		for _, key := range []string{"message_id", "id"} {
			if raw, ok := container[key]; ok {
				if id, ok := jsonInt(raw); ok {
					res.messageID = &id
					return
				}
			}
		}
	}
}

func decodeObject(raw json.RawMessage) map[string]json.RawMessage {
	if !isJSONObject(raw) {
		return nil
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	return m
}
