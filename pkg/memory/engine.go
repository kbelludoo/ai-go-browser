package memory

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Engine is the central memory manager coordinating playbooks, FTS5 page cache, and handoffs.
type Engine struct {
	mu      sync.RWMutex
	storage *Storage
	cache   map[string]*DomainPlaybook // in-memory hot cache for playbooks
}

// NewEngine creates or opens the Browser Memory engine.
func NewEngine(dbPath string) (*Engine, error) {
	storage, err := OpenStorage(dbPath)
	if err != nil {
		return nil, err
	}

	eng := &Engine{
		storage: storage,
		cache:   make(map[string]*DomainPlaybook),
	}

	// Seed universal defaults for common domains if not present
	eng.seedUniversalDefaults()

	return eng, nil
}

// Close gracefully flushes and closes storage.
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.storage.Close()
}

func (e *Engine) seedUniversalDefaults() {
	defaults := []DomainPlaybook{
		{
			Domain: "github.com",
			FastPaths: map[string]string{
				"search": "https://github.com/search?q={query}",
			},
			KnownSelectors: map[string]string{
				"search_input": "input.header-search-input, input[name='q'], [data-target='qbsearch-input.inputButtonText']",
				"sign_in":      "a[href*='/login']",
			},
			DismissModals: []string{
				".js-cookie-consent-banner button",
			},
		},
		{
			Domain: "google.com",
			FastPaths: map[string]string{
				"search": "https://www.google.com/search?q={query}",
			},
			KnownSelectors: map[string]string{
				"search_input": "textarea[name='q'], input[name='q']",
				"submit":       "input[name='btnK']",
			},
			DismissModals: []string{
				"#L2AGLb", "button[aria-label*='Concordo']", "button[aria-label*='Aceitar']",
			},
		},
		{
			Domain: "pkg.go.dev",
			FastPaths: map[string]string{
				"search": "https://pkg.go.dev/search?q={query}",
			},
			KnownSelectors: map[string]string{
				"search_input": "input[name='q']",
			},
		},
		{
			Domain: "wikipedia.org",
			FastPaths: map[string]string{
				"search": "https://en.wikipedia.org/wiki/Special:Search?search={query}",
			},
			KnownSelectors: map[string]string{
				"search_input": "input[name='search']",
			},
		},
	}

	for _, d := range defaults {
		existing, _ := e.storage.GetPlaybook(d.Domain)
		if existing == nil {
			_ = e.storage.SavePlaybook(&d)
			e.cache[d.Domain] = &d
		}
	}
}

// ExtractDomain extracts clean hostname (e.g. "github.com" from "https://github.com/torvalds/linux").
func ExtractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	h := strings.ToLower(u.Hostname())
	h = strings.TrimPrefix(h, "www.")
	return h
}

// RecordVisit stores a visited page into the FTS5 index and updates the domain last visited time.
func (e *Engine) RecordVisit(pageURL, title, markdownContent string, status int) error {
	domain := ExtractDomain(pageURL)
	rec := PageRecord{
		URL:       pageURL,
		Title:     title,
		Domain:    domain,
		Content:   markdownContent,
		VisitedAt: time.Now(),
		Status:    status,
	}

	if err := e.storage.IndexPage(rec); err != nil {
		return err
	}

	if domain != "" {
		pb := e.GetOrCreatePlaybook(domain)
		pb.LastVisited = time.Now()
		_ = e.storage.SavePlaybook(pb)
	}

	return nil
}

// SearchVisitedPages performs BM25 search over all visited web content.
func (e *Engine) SearchVisitedPages(query string, limit int) ([]PageRecord, error) {
	return e.storage.SearchPages(query, limit)
}

// GetOrCreatePlaybook returns cached or stored playbook for a domain.
func (e *Engine) GetOrCreatePlaybook(domain string) *DomainPlaybook {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.getOrCreatePlaybookLocked(domain)
}

