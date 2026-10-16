package browser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func relayTestConfig() TicketRelayConfig {
	return TicketRelayConfig{
		Enabled:       true,
		GameBase:      "https://game.invalid",
		SessionCookie: "sess-vps-123",
		CookieName:    "TIBIDLE_SESSION",
		RelayAddr:     "127.0.0.1:0",
		TunnelURL:     "https://tunnel.invalid",
		GHRepo:        "owner/repo",
		GHWorkflow:    "ticket-relay.yml",
		GHRef:         "main",
		GHToken:       "",
		TokenWait:     3 * time.Second,
		HTTPTimeout:   3 * time.Second,
	}
}

func TestTicketRelayReceiverFeedsChannel(t *testing.T) {
	relay := NewTicketRelay(relayTestConfig())
	if err := relay.StartReceiver(); err != nil {
		t.Fatalf("StartReceiver: %v", err)
	}
	defer relay.StopReceiver()
	addr := relay.ReceiverAddr()
	if addr == "" {
		t.Fatal("receiver not bound")
	}

	token := strings.Repeat("a", 40)
	push := func(contentType, body string) int {
		res, err := http.Post("http://"+addr+"/api/ticket", contentType, strings.NewReader(body))
		if err != nil {
			t.Fatalf("post: %v", err)
		}
		defer res.Body.Close()
		return res.StatusCode
	}

	if got := push("application/json", fmt.Sprintf(`{"turnstileToken":%q}`, token)); got != 200 {
		t.Fatalf("json push status = %d, want 200", got)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := relay.waitToken(ctx)
	if err != nil || got != token {
		t.Fatalf("waitToken = %q, %v; want %q", got, err, token)
	}

	if got := push("text/plain", fmt.Sprintf(`{"turnstileToken":%q}`, token)); got != 200 {
		t.Fatalf("text/plain push status = %d, want 200", got)
	}
	if got := push("application/json", `{"turnstileToken":"curto"}`); got != 400 {
		t.Fatalf("short token status = %d, want 400", got)
	}
	if got := push("application/json", `{}`); got != 400 {
		t.Fatalf("missing token status = %d, want 400", got)
	}
}

func TestTicketRelayLatestTokenWins(t *testing.T) {
	relay := NewTicketRelay(relayTestConfig())
	first := strings.Repeat("f", 40)
	second := strings.Repeat("s", 40)
	if err := relay.Ingest(first); err != nil {
		t.Fatal(err)
	}
	if err := relay.Ingest(second); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := relay.waitToken(ctx)
	if err != nil || got != second {
		t.Fatalf("waitToken = %q, %v; want %q", got, err, second)
	}
}

func TestIsGate(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   bool
	}{
		{4401, "", true},
		{200, `{"error":"captcha_required"}`, true},
		{403, `captcha_required: solve to continue`, true},
		{200, `{"ok":true}`, false},
		{0, "", false},
	}
	for _, c := range cases {
		if got := IsGate(c.status, c.body); got != c.want {
			t.Errorf("IsGate(%d, %q) = %v, want %v", c.status, c.body, got, c.want)
		}
	}
}

func TestResultIsGate(t *testing.T) {
	if !resultIsGate(Result{Resource: &Resource{Status: 200, Body: `{"error":"captcha_required"}`}}) {
		t.Error("resource body with captcha_required not detected")
	}
	if !resultIsGate(Result{Page: &Page{Status: 4401, Text: "upgrade"}}) {
		t.Error("page status 4401 not detected")
	}
	if resultIsGate(Result{Page: &Page{Status: 200, Text: "conteúdo comum"}}) {
		t.Error("plain page flagged as gate")
	}
}

func TestTicketRelayHookShape(t *testing.T) {
	hook := TicketRelayHook("https://tunnel.example")
	for _, want := range []string{
		"https://tunnel.example",
		"/api/ticket",
		"turnstileToken",
		"new Promise(() => {})",
		"DummyWebSocket",
		`/\/auth\/world-ticket(\?|$)/`,
		"__ticketRelayInstalled",
	} {
		if !strings.Contains(hook, want) {
			t.Errorf("hook missing %q", want)
		}
	}
}

