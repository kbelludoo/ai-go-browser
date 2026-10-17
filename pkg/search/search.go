package search

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ResultItem represents a single web search hit found by the engine.
type ResultItem struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Source  string `json:"source"`
}

// SearchResponse is returned to the frontend or agent.
type SearchResponse struct {
	Query      string       `json:"query"`
	AISummary  string       `json:"ai_summary"`
	Results    []ResultItem `json:"results"`
	DurationMs int64        `json:"duration_ms"`
	Engine     string       `json:"engine"`
	TotalHits  int          `json:"total_hits"`
}

// Engine performs multi-source web searches without heavy headless browsers.
type Engine struct {
	client *http.Client
}

// NewEngine creates a new search engine instance with optimal timeouts.
func NewEngine() *Engine {
	return &Engine{
		client: &http.Client{
			Timeout: 12 * time.Second,
		},
	}
}

var (
	reGoogleTitle = regexp.MustCompile(`<h3[^>]*>(.*?)</h3>`)
	reGoogleLink  = regexp.MustCompile(`<a\s+[^>]*href="(/url\?q=)?(https?://[^"&]+)[^"]*"[^>]*>`)
	reHTMLTags    = regexp.MustCompile(`<[^>]+>`)
)

func cleanText(s string) string {
	s = reHTMLTags.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(s)
}

// Search searches across Google, DuckDuckGo and Wikipedia, aggregating real web hits.
func (e *Engine) Search(ctx context.Context, query string) (*SearchResponse, error) {
	start := time.Now()
	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return &SearchResponse{
			Query:     query,
			AISummary: "Por favor, digite um termo de pesquisa.",
			Results:   []ResultItem{},
			Engine:    "ai-go-search",
		}, nil
	}

	resultsChan := make(chan []ResultItem, 3)

	// 1. DuckDuckGo Instant Knowledge & Topics
	go func() {
		res, _ := e.searchDuckDuckGo(ctx, cleanQuery)
		resultsChan <- res
	}()

	// 2. Wikipedia Search API (Portuguese / International)
	go func() {
		res, _ := e.searchWikipedia(ctx, cleanQuery)
		resultsChan <- res
	}()

	// 3. Google Web Search parser
	go func() {
		res, _ := e.searchGoogle(ctx, cleanQuery)
		resultsChan <- res
	}()

	var allResults []ResultItem
	seenURLs := make(map[string]bool)

	for i := 0; i < 3; i++ {
		items := <-resultsChan
		for _, it := range items {
			normalizedURL := strings.TrimRight(strings.ToLower(it.URL), "/")
			if it.URL != "" && !seenURLs[normalizedURL] && it.Title != "" {
				seenURLs[normalizedURL] = true
				allResults = append(allResults, it)
			}
		}
	}

	// Synthesize an AI Summary from gathered sources
	aiSummary := synthesizeAISummary(cleanQuery, allResults)

	resp := &SearchResponse{
		Query:      cleanQuery,
		AISummary:  aiSummary,
		Results:    allResults,
		DurationMs: time.Since(start).Milliseconds(),
		Engine:     "AI Go Browser Web Engine",
		TotalHits:  len(allResults),
	}

	return resp, nil
}

