package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func main() {
	client := &http.Client{Timeout: 15 * time.Second}
	u := "https://arena.ai/_next/static/chunks/2rtqob14b2sem.js?dpl=dpl_38nGKQtRZHSGWiHHYYh7Cu99ecdr"
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Erro:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	js := string(body)

	target := "recaptchaV3Token"
	idx := strings.Index(js, target)
	if idx != -1 {
		start := idx - 400
		if start < 0 {
			start = 0
		}
		end := idx + 400
		if end > len(js) {
			end = len(js)
		}
		fmt.Println("=== Contexto do recaptchaV3Token ===")
		fmt.Println(js[start:end])
	}
}
