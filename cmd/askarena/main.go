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
	fmt.Println("🚀 AI GO BROWSER — PERGUNTA DIRETA AO CLAUDE NO ARENA.AI (ZERO-BROWSER)")
	fmt.Println("================================================================================")

	// 1. Obter cookies de autenticação da sessão do Arena
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
	if wsURL == "" {
		for _, t := range tabs {
			if t.Type == "page" {
				wsURL = t.WebSocketDebuggerURL
				break
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		log.Fatalf("Erro Dial WebSocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	getCookiesReq, _ := json.Marshal(map[string]any{
		"id":     101,
		"method": "Network.getCookies",
		"params": map[string]any{
			"urls": []string{"https://arena.ai"},
		},
	})
	conn.Write(ctx, websocket.MessageText, getCookiesReq)

	var cookiePairs []string
	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			break
		}
		var parsed struct {
			ID     int `json:"id"`
			Result struct {
				Cookies []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"cookies"`
			} `json:"result"`
		}
		if json.Unmarshal(msg, &parsed) == nil && parsed.ID == 101 {
			for _, c := range parsed.Result.Cookies {
				cookiePairs = append(cookiePairs, fmt.Sprintf("%s=%s", c.Name, c.Value))
			}
			break
		}
	}

	rawCookies := strings.Join(cookiePairs, "; ")
	fmt.Printf("[1/4] %d Cookies obtidos com sucesso da sessão ativa.\n", len(cookiePairs))

	// 2. Inicializar cliente Go do Arena
	client := arena.NewClient()
	client.SetRawCookies(rawCookies)

	// 3. Validar usuário autenticado
	user, err := client.GetUser()
	if err != nil {
		log.Fatalf("Erro ao obter usuário autenticado: %v", err)
	}
	fmt.Printf("[2/4] Usuário autenticado: %s (%s)\n", user.Username, user.Email)

	// 4. Enviar pergunta ao Claude Sonnet 5
	prompt := "Olá Claude! Você é a versão mais recente da Anthropic disponível na Arena? Responda em 1 frase curta se apresentando em português."
	fmt.Printf("[3/4] Enviando pergunta ao Claude Sonnet 5:\n      %q\n", prompt)

	chatResp, err := client.CreateEvaluation(arena.ChatRequest{
		Prompt:   prompt,
		Mode:     "side-by-side",
		Modality: "chat",
		ModelA:   "claude-sonnet-5",
		ModelB:   "claude-haiku-4-5-20251001",
	})
	if err != nil {
		log.Fatalf("Erro ao criar avaliação com Claude: %v", err)
	}
	defer chatResp.Body.Close()

	fmt.Println("[4/4] Transmissão recebida! Streaming da resposta:")
	fmt.Println("--------------------------------------------------------------------------------")

	var modelAText strings.Builder
	var modelBText strings.Builder

	err = arena.ParseStream(ctx, chatResp.Body, func(ev arena.StreamEvent) error {
		switch ev.Kind {
		case "delta":
			if ev.Model == "a" || ev.Model == "" {
				fmt.Print(ev.Text)
				modelAText.WriteString(ev.Text)
			} else {
				modelBText.WriteString(ev.Text)
			}
		case "finish":
			if ev.Model == "a" {
				fmt.Println("\n[Modelo A finalizou]")
			}
		}
		return nil
	})
	if err != nil && err != io.EOF {
		fmt.Printf("Aviso stream: %v\n", err)
	}

	fmt.Println("\n--------------------------------------------------------------------------------")
	if modelAText.Len() > 0 {
		fmt.Println("🏆 RESPOSTA FINAL DO CLAUDE SONNET 5 (MODELO A):")
		fmt.Println(modelAText.String())
		fmt.Println("================================================================================")
	}
	if modelBText.Len() > 0 {
		fmt.Println("🏆 RESPOSTA DO MODELO B (Claude Haiku):")
		fmt.Println(modelBText.String())
		fmt.Println("================================================================================")
	}
}
