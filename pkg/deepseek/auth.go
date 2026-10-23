package deepseek

// Authentication: capture a signed-in DeepSeek Web session (bearer token from
// localStorage.userToken + cookies) from a real browser, either attached over
// CDP or launched by us. The browser is used only to establish the session; it
// is not used to chat. The captured session then drives the pure-HTTP client.
//
// Cassandra never solves or bypasses CAPTCHA / human-verification; the user
// completes those by hand in a visible browser.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"ai-go-browser/browser"
)

const (
	chatURL   = "https://chat.deepseek.com/"
	signInURL = "https://chat.deepseek.com/sign_in"

	// SessionMaxAge is how long a captured session is trusted before a refresh
	// is attempted (24h), mirroring the reference client.
	SessionMaxAge = 24 * time.Hour
)

// Session is a captured signed-in DeepSeek session.
type Session struct {
	Token      string            `json:"token"`
	Cookies    map[string]string `json:"cookies"`
	UserAgent  string            `json:"user_agent"`
	CapturedAt float64           `json:"captured_at"`
}

// Age reports how long ago the session was captured.
func (s *Session) Age() time.Duration {
	return time.Duration(time.Since(time.Unix(int64(s.CapturedAt), 0)))
}

func (s *Session) cookieHeader() string {
	if len(s.Cookies) == 0 {
		return ""
	}
	keys := make([]string, 0, len(s.Cookies))
	for k := range s.Cookies {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+s.Cookies[k])
	}
	return strings.Join(parts, "; ")
}

// DefaultSessionFile resolves the session cache path (env DEEPSEEK_SESSION_FILE,
// otherwise ./session/session.json).
func DefaultSessionFile() string {
	if v := strings.TrimSpace(os.Getenv("DEEPSEEK_SESSION_FILE")); v != "" {
		return v
	}
	return filepath.Join("session", "session.json")
}

// DefaultProfileDir resolves the persistent browser profile path.
func DefaultProfileDir() string {
	if v := strings.TrimSpace(os.Getenv("DEEPSEEK_PROFILE_DIR")); v != "" {
		return v
	}
	return filepath.Join("session", "profile")
}

