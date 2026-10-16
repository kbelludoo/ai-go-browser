package arena

// In-browser arena session: login, token mint, and create-evaluation all
// execute inside ONE stealth-controlled real browser that holds the arena
// cookies. This mirrors the official frontend exactly — the reCAPTCHA
// Enterprise token is minted in the same session context that submits the
// evaluation, which is what the server's validation expects.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ai-go-browser/browser"
)

// BrowserSession owns a stealth browser logged into arena.ai.
type BrowserSession struct {
	b         *browser.StealthSession
	LoggedIn  bool
	UserEmail string
	v2token   string
}

// StartBrowserSession launches the stealth browser and loads arena.ai.
// lang should match the exit IP's language (default pt-BR) — locale/IP
// coherence feeds the Enterprise score.
func StartBrowserSession(ctx context.Context, browserPath string, headful bool, lang string) (*BrowserSession, error) {
	if lang == "" {
		lang = "pt-BR"
	}
	b, err := browser.LaunchStealth(ctx, browser.StealthConfig{BrowserPath: browserPath, Headless: !headful, Lang: lang, ProfileDir: stealthProfileDir()})
	if err != nil {
		return nil, err
	}
	s := &BrowserSession{b: b}
	if err := b.Navigate(ctx, "https://arena.ai/"); err != nil {
		b.Close()
		return nil, fmt.Errorf("navigate arena.ai: %w", err)
	}
	// Let the SPA hydrate so fetch/routes exist before we call the API.
	time.Sleep(1500 * time.Millisecond)
	return s, nil
}

// stealthProfileDir returns the persistent profile path. reCAPTCHA keeps a
// per-profile reputation: after one manual challenge pass, later automated
// checkbox clicks skip the image challenge. AIGOBROWSER_EPHEMERAL=1 opts out.
func stealthProfileDir() string {
	if os.Getenv("AIGOBROWSER_EPHEMERAL") != "" {
		return ""
	}
	if dir := os.Getenv("AIGOBROWSER_PROFILE"); dir != "" {
		return dir
	}
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		return ""
	}
	return filepath.Join(base, "ai-go-browser", "stealth-profile")
}

// Close tears down the browser.
func (s *BrowserSession) Close() {
	if s != nil && s.b != nil {
		s.b.Close()
	}
}

func (s *BrowserSession) evalJS(ctx context.Context, expr string) (string, error) {
	raw, err := s.b.Evaluate(ctx, expr)
	if err != nil {
		return "", err
	}
	var out string
	if err := json.Unmarshal(raw, &out); err != nil {
		snippet := string(raw)
		if len(snippet) > 300 {
			snippet = snippet[:300] + "..."
		}
		return "", fmt.Errorf("js result not a string: %s", snippet)
	}
	return out, nil
}

