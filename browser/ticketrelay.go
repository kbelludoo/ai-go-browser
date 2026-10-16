package browser

// Ticket relay — captcha gate off the playing machine.
//
// Three legs, all on by default:
//
//  1. Runner (GitHub Actions, disposable): real Chromium under xvfb opens the
//     game with the VPS session cookie, the fetch hook steals turnstileToken
//     from the game's own POST /auth/world-ticket and freezes the request so
//     the token is never burned from the runner IP, DummyWebSocket keeps the
//     page from opening a game socket, and the token is pushed over the
//     Cloudflare tunnel to POST /api/ticket.
//  2. VPS (this process): the receiver feeds b.turnstileChan, then
//     POST /auth/world-nonce + /auth/world-ticket run from the VPS's own IP
//     and SID — Cloudflare validates the token, the WebSocket connects with
//     zero 4401. ticketd.exe on the desktop is retired by this path.
//  3. Autonomy: a response carrying captcha_required (or status 4401) trips
//     TryResolve, which dispatches ticket-relay.yml and repeats the loop with
//     no human and no PC in it.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	defaultRelayAddr   = "127.0.0.1:8787"
	defaultRelayCookie = "TIBIDLE_SESSION"
	defaultRelayWF     = "ticket-relay.yml"
	defaultRelayRef    = "main"
	relayTokenMinLen   = 20
	relayAutoInterval  = 45 * time.Second
)

// TicketRelayConfig is the relay's full contract. Zero value is disabled;
// DefaultTicketRelayConfig reads the environment and turns it on.
type TicketRelayConfig struct {
	Enabled       bool
	GameBase      string        // https://game.example — origin hosting /auth/world-*
	SessionCookie string        // value of the VPS session cookie
	CookieName    string        // cookie header name, default TIBIDLE_SESSION
	RelayAddr     string        // receiver bind address; empty disables the listener
	TunnelURL     string        // https://<tunnel> the runner pushes tokens to
	GHRepo        string        // owner/repo owning ticket-relay.yml
	GHWorkflow    string        // workflow file name
	GHRef         string        // branch the dispatch runs against
	GHToken       string        // GitHub token with actions:write (GH_TOKEN/GITHUB_TOKEN)
	TokenWait     time.Duration // budget waiting for the runner push (push ~40s)
	HTTPTimeout   time.Duration // budget for nonce + world-ticket round trips
	UserAgent     string
}