// Save writes the session as JSON.
func (s *Session) Save(path string) error {
	if path == "" {
		path = DefaultSessionFile()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

// LoadSession reads a session JSON file. A missing file returns (nil, nil).
func LoadSession(path string) (*Session, error) {
	if path == "" {
		path = DefaultSessionFile()
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// readTokenJS reads the bearer token the web app stores after login. Shape:
//
//	localStorage.userToken = {"value":"<TOKEN>","__version":"0"}
const readTokenJS = `(() => { try {
  const raw = window.localStorage.getItem('userToken');
  if (!raw) return null;
  const o = JSON.parse(raw);
  return (o && o.value) ? o.value : null;
} catch (e) { return null; } })()`

// LoginOptions configures Login.
type LoginOptions struct {
	// CDPURL attaches to an already-running Chrome started with
	// --remote-debugging-port (e.g. http://127.0.0.1:9222). Required for Google
	// sign-in and CAPTCHA loops.
	CDPURL string
	// ProfileDir is the persistent profile used when launching our own browser.
	ProfileDir string
	// BrowserPath overrides the browser executable.
	BrowserPath string
	// Headless launches without a window (only useful with a pre-signed profile).
	Headless bool
	// Timeout bounds how long we wait for the user to sign in.
	Timeout time.Duration
	// SessionFile is where the captured session is written.
	SessionFile string
}

// Login captures a session by waiting for the user to sign in (solving any
// human-check by hand) in a visible browser.
func Login(ctx context.Context, opts LoginOptions) (*Session, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 300 * time.Second
	}
	if opts.SessionFile == "" {
		opts.SessionFile = DefaultSessionFile()
	}

	sess, owned, err := openBrowser(ctx, opts)
	if err != nil {
		return nil, err
	}
	if owned {
		defer sess.Close()
	}

	token, err := captureToken(ctx, sess, opts)
	if err != nil {
		return nil, err
	}

	captured, err := captureSession(ctx, sess, token)
	if err != nil {
		return nil, err
	}
	return finishSession(ctx, captured, opts.SessionFile)
}

// RenewOptions configures Renew.
type RenewOptions struct {
	// ProfileDir is a persistent browser profile that is (or was) signed in.
	ProfileDir string
	// BrowserPath overrides the browser executable.
	BrowserPath string
	// CDPURL, when set, is preferred: renew from an already-running browser.
	CDPURL string
	// SessionFile is where the refreshed session is written.
	SessionFile string
	// Timeout bounds how long we wait for the app to read the token.
	Timeout time.Duration
}

// Renew re-captures the session from a persistent browser profile (headless)
// or from a running CDP browser, without showing a window. It only reuses an
// existing sign-in: if the profile is logged out it fails and asks for a manual
// `login`. It never solves or bypasses CAPTCHA.
func Renew(ctx context.Context, opts RenewOptions) (*Session, error) {
	if opts.SessionFile == "" {
		opts.SessionFile = DefaultSessionFile()
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 20 * time.Second
	}

	// A fresh cached session needs no refresh.
	if cached, err := LoadSession(opts.SessionFile); err == nil && cached != nil && cached.Age() < SessionMaxAge {
		return cached, nil
	}

	// Prefer an already-running browser when we know its debug endpoint.
	if url := strings.TrimSpace(firstNonEmpty(opts.CDPURL, os.Getenv("DEEPSEEK_CDP_URL"))); url != "" {
		if port, perr := cdpPort(url); perr == nil {
			if sess, aerr := browser.AttachTab(ctx, port, "chat.deepseek.com"); aerr == nil {
				if captured, cerr := grabSession(ctx, sess, opts.Timeout); cerr == nil {
					return finishSession(ctx, captured, opts.SessionFile)
				}
			}
		}
	}

	profileDir := opts.ProfileDir
	if profileDir == "" {
		profileDir = DefaultProfileDir()
	}
	sess, err := browser.LaunchStealth(ctx, browser.StealthConfig{
		BrowserPath: opts.BrowserPath,
		Headless:    true,
		ProfileDir:  profileDir,
	})
	if err != nil {
		return nil, fmt.Errorf("open headless profile %q: %w", profileDir, err)
	}
	defer sess.Close()

	captured, err := grabSession(ctx, sess, opts.Timeout)
	if err != nil {
		return nil, fmt.Errorf("%w (run `deepseek login` to sign in again)", err)
	}
	return finishSession(ctx, captured, opts.SessionFile)
}

// grabSession reads an existing sign-in from a browser, waiting briefly for a
// slow SPA. It never interacts with a login form.
func grabSession(ctx context.Context, sess *browser.StealthSession, timeout time.Duration) (*Session, error) {
	if !onDeepSeek(ctx, sess) {
		_ = sess.Navigate(ctx, chatURL)
	}
	token := evalToken(ctx, sess)
	deadline := time.Now().Add(timeout)
	for token == "" && time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
		token = evalToken(ctx, sess)
	}
	if token == "" {
		return nil, errors.New("no DeepSeek token in this browser profile (logged out)")
	}
	return captureSession(ctx, sess, token)
}

// finishSession validates, saves and returns a captured session.
func finishSession(ctx context.Context, s *Session, file string) (*Session, error) {
	if err := ProbeSession(ctx, s); err != nil {
		return nil, err
	}
	if err := s.Save(file); err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}
	return s, nil
}

// openBrowser attaches over CDP when a URL is given, otherwise launches a
// browser with a persistent profile. owned reports whether we must Close it.
func openBrowser(ctx context.Context, opts LoginOptions) (sess *browser.StealthSession, owned bool, err error) {
	if url := strings.TrimSpace(firstNonEmpty(opts.CDPURL, os.Getenv("DEEPSEEK_CDP_URL"))); url != "" {
		port, perr := cdpPort(url)
		if perr != nil {
			return nil, false, perr
		}
		attached, aerr := browser.AttachTab(ctx, port, "chat.deepseek.com")
		if aerr != nil {
			return nil, false, fmt.Errorf("attach to CDP browser at %s: %w", url, aerr)
		}
		return attached, false, nil
	}

	profileDir := opts.ProfileDir
	if profileDir == "" {
		profileDir = DefaultProfileDir()
	}
	launched, lerr := browser.LaunchStealth(ctx, browser.StealthConfig{
		BrowserPath: opts.BrowserPath,
		Headless:    opts.Headless,
		ProfileDir:  profileDir,
	})
	if lerr != nil {
		return nil, false, lerr
	}
	return launched, true, nil
}

// captureToken ensures the page is on chat.deepseek.com and waits for a token.
// A token already present skips the sign-in wait.
func captureToken(ctx context.Context, sess *browser.StealthSession, opts LoginOptions) (string, error) {
	if !onDeepSeek(ctx, sess) {
		_ = sess.Navigate(ctx, chatURL)
	}
	if token := evalToken(ctx, sess); token != "" {
		return token, nil
	}

	if !onDeepSeek(ctx, sess) {
		_ = sess.Navigate(ctx, signInURL)
	}
	deadline := time.Now().Add(opts.Timeout)
	for time.Now().Before(deadline) {
		if token := evalToken(ctx, sess); token != "" {
			return token, nil
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return "", fmt.Errorf("login timed out after %s — no token captured", opts.Timeout)
}

func onDeepSeek(ctx context.Context, sess *browser.StealthSession) bool {
	raw, err := sess.Evaluate(ctx, "location.href")
	if err != nil {
		return false
	}
	var href string
	if json.Unmarshal(raw, &href) != nil {
		return false
	}
	return strings.Contains(href, "chat.deepseek.com")
}

// evalToken reads localStorage.userToken, tolerating transient navigations that
// briefly destroy the page's JS execution context.
func evalToken(ctx context.Context, sess *browser.StealthSession) string {
	raw, err := sess.Evaluate(ctx, readTokenJS)
	if err != nil || len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var token string
	if json.Unmarshal(raw, &token) != nil {
		return ""
	}
	return token
}

func captureSession(ctx context.Context, sess *browser.StealthSession, token string) (*Session, error) {
	cookies := map[string]string{}
	if raw, err := sess.GetCookies(ctx, "https://chat.deepseek.com"); err == nil {
		var out struct {
			Cookies []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"cookies"`
		}
		if json.Unmarshal(raw, &out) == nil {
			for _, c := range out.Cookies {
				cookies[c.Name] = c.Value
			}
		}
	}

	ua := ""
	if raw, err := sess.Evaluate(ctx, "navigator.userAgent"); err == nil {
		_ = json.Unmarshal(raw, &ua)
	}

	return &Session{
		Token:      token,
		Cookies:    cookies,
		UserAgent:  ua,
		CapturedAt: float64(time.Now().Unix()),
	}, nil
}

// ProbeSession validates a token against the API by creating (and deleting) a
// throwaway chat session. Explicit auth rejection fails; network blips warn only.
func ProbeSession(ctx context.Context, s *Session) error {
	if s == nil || s.Token == "" {
		return errors.New("empty session")
	}
	body, _ := json.Marshal(map[string]any{})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, DeepSeekBase+"/api/v0/chat_session/create", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("authorization", "Bearer "+s.Token)
	req.Header.Set("content-type", "application/json")
	if s.UserAgent != "" {
		req.Header.Set("user-agent", s.UserAgent)
	}
	req.Header.Set("origin", DeepSeekBase)
	req.Header.Set("referer", DeepSeekBase+"/")
	if cookie := s.cookieHeader(); cookie != "" {
		req.Header.Set("cookie", cookie)
	}

	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil // warn-and-continue: the client reports real errors at chat time
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	text := strings.ToLower(string(raw))

	rejected := resp.StatusCode == 401 || resp.StatusCode == 403 || isAuthFailure(0, text)
	if rejected {
		return fmt.Errorf("the captured token was rejected by DeepSeek (invalid/expired); log out and sign in again")
	}

	// Delete the throwaway probe session to avoid clutter.
	var env struct {
		Data struct {
			BizData struct {
				ChatSession struct {
					ID string `json:"id"`
				} `json:"chat_session"`
			} `json:"biz_data"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &env) == nil && env.Data.BizData.ChatSession.ID != "" {
		del, _ := http.NewRequestWithContext(ctx, http.MethodPost, DeepSeekBase+"/api/v0/chat_session/delete",
			strings.NewReader(fmt.Sprintf(`{"chat_session_id":%q}`, env.Data.BizData.ChatSession.ID)))
		if del != nil {
			del.Header.Set("authorization", "Bearer "+s.Token)
			del.Header.Set("content-type", "application/json")
			if cookie := s.cookieHeader(); cookie != "" {
				del.Header.Set("cookie", cookie)
			}
			_, _ = (&http.Client{Timeout: 20 * time.Second}).Do(del)
		}
	}
	return nil
}

// GetSession returns a usable session: the cached file when fresh, otherwise it
// logs in interactively if allowed.
func GetSession(ctx context.Context, opts LoginOptions, allowInteractive bool) (*Session, error) {
	file := opts.SessionFile
	if file == "" {
		file = DefaultSessionFile()
	}
	if cached, err := LoadSession(file); err == nil && cached != nil && cached.Age() < SessionMaxAge {
		return cached, nil
	}
	if !allowInteractive {
		return nil, &AuthError{Message: "no DeepSeek session found; run `deepseek login` first"}
	}
	opts.SessionFile = file
	return Login(ctx, opts)
}

func cdpPort(raw string) (int, error) {
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid CDP url %q: %w", raw, err)
	}
	if p := u.Port(); p != "" {
		n, perr := strconv.Atoi(p)
		if perr != nil {
			return 0, fmt.Errorf("invalid CDP port %q", p)
		}
		return n, nil
	}
	return 9222, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
