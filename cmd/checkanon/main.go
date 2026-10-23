package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func main() {
	url := "https://arena.ai/_next/static/chunks/0ebx6ihugnhe1.js?dpl=dpl_5utUdh1kn7oYYXe1bxQeq4wahtbE"
	resp, _ := http.Get(url)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	txt := string(body)

	idx := strings.Index(txt, "et=async")
	if idx == -1 {
		idx = strings.Index(txt, "et=")
	}
	if idx != -1 {
		start := max(0, idx-50)
		end := min(len(txt), idx+300)
		fmt.Println(txt[start:end])
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