func TestTicketRelayResolveEndToEnd(t *testing.T) {
	var nonceHits, ticketHits int
	var gotCookie, gotBody string
	game := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case "/auth/world-nonce":
			nonceHits++
			gotCookie = r.Header.Get("Cookie")
			fmt.Fprint(w, `{"sid":"SID-9","nonce":"NONCE-7"}`)
		case "/auth/world-ticket":
			ticketHits++
			gotBody = string(body)
			fmt.Fprint(w, `{"ticket":"WS-TICKET-1"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer game.Close()

	cfg := relayTestConfig()
	cfg.GameBase = game.URL
	relay := NewTicketRelay(cfg)
	token := strings.Repeat("t", 48)
	relay.dispatch = func(ctx context.Context, c TicketRelayConfig) error {
		if c.GHRepo != "owner/repo" || c.GHWorkflow != "ticket-relay.yml" {
			t.Errorf("dispatch got repo=%q workflow=%q", c.GHRepo, c.GHWorkflow)
		}
		go func() { _ = relay.Ingest(token) }()
		return nil
	}

	grant, err := relay.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if nonceHits != 1 || ticketHits != 1 {
		t.Fatalf("hits nonce=%d ticket=%d, want 1/1", nonceHits, ticketHits)
	}
	if gotCookie != "TIBIDLE_SESSION=sess-vps-123" {
		t.Errorf("cookie = %q", gotCookie)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(gotBody), &payload); err != nil {
		t.Fatalf("world-ticket body: %v", err)
	}
	if payload["turnstileToken"] != token {
		t.Errorf("turnstileToken = %v", payload["turnstileToken"])
	}
	if payload["sid"] != "SID-9" || payload["nonce"] != "NONCE-7" {
		t.Errorf("identifiers = %v", payload)
	}
	if grant.SID != "SID-9" || grant.Nonce != "NONCE-7" {
		t.Errorf("grant = %+v", grant)
	}
	if !bytes.Contains(grant.Response, []byte("WS-TICKET-1")) {
		t.Errorf("grant.Response = %s", grant.Response)
	}
}

func TestTicketRelayTryResolveThrottles(t *testing.T) {
	relay := NewTicketRelay(relayTestConfig())
	relay.dispatch = func(ctx context.Context, c TicketRelayConfig) error {
		return fmt.Errorf("dispatch não deve ser chamado no intervalo")
	}
	relay.lastAuto = time.Now()
	_, err, attempted := relay.TryResolve(context.Background())
	if attempted {
		t.Error("second TryResolve inside the interval attempted anyway")
	}
	if err == nil {
		t.Error("throttled call should report the wait")
	}
}

func TestTicketRelayUnconfiguredStaysInert(t *testing.T) {
	relay := NewTicketRelay(TicketRelayConfig{Enabled: true})
	if relay.Configured() {
		t.Fatal("empty config must not be configured")
	}
	if _, err := relay.Resolve(context.Background()); err == nil {
		t.Fatal("Resolve on empty config must fail")
	}
	if err := relay.StartReceiver(); err != nil {
		t.Fatalf("StartReceiver on unconfigured relay: %v", err)
	}
	if relay.ReceiverAddr() != "" {
		t.Fatal("unconfigured relay must not bind a port")
	}
}

func TestExtractRelayTokenShapes(t *testing.T) {
	token := strings.Repeat("x", 40)
	cases := []string{
		fmt.Sprintf(`{"turnstileToken":%q}`, token),
		fmt.Sprintf(`{"token":%q,"source":"gh-runner"}`, token),
		fmt.Sprintf(`%q`, token),
	}
	for _, body := range cases {
		if got := extractRelayToken([]byte(body)); got != token {
			t.Errorf("extractRelayToken(%s) = %q, want %q", body, got, token)
		}
	}
	if got := extractRelayToken([]byte(`{}`)); got != "" {
		t.Errorf("empty body extracted %q", got)
	}
}
