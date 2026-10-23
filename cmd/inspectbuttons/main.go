package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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

	inspectJS := `(() => {
		const btns = Array.from(document.querySelectorAll('button'));
		return btns.map(b => {
			const r = b.getBoundingClientRect();
			return {
				text: b.innerText.trim(),
				w: r.width,
				h: r.height,
				top: r.top,
				left: r.left,
				display: window.getComputedStyle(b).display,
				visibility: window.getComputedStyle(b).visibility
			};
		}).filter(b => b.text.length > 0 || b.w > 0);
	})()`

	evalReq, _ := json.Marshal(map[string]any{
		"id": 1, "method": "Runtime.evaluate",
		"params": map[string]any{"expression": inspectJS, "returnByValue": true},
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
		if json.Unmarshal(msg, &parsed) == nil && parsed.ID == 1 {
			out, _ := json.MarshalIndent(parsed.Result.Result.Value, "", "  ")
			fmt.Println("VISIBLE BUTTONS:")
			fmt.Println(string(out))
			break
		}
	}
}
