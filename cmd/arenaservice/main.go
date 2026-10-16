// arenaservice — REST bridge over arena.ai (LMArena).
// Exposes the reverse-engineered nextjs-api as a local HTTP service:
//
//	GET  /health                 session status
//	GET  /v1/models              OpenAI-style model catalog
//	POST /ask                    {prompt,mode,modality} -> {model_a,model_b,session_id}
//	POST /v1/chat/completions    OpenAI-compatible (stream + non-stream)
//
// Anti-bot stack: login/session via Go HTTP client; every chat evaluation
// carries a reCAPTCHA v3 token minted inside a stealth CDP browser session
// (real Chromium binary, patched fingerprint, pointer telemetry).
//
// Usage:
//
//	set ARENA_EMAIL / ARENA_PASSWORD
//	go run ./cmd/arenaservice -addr :8080
//	curl localhost:8080/ask -d '{"prompt":"capital of France?"}'
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"ai-go-browser/browser"
	"ai-go-browser/pkg/arena"
)

// arenaSiteKey: reCAPTCHA Enterprise key recovered from the live arena.ai
// frontend (chunk_16: grecaptcha.enterprise.execute). Discovery at runtime
// still takes precedence — this is the pinned fallback.
const arenaSiteKey = "6LeTGMcsAAAAALuIlkVwIxaAuZA8VledA6d3Nnb0"

type config struct {
	addr        string
	email       string
	password    string
	cookie      string
	browserPath string
	action      string
	siteKey     string
	token       string
	headful     bool
	debugStream bool
	httpBridge  bool
	noBrowser   bool
}

type server struct {
	cfg    config
	client *arena.Client
	mint   *browser.RecaptchaMint
	log    *log.Logger

	sessMu      sync.Mutex
	user        *arena.User
	siteKeyOnce sync.Once
	siteKeyName string

	bseshMu sync.Mutex
	bsesh   *arena.BrowserSession
}

type askResult struct {
	SessionID string
	A         string
	B         string
	Unknown   string
}

type oaiContent string

func (c *oaiContent) UnmarshalJSON(b []byte) error {
	trimmed := strings.TrimSpace(string(b))
	if trimmed == "null" || trimmed == "" {
		*c = ""
		return nil
	}
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*c = oaiContent(s)
		return nil
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(b, &parts); err == nil {
		var sb strings.Builder
		for _, p := range parts {
			sb.WriteString(p.Text)
		}
		*c = oaiContent(sb.String())
		return nil
	}
	*c = ""
	return nil
}

type oaiMessage struct {
	Role    string     `json:"role"`
	Content oaiContent `json:"content"`
}

type oaiRequest struct {
	Model    string       `json:"model"`
	Messages []oaiMessage `json:"messages"`
	Stream   bool         `json:"stream"`
}

