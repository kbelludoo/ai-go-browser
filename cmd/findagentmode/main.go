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

	searchJS := `(async () => {
		const scripts = Array.from(document.querySelectorAll('script[src*="_next/static/chunks"]')).map(s => s.src);
		const hits = [];
		for (const src of scripts) {
			try {
				const txt = await (await fetch(src)).text();
				const idx = txt.indexOf('Agent Mode');
				if (idx !== -1) {
					hits.push({
						src,
						snippet: txt.substring(Math.max(0, idx - 200), Math.min(txt.length, idx + 400))
					});
				}
			} catch(e) {}
		}
		return JSON.stringify(hits);
	})()`

	evalReq, _ := json.Marshal(map[string]any{
		"id": 1, "method": "Runtime.evaluate",
		"params": map[string]any{
			"expression":    searchJS,
			"awaitPromise":  true,
			"returnByValue": true,
		},
	})
	conn.Write(ctx, websocket.MessageText, evalReq)

	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			break
		}
		var parsed struct {
			ID     int `json:"id"`
			Result struct {
				Result struct {
					Value string `json:"value"`
				} `json:"result"`
			} `json:"result"`
		}
		if json.Unmarshal(msg, &parsed) == nil && parsed.ID == 1 {
			fmt.Println("AGENT MODE IN CHUNKS:")
			fmt.Println(parsed.Result.Result.Value)
			break
		}
	}
}
