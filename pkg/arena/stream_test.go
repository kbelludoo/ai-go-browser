package arena

import (
	"context"
	"strings"
	"testing"
)

func TestParseStreamAISDKProtocol(t *testing.T) {
	// Exact wire format recovered from arena.ai frontend (chunk_42):
	// raw lines, <model><code>:<json>, no SSE framing.
	payload := strings.Join([]string{
		`a0:"A capital da Fran"`,
		`b0:"A capital é Par"`,
		`a0:"ça é Paris."`,
		`ag:"pensando..."`,
		`b0:"is."`,
		`ad:{"finishReason":"stop"}`,
		`bd:{"finishReason":"stop","usage":{"promptTokens":10,"completionTokens":20}}`,
	}, "\n") + "\n"

	var a, b, reasoning strings.Builder
	var finishes []string
	err := ParseStream(context.Background(), strings.NewReader(payload), func(ev StreamEvent) error {
		switch ev.Kind {
		case "delta":
			if ev.Model == "a" {
				a.WriteString(ev.Text)
			} else if ev.Model == "b" {
				b.WriteString(ev.Text)
			}
		case "reasoning":
			reasoning.WriteString(ev.Text)
		case "finish":
			finishes = append(finishes, ev.Model)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if a.String() != "A capital da França é Paris." {
		t.Fatalf("model a: %q", a.String())
	}
	if b.String() != "A capital é Paris." {
		t.Fatalf("model b: %q", b.String())
	}
	if reasoning.String() != "pensando..." {
		t.Fatalf("reasoning: %q", reasoning.String())
	}
	if len(finishes) != 2 || finishes[0] != "a" || finishes[1] != "b" {
		t.Fatalf("finishes: %+v", finishes)
	}
}

func TestParseStreamAIProtocolErrorLine(t *testing.T) {
	payload := "a0:\"ok\"\na3:\"recaptcha validation failed\"\n"
	var kinds []string
	var errMsg string
	err := ParseStream(context.Background(), strings.NewReader(payload), func(ev StreamEvent) error {
		kinds = append(kinds, ev.Kind)
		if ev.Kind == "error" {
			errMsg = ev.Text
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(kinds) != 2 || kinds[1] != "error" || errMsg != "recaptcha validation failed" {
		t.Fatalf("kinds=%+v msg=%q", kinds, errMsg)
	}
}

func TestParseStreamSSEAttributed(t *testing.T) {
	payload := strings.Join([]string{
		`data: {"model":"A","delta":"Hello"}`,
		``,
		`data: {"model":"B","delta":"Olá"}`,
		``,
		`data: {"modelId":"model_a","content":" world"}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	var got []StreamEvent
	err := ParseStream(context.Background(), strings.NewReader(payload), func(ev StreamEvent) error {
		got = append(got, ev)
		return nil
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("expected 4 events, got %d: %+v", len(got), got)
	}
	if got[0].Kind != "delta" || got[0].Model != "a" || got[0].Text != "Hello" {
		t.Fatalf("event0: %+v", got[0])
	}
	if got[1].Model != "b" || got[1].Text != "Olá" {
		t.Fatalf("event1: %+v", got[1])
	}
	if got[2].Model != "a" || got[2].Text != " world" {
		t.Fatalf("event2: %+v", got[2])
	}
	if got[3].Kind != "done" {
		t.Fatalf("event3: %+v", got[3])
	}
}

func TestParseStreamNestedMessageAndIndex(t *testing.T) {
	payload := strings.Join([]string{
		`data: {"index":1,"message":{"content":[{"type":"text","text":"capital is "},{"type":"text","text":"Paris"}]}}`,
		``,
		`data: {"index":0,"message":{"content":"A é Paris."}}`,
		``,
	}, "\n")

	var texts []string
	var models []string
	err := ParseStream(context.Background(), strings.NewReader(payload), func(ev StreamEvent) error {
		if ev.Kind == "delta" {
			texts = append(texts, ev.Text)
			models = append(models, ev.Model)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(texts) != 2 || texts[0] != "capital is Paris" || models[0] != "b" {
		t.Fatalf("nested: %+v %+v", texts, models)
	}
	if models[1] != "a" {
		t.Fatalf("index attribution: %+v", models)
	}
}

func TestParseStreamNDJSONFallback(t *testing.T) {
	payload := `{"delta":"x","model":"a"}
{"error":"recaptcha validation failed"}
`
	var kinds []string
	err := ParseStream(context.Background(), strings.NewReader(payload), func(ev StreamEvent) error {
		kinds = append(kinds, ev.Kind)
		return nil
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(kinds) != 2 || kinds[0] != "delta" || kinds[1] != "error" {
		t.Fatalf("kinds: %+v", kinds)
	}
}

func TestParseStreamMultilineData(t *testing.T) {
	payload := "data: {\"delta\":\"line1\",\ndata: \"model\":\"a\"}\n\n"
	var evs []StreamEvent
	err := ParseStream(context.Background(), strings.NewReader(payload), func(ev StreamEvent) error {
		if ev.Kind == "delta" {
			evs = append(evs, ev)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(evs) != 1 || evs[0].Text != "line1" || evs[0].Model != "a" {
		t.Fatalf("multiline: %+v", evs)
	}
}