// DefaultTicketRelayConfig builds the standard configuration from the
// environment. Missing values leave fields empty; Configured() reports
// whether the whole loop can run.
func DefaultTicketRelayConfig() TicketRelayConfig {
	return TicketRelayConfig{
		Enabled:       os.Getenv("TICKET_RELAY") != "0",
		GameBase:      strings.TrimRight(os.Getenv("TICKET_RELAY_GAME"), "/"),
		SessionCookie: os.Getenv("TIBIDLE_SESSION"),
		CookieName:    envOr("TICKET_RELAY_COOKIE_NAME", defaultRelayCookie),
		RelayAddr:     envOr("TICKET_RELAY_ADDR", defaultRelayAddr),
		TunnelURL:     strings.TrimRight(os.Getenv("TICKET_RELAY_TUNNEL"), "/"),
		GHRepo:        os.Getenv("TICKET_RELAY_REPO"),
		GHWorkflow:    envOr("TICKET_RELAY_WORKFLOW", defaultRelayWF),
		GHRef:         envOr("TICKET_RELAY_REF", defaultRelayRef),
		GHToken:       firstEnv("GH_TOKEN", "GITHUB_TOKEN"),
		TokenWait:     75 * time.Second,
		HTTPTimeout:   20 * time.Second,
		UserAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

// TicketGrant is what the VPS earned: the raw server answer to the world
// ticket plus the identifiers it was built from.
type TicketGrant struct {
	Token    string          `json:"token,omitempty"`
	SID      string          `json:"sid,omitempty"`
	Nonce    string          `json:"nonce,omitempty"`
	Response json.RawMessage `json:"response,omitempty"`
	At       time.Time       `json:"at"`
}

// TicketRelay owns the VPS leg: the /api/ticket receiver, the workflow
// dispatch, and the nonce → world-ticket completion.
type TicketRelay struct {
	cfg           TicketRelayConfig
	client        *http.Client
	turnstileChan chan string
	mu            sync.Mutex // serializes Resolve runs
	resolveMu     sync.Mutex // gate throttle bookkeeping
	lastAuto      time.Time
	srv           *http.Server
	ln            net.Listener
	dispatch      func(ctx context.Context, cfg TicketRelayConfig) error // workflow trigger; tests override
}

// NewTicketRelay builds the relay from cfg. The token channel is always live
// so a receiver can be fed even before Resolve is called.
func NewTicketRelay(cfg TicketRelayConfig) *TicketRelay {
	if cfg.CookieName == "" {
		cfg.CookieName = defaultRelayCookie
	}
	if cfg.GHWorkflow == "" {
		cfg.GHWorkflow = defaultRelayWF
	}
	if cfg.GHRef == "" {
		cfg.GHRef = defaultRelayRef
	}
	if cfg.TokenWait <= 0 {
		cfg.TokenWait = 75 * time.Second
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = 20 * time.Second
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
	}
	return &TicketRelay{
		cfg:           cfg,
		client:        &http.Client{Timeout: cfg.HTTPTimeout},
		turnstileChan: make(chan string, 1),
	}
}

// Configured reports whether the full loop (game + session + tunnel + repo)
// can run. Unconfigured relays are inert: no listener, no dispatches.
func (r *TicketRelay) Configured() bool {
	return r != nil && r.cfg.Enabled && r.cfg.GameBase != "" &&
		r.cfg.SessionCookie != "" && r.cfg.TunnelURL != "" && r.cfg.GHRepo != ""
}

// Config returns a copy of the active configuration.
func (r *TicketRelay) Config() TicketRelayConfig { return r.cfg }

// Handler serves the runner-facing endpoints: POST /api/ticket consumes the
// pushed turnstileToken, GET /healthz is a liveness probe for the tunnel.
// The body may arrive as JSON or as text/plain (the in-page hook posts
// text/plain to stay a CORS simple request and skip the preflight).
func (r *TicketRelay) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/ticket", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if req.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(req.Body, 1<<20))
		if err != nil {
			http.Error(w, "read failed", http.StatusBadRequest)
			return
		}
		token := extractRelayToken(body)
		if token == "" {
			http.Error(w, "turnstileToken missing", http.StatusBadRequest)
			return
		}
		if len(token) < relayTokenMinLen {
			http.Error(w, "turnstileToken too short", http.StatusBadRequest)
			return
		}
		if err := r.Ingest(token); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	})
	return mux
}

// Ingest validates a pushed token and stores it. Latest wins: a fresher
// runner push replaces a stale one still sitting in the buffer.
func (r *TicketRelay) Ingest(token string) error {
	token = strings.TrimSpace(token)
	if len(token) < relayTokenMinLen {
		return errors.New("turnstileToken too short")
	}
	select {
	case r.turnstileChan <- token:
	default:
		select {
		case <-r.turnstileChan:
		default:
		}
		select {
		case r.turnstileChan <- token:
		default:
		}
	}
	return nil
}

// StartReceiver binds RelayAddr and serves /api/ticket until StopReceiver.
// Only configured relays bind a port — the default browser run stays inert.
func (r *TicketRelay) StartReceiver() error {
	if r == nil || !r.cfg.Enabled || r.cfg.RelayAddr == "" {
		return nil
	}
	ln, err := net.Listen("tcp", r.cfg.RelayAddr)
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: r.Handler(), ReadHeaderTimeout: 5 * time.Second}
	r.ln = ln
	r.srv = srv
	go func() { _ = srv.Serve(ln) }()
	return nil
}

// ReceiverAddr returns the bound address, or empty when not listening.
func (r *TicketRelay) ReceiverAddr() string {
	if r == nil || r.ln == nil {
		return ""
	}
	return r.ln.Addr().String()
}

