// arenaprobe — diagnostic tool for the arena.ai anti-bot chain.
// Mints a real reCAPTCHA Enterprise token through the stealth browser and
// fires an unauthenticated create-evaluation with it. Reading the error:
//
//	"recaptcha validation failed" -> token rejected (score/enterprise mismatch)
//	401 / "unauthorized" / auth error -> token ACCEPTED, only login missing
//
// Usage:
//
//	go run ./cmd/arenaprobe [-action chat_submit] [-site-key <key>] [-browser <path>]
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"

	"ai-go-browser/browser"
	"ai-go-browser/pkg/arena"
)

func main() {
	action := flag.String("action", "chat_submit", "reCAPTCHA action string")
	siteKey := flag.String("site-key", "6LeTGMcsAAAAALuIlkVwIxaAuZA8VledA6d3Nnb0", "recaptcha enterprise site key")
	browserPath := flag.String("browser", "", "chromium binary (default: system)")
	headful := flag.Bool("headful", false, "visible sidecar")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	fmt.Println("[1] minting recaptcha enterprise token via stealth browser...")
	mint := &browser.RecaptchaMint{
		BrowserPath: *browserPath,
		Headless:    !*headful,
		Action:      *action,
		SiteKey:     *siteKey,
	}
	t0 := time.Now()
	token, err := mint.Token(ctx)
	if err != nil {
		fmt.Printf("FAIL mint: %v\n", err)
		return
	}
	fmt.Printf("OK token minted in %s (len=%d)\n", time.Since(t0).Round(time.Millisecond), len(token))

	fmt.Println("[2] firing unauthenticated create-evaluation with token...")
	payload := map[string]any{
		"id":               arena.NewUUIDv7(),
		"mode":             "battle",
		"modality":         "chat",
		"userMessageId":    arena.NewUUIDv7(),
		"modelAMessageId":  arena.NewUUIDv7(),
		"modelBMessageId":  arena.NewUUIDv7(),
		"userMessage":      map[string]any{"content": "responda apenas: ola"},
		"recaptchaV3Token": token,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://arena.ai/nextjs-api/stream/create-evaluation", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/json, */*")
	req.Header.Set("Origin", "https://arena.ai")
	req.Header.Set("Referer", "https://arena.ai/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("FAIL request: %v\n", err)
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2000))
	fmt.Printf("HTTP %d\n", resp.Status)
	fmt.Printf("BODY: %s\n", string(respBody))

	switch {
	case bytes.Contains(respBody, []byte("recaptcha validation failed")):
		fmt.Println("VERDICT: token REJECTED — score/enterprise path needs adjustment")
	case resp.StatusCode == 401 || bytes.Contains(respBody, []byte("unauthorized")) || bytes.Contains(respBody, []byte("log in")) || bytes.Contains(respBody, []byte("sign in")):
		fmt.Println("VERDICT: token ACCEPTED — anti-bot chain PASSED, only session/login missing")
	default:
		fmt.Println("VERDICT: check body above — stream may be starting (auth not required for anon?)")
	}
}
