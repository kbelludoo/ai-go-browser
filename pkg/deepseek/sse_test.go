package deepseek

import (
	"errors"
	"strings"
	"testing"
)

func TestParseSSEStreamsResponseText(t *testing.T) {
	stream := strings.Join([]string{
		`data: {"v":{"response":{"message_id":123,"fragments":[{"type":"RESPONSE","content":"Hello"}]}}}`,
		``,
		`data: {"p":"response/fragments/-1/content","o":"APPEND","v":" world"}`,
		`data: {"v":"!"}`,
		`data: [DONE]`,
		``,
	}, "\n")

	var got strings.Builder
	res, err := parseSSE(strings.NewReader(stream), func(chunk string) error {
		got.WriteString(chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("parseSSE: %v", err)
	}
	if got.String() != "Hello world!" {
		t.Fatalf("text = %q, want %q", got.String(), "Hello world!")
	}
	if res.messageID == nil || *res.messageID != 123 {
		t.Fatalf("messageID = %v, want 123", res.messageID)
	}
}

func TestParseSSEErrorFrame(t *testing.T) {
	stream := "data: {\"type\":\"error\",\"content\":\"backend exploded\",\"finish_reason\":\"error\"}\n"
	_, err := parseSSE(strings.NewReader(stream), func(string) error { return nil })
	var be *BackendError
	if !errors.As(err, &be) {
		t.Fatalf("err = %v, want BackendError", err)
	}
	if be.Message != "backend exploded" {
		t.Fatalf("message = %q", be.Message)
	}
}

func TestParseSSEAuthErrorFrame(t *testing.T) {
	stream := "data: {\"type\":\"error\",\"content\":\"invalid token\"}\n"
	_, err := parseSSE(strings.NewReader(stream), func(string) error { return nil })
	var ae *AuthError
	if !errors.As(err, &ae) {
		t.Fatalf("err = %v, want AuthError", err)
	}
}

func TestConversationIDRoundTrip(t *testing.T) {
	if got := encodeCID("sess", nil); got != "sess" {
		t.Fatalf("encodeCID(nil) = %q", got)
	}
	msg := int64(77)
	cid := encodeCID("sess", &msg)
	if cid != "sess:77" {
		t.Fatalf("encodeCID = %q", cid)
	}
	session, parent := decodeCID(cid)
	if session != "sess" || parent == nil || *parent != 77 {
		t.Fatalf("decodeCID(%q) = %q, %v", cid, session, parent)
	}
	session, parent = decodeCID("sess2")
	if session != "sess2" || parent != nil {
		t.Fatalf("decodeCID(sess2) = %q, %v", session, parent)
	}
}
