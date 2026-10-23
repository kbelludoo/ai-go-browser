package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func main() {
	url := "https://arena.ai/_next/static/chunks/2ovd9ml5m0q6-.js?dpl=dpl_5utUdh1kn7oYYXe1bxQeq4wahtbE"
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	txt := string(body)

	idx := strings.Index(txt, "modeSelector")
	if idx != -1 {
		start := max(0, idx-400)
		end := min(len(txt), idx+800)
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
