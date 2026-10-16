package main

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"

	"ai-go-browser/browser"
)

type BenchTarget struct {
	Name string
	URL  string
}

type BenchResult struct {
	Name         string
	URL          string
	HTMLBytes    int
	HTMLTokens   int
	MDBytes      int
	MDTokens     int
	Ratio        float64
	Links        int
	Buttons      int
	Forms        int
	Headings     int
	DurationMs   int64
	RSSMiB       float64
}

// estimateTokens estimates LLM token count using standard rule (1 token ~= 4 chars).
func estimateTokens(text string) int {
	return int(math.Ceil(float64(len(text)) / 4.0))
}

func main() {
	targets := []BenchTarget{
		{"Example Domain", "https://example.com"},
		{"Hacker News", "https://news.ycombinator.com"},
		{"Go Language Portal", "https://go.dev"},
		{"Wikipedia (Go Language)", "https://en.wikipedia.org/wiki/Go_(programming_language)"},
		{"Books to Scrape", "https://books.toscrape.com"},
		{"Httpbin HTML", "https://httpbin.org/html"},
		{"CNN Lite News", "https://lite.cnn.com"},
	}

	fmt.Println("=========================================================================================")
	fmt.Println("       AIGoBrowser - AI Agent Token Reduction & Action Coverage Benchmark                ")
	fmt.Println("=========================================================================================")

	b := browser.New([]string{"*"}, false)
	b.MaxText = 25000

	client := &http.Client{Timeout: 15 * time.Second}

	var results []BenchResult

	for _, target := range targets {
		fmt.Printf("Avaliando: %-25s (%s)... ", target.Name, target.URL)
		start := time.Now()

		// 1. Fetch raw HTML for baseline comparison
		req, _ := http.NewRequest("GET", target.URL, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("ERRO: %v\n", err)
			continue
		}
		rawBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		rawHTML := string(rawBytes)
		rawTokens := estimateTokens(rawHTML)

		// 2. Process via AIGoBrowser engine
		u, _ := url.Parse(target.URL)
		doc, err := browser.ParseDocument(rawBytes, u.String(), resp.StatusCode, len(rawBytes))
		if err != nil {
			fmt.Printf("PARSER ERRO: %v\n", err)
			continue
		}

		page := doc.Page(b.MaxText)
		md := page.Markdown
		if md == "" {
			md = page.Text
		}
		mdTokens := estimateTokens(md)

		// Token compression ratio: 1 - (mdTokens / rawTokens)
		reduction := 0.0
		if rawTokens > 0 {
			reduction = (1.0 - (float64(mdTokens) / float64(rawTokens))) * 100.0
		}

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		rss := float64(m.Sys) / (1024 * 1024)

		dur := time.Since(start).Milliseconds()

		res := BenchResult{
			Name:       target.Name,
			URL:        target.URL,
			HTMLBytes:  len(rawBytes),
			HTMLTokens: rawTokens,
			MDBytes:    len(md),
			MDTokens:   mdTokens,
			Ratio:      reduction,
			Links:      len(page.Links),
			Buttons:    len(page.Buttons),
			Forms:      len(page.Forms),
			Headings:   len(page.Headings),
			DurationMs: dur,
			RSSMiB:     rss,
		}
		results = append(results, res)
		fmt.Printf("OK (%.1f%% economizado)\n", reduction)
	}

	fmt.Println("\n-----------------------------------------------------------------------------------------")
	fmt.Printf("%-24s | %-12s | %-12s | %-9s | %-10s | %-7s\n",
		"Site", "Raw Tokens", "Agent Tokens", "Economia", "Ações (L/B)", "RAM RSS")
	fmt.Println("-----------------------------------------------------------------------------------------")

	totalRawTokens := 0
	totalMDTokens := 0
	totalActions := 0

	for _, r := range results {
		totalRawTokens += r.HTMLTokens
		totalMDTokens += r.MDTokens
		totalActions += (r.Links + r.Buttons + r.Forms)

		actionsStr := fmt.Sprintf("%d/%d", r.Links, r.Buttons)
		fmt.Printf("%-24s | %-12d | %-12d | %7.1f%%  | %-10s | %5.1f MiB\n",
			r.Name, r.HTMLTokens, r.MDTokens, r.Ratio, actionsStr, r.RSSMiB)
	}

	fmt.Println("=========================================================================================")
	totalReduction := (1.0 - (float64(totalMDTokens) / float64(totalRawTokens))) * 100.0
	fmt.Printf("TOTAL AGREGADO:\n")
	fmt.Printf("  • Tokens Brutos (HTML):     %d tokens\n", totalRawTokens)
	fmt.Printf("  • Tokens para o LLM:        %d tokens\n", totalMDTokens)
	fmt.Printf("  • Economia de Contexto:     %.1f%% MENOS TOKENS\n", totalReduction)
	fmt.Printf("  • Total de Ações Mapeadas:  %d elementos interativos\n", totalActions)
	fmt.Printf("  • Custo Economizado:        ~%.1fx mais barato para a IA processar\n", float64(totalRawTokens)/float64(totalMDTokens))
	fmt.Println("=========================================================================================")

	_ = os.Stdout.Sync()
}
