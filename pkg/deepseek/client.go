package deepseek

// Pure-HTTP DeepSeek chat client.
//
// Speaks chat.deepseek.com's internal API directly using a captured signed-in
// session (see auth.go). For each message it:
//
//  1. creates a chat session   (POST /api/v0/chat_session/create)
//  2. fetches a PoW challenge   (POST /api/v0/chat/create_pow_challenge)
//  3. solves it via the WASM    (pow.go)
//  4. POSTs the completion       with the x-ds-pow-response header
//  5. parses the SSE stream      into text

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// BaseURL is DeepSeek's public chat host.
	DeepSeekBase = "https://chat.deepseek.com"
	// CompletionPath is the PoW-gated chat completion endpoint.
	CompletionPath = "/api/v0/chat/completion"

	// DefaultModelType is DeepSeek's mode pill sent as model_type. "default" is
	// Instant (the fast model); "expert" is the stronger, slower model.
	DefaultModelType = "default"

	cidSep = ":"
)

// authMarkers are body substrings that mean the server rejected our
// credentials (as opposed to rate/WAF blocks).
var authMarkers = []string{
	"authorization failed",
	"invalid token",
	"missing token",
	"unauthorized",
	"token expired",
	"token invalid",
	"login required",
	"please login",
	"please log in",
	"not logged in",
}

func isAuthFailure(status int, body string) bool {
	if status == 401 {
		return true
	}
	text := strings.ToLower(body)
	for _, m := range authMarkers {
		if strings.Contains(text, m) {
			return true
		}
	}
	return false
}

// AuthError signals that the saved session was rejected and the user must log
// in again. The stale cache is deleted before it is returned.
type AuthError struct{ Message string }

func (e *AuthError) Error() string { return e.Message }

// BackendError is returned when DeepSeek accepted the request (HTTP 200) but
// its own generation backend failed (an `event: hint` error frame). It is a
// provider-side, usually transient failure and does not trip the safety guard.
type BackendError struct {
	Message      string
	FinishReason string
}

func (e *BackendError) Error() string {
	suffix := ""
	if e.FinishReason != "" {
		suffix = fmt.Sprintf(" (finish_reason=%s)", e.FinishReason)
	}
	return "DeepSeek backend error: " + e.Message + suffix
}

// Reply is a completed chat reply plus the id to resume the conversation.
type Reply struct {
	Text           string
	ConversationID string
}

// ClientOption customizes a Client.
type ClientOption func(*Client)

// WithBaseURL overrides the API origin (useful for tests).
func WithBaseURL(base string) ClientOption {
	return func(c *Client) { c.base = strings.TrimRight(base, "/") }
}

// WithHTTPClient injects a custom HTTP client.
func WithHTTPClient(h *http.Client) ClientOption {
	return func(c *Client) { c.http = h }
}

// WithRequestGuard injects a pre-configured request guard.
func WithRequestGuard(g *RequestGuard) ClientOption {
	return func(c *Client) { c.guard = g }
}

// Client is a stateful DeepSeek Web chat client.
type Client struct {
	session *Session
	http    *http.Client
	pow     *PowSolver
	guard   *RequestGuard
	base    string
}

// MinRequestInterval returns the pacing interval from
// DEEPSEEK_MIN_REQUEST_INTERVAL (default 3s, clamped to >= 1s).
func MinRequestInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("DEEPSEEK_MIN_REQUEST_INTERVAL"))
	if raw == "" {
		return 3 * time.Second
	}
	secs, err := strconv.ParseFloat(raw, 64)
	if err != nil || secs <= 0 {
		return 3 * time.Second
	}
	return time.Duration(secs * float64(time.Second))
}

// NewClient builds a client for a session, loading the PoW module once.
func NewClient(ctx context.Context, session *Session, opts ...ClientOption) (*Client, error) {
	if session == nil {
		return nil, errors.New("deepseek: nil session")
	}
	c := &Client{
		session: session,
		base:    DeepSeekBase,
		http: &http.Client{
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          10,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   30 * time.Second,
				ResponseHeaderTimeout: 120 * time.Second,
			},
		},
		guard: NewRequestGuard(MinRequestInterval()),
	}
	for _, opt := range opts {
		opt(c)
	}
	pow, err := NewPowSolver(ctx)
	if err != nil {
		return nil, err
	}
	c.pow = pow
	return c, nil
}

// Close releases the PoW module and idle connections.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	if c.http != nil {
		c.http.CloseIdleConnections()
	}
	if c.pow != nil {
		return c.pow.Close()
	}
	return nil
}

// Guard exposes the request guard (e.g. to reset after a manual pause).
func (c *Client) Guard() *RequestGuard { return c.guard }

