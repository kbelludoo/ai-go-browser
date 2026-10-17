package resolver

import (
	"testing"

	"ai-go-browser/browser"
)

func TestSelfHealingResolver(t *testing.T) {
	elements := []browser.Element{
		{Ref: "#ref:1", Tag: "button", Text: "Enviar Mensagem", Label: "Send message"},
		{Ref: "#ref:2", Tag: "input", Label: "Digite seu e-mail", Name: "email"},
		{Ref: "#ref:3", Tag: "a", Text: "Saiba mais", Href: "/about"},
	}

	res := NewSelfHealingResolver()

	// 1. Exact ref match
	m1 := res.Resolve(elements, MatchTarget{Ref: "#ref:1"})
	if m1 == nil || m1.Element.Ref != "#ref:1" || m1.Healed {
		t.Fatalf("expected exact ref match, got %+v", m1)
	}

	// 2. Element shifted its ref (was #ref:1, agent requests with text "Enviar Mensagem" and old ref "#ref:99")
	m2 := res.Resolve(elements, MatchTarget{Ref: "#ref:99", Text: "Enviar Mensagem"})
	if m2 == nil || m2.Element.Ref != "#ref:1" || !m2.Healed {
		t.Fatalf("expected healed match by text, got %+v", m2)
	}
	if m2.Reason != "exact_text_match" {
		t.Fatalf("expected exact_text_match, got %s", m2.Reason)
	}

	// 3. Typo in text: "Enviar Mensgem" (missing 'a')
	m3 := res.Resolve(elements, MatchTarget{Ref: "#ref:99", Text: "Enviar Mensgem"})
	if m3 == nil || m3.Element.Ref != "#ref:1" || !m3.Healed {
		t.Fatalf("expected fuzzy match on typo, got %+v", m3)
	}
	if m3.Confidence < 0.85 {
		t.Fatalf("expected high confidence for 1-char typo, got %f", m3.Confidence)
	}

	// 4. Placeholder match
	m4 := res.Resolve(elements, MatchTarget{Placeholder: "Digite seu e-mail"})
	if m4 == nil || m4.Element.Ref != "#ref:2" {
		t.Fatalf("expected placeholder match, got %+v", m4)
	}
}
