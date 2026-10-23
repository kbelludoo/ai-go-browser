package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func main() {
	url := "https://arena.ai/_next/static/chunks/16qipkvp-_40x.js?dpl=dpl_5utUdh1kn7oYYXe1bxQeq4wahtbE"
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	txt := string(body)

	matches := []string{`"agent"`, `'agent'`, `mode==="agent"`, `mode=="agent"`, `mode:"agent"`}
	for _, m := range matches {
		idx := 0
		for {
			pos := strings.Index(txt[idx:], m)
			if pos == -1 {
				break
			}
			actual := idx + pos
			start := max(0, actual-100)
			end := min(len(txt), actual+200)
			fmt.Printf("Match [%s] at %d:\n%s\n---\n", m, actual, txt[start:end])
			idx = actual + len(m)
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
