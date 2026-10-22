package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"ai-go-browser/browser"
	"ai-go-browser/pkg/resolver"
)

// generateTestDOM creates a realistic DOM tree with 500 elements (links, buttons, inputs, spans, divs).
func generateTestDOM() []browser.Element {
	elements := make([]browser.Element, 500)
	for i := 0; i < 500; i++ {
		tag := "div"
		role := "generic"
		switch i % 5 {
		case 0:
			tag = "a"
			role = "link"
		case 1:
			tag = "button"
			role = "button"
		case 2:
			tag = "input"
			role = "textbox"
		case 3:
			tag = "span"
		}

		elements[i] = browser.Element{
			Ref:   fmt.Sprintf("#ref:%d", i),
			Tag:   tag,
			ID:    fmt.Sprintf("el_node_%d", i),
			Role:  role,
			Text:  fmt.Sprintf("Navigation Item %d - Overview and Details", i),
			Label: fmt.Sprintf("Label for item %d", i),
			Name:  fmt.Sprintf("field_%d", i),
		}
	}

	// Insert realistic target at index 420 (deep in the DOM)
	elements[420] = browser.Element{
		Ref:   "#ref:420",
		Tag:   "input",
		ID:    "global_search_input",
		Role:  "textbox",
		Text:  "Search or jump to...",
		Label: "Search GitHub",
		Name:  "q",
	}

	return elements
}

// BenchmarkColdFuzzyResolution measures element resolution without memory (pure heuristic + fuzzy Levenshtein).
func BenchmarkColdFuzzyResolution(b *testing.B) {
	elements := generateTestDOM()
	r := resolver.NewSelfHealingResolver()

	target := resolver.MatchTarget{
		Domain: "github.com",
		Action: "search_input",
		Text:   "Search or jump",
		Role:   "textbox",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		match := r.Resolve(elements, target)
		if match == nil {
			b.Fatal("cold match failed")
		}
	}
}

// BenchmarkWarmMemoryResolution measures element resolution WITH learned memory fast-path.
func BenchmarkWarmMemoryResolution(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "bench_mem_*")
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "bench_memory.db")
	eng, err := NewEngine(dbPath)
	if err != nil {
		b.Fatalf("failed to create engine: %v", err)
	}
	defer eng.Close()

	// Seed the learned selector for github.com search_input
	eng.RecordFeedback("github.com", "search_input", "global_search_input", true)

	elements := generateTestDOM()
	r := resolver.NewSelfHealingResolver().WithMemory(eng)

	target := resolver.MatchTarget{
		Domain: "github.com",
		Action: "search_input",
		Text:   "Search or jump",
		Role:   "textbox",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		match := r.Resolve(elements, target)
		if match == nil || match.Reason != "learned_memory_fastpath" {
			b.Fatalf("warm match failed or reason not learned: %+v", match)
		}
	}
}

// BenchmarkFTS5PageSearch measures BM25 full-text search speed across indexed web pages.
func BenchmarkFTS5PageSearch(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "bench_fts_*")
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "bench_fts.db")
	eng, err := NewEngine(dbPath)
	if err != nil {
		b.Fatalf("failed to create engine: %v", err)
	}
	defer eng.Close()

	// Populate 100 pages of simulated web content
	for i := 0; i < 100; i++ {
		content := fmt.Sprintf("Page %d contains detailed technical specifications about Go concurrency, channels, mutexes, memory compaction, and garbage collection metrics.", i)
		_ = eng.RecordVisit(fmt.Sprintf("https://golang.org/doc/page_%d", i), fmt.Sprintf("Go Doc %d", i), content, 200)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := eng.SearchVisitedPages("concurrency compaction channels", 5)
		if err != nil || len(results) == 0 {
			b.Fatal("FTS5 search returned 0 results or error")
		}
	}
}

// BenchmarkFastPathRouteResolution measures instant O(1) direct URL resolution.
func BenchmarkFastPathRouteResolution(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "bench_fp_*")
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "bench_fp.db")
	eng, err := NewEngine(dbPath)
	if err != nil {
		b.Fatalf("failed to create engine: %v", err)
	}
	defer eng.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		targetURL, ok := eng.ResolveFastPath("github.com", "search", "kubernetes operator")
		if !ok || targetURL == "" {
			b.Fatal("fastpath failed")
		}
	}
}