// Login authenticates inside the page via same-origin fetch, so cookies,
// TLS session, and client hints all stay in the browser.
func (s *BrowserSession) Login(ctx context.Context, email, password string) error {
	expr := fmt.Sprintf(`(async () => {
  const r = await fetch('/nextjs-api/sign-in/email', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ email: %s, password: %s, shouldLinkHistory: false })
  });
  const t = await r.text();
  return JSON.stringify({ status: r.status, body: t });
})()`, strconv.Quote(email), strconv.Quote(password))

	out, err := s.evalJS(ctx, expr)
	if err != nil {
		return err
	}
	var res struct {
		Status int    `json:"status"`
		Body   string `json:"body"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		return fmt.Errorf("login response: %w", err)
	}
	if res.Status != 200 {
		return fmt.Errorf("login HTTP %d: %s", res.Status, truncate(res.Body, 300))
	}
	var parsed struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		User    struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	if err := json.Unmarshal([]byte(res.Body), &parsed); err != nil {
		return fmt.Errorf("login body: %w (%s)", err, truncate(res.Body, 300))
	}
	if !parsed.Success {
		return fmt.Errorf("login rejected: %s", parsed.Error)
	}
	s.LoggedIn = true
	s.UserEmail = parsed.User.Email
	return nil
}

// SubmitEvaluation mints a fresh Enterprise v3 token and posts
// create-evaluation from inside the logged-in page. If the server rejects
// the v3 token, falls back to the interactive v2 checkbox — the same
// escalation the official frontend performs — and retries with the v2 token.
func (s *BrowserSession) SubmitEvaluation(ctx context.Context, req ChatRequest, siteKey, action string) (string, error) {
	if req.Mode == "" {
		req.Mode = "battle"
	}
	if req.Modality == "" {
		req.Modality = "chat"
	}
	payload := map[string]any{
		"id":               NewUUIDv7(),
		"mode":             req.Mode,
		"modality":         req.Modality,
		"userMessageId":    NewUUIDv7(),
		"modelAMessageId":  NewUUIDv7(),
		"modelBMessageId":  NewUUIDv7(),
		"userMessage":      map[string]any{"content": req.Prompt},
		"recaptchaV3Token": "", // filled in-page, keeps token and submit atomic
	}
	bodyJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// Behavioral warmup: pointer + typing telemetry before the Enterprise
	// mint. The score reacts to what the browser was doing in the seconds
	// before execute(), not just the environment flags.
	if os.Getenv("AIGOBROWSER_NO_WARMUP") == "" {
		s.b.MouseJitter(ctx, 5)
		s.b.KeyActivity(ctx, "arena battle")
		time.Sleep(1200 * time.Millisecond)
	}

	bodyText, err := s.submitV3(ctx, string(bodyJSON), siteKey, action)
	if err == nil {
		return bodyText, nil
	}
	if !IsRecaptchaRejected(err) {
		return "", err
	}

	// v3 score rejected — escalate exactly like the frontend: interactive
	// v2 checkbox, solved in this browser, retried as recaptchaV2Token.
	var v2err error
	s.v2token, v2err = s.SolveV2(ctx)
	if v2err != nil {
		return "", fmt.Errorf("v3 rejected, v2 fallback failed: %w", v2err)
	}
	return s.submitV2(ctx, string(bodyJSON))
}

func (s *BrowserSession) submitV3(ctx context.Context, bodyJSON, siteKey, action string) (string, error) {
	expr := fmt.Sprintf(`(async () => {
  const payload = %s;
  try {
    payload.recaptchaV3Token = await %s;
  } catch (e) {
    return JSON.stringify({ status: 0, tokenLen: 0, body: 'token_error: ' + ((e && e.message) || String(e)) });
  }
  const r = await fetch('/nextjs-api/stream/create-evaluation', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Accept': 'text/event-stream, application/json, */*' },
    credentials: 'include',
    body: JSON.stringify(payload)
  });
  const t = await r.text();
  return JSON.stringify({ status: r.status, tokenLen: payload.recaptchaV3Token.length, body: t });
})()`, bodyJSON, browser.RecaptchaExecJS(siteKey, action))
	return s.evalSubmit(ctx, expr)
}

func (s *BrowserSession) submitV2(ctx context.Context, bodyJSON string) (string, error) {
	expr := fmt.Sprintf(`(async () => {
  const payload = %s;
  delete payload.recaptchaV3Token;
  payload.recaptchaV2Token = %s;
  const r = await fetch('/nextjs-api/stream/create-evaluation', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Accept': 'text/event-stream, application/json, */*' },
    credentials: 'include',
    body: JSON.stringify(payload)
  });
  const t = await r.text();
  return JSON.stringify({ status: r.status, tokenLen: payload.recaptchaV2Token.length, body: t });
})()`, bodyJSON, strconv.Quote(s.v2token))
	return s.evalSubmit(ctx, expr)
}

func (s *BrowserSession) evalSubmit(ctx context.Context, expr string) (string, error) {
	out, err := s.evalJS(ctx, expr)
	if err != nil {
		return "", err
	}
	var res struct {
		Status   int    `json:"status"`
		TokenLen int    `json:"tokenLen"`
		Body     string `json:"body"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		return "", fmt.Errorf("submit response: %w", err)
	}
	if res.Status == 0 {
		return "", fmt.Errorf("in-page submit failed: %s", res.Body)
	}
	if res.Status >= 400 {
		return "", fmt.Errorf("API erro %d: %s", res.Status, truncate(res.Body, 500))
	}
	return res.Body, nil
}

// v2 sitekey: interactive checkbox key rendered via
// grecaptcha.enterprise.render (chunk_16). Distinct from the v3 score key.
const v2SiteKey = "6Le3_cYsAAAAAGwWOK2RLDgNI15Bh8C0yLBOL1yL"