// StopReceiver shuts the listener down. Safe to call repeatedly.
func (r *TicketRelay) StopReceiver() {
	if r == nil || r.srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_ = r.srv.Shutdown(ctx)
	cancel()
	r.srv = nil
	r.ln = nil
}

func (r *TicketRelay) drainToken() {
	for {
		select {
		case <-r.turnstileChan:
		default:
			return
		}
	}
}

func (r *TicketRelay) waitToken(ctx context.Context) (string, error) {
	select {
	case tok := <-r.turnstileChan:
		return tok, nil
	case <-ctx.Done():
		return "", fmt.Errorf("sem push do runner em %s", r.cfg.TokenWait)
	}
}

// dispatchWorkflow queues ticket-relay.yml on GitHub. With a token it hits
// the REST dispatch endpoint; without one it falls back to the gh CLI, which
// is the same workflow_dispatch the diagram's `gh workflow run` triggers.
func (r *TicketRelay) dispatchWorkflow(ctx context.Context) error {
	if r.dispatch != nil {
		return r.dispatch(ctx, r.cfg)
	}
	if r.cfg.GHToken != "" {
		return r.dispatchViaAPI(ctx)
	}
	return r.dispatchViaCLI(ctx)
}

func (r *TicketRelay) dispatchViaAPI(ctx context.Context) error {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/actions/workflows/%s/dispatches",
		r.cfg.GHRepo, url.PathEscape(r.cfg.GHWorkflow))
	payload := map[string]any{"ref": r.cfg.GHRef}
	inputs := map[string]string{}
	if r.cfg.TunnelURL != "" {
		inputs["tunnel"] = r.cfg.TunnelURL
	}
	if r.cfg.GameBase != "" {
		inputs["game"] = r.cfg.GameBase
	}
	if len(inputs) > 0 {
		payload["inputs"] = inputs
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.cfg.GHToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		blob, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("workflow dispatch HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(blob)))
	}
	return nil
}

// execCommandContext is a seam so tests can swap the gh CLI invocation.
var execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

func (r *TicketRelay) dispatchViaCLI(ctx context.Context) error {
	args := []string{"workflow", "run", r.cfg.GHWorkflow, "-R", r.cfg.GHRepo, "-r", r.cfg.GHRef}
	if r.cfg.TunnelURL != "" {
		args = append(args, "-f", "tunnel="+r.cfg.TunnelURL)
	}
	if r.cfg.GameBase != "" {
		args = append(args, "-f", "game="+r.cfg.GameBase)
	}
	cmd := execCommandContext(ctx, "gh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh workflow run: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// CompleteTicket spends the runner's token from this machine's IP: nonce
// first to get the session's SID, then the world-ticket carrying the token.
// Field names follow the game's contract; adjust buildWorldTicket only.
func (r *TicketRelay) CompleteTicket(ctx context.Context, token string) (*TicketGrant, error) {
	if !r.Configured() {
		return nil, errors.New("ticket relay não configurado (TICKET_RELAY_GAME, TIBIDLE_SESSION, TICKET_RELAY_TUNNEL, TICKET_RELAY_REPO)")
	}
	nonceBody, err := r.postJSON(ctx, "/auth/world-nonce", map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("world-nonce: %w", err)
	}
	sid, nonce := extractSIDNonce(nonceBody)
	payload := buildWorldTicket(token, sid, nonce)
	respBody, err := r.postJSON(ctx, "/auth/world-ticket", payload)
	if err != nil {
		return nil, fmt.Errorf("world-ticket: %w", err)
	}
	return &TicketGrant{
		Token:    token,
		SID:      sid,
		Nonce:    nonce,
		Response: json.RawMessage(respBody),
		At:       time.Now().UTC(),
	}, nil
}

func (r *TicketRelay) postJSON(ctx context.Context, path string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, r.cfg.HTTPTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.cfg.GameBase+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", r.cfg.UserAgent)
	req.Header.Set("Cookie", r.cfg.CookieName+"="+r.cfg.SessionCookie)
	res, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	blob, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(blob)))
	}
	return blob, nil
}