func (c *Client) baseHeaders() http.Header {
	h := http.Header{}
	h.Set("authorization", "Bearer "+c.session.Token)
	h.Set("accept", "*/*")
	h.Set("content-type", "application/json")
	h.Set("origin", c.base)
	h.Set("referer", c.base+"/")
	h.Set("x-app-version", "2.0.0")
	h.Set("x-client-version", "2.0.0")
	h.Set("x-client-platform", "web")
	h.Set("x-client-locale", "en_US")
	h.Set("x-client-bundle-id", "com.deepseek.chat")
	h.Set("x-client-timezone-offset", "19800")
	if c.session.UserAgent != "" {
		h.Set("user-agent", c.session.UserAgent)
	}
	if cookie := c.session.cookieHeader(); cookie != "" {
		h.Set("cookie", cookie)
	}
	return h
}

// doJSON issues a JSON request and returns the HTTP response and its body.
func (c *Client) doJSON(ctx context.Context, method, path string, payload any, extra map[string]string) (*http.Response, []byte, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return nil, nil, err
	}
	req.Header = c.baseHeaders()
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("deepseek %s: %w", path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return resp, nil, err
	}
	return resp, data, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// authFailed drops the stale cache (when it matches) and returns an AuthError.
func (c *Client) authFailed() {
	if path := DefaultSessionFile(); path != "" {
		if cached, err := LoadSession(path); err == nil && cached != nil && cached.Token == c.session.Token {
			_ = os.Remove(path)
		}
	}
}

// CreateChatSession opens a new chat session and returns its id.
func (c *Client) CreateChatSession(ctx context.Context) (string, error) {
	if err := c.guard.Wait(); err != nil {
		return "", err
	}
	resp, body, err := c.doJSON(ctx, http.MethodPost, "/api/v0/chat_session/create", map[string]any{}, nil)
	if err != nil {
		return "", err
	}
	if err := c.classify(resp, body, "during session creation"); err != nil {
		return "", err
	}
	biz, err := unwrapBiz(body)
	if err != nil {
		return "", c.maybeAuth(err)
	}
	var out struct {
		ChatSession struct {
			ID string `json:"id"`
		} `json:"chat_session"`
	}
	if err := json.Unmarshal(biz, &out); err != nil {
		return "", fmt.Errorf("decode chat_session: %w", err)
	}
	return out.ChatSession.ID, nil
}

