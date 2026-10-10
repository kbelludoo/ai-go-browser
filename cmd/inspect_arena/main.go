package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func main() {
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", "https://arena.ai", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Erro:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// Procurar por chaves de recaptcha no HTML
	reKey := regexp.MustCompile(`(?i)(6L[a-zA-Z0-9_\-]{38}|recaptcha[^"'\s>]+)`)
	matches := reKey.FindAllString(html, -1)
	fmt.Println("Ocorrências no HTML:", matches)

	// Procurar nos chunks
	reScript := regexp.MustCompile(`src=["'](/_next/static/chunks/[^"']+\.js[^"']*)["']`)
	scripts := reScript.FindAllStringSubmatch(html, -1)
	for _, s := range scripts {
		u := "https://arena.ai" + s[1]
		reqS, _ := http.NewRequest("GET", u, nil)
		resS, err := client.Do(reqS)
		if err != nil {
			continue
		}
		jsBytes, _ := io.ReadAll(resS.Body)
		resS.Body.Close()
		js := string(jsBytes)
		if strings.Contains(js, "execute") && strings.Contains(js, "recaptcha") {
			fmt.Println("Chunk com recaptcha execute:", u)
			idx := strings.Index(js, "execute")
			start := idx - 100
			if start < 0 {
				start = 0
			}
			end := idx + 200
			if end > len(js) {
				end = len(js)
			}
			fmt.Println(js[start:end])
			break
		}
	}
}
