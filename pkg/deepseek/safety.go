package deepseek

// Account-safety controls for the DeepSeek Web client.
//
// These controls intentionally fail closed. They avoid request bursts and
// repeated authentication attempts; they do not bypass provider limits,
// CAPTCHAs, WAFs, or account controls.

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// SafetyTrip is raised when the client must stop instead of retrying
// automatically (a CAPTCHA/WAF/rate-limit signal was observed).
type SafetyTrip struct{ Reason string }

func (e *SafetyTrip) Error() string {
	if e.Reason == "" {
		return "DeepSeek safety guard is active."
	}
	return e.Reason
}

// RequestGuard paces requests and latches a circuit breaker on account-
// protection responses.
type RequestGuard struct {
	mu          sync.Mutex
	minInterval time.Duration
	lastRequest time.Time
	tripped     bool
	reason      string
}

// NewRequestGuard builds a guard that enforces at least minInterval between
// requests. Intervals below one second are clamped, matching the reference
// fail-closed policy.
func NewRequestGuard(minInterval time.Duration) *RequestGuard {
	if minInterval < time.Second {
		minInterval = time.Second
	}
	return &RequestGuard{minInterval: minInterval}
}

// Wait blocks until the minimum interval has elapsed, or fails when tripped.
func (g *RequestGuard) Wait() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.tripped {
		if g.reason == "" {
			return &SafetyTrip{Reason: "DeepSeek safety guard is active."}
		}
		return &SafetyTrip{Reason: g.reason}
	}
	if delay := g.minInterval - time.Since(g.lastRequest); delay > 0 {
		time.Sleep(delay)
	}
	g.lastRequest = time.Now()
	return nil
}

// Trip latches the circuit breaker with a human-readable reason.
func (g *RequestGuard) Trip(reason string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.tripped = true
	g.reason = reason
}

// Reset clears the breaker.
func (g *RequestGuard) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.tripped = false
	g.reason = ""
	g.lastRequest = time.Now()
}

// IsAccountProtectionResponse reports whether an HTTP status/body indicates
// account protection (CAPTCHA, WAF, rate limiting) rather than a plain error.
func IsAccountProtectionResponse(status int, body string) bool {
	if status == 401 || status == 403 || status == 429 {
		return true
	}
	text := strings.ToLower(body)
	for _, m := range []string{
		"captcha", "recaptcha", "verify you are human", "human verification",
		"too many requests", "rate limit", "forbidden", "account suspended",
		"account disabled", "risk control", "security verification",
	} {
		if strings.Contains(text, m) {
			return true
		}
	}
	return false
}

func safetyStop(status int, phase string) *SafetyTrip {
	return &SafetyTrip{Reason: fmt.Sprintf(
		"DeepSeek returned HTTP %d %s; Cassandra stopped to avoid repeated account-security requests.",
		status, phase)}
}
