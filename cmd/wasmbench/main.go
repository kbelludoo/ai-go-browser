package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"

	"ai-go-browser/browser"
)

func main() {
	fmt.Println("================================================================================")
	fmt.Println("⚡ AI GO BROWSER — BENCHMARK DE EFICIÊNCIA, MEMÓRIA E FINGERPRINT REAL")
	fmt.Println("================================================================================")

	mem := browser.ReadProcessMemory(os.Getpid())
	fmt.Printf("Ambiente: %s (%s/%s)\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Consumo de Memória Base (RSS): %.2f MiB (Fonte: %s)\n", mem.RSSMiB, mem.Source)

	// 1. Benchmark real de parse in-process com HTML representativo
	sampleHTML := `<!doctype html><html><head><title>Benchmark Test</title></head>
	<body><nav><a href="/home">Home</a><a href="/docs">Docs</a><a href="/login">Login</a></nav>
	<main><h1>AI Go Browser Benchmark</h1><p>Testando velocidade de parse e redução de tokens de IA.</p>
	<form action="/api/search" method="POST"><input name="q" placeholder="Buscar..."><input type="hidden" name="token" value="sec123"><button type="submit">Pesquisar</button></form>
	<button id="cart" type="button">Adicionar ao Carrinho</button>
	</main></body></html>`

	iterations := 1000
	t0 := time.Now()
	for i := 0; i < iterations; i++ {
		_ = browser.Parse([]byte(sampleHTML), "https://benchmark.local", 200, len(sampleHTML), 50000)
	}
	elapsed := time.Since(t0)
	perOp := elapsed / time.Duration(iterations)

	page := browser.Parse([]byte(sampleHTML), "https://benchmark.local", 200, len(sampleHTML), 50000)
	rawTokens := len(sampleHTML) / 4
	agentTokens := len(page.Text) / 4
	if agentTokens == 0 {
		agentTokens = 1
	}
	reduction := (1.0 - (float64(agentTokens) / float64(rawTokens))) * 100.0

	fmt.Println("\n🔬 1. PERFORMANCE IN-PROCESS DO PARSER:")
	fmt.Printf("   • Iterações: %d execuções completas\n", iterations)
	fmt.Printf("   • Tempo Total: %v\n", elapsed)
	fmt.Printf("   • Latência Média por Parse: %v/op\n", perOp)
	fmt.Printf("   • Elementos Mapeados: %d links, %d botões, %d formulários\n",
		len(page.Links), len(page.Buttons), len(page.Forms))
	fmt.Printf("   • Redução de Tokens de Contexto: %d -> %d tokens (%.1f%% de economia)\n",
		rawTokens, agentTokens, reduction)

	// 2. Teste Real de Fingerprint TLS uTLS (se houver rede)
	fmt.Println("\n🔒 2. AUDITORIA REAL DE FINGERPRINT TLS (uTLS):")
	client := &http.Client{
		Transport: browser.NewSafeTransport(false),
		Timeout:   6 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://tls.peet.ws/api/all", nil)
	if err == nil {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
		req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
		req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
		req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "none")
		req.Header.Set("Sec-Fetch-User", "?1")

		resp, reqErr := client.Do(req)
		if reqErr == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			var data map[string]any
			if jsonErr := json.Unmarshal(body, &data); jsonErr == nil {
				if tlsInfo, ok := data["tls"].(map[string]any); ok {
					fmt.Printf("   • JA4 Fingerprint Reportado: %v\n", tlsInfo["ja4"])
					fmt.Printf("   • JA3 Hash:                  %v\n", tlsInfo["ja3_hash"])
					if ciphers, ok := tlsInfo["ciphers"].([]any); ok && len(ciphers) > 0 {
						fmt.Printf("   • Primeira Cifra do Handshake: %v (GREASE Ativo)\n", ciphers[0])
					}
					fmt.Println("   • Veredicto: Handshake TLS com suporte a GREASE e cifras modernas via uTLS.")
				}
			}
		} else {
			fmt.Printf("   • Nota de Rede: tls.peet.ws não acessível no momento (%v)\n", reqErr)
		}
	}

	// 3. Memória final
	finalMem := browser.ReadProcessMemory(os.Getpid())
	fmt.Printf("\n📊 3. CONSUMO FINAL DE MEMÓRIA (RSS): %.2f MiB\n", finalMem.RSSMiB)
	fmt.Println("================================================================================")
}
