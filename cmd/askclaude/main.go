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

	"github.com/coder/websocket"
)

func main() {
	fmt.Println("================================================================================")
	fmt.Println("🤖 AI GO BROWSER — ACESSANDO CLAUDE NO ARENA.AI")
	fmt.Println("================================================================================")

	// 1. Obter a aba do Arena.ai
	resp, err := http.Get("http://127.0.0.1:9222/json/list")
	if err != nil {
		log.Fatalf("Erro ao conectar no navegador na porta 9222: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var tabs []struct {
		ID                   string `json:"id"`
		Title                string `json:"title"`
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

	if wsURL == "" {
		log.Fatal("Aba do Arena.ai não encontrada!")
	}

	fmt.Printf("✅ Aba conectada: %s\n", wsURL)

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		log.Fatalf("Erro Dial WebSocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// 2. Navegar para claude-sonnet-5 diretamente
	targetURL := "https://arena.ai/text/direct?model_a=claude-sonnet-5"
	fmt.Printf("[1/4] Selecionando o modelo mais recente: %s\n", targetURL)
	navReq, _ := json.Marshal(map[string]any{
		"id":     1,
		"method": "Page.navigate",
		"params": map[string]any{"url": targetURL},
	})
	conn.Write(ctx, websocket.MessageText, navReq)
	_, _, _ = conn.Read(ctx)

	time.Sleep(5 * time.Second)

	// 3. Clicar no botão do modelo ("Max") e selecionar Claude
	fmt.Println("[2/4] Abrindo catálogo e selecionando Claude Sonnet...")
	clickMax, _ := json.Marshal(map[string]any{
		"id": 10,
		"method": "Runtime.evaluate",
		"params": map[string]any{
			"expression": `(() => {
				const maxBtn = Array.from(document.querySelectorAll('button')).find(b => b.innerText.trim() === 'Max' || (b.className && b.className.includes('178px')));
				if (maxBtn) {
					maxBtn.click();
					return "model_dropdown_clicked: " + maxBtn.innerText.trim();
				}
				return "model_button_not_found";
			})()`,
			"returnByValue": true,
		},
	})
	conn.Write(ctx, websocket.MessageText, clickMax)
	_, clickMsg, _ := conn.Read(ctx)
	fmt.Printf("      %s\n", string(clickMsg))
	time.Sleep(1 * time.Second)

	// Filter and select Claude
	selectClaude, _ := json.Marshal(map[string]any{
		"id": 11,
		"method": "Runtime.evaluate",
		"params": map[string]any{
			"expression": `(() => {
				const search = document.querySelector('input[placeholder*="Search"], input[type="text"]');
				if (search) {
					search.value = 'claude';
					search.dispatchEvent(new Event('input', { bubbles: true }));
				}
				const items = Array.from(document.querySelectorAll('[role="option"], li, div[role="menuitem"], button')).filter(el => el.innerText.toLowerCase().includes('claude'));
				if (items.length > 0) {
					const sonnet = items.find(el => el.innerText.toLowerCase().includes('sonnet')) || items[0];
					const name = sonnet.innerText.trim();
					sonnet.click();
					return "selected_model: " + name;
				}
				return "claude_option_not_found";
			})()`,
			"returnByValue": true,
		},
	})
	conn.Write(ctx, websocket.MessageText, selectClaude)
	_, selectMsg, _ := conn.Read(ctx)
	fmt.Printf("      %s\n", string(selectMsg))
	time.Sleep(1500 * time.Millisecond)

	// 4. Digitar o prompt no campo textarea
	prompt := "Olá Claude! Você é a versão mais recente disponível na Arena? Responda em 1 frase curta se apresentando."
	fmt.Printf("[3/4] Enviando pergunta para o Claude:\n      %q\n", prompt)

	focusReq, _ := json.Marshal(map[string]any{
		"id":     3,
		"method": "Runtime.evaluate",
		"params": map[string]any{
			"expression": `(() => {
				const ta = document.querySelector('textarea');
				if (ta) {
					ta.focus();
					return true;
				}
				return false;
			})()`,
			"returnByValue": true,
		},
	})
	conn.Write(ctx, websocket.MessageText, focusReq)
	_, _, _ = conn.Read(ctx)
	time.Sleep(300 * time.Millisecond)

	insertReq, _ := json.Marshal(map[string]any{
		"id":     4,
		"method": "Input.insertText",
		"params": map[string]any{"text": prompt},
	})
	conn.Write(ctx, websocket.MessageText, insertReq)
	_, _, _ = conn.Read(ctx)
	time.Sleep(500 * time.Millisecond)

	// 5. Clicar em Send message ou Enter
	clickSend, _ := json.Marshal(map[string]any{
		"id":     5,
		"method": "Runtime.evaluate",
		"params": map[string]any{
			"expression": `(() => {
				const btn = document.querySelector('button[aria-label="Send message"], button:has(svg[data-icon="arrow-up"])');
				if (btn) {
					btn.disabled = false;
					btn.click();
					return "clicked_send_btn";
				}
				return "send_btn_not_found";
			})()`,
			"returnByValue": true,
		},
	})
	conn.Write(ctx, websocket.MessageText, clickSend)
	_, sendRes, _ := conn.Read(ctx)
	fmt.Printf("[4/4] Submissão: %s\n", string(sendRes))

	// Also send Enter key as backup
	kd, _ := json.Marshal(map[string]any{
		"id":     6,
		"method": "Input.dispatchKeyEvent",
		"params": map[string]any{"type": "keyDown", "key": "Enter", "code": "Enter", "windowsVirtualKeyCode": 13, "nativeVirtualKeyCode": 13},
	})
	conn.Write(ctx, websocket.MessageText, kd)
	_, _, _ = conn.Read(ctx)

	ku, _ := json.Marshal(map[string]any{
		"id":     7,
		"method": "Input.dispatchKeyEvent",
		"params": map[string]any{"type": "keyUp", "key": "Enter", "code": "Enter"},
	})
	conn.Write(ctx, websocket.MessageText, ku)
	_, _, _ = conn.Read(ctx)

	// 6. Aguardar streaming da resposta do Claude
	fmt.Println("\n⏳ Aguardando Claude responder (até 35s)...")
	t0 := time.Now()
	for time.Since(t0) < 35*time.Second {
		time.Sleep(2 * time.Second)
		checkReq, _ := json.Marshal(map[string]any{
			"id":     8,
			"method": "Runtime.evaluate",
			"params": map[string]any{
				"expression": `(() => {
					const markdowns = Array.from(document.querySelectorAll('.markdown, [class*="prose"], div[data-message-author="assistant"]'));
					for (const m of markdowns) {
						const txt = m.innerText.trim();
						if (txt.length > 5 && !txt.includes("Olá Claude!")) {
							return txt;
						}
					}
					return "";
				})()`,
				"returnByValue": true,
			},
		})
		conn.Write(ctx, websocket.MessageText, checkReq)
		_, rMsg, err := conn.Read(ctx)
		if err != nil {
			break
		}

		var parsed struct {
			Result struct {
				Result struct {
					Value string `json:"value"`
				} `json:"result"`
			} `json:"result"`
		}
		if json.Unmarshal(rMsg, &parsed) == nil {
			val := parsed.Result.Result.Value
			if len(val) > 10 {
				fmt.Println("\n================================================================================")
				fmt.Println("🏆 RESPOSTA TRANSMITIDA PELO CLAUDE NO ARENA.AI:")
				fmt.Println("================================================================================")
				fmt.Println(val)
				fmt.Println("================================================================================")
				fmt.Println("✅ Missão concluída com sucesso!")
				return
			}
		}
	}
	fmt.Println("Tempo esgotado aguardando geração do Claude.")
}
