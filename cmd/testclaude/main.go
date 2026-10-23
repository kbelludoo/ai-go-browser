package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"ai-go-browser/browser"
)

func main() {
	fmt.Println("================================================================================")
	fmt.Println("🤖 AI GO BROWSER — SELEÇÃO AUTOMATIZADA DO CLAUDE NO ARENA.AI")
	fmt.Println("================================================================================")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	session, err := browser.LaunchStealth(ctx, browser.StealthConfig{
		Headless: false,
		Width:    1366,
		Height:   768,
	})
	if err != nil {
		log.Fatalf("Erro ao iniciar navegador stealth: %v", err)
	}
	defer session.Close()

	fmt.Println("[1/5] Navegando para o modo Direct do Arena.ai...")
	if err := session.Navigate(ctx, "https://arena.ai/?mode=direct"); err != nil {
		log.Fatalf("Erro ao navegar: %v", err)
	}
	time.Sleep(5 * time.Second)

	// Step 2: Click the model selector button (next to 'Direct')
	fmt.Println("[2/5] Abrindo seletor de modelos...")
	openModelSelectorJS := `(() => {
		// The button next to Direct has class containing 'max-w-[178px]' or contains model name
		const directBtn = Array.from(document.querySelectorAll('button')).find(b => b.innerText.trim() === 'Direct');
		if (directBtn) {
			// Find adjacent model button
			const container = directBtn.parentElement;
			const buttons = Array.from(container.querySelectorAll('button')).filter(b => b !== directBtn);
			if (buttons.length > 0) {
				buttons[0].click();
				return "clicked_adjacent_model_btn: " + buttons[0].innerText.trim();
			}
		}
		// Fallback: any button with max-w-[178px]
		const btn = document.querySelector('button[class*="178px"], button[role="combobox"]:not(:first-child)');
		if (btn) {
			btn.click();
			return "clicked_by_class: " + btn.innerText.trim();
		}
		return "model_button_not_found";
	})()`

	resOpen, _ := session.Evaluate(ctx, openModelSelectorJS)
	fmt.Printf("      %s\n", string(resOpen))
	time.Sleep(1500 * time.Millisecond)

	// Step 3: Search and select Claude
	fmt.Println("[3/5] Pesquisando e selecionando o modelo mais recente da Claude...")
	selectClaudeJS := `(() => {
		// Type 'claude' into search box if exists
		const searchInput = document.querySelector('input[placeholder*="Search"], input[type="text"]:not([class*="hidden"])');
		if (searchInput) {
			searchInput.value = 'claude';
			searchInput.dispatchEvent(new Event('input', { bubbles: true }));
		}
		
		// Find claude in dropdown options
		const items = Array.from(document.querySelectorAll('[role="option"], div[role="menuitem"], li, button')).filter(el => {
			const txt = el.innerText.toLowerCase();
			return txt.includes('claude') && (txt.includes('sonnet') || txt.includes('haiku') || txt.includes('opus') || txt.includes('3.5') || txt.includes('5'));
		});

		if (items.length > 0) {
			// Prioritize sonnet / latest
			const sonnet = items.find(el => el.innerText.toLowerCase().includes('sonnet')) || items[0];
			const name = sonnet.innerText.trim();
			sonnet.click();
			return "selected_claude: " + name;
		}
		return "claude_not_found_in_list (total items: " + items.length + ")";
	})()`

	resSelect, _ := session.Evaluate(ctx, selectClaudeJS)
	fmt.Printf("      %s\n", string(resSelect))
	time.Sleep(2 * time.Second)

	// Step 4: Focus and type prompt into textarea
	prompt := "Olá Claude! Você é a versão mais recente? Responda em 1 frase se apresentando e dizendo sua versão."
	fmt.Printf("[4/5] Digitando prompt nativamente:\n      %q\n", prompt)

	typeJS := fmt.Sprintf(`(() => {
		const ta = document.querySelector('textarea');
		if (ta) {
			ta.focus();
			ta.value = %q;
			ta.dispatchEvent(new Event('input', { bubbles: true }));
			return "prompt_inserted";
		}
		return "textarea_not_found";
	})()`, prompt)

	resType, _ := session.Evaluate(ctx, typeJS)
	fmt.Printf("      %s\n", string(resType))
	time.Sleep(500 * time.Millisecond)

	// Step 5: Click Send button
	fmt.Println("[5/5] Enviando mensagem para o Claude...")
	sendJS := `(() => {
		const sendBtn = document.querySelector('button[aria-label="Send message"], button:has(svg[data-icon="arrow-up"]), button[type="submit"]');
		if (sendBtn) {
			sendBtn.disabled = false;
			sendBtn.click();
			return "send_btn_clicked";
		}
		// Fallback: Enter on textarea
		const ta = document.querySelector('textarea');
		if (ta) {
			ta.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', code: 'Enter', keyCode: 13, which: 13, bubbles: true }));
			return "enter_pressed";
		}
		return "send_failed";
	})()`

	resSend, _ := session.Evaluate(ctx, sendJS)
	fmt.Printf("      %s\n", string(resSend))

	fmt.Println("\n⏳ Aguardando streaming de resposta do Claude no Arena.ai...")
	t0 := time.Now()
	for time.Since(t0) < 40*time.Second {
		time.Sleep(2 * time.Second)
		readJS := `(() => {
			const msgs = Array.from(document.querySelectorAll('.markdown, [class*="prose"], div[data-message-author="assistant"]'));
			for (const m of msgs) {
				const text = m.innerText.trim();
				if (text.length > 5 && !text.includes("Olá Claude!")) {
					return text;
				}
			}
			return "";
		})()`

		rawAns, _ := session.Evaluate(ctx, readJS)
		var ans string
		json.Unmarshal(rawAns, &ans)
		if strings.TrimSpace(ans) != "" {
			fmt.Println("\n================================================================================")
			fmt.Println("🏆 RESPOSTA OBTIDA DO CLAUDE NO ARENA.AI:")
			fmt.Println("================================================================================")
			fmt.Println(ans)
			fmt.Println("================================================================================")
			fmt.Println("✅ Sucesso total! O modelo mais recente da Claude respondeu perfeitamente.")
			return
		}
	}
	fmt.Println("Aviso: tempo limite atingido aguardando resposta.")
}
