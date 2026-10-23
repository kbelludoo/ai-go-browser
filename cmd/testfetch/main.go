package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

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
		if t.Type == "page" && strings.Contains(t.URL, "github.io/ai-go-browser") {
			wsURL = t.WebSocketDebuggerURL
			break
		}
	}

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	testJS := `(async () => {
		try {
			const res = await fetch('https://finance-monitor-physical-beautiful.trycloudflare.com/ask', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'X-Recaptcha-Token': 'test_token'
				},
				body: JSON.stringify({ prompt: 'teste', mode: 'battle', recaptcha_token: 'test' })
			});
			return "STATUS: " + res.status + " BODY: " + (await res.text());
		} catch (e) {
			return "FAILED: " + e.name + ": " + e.message;
		}
	})()`

	req, _ := json.Marshal(map[string]any{
		"id": 1, "method": "Runtime.evaluate",
		"params": map[string]any{"expression": testJS, "awaitPromise": true, "returnByValue": true},
	})
	conn.Write(ctx, websocket.MessageText, req)
	_, msg, _ := conn.Read(ctx)
	fmt.Println("RESULT:", string(msg))
}
