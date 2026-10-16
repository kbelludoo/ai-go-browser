package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

func main() {
	port := flag.Int("port", 9222, "Porta CDP do navegador Brave/Chrome")
	prompt := flag.String("prompt", "Qual é a montanha mais alta do mundo e sua altitude em metros? Responda em 1 frase.", "Prompt para enviar à batalha do Arena.ai")
	flag.Parse()

	fmt.Println("================================================================================")
	fmt.Println("🏟️  AI GO BROWSER — CONTROLADOR OFICIAL DO ARENA.AI (LMSYS)")
	fmt.Printf("   Porta CDP: %d | Prompt: %q\n", *port, *prompt)
	fmt.Println("================================================================================")

	// 1. Obter abas ativas
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/json/list", *port))
	if err != nil {
		log.Fatalf("Erro ao conectar na porta %d: %v", *port, err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var tabs []struct {
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
		log.Fatal("Aba do Arena.ai não encontrada! Abra https://arena.ai no navegador.")
	}

	fmt.Printf("✅ Aba do Arena.ai conectada (%s)\n", wsURL)

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		log.Fatalf("Erro Dial WebSocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// 2. Focar textarea e digitar prompt via Input.insertText
	fmt.Println("[1/4] Focando campo de pergunta do Arena.ai...")
	focusReq := map[string]any{
		"id":     1,
		"method": "Runtime.evaluate",
		"params": map[string]any{
			"expression": `(() => {
				const textarea = document.querySelector('textarea[placeholder*="Ask"], textarea');
				if (textarea) {
					textarea.focus();
					return true;
				}
				return false;
			})()`,
			"returnByValue": true,
		},
	}
	focusBytes, _ := json.Marshal(focusReq)
	conn.Write(ctx, websocket.MessageText, focusBytes)
	_, _, _ = conn.Read(ctx)

	time.Sleep(300 * time.Millisecond)

	fmt.Printf("[2/4] Digitando prompt nativamente:\n      %q\n", *prompt)
	insertReq := map[string]any{
		"id":     2,
		"method": "Input.insertText",
		"params": map[string]any{
			"text": *prompt,
		},
	}
	insertBytes, _ := json.Marshal(insertReq)
	conn.Write(ctx, websocket.MessageText, insertBytes)
	_, _, _ = conn.Read(ctx)

	time.Sleep(500 * time.Millisecond)

	// 3. Pressionar Enter para enviar
	fmt.Println("[3/4] Enviando prompt para a Batalha de Modelos...")
	keyDown := map[string]any{
		"id":     3,
		"method": "Input.dispatchKeyEvent",
		"params": map[string]any{
			"type":                  "keyDown",
			"key":                   "Enter",
			"code":                  "Enter",
			"windowsVirtualKeyCode": 13,
			"nativeVirtualKeyCode":  13,
		},
	}
	kdBytes, _ := json.Marshal(keyDown)
	conn.Write(ctx, websocket.MessageText, kdBytes)
	_, _, _ = conn.Read(ctx)

	keyUp := map[string]any{
		"id":     4,
		"method": "Input.dispatchKeyEvent",
		"params": map[string]any{
			"type": "keyUp",
			"key":  "Enter",
			"code": "Enter",
		},
	}
	kuBytes, _ := json.Marshal(keyUp)
	conn.Write(ctx, websocket.MessageText, kuBytes)
	_, _, _ = conn.Read(ctx)

	// 4. Aguardar respostas do Modelo A e Modelo B
	fmt.Println("[4/4] Aguardando respostas dos Modelos A e B no Arena.ai...")
	time.Sleep(6 * time.Second)

	readReq := map[string]any{
		"id":     5,
		"method": "Runtime.evaluate",
		"params": map[string]any{
			"expression": `(() => {
				const markdowns = document.querySelectorAll('.markdown, [class*="prose"]');
				const results = [];
				for (const m of markdowns) {
					const text = m.innerText.trim();
					if (text.length > 5) {
						results.push(text);
					}
				}
				return JSON.stringify({
					total: results.length,
					answers: results
				});
			})()`,
			"returnByValue": true,
		},
	}
	rbData, _ := json.Marshal(readReq)
	conn.Write(ctx, websocket.MessageText, rbData)

	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			break
		}
		var r struct {
			ID     int `json:"id"`
			Result struct {
				Result struct {
					Value string `json:"value"`
				} `json:"result"`
			} `json:"result"`
		}
		if json.Unmarshal(msg, &r) == nil && r.ID == 5 {
			var d struct {
				Total   int      `json:"total"`
				Answers []string `json:"answers"`
			}
			json.Unmarshal([]byte(r.Result.Result.Value), &d)

			fmt.Println("\n================================================================================")
			fmt.Println("🏆 RESPOSTAS CAPTURADAS DA BATALHA DE MODELOS NO ARENA.AI:")
			fmt.Println("================================================================================")
			for i, ans := range d.Answers {
				if i == 0 {
					fmt.Printf("\n🤖 [MODELO A]:\n%s\n", ans)
				} else if i == 1 {
					fmt.Printf("\n🤖 [MODELO B]:\n%s\n", ans)
				}
			}
			fmt.Println("================================================================================")
			break
		}
	}
}
