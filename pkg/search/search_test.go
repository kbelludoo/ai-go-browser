package search

import (
	"context"
	"testing"
	"time"
)

func TestEngineSearch(t *testing.T) {
	eng := NewEngine()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := eng.Search(ctx, "Golang")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if resp.Query != "Golang" {
		t.Errorf("expected query 'Golang', got '%s'", resp.Query)
	}

	if len(resp.Results) == 0 {
		t.Logf("no results returned (network may be restricted), but no error occurred")
	} else {
		t.Logf("found %d results in %d ms", len(resp.Results), resp.DurationMs)
		for i, r := range resp.Results {
			if i < 3 {
				t.Logf("  [%s] %s -> %s", r.Source, r.Title, r.URL)
			}
		}
	}

	if resp.AISummary == "" {
		t.Errorf("expected non-empty AI summary")
	}
}