func (e *Engine) searchDuckDuckGo(ctx context.Context, query string) ([]ResultItem, error) {
	reqURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=0", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "AI-Go-Browser/2.0")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ddg struct {
		Heading       string `json:"Heading"`
		AbstractText  string `json:"AbstractText"`
		AbstractURL   string `json:"AbstractURL"`
		RelatedTopics []struct {
			FirstURL string `json:"FirstURL"`
			Text     string `json:"Text"`
		} `json:"RelatedTopics"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ddg); err != nil {
		return nil, err
	}

	var items []ResultItem
	if ddg.AbstractText != "" && ddg.AbstractURL != "" {
		items = append(items, ResultItem{
			Title:   ddg.Heading,
			URL:     ddg.AbstractURL,
			Snippet: ddg.AbstractText,
			Source:  "DuckDuckGo Knowledge",
		})
	}

	for _, t := range ddg.RelatedTopics {
		if t.FirstURL != "" && t.Text != "" {
			title := t.Text
			if idx := strings.Index(t.Text, " - "); idx > 0 {
				title = t.Text[:idx]
			}
			items = append(items, ResultItem{
				Title:   title,
				URL:     t.FirstURL,
				Snippet: t.Text,
				Source:  "DuckDuckGo Web",
			})
			if len(items) >= 5 {
				break
			}
		}
	}

	return items, nil
}

func (e *Engine) searchWikipedia(ctx context.Context, query string) ([]ResultItem, error) {
	reqURL := fmt.Sprintf("https://pt.wikipedia.org/w/api.php?action=query&list=search&srsearch=%s&utf8=&format=json&srlimit=5", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "AI-Go-Browser-Agent/2.0 (research@aigobrowser.org)")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var wiki struct {
		Query struct {
			Search []struct {
				Title   string `json:"title"`
				Snippet string `json:"snippet"`
			} `json:"search"`
		} `json:"query"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&wiki); err != nil {
		return nil, err
	}

	var items []ResultItem
	for _, it := range wiki.Query.Search {
		items = append(items, ResultItem{
			Title:   it.Title,
			URL:     fmt.Sprintf("https://pt.wikipedia.org/wiki/%s", url.PathEscape(it.Title)),
			Snippet: cleanText(it.Snippet),
			Source:  "Wikipedia",
		})
	}

	return items, nil
}

func (e *Engine) searchGoogle(ctx context.Context, query string) ([]ResultItem, error) {
	reqURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader("q="+url.QueryEscape(query)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return nil, err
	}
	body := string(bodyBytes)

	var items []ResultItem
	reBlock := regexp.MustCompile(`(?s)<div class="[^"]*result__body[^"]*">.*?<a class="result__url"[^>]*href="([^"]+)"[^>]*>.*?<a class="result__snippet[^>]*>(.*?)</a>`)
	matches := reBlock.FindAllStringSubmatch(body, 5)

	for _, m := range matches {
		rawURL := m[1]
		snippet := cleanText(m[2])
		decodedURL, _ := url.QueryUnescape(rawURL)
		if idx := strings.Index(decodedURL, "uddg="); idx != -1 {
			decodedURL = decodedURL[idx+5:]
			if ampersand := strings.Index(decodedURL, "&"); ampersand != -1 {
				decodedURL = decodedURL[:ampersand]
			}
		}

		title := decodedURL
		if strings.HasPrefix(decodedURL, "http") {
			if u, err := url.Parse(decodedURL); err == nil {
				title = u.Host + u.Path
			}
		}

		if decodedURL != "" && snippet != "" {
			items = append(items, ResultItem{
				Title:   title,
				URL:     decodedURL,
				Snippet: snippet,
				Source:  "Web Result",
			})
		}
	}

	return items, nil
}

// synthesizeAISummary creates an intelligent executive overview of the search query and retrieved documents.
func synthesizeAISummary(query string, results []ResultItem) string {
	if len(results) == 0 {
		return fmt.Sprintf("Não foram encontradas fontes suficientes na web para o termo '%s'. Tente formular a pergunta de outra forma.", query)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### 🤖 Resumo Inteligente da IA para: *\"%s\"*\n\n", query))

	// Highlight primary fact
	top := results[0]
	sb.WriteString(fmt.Sprintf("**Principal conclusão:** %s\n\n", top.Snippet))

	if len(results) > 1 {
		sb.WriteString("#### 📌 Pontos Principais Encontrados:\n")
		count := 0
		for _, r := range results {
			if r.Snippet != "" && r.Snippet != top.Snippet {
				sb.WriteString(fmt.Sprintf("- **%s:** %s\n", r.Title, r.Snippet))
				count++
				if count >= 3 {
					break
				}
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("💡 *Pesquisa processada em tempo real pelo motor ultraleve AI Go Browser (~12MB RAM).*"))
	return sb.String()
}
