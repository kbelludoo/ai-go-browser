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
	fmt.Println("🤖 AI GO BROWSER — EXECUÇÃO DE PERGUNTA DIRETA AO CLAUDE NO ARENA.AI")
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

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		log.Fatalf("Erro Dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// 1. Mudar o modo para "Direct"
	fmt.Println("[1/5] Selecionando modo 'Direct'...")
	selectDirectJS := `(() => {
		// Encontrar botão Battle ou menu de modo
		const battleBtn = Array.from(document.querySelectorAll('button')).find(b => b.innerText.trim() === 'Battle' || b.innerText.trim() === 'Direct');
		if (battleBtn && battleBtn.innerText.trim() === 'Battle') {
			battleBtn.click();
			return "clicked_battle_menu";
		}
		return "already_or_not_found: " + (battleBtn ? battleBtn.innerText.trim() : 'none');
	})()`

	evalReq1, _ := json.Marshal(map[string]any{
		"id": 1, "method": "Runtime.evaluate",
		"params": map[string]any{"expression": selectDirectJS, "returnByValue": true},
	})
	conn.Write(ctx, websocket.MessageText, evalReq1)
	_, m1, _ := conn.Read(ctx)
	fmt.Printf("      %s\n", string(m1))
	time.Sleep(1 * time.Second)

	// Clicar na opção Direct
	chooseDirectJS := `(() => {
		const directOption = Array.from(document.querySelectorAll('[role="option"], [role="menuitem"], div, span, button')).find(el => {
			const t = el.innerText.trim();
			return t === 'Direct' || t.startsWith('Direct\n');
		});
		if (directOption) {
			directOption.click();
			return "clicked_direct_option";
		}
		return "direct_option_not_found";
	})()`

	evalReq2, _ := json.Marshal(map[string]any{
		"id": 2, "method": "Runtime.evaluate",
		"params": map[string]any{"expression": chooseDirectJS, "returnByValue": true},
	})
	conn.Write(ctx, websocket.MessageText, evalReq2)
	_, m2, _ := conn.Read(ctx)
	fmt.Printf("      %s\n", string(m2))
	time.Sleep(2 * time.Second)

	// 2. Abrir seletor de modelo (ao lado do botão Direct)
	fmt.Println("[2/5] Abrindo catálogo de modelos...")
	openCatalogJS := `(() => {
		// O botão do modelo fica logo ao lado do botão Direct
		const directBtn = Array.from(document.querySelectorAll('button')).find(b => b.innerText.trim() === 'Direct');
		if (directBtn) {
			const container = directBtn.parentElement;
			const nextBtn = Array.from(container.querySelectorAll('button')).find(b => b !== directBtn);
			if (nextBtn) {
				nextBtn.click();
				return "clicked_model_selector: " + nextBtn.innerText.trim();
			}
		}
		// Fallback: qualquer botão com classe 178px
		const btn = document.querySelector('button[class*="178px"]');
		if (btn) {
			btn.click();
			return "clicked_by_class: " + btn.innerText.trim();
		}
		return "model_selector_not_found";
	})()`

	evalReq3, _ := json.Marshal(map[string]any{
		"id": 3, "method": "Runtime.evaluate",
		"params": map[string]any{"expression": openCatalogJS, "returnByValue": true},
	})
	conn.Write(ctx, websocket.MessageText, evalReq3)
	_, m3, _ := conn.Read(ctx)
	fmt.Printf("      %s\n", string(m3))
	time.Sleep(1500 * time.Millisecond)

	// 3. Buscar e escolher Claude
	fmt.Println("[3/5] Escolhendo o modelo mais recente da Claude...")
	chooseClaudeJS := `(() => {
		// Digitar claude no input de busca se houver
		const search = document.querySelector('input[placeholder*="Search"], input[type="text"]:not([class*="hidden"])');
		if (search) {
			search.value = 'claude';
			search.dispatchEvent(new Event('input', { bubbles: true }));
		}
		// Encontrar os itens com claude
		const items = Array.from(document.querySelectorAll('[role="option"], [role="menuitem"], button, div')).filter(el => {
			const t = (el.innerText || '').toLowerCase();
			return t.includes('claude') && (t.includes('sonnet') || t.includes('3.5') || t.includes('5'));
		});
		if (items.length > 0) {
			// Priorizar Claude 3.5 Sonnet / Sonnet 5
			const best = items.find(el => el.innerText.toLowerCase().includes('sonnet')) || items[0];
			const name = best.innerText.trim().replace(/\n/g, ' ');
			best.click();
			return "selected_model: " + name;
		}
		return "claude_not_found (items found: " + items.length + ")";
	})()`

	evalReq4, _ := json.Marshal(map[string]any{
		"id": 4, "method": "Runtime.evaluate",
		"params": map[string]any{"expression": chooseClaudeJS, "returnByValue": true},
	})
	conn.Write(ctx, websocket.MessageText, evalReq4)
	_, m4, _ := conn.Read(ctx)
	fmt.Printf("      %s\n", string(m4))
	time.Sleep(1500 * time.Millisecond)

	// 4. Digitar prompt e Enviar
	prompt := "Olá Claude! Você é a versão mais recente? Responda em 1 frase curta se apresentando."
	fmt.Printf("[4/5] Digitando e enviando a pergunta:\n      %q\n", prompt)

	sendPromptJS := fmt.Sprintf(`(() => {
		const ta = document.querySelector('textarea');
		if (!ta) return "textarea_not_found";
		ta.focus();
		ta.value = %q;
		ta.dispatchEvent(new Event('input', { bubbles: true }));
		
		const sendBtn = document.querySelector('button[aria-label="Send message"], button:has(svg[data-icon="arrow-up"]), button[type="submit"]');
		if (sendBtn) {
			sendBtn.disabled = false;
			sendBtn.click();
			return "prompt_sent_via_btn";
		}
		// Fallback enter
		ta.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', code: 'Enter', keyCode: 13, which: 13, bubbles: true }));
		return "prompt_sent_via_enter";
	})()`, prompt)

	evalReq5, _ := json.Marshal(map[string]any{
		"id": 5, "method": "Runtime.evaluate",
		"params": map[string]any{"expression": sendPromptJS, "returnByValue": true},
	})
	conn.Write(ctx, websocket.MessageText, evalReq5)
	_, m5, _ := conn.Read(ctx)
	fmt.Printf("      %s\n", string(m5))

	// 5. Aguardar resposta
	fmt.Println("\n⏳ [5/5] Aguardando Claude responder ao vivo...")
	t0 := time.Now()
	for time.Since(t0) < 45*time.Second {
		time.Sleep(2 * time.Second)

		readAnswerJS := `(() => {
			const msgs = Array.from(document.querySelectorAll('.markdown, [class*="prose"], div[data-message-author="assistant"]'));
			for (const m of msgs) {
				const txt = m.innerText.trim();
				if (txt.length > 5 && !txt.includes("Olá Claude!")) {
					return txt;
				}
			}
			return "";
		})()`

		evalReq6, _ := json.Marshal(map[string]any{
			"id": 6, "method": "Runtime.evaluate",
			"params": map[string]any{"expression": readAnswerJS, "returnByValue": true},
		})
		conn.Write(ctx, websocket.MessageText, evalReq6)

		var parsed struct {
			ID     int `json:"id"`
			Result struct {
				Result struct {
					Value string `json:"value"`
				} `json:"result"`
			} `json:"result"`
		}
		for {
			_, m6, err := conn.Read(ctx)
			if err != nil {
				break
			}
			if json.Unmarshal(m6, &parsed) == nil && parsed.ID == 6 {
				val := parsed.Result.Result.Value
				if len(val) > 5 {
					fmt.Println("\n================================================================================")
					fmt.Println("🏆 RESPOSTA TRANSMITIDA PELO CLAUDE NO ARENA.AI:")
					fmt.Println("================================================================================")
					fmt.Println(val)
					fmt.Println("================================================================================")
					fmt.Println("✅ Sucesso absoluto! A pergunta foi respondida com sucesso pelo Claude.")
					return
				}
				break
			}
		}
	}

	fmt.Println("Tempo esgotado aguardando resposta.")
}
