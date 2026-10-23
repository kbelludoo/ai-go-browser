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
		const iframes = Array.from(document.querySelectorAll('iframe')).map(f => ({
			src: f.src.substring(0, 100),
			w: f.getBoundingClientRect().width,
			h: f.getBoundingClientRect().height,
			vis: window.getComputedStyle(f).visibility
		}));
		const recaptchas = Array.from(document.querySelectorAll('[class*="recaptcha"], [id*="recaptcha"], [class*="captcha"]')).map(el => ({
			tag: el.tagName,
			id: el.id,
			class: el.className,
			w: el.getBoundingClientRect().width,
			h: el.getBoundingClientRect().height
		}));
		return { iframes, recaptchas };
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
			fmt.Println("CAPTCHA INSPECT:")
			fmt.Println(string(out))
			break
		}
	}
}