func (e *Engine) getOrCreatePlaybookLocked(domain string) *DomainPlaybook {
	domain = strings.TrimPrefix(strings.ToLower(domain), "www.")
	if pb, ok := e.cache[domain]; ok {
		return pb
	}

	pb, err := e.storage.GetPlaybook(domain)
	if err != nil || pb == nil {
		pb = &DomainPlaybook{
			Domain:         domain,
			FastPaths:      make(map[string]string),
			KnownSelectors: make(map[string]string),
			DismissModals:  make([]string, 0),
			LastVisited:    time.Now(),
		}
	}

	e.cache[domain] = pb
	return pb
}

// ResolveFastPath checks if a domain has a learned direct URL for an intent (e.g., "search").
// If found, formats the URL with query parameter, bypassing multiple homepage steps.
func (e *Engine) ResolveFastPath(domain, intent, query string) (string, bool) {
	pb := e.GetOrCreatePlaybook(domain)
	if pb == nil || pb.FastPaths == nil {
		return "", false
	}

	template, ok := pb.FastPaths[intent]
	if !ok || template == "" {
		return "", false
	}

	escapedQuery := url.QueryEscape(query)
	targetURL := strings.ReplaceAll(template, "{query}", escapedQuery)
	return targetURL, true
}

// GetLearnedSelector returns a proven selector for an action on a domain.
func (e *Engine) GetLearnedSelector(domain, action string) (string, bool) {
	pb := e.GetOrCreatePlaybook(domain)
	if pb == nil || pb.KnownSelectors == nil {
		return "", false
	}

	selector, ok := pb.KnownSelectors[action]
	return selector, ok && selector != ""
}

// RecordFeedback updates the playbook with success/failure feedback for self-healing.
func (e *Engine) RecordFeedback(domain, action, selector string, success bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	pb := e.getOrCreatePlaybookLocked(domain)
	if success {
		pb.SuccessCount++
		if pb.KnownSelectors == nil {
			pb.KnownSelectors = make(map[string]string)
		}
		pb.KnownSelectors[action] = selector
	} else {
		pb.FailureCount++
		// If the failed selector is the current known one, clear it so fallback heals
		if pb.KnownSelectors != nil && pb.KnownSelectors[action] == selector {
			delete(pb.KnownSelectors, action)
		}
	}
	_ = e.storage.SavePlaybook(pb)
}

// GetDismissModals returns all known modal/cookie dismiss selectors for a domain.
func (e *Engine) GetDismissModals(domain string) []string {
	pb := e.GetOrCreatePlaybook(domain)
	if pb == nil {
		return nil
	}
	return pb.DismissModals
}

// RegisterDismissModal registers a new modal dismiss selector for a domain.
func (e *Engine) RegisterDismissModal(domain, selector string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	pb := e.getOrCreatePlaybookLocked(domain)
	for _, s := range pb.DismissModals {
		if s == selector {
			return
		}
	}
	pb.DismissModals = append(pb.DismissModals, selector)
	_ = e.storage.SavePlaybook(pb)
}

// ExportHandoff creates a cross-agent handoff packet.
func (e *Engine) ExportHandoff(currentURL, title, goal string, steps []string, cookies map[string]string) (*HandoffSession, error) {
	idBytes := make([]byte, 8)
	_, _ = rand.Read(idBytes)
	id := fmt.Sprintf("b-handoff-%s", hex.EncodeToString(idBytes))

	session := &HandoffSession{
		ID:         id,
		URL:        currentURL,
		Title:      title,
		Goal:       goal,
		StepsTaken: steps,
		Cookies:    cookies,
		SavedAt:    time.Now(),
	}

	if err := e.storage.SaveHandoff(session); err != nil {
		return nil, err
	}
	return session, nil
}

// ResumeHandoff loads a saved handoff session.
func (e *Engine) ResumeHandoff(id string) (*HandoffSession, error) {
	return e.storage.GetHandoff(id)
}