// DeleteChatSession removes a chat session (best effort).
func (c *Client) DeleteChatSession(ctx context.Context, chatSessionID string) bool {
	if chatSessionID == "" {
		return false
	}
	resp, _, err := c.doJSON(ctx, http.MethodPost, "/api/v0/chat_session/delete",
		map[string]any{"chat_session_id": chatSessionID}, nil)
	if err != nil {
		return false
	}
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// PowHeader fetches a PoW challenge for targetPath and solves it.
func (c *Client) PowHeader(ctx context.Context, targetPath string) (string, error) {
	if err := c.guard.Wait(); err != nil {
		return "", err
	}
	resp, body, err := c.doJSON(ctx, http.MethodPost, "/api/v0/chat/create_pow_challenge",
		map[string]any{"target_path": targetPath}, nil)
	if err != nil {
		return "", err
	}
	if err := c.classify(resp, body, "while issuing a PoW challenge"); err != nil {
		return "", err
	}
	biz, err := unwrapBiz(body)
	if err != nil {
		return "", c.maybeAuth(err)
	}
	var out struct {
		Challenge PowChallenge `json:"challenge"`
	}
	if err := json.Unmarshal(biz, &out); err != nil {
		return "", fmt.Errorf("decode pow challenge: %w", err)
	}
	return c.pow.MakeHeader(ctx, out.Challenge)
}

// classify returns AuthError/SafetyTrip/status error as appropriate.
func (c *Client) classify(resp *http.Response, body []byte, phase string) error {
	text := string(body)
	if isAuthFailure(resp.StatusCode, truncate(text, 1000)) {
		c.authFailed()
		return &AuthError{Message: loginRequiredMessage}
	}
	if IsAccountProtectionResponse(resp.StatusCode, text) {
		trip := safetyStop(resp.StatusCode, phase)
		c.guard.Trip(trip.Reason)
		return trip
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("deepseek %s: HTTP %d", phase, resp.StatusCode)
	}
	return nil
}

// maybeAuth converts an unwrapBiz AuthError into the LoginRequired form.
func (c *Client) maybeAuth(err error) error {
	var ae *AuthError
	if errors.As(err, &ae) {
		c.authFailed()
		return &AuthError{Message: loginRequiredMessage}
	}
	return err
}

const loginRequiredMessage = "DeepSeek rejected the saved session (invalid/expired token). " +
	"Log in again by running `deepseek login` (use --cdp-url for Google sign-in or CAPTCHA loops)."

// unwrapBiz unwraps DeepSeek's data.biz_data envelope.
func unwrapBiz(body []byte) (json.RawMessage, error) {
	var envelope struct {
		Code *int   `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			BizData json.RawMessage `json:"biz_data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if envelope.Code == nil || *envelope.Code != 0 {
		msg := envelope.Msg
		if msg == "" {
			msg = string(body)
		}
		if isAuthFailure(0, msg) {
			return nil, &AuthError{Message: msg}
		}
		return nil, fmt.Errorf("DeepSeek API error: %s", msg)
	}
	if envelope.Data.BizData == nil {
		return nil, fmt.Errorf("unexpected response shape: %s", truncate(string(body), 500))
	}
	return envelope.Data.BizData, nil
}

// StreamOptions controls a completion request.
type StreamOptions struct {
	// ConversationID resumes a thread. When set, Model must be empty (a
	// thread's model is fixed when the thread is created).
	ConversationID string
	// Model is DeepSeek's model_type: "default" (Instant) or "expert".
	Model string
	// Thinking enables DeepThink reasoning.
	Thinking bool
	// Search enables web search.
	Search bool
}

// StreamChat sends prompt and invokes onChunk for each reply-text delta,
// returning the conversation id for resuming the thread. onChunk may be nil.
func (c *Client) StreamChat(ctx context.Context, prompt string, opts StreamOptions, onChunk func(string) error) (string, error) {
	if opts.ConversationID != "" && opts.Model != "" {
		return "", errors.New("Model cannot be set together with ConversationID; a thread's model is fixed when it is created")
	}
	sessionID, parentID := decodeCID(opts.ConversationID)
	modelType := ""
	if sessionID == "" {
		id, err := c.CreateChatSession(ctx)
		if err != nil {
			return "", err
		}
		sessionID = id
		modelType = opts.Model
		if modelType == "" {
			modelType = DefaultModelType
		}
	}

	body := map[string]any{
		"chat_session_id":   sessionID,
		"parent_message_id": parentID,
		"prompt":            prompt,
		"ref_file_ids":      []string{},
		"thinking_enabled":  opts.Thinking,
		"search_enabled":    opts.Search,
		"action":            nil,
		"preempt":           false,
	}
	if modelType != "" {
		body["model_type"] = modelType
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	// PoW challenges are short-lived, so solve right before the request.
	powHeader, err := c.PowHeader(ctx, CompletionPath)
	if err != nil {
		return "", err
	}
	if err := c.guard.Wait(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+CompletionPath, bytes.NewReader(rawBody))
	if err != nil {
		return "", err
	}
	req.Header = c.baseHeaders()
	req.Header.Set("x-ds-pow-response", powHeader)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("deepseek completion: %w", err)
	}
	defer resp.Body.Close()

	if isAuthFailure(resp.StatusCode, "") {
		c.authFailed()
		return "", &AuthError{Message: loginRequiredMessage}
	}
	if IsAccountProtectionResponse(resp.StatusCode, "") {
		trip := safetyStop(resp.StatusCode, "during completion")
		c.guard.Trip(trip.Reason)
		return "", trip
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("deepseek completion: HTTP %d", resp.StatusCode)
	}

	emit := func(string) error { return nil }
	if onChunk != nil {
		emit = onChunk
	}
	res, err := parseSSE(resp.Body, emit)
	if err != nil {
		var ae *AuthError
		if errors.As(err, &ae) {
			c.authFailed()
			return "", &AuthError{Message: loginRequiredMessage}
		}
		return "", err
	}
	return encodeCID(sessionID, res.messageID), nil
}

// Chat returns the complete reply plus its conversation id.
func (c *Client) Chat(ctx context.Context, prompt string, opts StreamOptions) (Reply, error) {
	var sb strings.Builder
	cid, err := c.StreamChat(ctx, prompt, opts, func(chunk string) error {
		sb.WriteString(chunk)
		return nil
	})
	if err != nil {
		return Reply{}, err
	}
	return Reply{Text: sb.String(), ConversationID: cid}, nil
}

func encodeCID(sessionID string, messageID *int64) string {
	if messageID == nil {
		return sessionID
	}
	return sessionID + cidSep + strconv.FormatInt(*messageID, 10)
}

func decodeCID(conversationID string) (string, *int64) {
	if conversationID == "" {
		return "", nil
	}
	sessionID, msg, _ := strings.Cut(conversationID, cidSep)
	var parent *int64
	if msg != "" {
		if n, err := strconv.ParseInt(msg, 10, 64); err == nil {
			parent = &n
		}
	}
	return sessionID, parent
}
