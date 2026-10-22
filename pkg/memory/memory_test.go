package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserMemoryEngine(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "browser_mem_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test_memory.db")
	eng, err := NewEngine(dbPath)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer eng.Close()

	// 1. Test Seed Defaults
	targetURL, found := eng.ResolveFastPath("github.com", "search", "golang agent")
	if !found {
		t.Errorf("expected fast-path for github.com search")
	}
	expected := "https://github.com/search?q=golang+agent"
	if targetURL != expected {
		t.Errorf("expected %q, got %q", expected, targetURL)
	}

	// 2. Test Record Visit & FTS5 Full-Text Search
	doc1 := "Golang is an open-source programming language created by Google for high concurrency and microservices."
	if err := eng.RecordVisit("https://go.dev/about", "About Go", doc1, 200); err != nil {
		t.Fatalf("failed to record visit: %v", err)
	}

	doc2 := "Python is a high-level interpreted programming language often used in data science."
	if err := eng.RecordVisit("https://python.org/about", "About Python", doc2, 200); err != nil {
		t.Fatalf("failed to record visit: %v", err)
	}

	hits, err := eng.SearchVisitedPages("concurrency microservices", 5)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(hits) == 0 {
		t.Fatalf("expected search hits for 'concurrency microservices', got 0")
	}
	if hits[0].URL != "https://go.dev/about" {
		t.Errorf("expected top result to be go.dev, got %s", hits[0].URL)
	}

	// 3. Test Learned Selector Feedback
	domain := "custom-portal.org"
	sel, ok := eng.GetLearnedSelector(domain, "search_input")
	if ok {
		t.Errorf("expected no learned selector initially")
	}

	// Record success
	eng.RecordFeedback(domain, "search_input", "#main-search-input", true)
	sel, ok = eng.GetLearnedSelector(domain, "search_input")
	if !ok || sel != "#main-search-input" {
		t.Errorf("expected learned selector '#main-search-input', got %q", sel)
	}

	// Record failure (self-healing invalidation)
	eng.RecordFeedback(domain, "search_input", "#main-search-input", false)
	_, ok = eng.GetLearnedSelector(domain, "search_input")
	if ok {
		t.Errorf("expected selector to be invalidated after failure")
	}

	// 4. Test Cross-Agent Handoff
	handoff, err := eng.ExportHandoff("https://pkg.go.dev/net/http", "Package http", "Find ServeMux examples", []string{"navigated", "scrolled"}, map[string]string{"session_id": "xyz123"})
	if err != nil {
		t.Fatalf("failed to export handoff: %v", err)
	}
	if handoff.ID == "" {
		t.Fatalf("expected valid handoff ID")
	}

	resumed, err := eng.ResumeHandoff(handoff.ID)
	if err != nil {
		t.Fatalf("failed to resume handoff: %v", err)
	}
	if resumed.URL != "https://pkg.go.dev/net/http" || resumed.Goal != "Find ServeMux examples" {
		t.Errorf("resumed handoff data mismatch: %+v", resumed)
	}
}
