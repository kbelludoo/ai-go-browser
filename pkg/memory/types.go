package memory

import (
	"time"
)

// DomainPlaybook stores learned knowledge, successful selectors, routes, and connection profiles for a web domain.
type DomainPlaybook struct {
	Domain          string            `json:"domain"`
	FastPaths       map[string]string `json:"fast_paths"`       // e.g. "search": "https://domain.com/search?q={query}"
	KnownSelectors  map[string]string `json:"known_selectors"`  // e.g. "search_input": "input[name='q']", "submit": "button[type='submit']"
	DismissModals   []string          `json:"dismiss_modals"`   // Known cookie/popup dismiss selectors (e.g. "#accept-cookies")
	RequiresWARP    bool              `json:"requires_warp"`    // Domain blocks datacenter IPs, requires Cloudflare WARP tunnel
	InsecureTLS     bool              `json:"insecure_tls"`     // Domain uses broken x509 chain (ICP-Brasil/gov.br) needing TLS bypass
	LastStatusCode  int               `json:"last_status_code"` // Last observed HTTP response status code
	LastDiagnosis   string            `json:"last_diagnosis"`   // Self-healing diagnostic reason if failed
	SuccessCount    int               `json:"success_count"`
	FailureCount    int               `json:"failure_count"`
	LastVisited     time.Time         `json:"last_visited"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// PageRecord represents an indexed web page with full text for instant retrieval.
type PageRecord struct {
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Domain    string    `json:"domain"`
	Snippet   string    `json:"snippet,omitempty"`
	Content   string    `json:"content,omitempty"`
	Rank      float64   `json:"rank,omitempty"`
	VisitedAt time.Time `json:"visited_at"`
	Status    int       `json:"status"`
}

// ActionFeedback represents real-time execution feedback for self-healing playbooks.
type ActionFeedback struct {
	Domain    string `json:"domain"`
	Action    string `json:"action"` // e.g. "search_input", "submit_button", "cookie_consent"
	Selector  string `json:"selector"`
	Success   bool   `json:"success"`
	LatencyMs int64  `json:"latency_ms"`
}

// HandoffSession allows transferring live navigation context between agents (Antigravity <-> Agent-Go).
type HandoffSession struct {
	ID          string            `json:"id"`
	URL         string            `json:"url"`
	Title       string            `json:"title"`
	Goal        string            `json:"goal"`
	StepsTaken  []string          `json:"steps_taken"`
	Cookies     map[string]string `json:"cookies,omitempty"`
	SavedAt     time.Time         `json:"saved_at"`
}