// SolveV2 renders the interactive reCAPTCHA v2 widget inside the page,
// clicks the checkbox via CDP at real coordinates, and waits for the
// callback token. Returns an error if an image challenge appears (needs
// -headful + manual solve).
func (s *BrowserSession) SolveV2(ctx context.Context) (string, error) {
	renderExpr := fmt.Sprintf(`(async () => {
  window.__arena_v2_token = null;
  window.__arena_v2_error = false;
  const old = document.getElementById('__arena_c2');
  if (old) old.remove();
  const div = document.createElement('div');
  div.id = '__arena_c2';
  div.style.cssText = 'position:fixed;left:20px;top:20px;z-index:2147483647;width:304px;height:78px;background:#fff;';
  document.body.appendChild(div);
  const rc = window.grecaptcha.enterprise || window.grecaptcha;
  if (!rc || !rc.render) throw new Error('grecaptcha render surface missing');
  await new Promise((resolve) => rc.ready(resolve));
  rc.render(div, {
    sitekey: %s,
    theme: 'light',
    callback: (t) => { window.__arena_v2_token = t; },
    'error-callback': () => { window.__arena_v2_error = true; }
  });
  const deadline = Date.now() + 10000;
  while (Date.now() < deadline) {
    const f = document.querySelector('#__arena_c2 iframe[src*="anchor"]');
    if (f) {
      const r = f.getBoundingClientRect();
      return JSON.stringify({ x: r.x, y: r.y, w: r.width, h: r.height });
    }
    if (window.__arena_v2_error) return JSON.stringify({ error: 'widget error-callback fired' });
    await new Promise((r2) => setTimeout(r2, 100));
  }
  return JSON.stringify({ error: 'anchor iframe never appeared' });
})()`, strconv.Quote(v2SiteKey))

	out, err := s.evalJS(ctx, renderExpr)
	if err != nil {
		return "", err
	}
	var rect struct {
		X     float64 `json:"x"`
		Y     float64 `json:"y"`
		W     float64 `json:"w"`
		H     float64 `json:"h"`
		Error string  `json:"error"`
	}
	if err := json.Unmarshal([]byte(out), &rect); err != nil {
		return "", fmt.Errorf("v2 render: %w", err)
	}
	if rect.Error != "" {
		return "", fmt.Errorf("v2 render: %s", rect.Error)
	}

	// Let the widget paint before interacting — clicking a freshly mounted
	// iframe whose internal document is still loading gets swallowed.
	time.Sleep(900 * time.Millisecond)

	// Checkbox sits ~28px from the left and ~28px from the top of the
	// anchor iframe. Hover, jiggle, then click with real input events.
	cx := rect.X + 28
	cy := rect.Y + 28
	if os.Getenv("AIGOBROWSER_V2_DEBUG") != "" {
		if img, serr := s.b.Screenshot(ctx); serr == nil {
			_ = os.WriteFile("v2-preclick.png", img, 0o644)
		}
	}
	s.b.MouseJitter(ctx, 2)
	if err := s.b.ClickAt(ctx, cx-6, cy-4); err != nil {
		return "", fmt.Errorf("v2 click: %w", err)
	}
	time.Sleep(250 * time.Millisecond)
	if err := s.b.ClickAt(ctx, cx, cy); err != nil {
		return "", fmt.Errorf("v2 click: %w", err)
	}
	if os.Getenv("AIGOBROWSER_V2_DEBUG") != "" {
		time.Sleep(2 * time.Second)
		if img, serr := s.b.Screenshot(ctx); serr == nil {
			_ = os.WriteFile("v2-postclick.png", img, 0o644)
		}
	}

	pollOnce := func(ms int) (string, error) {
		expr := fmt.Sprintf(`(async () => {
  const deadline = Date.now() + %d;
  while (Date.now() < deadline) {
    if (window.__arena_v2_token) return window.__arena_v2_token;
    if (window.__arena_v2_error) return JSON.stringify({ error: 'widget error-callback' });
    const ch = document.querySelector('iframe[src*="bframe"]');
    if (ch) {
      const r = ch.getBoundingClientRect();
      // Hidden bframe is 0x0; a live challenge is a full-size window.
      if (r.width > 100 && r.height > 100) return JSON.stringify({ challenge: true });
    }
    await new Promise((r) => setTimeout(r, 250));
  }
  return JSON.stringify({ error: 'v2 token timeout' });
})()`, ms)
		return s.evalJS(ctx, expr)
	}

	out, err = pollOnce(4000)
	if err != nil {
		return "", err
	}
	var tok struct {
		Challenge bool   `json:"challenge"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal([]byte(out), &tok); err == nil && tok.Error == "v2 token timeout" {
		// First click may have been swallowed — try once more, then wait.
		if err := s.b.ClickAt(ctx, cx, cy); err != nil {
			return "", fmt.Errorf("v2 re-click: %w", err)
		}
		out, err = pollOnce(26000)
		if err != nil {
			return "", err
		}
	}
	if err := json.Unmarshal([]byte(out), &tok); err == nil && tok.Challenge {
		if os.Getenv("AIGOBROWSER_V2_MANUAL") != "" {
			fmt.Println(">> reCAPTCHA image challenge open in the browser window — solve it manually; waiting for the token...")
			out, err = pollOnce(150000)
			if err != nil {
				return "", err
			}
			if err := json.Unmarshal([]byte(out), &tok); err == nil && (tok.Challenge || tok.Error != "") {
				if tok.Challenge {
					return "", errors.New("v2 challenge still open after manual window — nothing solved")
				}
				return "", fmt.Errorf("v2: %s", tok.Error)
			}
		} else {
			return "", errors.New("v2 image challenge appeared — rerun with -headful AIGOBROWSER_V2_MANUAL=1 and solve it once manually (profile persists the pass)")
		}
	}
	if err := json.Unmarshal([]byte(out), &tok); err == nil && tok.Error != "" {
		return "", fmt.Errorf("v2: %s", tok.Error)
	}
	if len(out) < 20 {
		return "", fmt.Errorf("v2 token too short: %q", out)
	}
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// IsAuthError reports whether an error string indicates a dead session.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, needle := range []string{"401", "unauthorized", "user not found", "not authenticated", "invalid session"} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

// IsRecaptchaRejected reports whether the server refused the token.
func IsRecaptchaRejected(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "recaptcha validation failed")
}
