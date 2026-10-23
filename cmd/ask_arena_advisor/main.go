package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ai-go-browser/browser"
	"ai-go-browser/pkg/arena"
)

func main() {
	fmt.Println("================================================================================")
	fmt.Println("🤖 CONSULTORIA DE ARQUITETURA NO ARENA.AI (MODO AGENTE / CLAUDE SONNET 5)")
	fmt.Println("================================================================================")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	fmt.Println("[1/4] Inicializando sessão segura de navegação...")
	session, err := browser.LaunchStealth(ctx, browser.StealthConfig{
		Headless: true,
		Width:    1366,
		Height:   768,
	})
	if err != nil {
		fmt.Printf("Erro ao iniciar navegador: %v\n", err)
		return
	}
	defer session.Close()

	fmt.Println("[2/4] Conectando ao Arena.ai...")
	if err := session.Navigate(ctx, "https://arena.ai/"); err != nil {
		fmt.Printf("Erro ao navegar: %v\n", err)
		return
	}
	time.Sleep(5 * time.Second)

	prompt := `Atue como um Engenheiro de Software Principal especialista em Go e Agentes de IA.
Analise a arquitetura do meu projeto "AI Go Browser":
- Navegador headless ultraleve em Go para agentes de IA (<10 MB RAM na VPS de 1GB).
- Action Resolver determinístico com tags #ref:1, #ref:2 economizando 90% dos tokens de HTML.
- Bypass anti-bot via TLS fingerprinting e reCAPTCHA Enterprise.
- Headless API na nuvem + Frontend GitHub Pages.

Pergunta:
Quais são as 4 melhorias de maior impacto técnico e arquitetural que devemos implementar a seguir para torná-lo o padrão de mercado para ferramentas de IA (como Claude Code, Cursor e AutoGen)? Seja direto, prático e cite bibliotecas ou abordagens recomendadas em Go.`

	fmt.Println("[3/4] Gerando token reCAPTCHA Enterprise e transmitindo pergunta técnica...")

	evalID := arena.NewUUIDv7()
	payload := map[string]any{
		"id":              evalID,
		"mode":            "battle",
		"modality":        "chat",
		"userMessageId":   arena.NewUUIDv7(),
		"modelAMessageId": arena.NewUUIDv7(),
		"modelBMessageId": arena.NewUUIDv7(),
		"userMessage": map[string]any{
			"content": prompt,
		},
	}
	payloadBytes, _ := json.Marshal(payload)

	execJS := fmt.Sprintf(`(async () => {
		try {
			const payload = %s;
			const siteKey = '6LeTGMcsAAAAALuIlkVwIxaAuZA8VledA6d3Nnb0';
			
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

	rawRes, err := session.Evaluate(ctx, execJS)
	if err != nil {
		fmt.Printf("Erro ao avaliar script: %v\n", err)
		return
	}

	var resStr string
	json.Unmarshal(rawRes, &resStr)

	var res struct {
		Status int    `json:"status"`
		Error  string `json:"error"`
		Body   string `json:"body"`
	}
	json.Unmarshal([]byte(resStr), &res)

	if res.Error != "" {
		fmt.Printf("❌ Erro da API: %s\n", res.Error)
		return
	}

	fmt.Println("\n[4/4] Resposta recebida da IA do Arena.ai!")
	fmt.Println("================================================================================")

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
		fmt.Println("🏆 PARECER TÉCNICO DA IA (MODELO A):")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println(modelAText.String())
		fmt.Println("================================================================================")
	}
	if modelBText.Len() > 0 {
		fmt.Println("🏆 PARECER TÉCNICO DA IA (MODELO B):")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println(modelBText.String())
		fmt.Println("================================================================================")
	}
}
