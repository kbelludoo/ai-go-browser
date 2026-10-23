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
	resp, err := http.Get("http://127.0.0.1:9222/json/list")
	if err != nil {
		log.Fatal(err)
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
		if t.Type == "page" && (strings.Contains(t.URL, "github.io/ai-go-browser") || strings.Contains(t.URL, "147.15.109.14:8080")) {
			wsURL = t.WebSocketDebuggerURL
			break
		}
	}

	if wsURL == "" {
		log.Fatal("Aba do painel não encontrada!")
	}

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Navegar para http://147.15.109.14:8080/ (servido diretamente pela VPS sem mixed content!)
	fmt.Println("[1/4] Acessando painel direto na VPS: http://147.15.109.14:8080/ ...")
	navReq, _ := json.Marshal(map[string]any{
		"id": 1, "method": "Page.navigate",
		"params": map[string]any{"url": "http://147.15.109.14:8080/"},
	})
	conn.Write(ctx, websocket.MessageText, navReq)
	_, _, _ = conn.Read(ctx)

	time.Sleep(3 * time.Second)

	// Verificar status da conexão
	checkJS := `(() => {
		const dot = document.getElementById('statusDot');
		const txt = document.getElementById('statusText');
		const ep = document.getElementById('endpointInput');
		return JSON.stringify({
			dot: dot ? dot.className : '',
			text: txt ? txt.innerText : '',
			endpoint: ep ? ep.value : ''
		});
	})()`
	evalCheck, _ := json.Marshal(map[string]any{
		"id": 2, "method": "Runtime.evaluate",
		"params": map[string]any{"expression": checkJS, "returnByValue": true},
	})
	conn.Write(ctx, websocket.MessageText, evalCheck)
	_, rCheck, _ := conn.Read(ctx)
	fmt.Printf("[2/4] Status da VPS: %s\n", string(rCheck))

	// Preencher prompt
	prompt := "Qual é a capital do Brasil? Responda em 1 frase curta."
	fmt.Printf("[3/4] Preenchendo prompt:\n      %q\n", prompt)
	fillJS := fmt.Sprintf(`(() => {
		const ta = document.getElementById('promptInput');
		if (ta) {
			ta.value = %q;
			ta.dispatchEvent(new Event('input', { bubbles: true }));
			return "prompt_set";
		}
		return "ta_not_found";
	})()`, prompt)
	evalFill, _ := json.Marshal(map[string]any{
		"id": 3, "method": "Runtime.evaluate",
		"params": map[string]any{"expression": fillJS, "returnByValue": true},
	})
	conn.Write(ctx, websocket.MessageText, evalFill)
	_, _, _ = conn.Read(ctx)

	// Clicar no botão enviar
	fmt.Println("[4/4] Enviando pergunta...")
	clickJS := `(() => {
		const btn = document.getElementById('sendBtn');
		if (btn) { btn.click(); return "clicked"; }
		return "btn_not_found";
	})()`
	evalClick, _ := json.Marshal(map[string]any{
		"id": 4, "method": "Runtime.evaluate",
		"params": map[string]any{"expression": clickJS, "returnByValue": true},
	})
	conn.Write(ctx, websocket.MessageText, evalClick)
	_, rClick, _ := conn.Read(ctx)
	fmt.Printf("      %s\n", string(rClick))

	// Aguardar streaming de resposta
	t0 := time.Now()
	for time.Since(t0) < 30*time.Second {
		time.Sleep(2 * time.Second)
		readJS := `(() => {
			const a = document.getElementById('outputA') ? document.getElementById('outputA').innerText.trim() : '';
			const meta = document.getElementById('metaInfo') ? document.getElementById('metaInfo').innerText.trim() : '';
			return JSON.stringify({ a, meta });
		})()`
		evalRead, _ := json.Marshal(map[string]any{
			"id": 5, "method": "Runtime.evaluate",
			"params": map[string]any{"expression": readJS, "returnByValue": true},
		})
		conn.Write(ctx, websocket.MessageText, evalRead)
		_, rRead, err := conn.Read(ctx)
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
		if json.Unmarshal(rRead, &parsed) == nil {
			var d struct {
				A    string `json:"a"`
				Meta string `json:"meta"`
			}
			json.Unmarshal([]byte(parsed.Result.Result.Value), &d)
			if d.A != "" {
				fmt.Println("\n================================================================================")
				fmt.Println("🏆 RESPOSTA OBTIDA NO PAINEL WEB:")
				fmt.Println("================================================================================")
				fmt.Println(d.A)
				fmt.Println("\nInfo:", d.Meta)
				fmt.Println("================================================================================")
				return
			}
		}
	}
}
