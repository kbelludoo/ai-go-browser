package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

func main() {
	resp, err := http.Get("http://127.0.0.1:9222/json/list")
	if err != nil {
		fmt.Printf("HTTP err: %v\n", err)
		return
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var tabs []struct {
		URL string `json:"url"`
		Type string `json:"type"`
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
		fmt.Println("No arena tab")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		fmt.Printf("Dial err: %v\n", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Check text of page
	evalReq, _ := json.Marshal(map[string]any{
		"id": 999,
		"method": "Network.getCookies",
		"params": map[string]any{
			"urls": []string{"https://arena.ai"},
		},
	})
	conn.Write(ctx, websocket.MessageText, evalReq)

	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			fmt.Printf("Read err: %v\n", err)
			break
		}
		var raw map[string]any
		if json.Unmarshal(msg, &raw) == nil {
			if id, ok := raw["id"].(float64); ok && int(id) == 999 {
				resJSON, _ := json.MarshalIndent(raw["result"], "", "  ")
				fmt.Println("COOKIES RESULT:")
				fmt.Println(string(resJSON))
				break
			}
		}
	}
}
