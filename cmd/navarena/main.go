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
	resp, _ := http.Get("http://127.0.0.1:9222/json/list")
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
		fmt.Println("Dial err:", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	fmt.Println("Navigating to https://arena.ai/ ...")
	navReq, _ := json.Marshal(map[string]any{
		"id":     1,
		"method": "Page.navigate",
		"params": map[string]any{"url": "https://arena.ai/"},
	})
	conn.Write(ctx, websocket.MessageText, navReq)
	_, _, _ = conn.Read(ctx)

	time.Sleep(6 * time.Second)

	checkJS := `(() => {
		const ta = document.querySelector('textarea');
		const btns = Array.from(document.querySelectorAll('button')).map(b => b.innerText.trim()).filter(t => t.length > 0);
		return {
			url: window.location.href,
			hasTextarea: !!ta,
			btns: btns.slice(0, 10)
		};
	})()`

	evalReq, _ := json.Marshal(map[string]any{
		"id": 2, "method": "Runtime.evaluate",
		"params": map[string]any{"expression": checkJS, "returnByValue": true},
	})
	conn.Write(ctx, websocket.MessageText, evalReq)

	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			break
		}
		var parsed struct {
			ID int `json:"id"`
			Result struct {
				Result struct {
					Value any `json:"value"`
				} `json:"result"`
			} `json:"result"`
		}
		if json.Unmarshal(msg, &parsed) == nil && parsed.ID == 2 {
			out, _ := json.MarshalIndent(parsed.Result.Result.Value, "", "  ")
			fmt.Println("PAGE STATE:")
			fmt.Println(string(out))
			break
		}
	}
}