func main() {
	cfg := config{}
	fs := flag.NewFlagSet("arenaservice", flag.ExitOnError)
	fs.StringVar(&cfg.addr, "addr", ":8080", "listen address")
	fs.StringVar(&cfg.email, "email", os.Getenv("ARENA_EMAIL"), "arena.ai email (or ARENA_EMAIL)")
	fs.StringVar(&cfg.password, "password", os.Getenv("ARENA_PASSWORD"), "arena.ai password (or ARENA_PASSWORD)")
	fs.StringVar(&cfg.cookie, "cookie", os.Getenv("ARENA_COOKIE"), "arena.ai session cookie (or ARENA_COOKIE)")
	fs.StringVar(&cfg.browserPath, "browser", "", "chromium binary path (default: system Edge/Chrome)")
	fs.StringVar(&cfg.action, "recaptcha-action", "chat_submit", "reCAPTCHA v3/Enterprise action string")
	fs.StringVar(&cfg.siteKey, "site-key", arenaSiteKey, "reCAPTCHA site key override (default: live-discovered, falls back to known key)")
	fs.StringVar(&cfg.token, "token", "", "require Authorization: Bearer <token> on all routes")
	fs.BoolVar(&cfg.headful, "headful", false, "run the token sidecar visible (debugging)")
	fs.BoolVar(&cfg.debugStream, "debug-stream", false, "log every raw stream event to stderr")
	fs.BoolVar(&cfg.httpBridge, "http-bridge", false, "legacy mode: Go HTTP client + sidecar mint (no in-browser session)")
	fs.BoolVar(&cfg.noBrowser, "no-browser", os.Getenv("ARENA_NO_BROWSER") != "", "disable any local browser (100% pure HTTP for 512MB/1GB VPS)")
	_ = fs.Parse(os.Args[1:])

	if cfg.noBrowser {
		cfg.httpBridge = true
	}

	logger := log.New(os.Stderr, "[arenaservice] ", log.LstdFlags|log.Lmicroseconds)

	client := arena.NewClient()
	if cfg.cookie != "" {
		client.SetRawCookies(cfg.cookie)
		logger.Printf("loaded custom session cookie from flags/env (headless pure HTTP mode)")
	}

	var mint *browser.RecaptchaMint
	if !cfg.noBrowser {
		mint = &browser.RecaptchaMint{
			BrowserPath: cfg.browserPath,
			Headless:    !cfg.headful,
			Action:      cfg.action,
			SiteKey:     cfg.siteKey,
		}
	}

	s := &server{
		cfg:    cfg,
		client: client,
		mint:   mint,
		log:    logger,
	}

	if cfg.email != "" && cfg.password != "" {
		user, err := client.Login(cfg.email, cfg.password)
		if err != nil {
			logger.Printf("login failed at startup: %v", err)
		} else {
			s.setUser(user)
			logger.Printf("logged in as %s (%s)", user.Username, user.Email)
		}
	} else if cfg.cookie != "" {
		logger.Printf("running with session cookie — pure Go HTTP client without local browser")
	} else {
		logger.Printf("no credentials — /health will report unauthenticated; set ARENA_EMAIL/ARENA_PASSWORD or ARENA_COOKIE")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /v1/models", s.handleModels)
	mux.HandleFunc("POST /ask", s.handleAsk)
	mux.HandleFunc("POST /v1/chat/completions", s.handleChatCompletions)

	handler := s.cors(s.requireAuth(mux))

	srv := &http.Server{
		Addr:              cfg.addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Printf("listening on %s", cfg.addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Printf("server error: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	s.bseshMu.Lock()
	if s.bsesh != nil {
		s.bsesh.Close()
		s.bsesh = nil
	}
	s.bseshMu.Unlock()
	logger.Printf("shutdown complete")
}

// --- handlers ---

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser()
	healthy := map[string]any{"ok": true, "authenticated": user != nil}
	if user != nil {
		healthy["user"] = user.Username
		healthy["email"] = user.Email
	}
	writeJSON(w, http.StatusOK, healthy)
}

func (s *server) handleModels(w http.ResponseWriter, r *http.Request) {
	models, err := s.client.GetModelCatalog()
	if err != nil {
		s.writeAPIError(w, http.StatusBadGateway, fmt.Sprintf("catalog unavailable: %v", err))
		return
	}
	data := make([]map[string]any, 0, len(models))
	for _, m := range models {
		id := m.PublicName
		if id == "" {
			id = m.ID
		}
		data = append(data, map[string]any{
			"id":       id,
			"object":   "model",
			"created":  0,
			"owned_by": m.Organization,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": data})
}

func (s *server) handleAsk(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Prompt         string `json:"prompt"`
		Mode           string `json:"mode"`
		Modality       string `json:"modality"`
		Model          string `json:"model"`
		RecaptchaToken string `json:"recaptcha_token"`
		Cookie         string `json:"cookie"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		s.writeAPIError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
		return
	}
	if strings.TrimSpace(body.Prompt) == "" {
		s.writeAPIError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	recaptchaToken := body.RecaptchaToken
	if recaptchaToken == "" {
		recaptchaToken = r.Header.Get("X-Recaptcha-Token")
	}
	if body.Cookie != "" {
		s.client.SetRawCookies(body.Cookie)
	} else if c := r.Header.Get("X-Arena-Cookie"); c != "" {
		s.client.SetRawCookies(c)
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	res, err := s.runEvaluation(ctx, body.Prompt, body.Mode, body.Modality, body.Model, recaptchaToken, func(ev arena.StreamEvent) error {
		return nil
	})
	if err != nil {
		s.writeAPIError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"model_a":      res.A,
		"model_b":      res.B,
		"session_id":   res.SessionID,
		"duration_ms":  time.Since(start).Milliseconds(),
		"unattributed": res.Unknown,
	})
}

func (s *server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	var req oaiRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&req); err != nil {
		s.writeOAIError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
		return
	}
	prompt := lastUserPrompt(req.Messages)
	if prompt == "" {
		s.writeOAIError(w, http.StatusBadRequest, "no user message found in messages[]")
		return
	}
	if req.Model == "" {
		req.Model = "arena-battle"
	}

	recaptchaToken := r.Header.Get("X-Recaptcha-Token")
	if c := r.Header.Get("X-Arena-Cookie"); c != "" {
		s.client.SetRawCookies(c)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	if req.Stream {
		s.streamChatCompletion(ctx, w, req, prompt, recaptchaToken)
		return
	}

	start := time.Now()
	res, err := s.runEvaluation(ctx, prompt, modeFromModel(req.Model), "chat", req.Model, recaptchaToken, func(ev arena.StreamEvent) error {
		return nil
	})
	if err != nil {
		s.writeOAIError(w, http.StatusBadGateway, err.Error())
		return
	}
	created := time.Now().Unix()
	id := "chatcmpl-" + res.SessionID
	promptTokens := (len(prompt) + 3) / 4
	completionTokens := (len(res.A) + len(res.B) + 3) / 4
	writeJSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"object":  "chat.completion",
		"created": created,
		"model":   req.Model,
		"choices": []map[string]any{
			{"index": 0, "message": map[string]any{"role": "assistant", "content": res.A}, "finish_reason": "stop"},
			{"index": 1, "message": map[string]any{"role": "assistant", "content": res.B}, "finish_reason": "stop"},
		},
		"usage": map[string]any{
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
			"total_tokens":      promptTokens + completionTokens,
		},
		"session_id":  res.SessionID,
		"duration_ms": time.Since(start).Milliseconds(),
	})
}

func (s *server) streamChatCompletion(ctx context.Context, w http.ResponseWriter, req oaiRequest, prompt, recaptchaToken string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeOAIError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	id := "chatcmpl-" + fmt.Sprintf("%d", time.Now().UnixNano())
	created := time.Now().Unix()

	res, err := s.runEvaluation(ctx, prompt, modeFromModel(req.Model), "chat", req.Model, recaptchaToken, func(ev arena.StreamEvent) error {
		switch ev.Kind {
		case "delta":
			switch ev.Model {
			case "a":
				if err := writeOAIChunk(w, id, created, "arena-a", 0, map[string]any{"content": ev.Text}, nil); err != nil {
					return err
				}
			case "b":
				if err := writeOAIChunk(w, id, created, "arena-b", 1, map[string]any{"content": ev.Text}, nil); err != nil {
					return err
				}
			default:
				// Attribution unknown: emit on its own index so clients can
				// route it; full text also lands in /ask aggregation.
				if err := writeOAIChunk(w, id, created, "arena-unknown", 2, map[string]any{"content": ev.Text}, nil); err != nil {
					return err
				}
			}
			flusher.Flush()
			return nil
		case "error":
			return arena.ErrStreamServer(ev.Text)
		default:
			return nil
		}
	})
	if err != nil {
		s.log.Printf("stream evaluation failed (session=%s): %v", res.SessionID, err)
		writeOAIChunk(w, id, created, req.Model, 0, map[string]any{"error": err.Error()}, nil)
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
		return
	}
	stop := "stop"
	_ = writeOAIChunk(w, id, created, "arena-a", 0, map[string]any{}, &stop)
	_ = writeOAIChunk(w, id, created, "arena-b", 1, map[string]any{}, &stop)
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func writeOAIChunk(w io.Writer, id string, created int64, model string, index int, delta map[string]any, finish *string) error {
	chunk := map[string]any{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []map[string]any{
			{"index": index, "delta": delta, "finish_reason": finish},
		},
	}
	buf, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", buf)
	return err
}

// --- evaluation core ---

func (s *server) runEvaluation(ctx context.Context, prompt, mode, modality, modelA, recaptchaToken string, onEvent func(arena.StreamEvent) error) (askResult, error) {
	if s.cfg.noBrowser || s.cfg.httpBridge || recaptchaToken != "" {
		return s.runEvaluationHTTP(ctx, prompt, mode, modality, modelA, recaptchaToken, onEvent)
	}
	return s.runEvaluationBrowser(ctx, prompt, mode, modality, onEvent)
}

// runEvaluationBrowser: login + token + submit all inside ONE stealth
// browser session holding the arena cookies — mirrors the official frontend.
func (s *server) runEvaluationBrowser(ctx context.Context, prompt, mode, modality string, onEvent func(arena.StreamEvent) error) (askResult, error) {
	if s.cfg.email == "" || s.cfg.password == "" {
		return askResult{}, errors.New("not authenticated: set ARENA_EMAIL and ARENA_PASSWORD")
	}
	if mode == "" {
		mode = "battle"
	}
	if modality == "" {
		modality = "chat"
	}

	aggregate := func(bodyText string) (askResult, error) {
		var out askResult
		start := time.Now()
		err := arena.ParseStream(ctx, strings.NewReader(bodyText), func(ev arena.StreamEvent) error {
			s.logStreamEvent(ev)
			switch ev.Kind {
			case "delta":
				switch ev.Model {
				case "a":
					out.A += ev.Text
				case "b":
					out.B += ev.Text
				default:
					out.Unknown += ev.Text
				}
			case "error":
				return arena.ErrStreamServer(ev.Text)
			}
			return onEvent(ev)
		})
		s.log.Printf("evaluation finished in %s (a=%dB b=%dB unattr=%dB err=%v)",
			time.Since(start).Round(time.Millisecond), len(out.A), len(out.B), len(out.Unknown), err)
		return out, err
	}

	attempt := func() (askResult, error) {
		if err := s.ensureBrowserSession(ctx); err != nil {
			return askResult{}, err
		}
		bodyText, err := s.bsesh.SubmitEvaluation(ctx, arena.ChatRequest{
			Prompt:   prompt,
			Mode:     mode,
			Modality: modality,
		}, s.discoverSiteKey(), s.cfg.action)
		if err != nil {
			return askResult{}, err
		}
		return aggregate(bodyText)
	}

	out, err := attempt()
	if err != nil && arena.IsAuthError(err) {
		s.log.Printf("browser session expired, re-login and retry once: %v", err)
		if rerr := s.resetBrowserSession(ctx); rerr != nil {
			return out, fmt.Errorf("%v (re-login failed: %w)", err, rerr)
		}
		return attempt()
	}
	if err != nil && arena.IsRecaptchaRejected(err) {
		// Score variance: give the session more history, then retry once.
		s.log.Printf("recaptcha rejected, warming up and retrying once: %v", err)
		time.Sleep(8 * time.Second)
		return attempt()
	}
	return out, err
}

func (s *server) logStreamEvent(ev arena.StreamEvent) {
	if !s.cfg.debugStream {
		return
	}
	raw := string(ev.Raw)
	if len(raw) > 300 {
		raw = raw[:300] + "..."
	}
	s.log.Printf("stream kind=%s model=%s text=%q raw=%s", ev.Kind, ev.Model, ev.Text, raw)
}

func (s *server) ensureBrowserSession(ctx context.Context) error {
	s.bseshMu.Lock()
	defer s.bseshMu.Unlock()
	if s.bsesh != nil && s.bsesh.LoggedIn {
		return nil
	}
	if s.bsesh == nil {
		s.log.Printf("starting stealth browser session...")
		t0 := time.Now()
		bs, err := arena.StartBrowserSession(ctx, s.cfg.browserPath, s.cfg.headful, "")
		if err != nil {
			return fmt.Errorf("browser session: %w", err)
		}
		s.bsesh = bs
		s.log.Printf("browser session up in %s", time.Since(t0).Round(time.Millisecond))
	}
	if err := s.bsesh.Login(ctx, s.cfg.email, s.cfg.password); err != nil {
		return fmt.Errorf("browser login: %w", err)
	}
	s.log.Printf("browser login ok (%s)", s.bsesh.UserEmail)
	s.setUser(&arena.User{Email: s.bsesh.UserEmail})
	return nil
}

func (s *server) resetBrowserSession(ctx context.Context) error {
	s.bseshMu.Lock()
	if s.bsesh != nil {
		s.bsesh.Close()
		s.bsesh = nil
	}
	s.bseshMu.Unlock()
	return s.ensureBrowserSession(ctx)
}

// runEvaluationHTTP is the legacy path: Go HTTP client session + detached
// sidecar token mint. Kept for debugging; the server may reject the token
// when it is not minted inside the session browser.
func (s *server) runEvaluationHTTP(ctx context.Context, prompt, mode, modality, modelA, recaptchaToken string, onEvent func(arena.StreamEvent) error) (askResult, error) {
	if err := s.ensureSession(); err != nil {
		return askResult{}, err
	}
	attempt := func() (askResult, error) {
		var out askResult
		token := recaptchaToken
		if token == "" {
			if s.cfg.noBrowser || s.mint == nil {
				return out, errors.New("recaptcha_token required in headless no-browser mode. Provide 'recaptcha_token' in JSON or header 'X-Recaptcha-Token' (minted via GitHub Pages / client)")
			}
			s.mint.SiteKey = s.discoverSiteKey()
			var err error
			token, err = s.mint.Token(ctx)
			if err != nil {
				return out, fmt.Errorf("recaptcha token mint failed: %w", err)
			}
		}
		resp, err := s.client.CreateEvaluation(arena.ChatRequest{
			Prompt:           prompt,
			Mode:             mode,
			Modality:         modality,
			ModelA:           modelA,
			RecaptchaV3Token: token,
		})
		if err != nil {
			return out, err
		}
		defer resp.Body.Close()
		out.SessionID = resp.EvaluationID
		start := time.Now()
		err = arena.ParseStream(ctx, resp.Body, func(ev arena.StreamEvent) error {
			s.logStreamEvent(ev)
			switch ev.Kind {
			case "delta":
				switch ev.Model {
				case "a":
					out.A += ev.Text
				case "b":
					out.B += ev.Text
				default:
					out.Unknown += ev.Text
				}
			case "error":
				return arena.ErrStreamServer(ev.Text)
			}
			return onEvent(ev)
		})
		s.log.Printf("evaluation %s finished in %s (a=%dB b=%dB unattr=%dB err=%v)",
			resp.EvaluationID, time.Since(start).Round(time.Millisecond), len(out.A), len(out.B), len(out.Unknown), err)
		return out, err
	}

	out, err := attempt()
	if err != nil && isSessionError(err) {
		s.log.Printf("session error, re-login and retry once: %v", err)
		if rerr := s.relogin(); rerr != nil {
			return out, fmt.Errorf("%v (re-login failed: %w)", err, rerr)
		}
		return attempt()
	}
	return out, err
}

func (s *server) ensureSession() error {
	if s.currentUser() != nil {
		return nil
	}
	if s.cfg.cookie != "" {
		return nil
	}
	if s.cfg.email == "" || s.cfg.password == "" {
		return errors.New("not authenticated: set ARENA_EMAIL and ARENA_PASSWORD or provide session cookie")
	}
	return s.relogin()
}

func (s *server) relogin() error {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	user, err := s.client.Login(s.cfg.email, s.cfg.password)
	if err != nil {
		return err
	}
	s.user = user
	s.log.Printf("re-login ok as %s", user.Username)
	return nil
}

func (s *server) setUser(u *arena.User) {
	s.sessMu.Lock()
	s.user = u
	s.sessMu.Unlock()
}

func (s *server) currentUser() *arena.User {
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	return s.user
}

func (s *server) discoverSiteKey() string {
	s.siteKeyOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://arena.ai/", nil)
		if err == nil {
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
			if resp, err := http.DefaultClient.Do(req); err == nil {
				body, rerr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
				resp.Body.Close()
				if rerr == nil {
					re := regexp.MustCompile(`6L[a-zA-Z0-9_-]{38}`)
					if m := re.Find(body); m != nil {
						s.siteKeyName = string(m)
						s.log.Printf("recaptcha sitekey discovered live: %s...", s.siteKeyName[:8])
					}
				}
			}
		}
		if s.siteKeyName == "" {
			s.siteKeyName = s.cfg.siteKey
			s.log.Printf("using pinned sitekey: %s...", s.siteKeyName[:8])
		}
	})
	return s.siteKeyName
}

// --- helpers ---

func isSessionError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, needle := range []string{"401", "403", "unauthorized", "not authenticated", "invalid session", "sessão inválida"} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

func modeFromModel(model string) string {
	switch strings.ToLower(model) {
	case "arena-side-by-side", "arena-sbs", "side-by-side":
		return "side-by-side"
	case "arena-direct":
		return "direct"
	default:
		return "battle"
	}
}

func lastUserPrompt(messages []oaiMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.EqualFold(messages[i].Role, "user") {
			return strings.TrimSpace(string(messages[i].Content))
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *server) writeAPIError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

func (s *server) writeOAIError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{"message": msg, "type": "server_error"},
	})
}

func (s *server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Recaptcha-Token, X-Arena-Cookie")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) requireAuth(next http.Handler) http.Handler {
	if s.cfg.token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+s.cfg.token {
			s.writeAPIError(w, http.StatusUnauthorized, "missing or invalid bearer token")
			return
		}
		next.ServeHTTP(w, r)
	})
}
