package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"ai-go-browser/pkg/arena"
	"github.com/coder/websocket"
)

func main() {
	fmt.Println("================================================================================")
	fmt.Println("🚀 AI GO BROWSER — EXECUÇÃO DE PERGUNTA DIRETA AO CLAUDE COM TOKEN ENTERPRISE")
	fmt.Println("================================================================================")

	resp, err := http.Get("http://127.0.0.1:9222/json/list")
	if err != nil {
		log.Fatalf("Erro ao contactar CDP: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var tabs []struct {
		URL                  string `json:"url"`
		Type                 string `json:"type"`
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	json.Unmarshal(body, &tabs)

	var wsURL string
	for _, t := range tabs {
		if t.Type == "page" && strings.Contains(t.URL, "arena.ai") {
			wsURL = t.WebSocketDebuggerURL
			break
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		log.Fatalf("Erro Dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	prompt := "Olá Claude! Você é a versão mais recente? Responda em 1 frase curta se apresentando em português."
	fmt.Printf("[1/3] Preparando requisição com Claude Sonnet 5:\n      Prompt: %q\n", prompt)

	// Side-by-side mode with Claude Sonnet 5 as Model A
	evalID := arena.NewUUIDv7()
	payload := map[string]any{
		"id":              evalID,
		"mode":            "side-by-side",
		"modality":        "chat",
		"userMessageId":   arena.NewUUIDv7(),
		"modelAMessageId": arena.NewUUIDv7(),
		"modelBMessageId": arena.NewUUIDv7(),
		"modelAId":        "019f19f2-41f1-7c6d-9891-48d02fd9952c", // claude-sonnet-5
		"modelBId":        "0199e8e9-01ed-73e0-96ba-cf43b286bf10", // claude-haiku-4-5-20251001
		"userMessage": map[string]any{
			"content": prompt,
		},
	}
	payloadBytes, _ := json.Marshal(payload)

	fmt.Println("[2/3] Gerando token reCAPTCHA Enterprise e disparando stream na Arena...")

	execJS := fmt.Sprintf(`(async () => {
		try {
			const payload = %s;
			const siteKey = '6LeTGMcsAAAAALuIlkVwIxaAuZA8VledA6d3Nnb0';
			
			// Wait for grecaptcha if needed
			if (typeof grecaptcha === 'undefined' || !grecaptcha.enterprise) {
				return JSON.stringify({ error: 'grecaptcha_enterprise_not_loaded' });
			}
			
			await new Promise(r => grecaptcha.enterprise.ready(r));
			const token = await grecaptcha.enterprise.execute(siteKey, { action: 'chat_submit' });
			payload.recaptchaV3Token = token;

			const resp = await fetch('/nextjs-api/stream/create-evaluation', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'Accept': 'text/event-stream, application/json, */*'
				},
				credentials: 'include',
				body: JSON.stringify(payload)
			});

			if (!resp.ok) {
				const errText = await resp.text();
				return JSON.stringify({ status: resp.status, error: errText });
			}

			// Read stream
			const reader = resp.body.getReader();
			const decoder = new TextDecoder();
			let done = false;
			let fullBody = '';
			while (!done) {
				const { value, done: readerDone } = await reader.read();
				done = readerDone;
				if (value) {
					fullBody += decoder.decode(value, { stream: true });
				}
			}

			return JSON.stringify({ status: 200, body: fullBody });
		} catch (e) {
			return JSON.stringify({ error: e.name + ': ' + e.message });
		}
	})()`, string(payloadBytes))

	evalReq, _ := json.Marshal(map[string]any{
		"id": 500, "method": "Runtime.evaluate",
		"params": map[string]any{
			"expression":    execJS,
			"awaitPromise":  true,
			"returnByValue": true,
		},
	})
	conn.Write(ctx, websocket.MessageText, evalReq)

	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			log.Fatalf("Erro ao ler resposta: %v", err)
		}
		var parsed struct {
			ID     int `json:"id"`
			Result struct {
				Result struct {
					Value string `json:"value"`
				} `json:"result"`
			} `json:"result"`
		}
		if json.Unmarshal(msg, &parsed) == nil && parsed.ID == 500 {
			val := parsed.Result.Result.Value
			var res struct {
				Status int    `json:"status"`
				Error  string `json:"error"`
				Body   string `json:"body"`
			}
			json.Unmarshal([]byte(val), &res)

			if res.Error != "" {
				fmt.Printf("❌ Resposta com erro: %s\n", res.Error)
				return
			}

			fmt.Println("\n[3/3] Stream completo recebido do Arena.ai!")
			fmt.Println("--------------------------------------------------------------------------------")

			// Parse response using arena.ParseStream
			var modelAText strings.Builder
			var modelBText strings.Builder

			arena.ParseStream(ctx, strings.NewReader(res.Body), func(ev arena.StreamEvent) error {
				if ev.Kind == "delta" {
					if ev.Model == "a" || ev.Model == "" {
						modelAText.WriteString(ev.Text)
					} else {
						modelBText.WriteString(ev.Text)
					}
				}
				return nil
			})

			if modelAText.Len() > 0 {
				fmt.Println("🏆 RESPOSTA DO CLAUDE SONNET 5:")
				fmt.Println(modelAText.String())
				fmt.Println("--------------------------------------------------------------------------------")
			} else {
				fmt.Println("Corpo bruto retornado:")
				fmt.Println(res.Body[:min(len(res.Body), 1000)])
			}

			if modelBText.Len() > 0 {
				fmt.Println("🏆 RESPOSTA DO MODELO B (Claude Haiku):")
				fmt.Println(modelBText.String())
				fmt.Println("--------------------------------------------------------------------------------")
			}
			fmt.Println("================================================================================")
			fmt.Println("✅ Sucesso absoluto!")
			return
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
