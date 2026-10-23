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

	inspectJS := `(() => {
		const btns = Array.from(document.querySelectorAll('button'));
		const maxBtn = btns.find(b => b.className && b.className.includes('178px'));
		if (!maxBtn) return "not_found";

		const rect = maxBtn.getBoundingClientRect();
		const x = rect.left + rect.width / 2;
		const y = rect.top + rect.height / 2;

		['pointerdown', 'mousedown', 'pointerup', 'mouseup', 'click'].forEach(evtType => {
			maxBtn.dispatchEvent(new MouseEvent(evtType, {
				bubbles: true,
				cancelable: true,
				view: window,
				clientX: x,
				clientY: y
			}));
		});

		return { x, y, text: maxBtn.innerText };
	})()`

	evalReq, _ := json.Marshal(map[string]any{
		"id": 1, "method": "Runtime.evaluate",
		"params": map[string]any{"expression": inspectJS, "returnByValue": true},
	})
	conn.Write(ctx, websocket.MessageText, evalReq)
	_, m1, _ := conn.Read(ctx)
	fmt.Println("EVENTS DISPATCHED:", string(m1))

	time.Sleep(1 * time.Second)

	dumpJS := `(() => {
		const all = Array.from(document.querySelectorAll('*')).filter(el => {
			const txt = el.innerText || '';
			return txt.toLowerCase().includes('claude') || txt.toLowerCase().includes('gpt-4');
		});
		return all.slice(0, 5).map(el => ({ tag: el.tagName, class: el.className, text: el.innerText.substring(0, 100) }));
	})()`

	evalReq2, _ := json.Marshal(map[string]any{
		"id": 2, "method": "Runtime.evaluate",
		"params": map[string]any{"expression": dumpJS, "returnByValue": true},
	})
	conn.Write(ctx, websocket.MessageText, evalReq2)
	for {
		_, m2, err := conn.Read(ctx)
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
		if json.Unmarshal(m2, &parsed) == nil && parsed.ID == 2 {
			out, _ := json.MarshalIndent(parsed.Result.Result.Value, "", "  ")
			fmt.Println("SEARCH FOR CLAUDE IN DOM:")
			fmt.Println(string(out))
			break
		}
	}
}