// Resolve runs the full loop: dispatch the runner, wait for the pushed
// token, spend it here. Callers inside a gate use TryResolve; the manual
// ticket command calls this directly.
func (r *TicketRelay) Resolve(ctx context.Context) (*TicketGrant, error) {
	if !r.Configured() {
		return nil, errors.New("ticket relay não configurado (TICKET_RELAY_GAME, TIBIDLE_SESSION, TICKET_RELAY_TUNNEL, TICKET_RELAY_REPO)")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.drainToken()
	if err := r.dispatchWorkflow(ctx); err != nil {
		return nil, fmt.Errorf("disparar %s: %w", r.cfg.GHWorkflow, err)
	}
	waitCtx, cancel := context.WithTimeout(ctx, r.cfg.TokenWait)
	token, err := r.waitToken(waitCtx)
	cancel()
	if err != nil {
		return nil, err
	}
	return r.CompleteTicket(ctx, token)
}

// TryResolve is the gate path: same loop as Resolve, throttled so a page
// that keeps answering captcha_required does not stack workflow runs.
func (r *TicketRelay) TryResolve(ctx context.Context) (*TicketGrant, error, bool) {
	r.resolveMu.Lock()
	if !r.lastAuto.IsZero() && time.Since(r.lastAuto) < relayAutoInterval {
		r.resolveMu.Unlock()
		return nil, errors.New("gate recorrente; aguardando intervalo do relay"), false
	}
	r.lastAuto = time.Now()
	r.resolveMu.Unlock()
	grant, err := r.Resolve(ctx)
	return grant, err, true
}

// IsGate reports whether a response is the game's ticket wall: the
// captcha_required marker or the 4401 the WebSocket answers without a ticket.
func IsGate(status int, body string) bool {
	if status == 4401 {
		return true
	}
	return strings.Contains(body, "captcha_required")
}

// resultIsGate runs IsGate over whatever a command produced: the resource
// body from an API answer, or the rendered page text from a live session.
func resultIsGate(r Result) bool {
	if r.Resource != nil && IsGate(r.Resource.Status, r.Resource.Body) {
		return true
	}
	if r.Page != nil && IsGate(r.Page.Status, r.Page.Text) {
		return true
	}
	return false
}

// extractRelayToken pulls turnstileToken out of a pushed body in any shape
// the runner might send it: JSON object, wrapped, or bare string.
func extractRelayToken(body []byte) string {
	var direct struct {
		TurnstileToken string `json:"turnstileToken"`
	}
	if err := json.Unmarshal(body, &direct); err == nil && direct.TurnstileToken != "" {
		return strings.TrimSpace(direct.TurnstileToken)
	}
	var wrap map[string]json.RawMessage
	if err := json.Unmarshal(body, &wrap); err == nil {
		for _, key := range []string{"turnstileToken", "token"} {
			if raw, ok := wrap[key]; ok {
				var s string
				if json.Unmarshal(raw, &s) == nil && s != "" {
					return strings.TrimSpace(s)
				}
			}
		}
	}
	var bare string
	if err := json.Unmarshal(body, &bare); err == nil && bare != "" {
		return strings.TrimSpace(bare)
	}
	return ""
}

// extractSIDNonce reads the identifiers the world-ticket must carry from the
// world-nonce answer, tolerating the key spellings the game might use.
func extractSIDNonce(body []byte) (sid, nonce string) {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return "", ""
	}
	for _, k := range []string{"sid", "session_id", "sessionId", "SID"} {
		if v, ok := m[k].(string); ok && v != "" {
			sid = v
			break
		}
	}
	for _, k := range []string{"nonce", "challenge", "ticket_nonce"} {
		if v, ok := m[k].(string); ok && v != "" {
			nonce = v
			break
		}
	}
	return sid, nonce
}

// buildWorldTicket assembles the POST body the Cloudflare-validated ticket
// request expects. Empty identifiers are omitted so the server can treat
// them as optional.
func buildWorldTicket(token, sid, nonce string) map[string]any {
	payload := map[string]any{"turnstileToken": token}
	if sid != "" {
		payload["sid"] = sid
	}
	if nonce != "" {
		payload["nonce"] = nonce
	}
	return payload
}
