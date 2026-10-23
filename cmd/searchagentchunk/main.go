package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func main() {
	chunks := []string{
		"https://arena.ai/_next/static/chunks/16qipkvp-_40x.js?dpl=dpl_5utUdh1kn7oYYXe1bxQeq4wahtbE",
		"https://arena.ai/_next/static/chunks/0ebx6ihugnhe1.js?dpl=dpl_5utUdh1kn7oYYXe1bxQeq4wahtbE",
		"https://arena.ai/_next/static/chunks/3ka-dadvvcrgc.js?dpl=dpl_5utUdh1kn7oYYXe1bxQeq4wahtbE",
	}

	for _, url := range chunks {
		resp, err := http.Get(url)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		txt := string(body)

		idx := strings.Index(txt, "Agent Mode")
		if idx != -1 {
			start := max(0, idx-200)
			end := min(len(txt), idx+400)
			fmt.Printf("Found in %s:\n%s\n\n", url, txt[start:end])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
