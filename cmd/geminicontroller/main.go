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
	prompt := flag.String("prompt", "Olá Gemini! Esta mensagem foi enviada pelo navegador em Go.", "Pergunta a enviar para o Gemini")
	flag.Parse()

	fmt.Println("================================================================================")
	fmt.Println("🤖 AI GO BROWSER — CONTROLADOR AUTOMÁTICO DO GOOGLE GEMINI")
	fmt.Printf("   Porta CDP: %d | Prompt: %q\n", *port, *prompt)
	fmt.Println("================================================================================")

	// 1. Localizar aba do Gemini
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/json/list", *port))
	if err != nil {
		log.Fatalf("Erro ao conectar na porta %d: %v\nCertifique-se de que o Brave está rodando com --remote-debugging-port=%d", *port, err, *port)
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
		if t.Type == "page" && (strings.Contains(t.URL, "gemini.google.com") || strings.Contains(t.Title, "Gemini")) {
			wsURL = t.WebSocketDebuggerURL
			break
		}
	}

	if wsURL == "" {
		log.Fatal("Aba do Gemini não encontrada! Abra https://gemini.google.com no navegador.")
	}

	fmt.Printf("✅ Aba do Gemini conectada via CDP (%s)\n", wsURL)

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		log.Fatalf("Erro Dial WebSocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// 2. Focar editor do Gemini
	fmt.Println("[1/4] Focando caixa de texto do Gemini...")
	focusReq := map[string]any{
		"id":     1,
		"method": "Runtime.evaluate",
		"params": map[string]any{
			"expression": `(() => {
				const editor = document.querySelector('rich-textarea .ql-editor, [contenteditable="true"]');
				if (editor) {
					editor.focus();
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

	time.Sleep(200 * time.Millisecond)

	// 3. Inserir texto nativamente (contorna proteção TrustedHTML do Google)
	fmt.Printf("[2/4] Digitando prompt nativamente via Input.insertText:\n      %q\n", *prompt)
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

	time.Sleep(400 * time.Millisecond)

	// 4. Pressionar Enter para submeter
	fmt.Println("[3/4] Enviando mensagem...")
	keyDownReq := map[string]any{
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
	kdBytes, _ := json.Marshal(keyDownReq)
	conn.Write(ctx, websocket.MessageText, kdBytes)
	_, _, _ = conn.Read(ctx)

	keyUpReq := map[string]any{
		"id":     4,
		"method": "Input.dispatchKeyEvent",
		"params": map[string]any{
			"type": "keyUp",
			"key":  "Enter",
			"code": "Enter",
		},
	}
	kuBytes, _ := json.Marshal(keyUpReq)
	conn.Write(ctx, websocket.MessageText, kuBytes)
	_, _, _ = conn.Read(ctx)

	// 5. Ler resposta
	fmt.Println("[4/4] Aguardando resposta do Gemini...")
	time.Sleep(3 * time.Second)

	readReq := map[string]any{
		"id":     5,
		"method": "Runtime.evaluate",
		"params": map[string]any{
			"expression": `(() => {
				const modelResponses = document.querySelectorAll('message-content, .model-response-text, [data-test-id="model-response"]');
				if (modelResponses.length > 0) {
					return modelResponses[modelResponses.length - 1].innerText.trim();
				}
				const paragraphs = Array.from(document.querySelectorAll('.markdown p, p')).map(p => p.innerText.trim()).filter(t => t.length > 0);
				return paragraphs[paragraphs.length - 1] || "";
			})()`,
			"returnByValue": true,
		},
	}
	readBytes, _ := json.Marshal(readReq)
	conn.Write(ctx, websocket.MessageText, readBytes)

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
			fmt.Println("\n================================================================================")
			fmt.Println("💬 RESPOSTA DO GOOGLE GEMINI:")
			fmt.Printf("   \"%s\"\n", r.Result.Result.Value)
			fmt.Println("================================================================================")
			break
		}
	}
}
