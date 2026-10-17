package search

import (
	"context"
	"testing"
	"time"
)

func TestEngineSearch(t *testing.T) {
	eng := NewEngine()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := eng.Search(ctx, "Golang")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if resp.Query != "Golang" {
		t.Errorf("expected query 'Golang', got '%s'", resp.Query)
	}

	t.Logf("AI Summary:\n%s", resp.AISummary)

	if len(resp.Results) == 0 {
		t.Logf("no results returned (network may be restricted), but no error occurred")
	} else {
		t.Logf("found %d results in %d ms", len(resp.Results), resp.DurationMs)
	}

	if resp.AISummary == "" {
		t.Errorf("expected non-empty AI summary")
	}
}
