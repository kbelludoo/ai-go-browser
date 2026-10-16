package browser

// reCAPTCHA v3 token minting through a stealth-controlled real browser.
// Tokens are scored by Google from the browser environment (webdriver flag,
// chrome runtime, pointer history, cookies) — minting inside the patched
// session is what keeps the score above the server's threshold. Tokens live
// ~2 minutes; the mint caches them and collapses concurrent requests into
// one browser launch.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// RecaptchaMint mints and caches reCAPTCHA v3 tokens for arena.ai.
type RecaptchaMint struct {
	BrowserPath string
	Headless    bool
	Action      string
	SiteKey     string // optional; discovered from the live page when empty
	Referrer    string // page that loads the reCAPTCHA script

	mu       sync.Mutex
	token    string
	expiry   time.Time
	inflight chan struct{}
}

func (m *RecaptchaMint) referrer() string {
	if m.Referrer != "" {
		return m.Referrer
	}
	return "https://arena.ai/"
}

// Token returns a cached token or mints a fresh one. Safe for concurrent use.
func (m *RecaptchaMint) Token(ctx context.Context) (string, error) {
	for {
		m.mu.Lock()
		if m.token != "" && time.Now().Before(m.expiry) {
			tok := m.token
			m.mu.Unlock()
			return tok, nil
		}
		if m.inflight != nil {
			wait := m.inflight
			m.mu.Unlock()
			select {
			case <-wait:
				continue
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		flight := make(chan struct{})
		m.inflight = flight
		m.mu.Unlock()

		tok, err := m.mintOnce(ctx)
		m.mu.Lock()
		if err == nil {
			m.token = tok
			// 90s: safe margin under the ~2min validity window.
			m.expiry = time.Now().Add(90 * time.Second)
		}
		m.inflight = nil
		m.mu.Unlock()
		close(flight)
		return tok, err
	}
}

func (m *RecaptchaMint) mintOnce(ctx context.Context) (string, error) {
	mctx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()

	s, err := LaunchStealth(mctx, StealthConfig{BrowserPath: m.BrowserPath, Headless: m.Headless})
	if err != nil {
		return "", err
	}
	defer s.Close()

	if err := s.Navigate(mctx, m.referrer()); err != nil {
		return "", fmt.Errorf("navigate: %w", err)
	}
	// Warm the session: recaptcha scripts load async and telemetry wants a
	// page that has actually been "used" for a moment.
	time.Sleep(1200 * time.Millisecond)
	s.MouseJitter(mctx, 3)

	expr := recaptchaExecJS(m.SiteKey, m.Action)
	raw, err := s.Evaluate(mctx, expr)
	if err != nil {
		return "", fmt.Errorf("evaluate: %w", err)
	}
	var token string
	if err := json.Unmarshal(raw, &token); err != nil {
		return "", fmt.Errorf("token was not a string: %s", string(raw))
	}
	if len(token) < 20 {
		return "", errors.New("recaptcha token suspiciously short")
	}
	return token, nil
}

// RecaptchaExecJS exposes the existing token expression to browser sessions.
func RecaptchaExecJS(siteKey, action string) string {
	return recaptchaExecJS(siteKey, action)
}

func recaptchaExecJS(siteKey, action string) string {
	if action == "" {
		action = "chat_submit"
	}
	return fmt.Sprintf(`(async () => {
  const deadline = Date.now() + 12000;
  while (!window.grecaptcha && Date.now() < deadline) {
    await new Promise((r) => setTimeout(r, 100));
  }
  if (!window.grecaptcha) throw new Error('grecaptcha script not loaded');
  const rc = window.grecaptcha.enterprise || window.grecaptcha;
  if (!rc || !rc.ready || !rc.execute) throw new Error('grecaptcha execute surface not loaded');
  let key = %s;
  const el = document.querySelector('[data-sitekey]');
  if (el && el.getAttribute('data-sitekey')) key = el.getAttribute('data-sitekey');
  const frame = document.querySelector('iframe[src*="recaptcha"]');
  if (frame) {
    try {
      const render = new URL(frame.src).searchParams.get('render');
      if (render && render !== 'explicit') key = render;
    } catch (e) {}
  }
  if (!key) throw new Error('recaptcha sitekey not found on page');
  await new Promise((resolve) => rc.ready(resolve));
  return await rc.execute(key, { action: %s });
})()`, strconv.Quote(siteKey), strconv.Quote(action))
}
