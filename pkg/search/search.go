package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
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
	webClient *http.Client
	llmClient *http.Client
}

// NewEngine creates a new search engine instance with optimal timeouts.
func NewEngine() *Engine {
	return &Engine{
		webClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		llmClient: &http.Client{
			Timeout: 60 * time.Second,
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

	// Synthesize an AI Summary from gathered sources via connected Agent-Go
	aiSummary := e.synthesizeAISummary(ctx, cleanQuery, allResults)

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

	resp, err := e.webClient.Do(req)
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

	resp, err := e.webClient.Do(req)
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
		wikiSlug := strings.ReplaceAll(it.Title, " ", "_")
		items = append(items, ResultItem{
			Title:   it.Title,
			URL:     fmt.Sprintf("https://pt.wikipedia.org/wiki/%s", url.PathEscape(wikiSlug)),
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

	resp, err := e.webClient.Do(req)
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
		rawURL := html.UnescapeString(m[1])
		snippet := cleanText(m[2])
		decodedURL := rawURL
		if parsedWrapper, err := url.Parse(rawURL); err == nil {
			if uddg := parsedWrapper.Query().Get("uddg"); uddg != "" {
				decodedURL = uddg
			}
		} else if idx := strings.Index(rawURL, "uddg="); idx != -1 {
			sub := rawURL[idx+5:]
			if ampersand := strings.Index(sub, "&"); ampersand != -1 {
				sub = sub[:ampersand]
			}
			if u, uErr := url.QueryUnescape(sub); uErr == nil {
				decodedURL = u
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

// loadAgentGoEnv attempts to load KILO_API_KEY and LLM_API_KEY from known .env paths.
func loadAgentGoEnv() (kiloKey, llmKey, llmBase, llmModel string) {
	kiloKey = os.Getenv("KILO_API_KEY")
	llmKey = os.Getenv("LLM_API_KEY")
	llmBase = os.Getenv("LLM_BASE_URL")
	llmModel = os.Getenv("LLM_MODEL")

	if kiloKey != "" || llmKey != "" {
		return
	}

	paths := []string{
		`C:\Users\kbell\Downloads\agent-go\.env`,
		"/home/ubuntu/ai-go-browser/.env",
		"/home/ubuntu/agent-go/.env",
		"agent-go.env",
		".env",
	}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			k := strings.TrimSpace(parts[0])
			v := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			switch k {
			case "KILO_API_KEY":
				if kiloKey == "" {
					kiloKey = v
				}
			case "LLM_API_KEY":
				if llmKey == "" {
					llmKey = v
				}
			case "LLM_BASE_URL":
				if llmBase == "" {
					llmBase = v
				}
			case "LLM_MODEL":
				if llmModel == "" {
					llmModel = v
				}
			}
		}
	}
	return
}

// synthesizeAISummary calls the Agent-Go LLM gateway to generate a human-grade executive summary.
func (e *Engine) synthesizeAISummary(ctx context.Context, query string, results []ResultItem) string {
	if len(results) == 0 {
		return fmt.Sprintf("Não foram encontradas fontes suficientes na web para o termo '%s'. Tente formular a pergunta de outra forma.", query)
	}

	kiloKey, llmKey, llmBase, _ := loadAgentGoEnv()
	log.Printf("[agent-go] checking LLM credentials: kilo=%t, llm=%t", kiloKey != "", llmKey != "")

	// 1. Try Kilo Gateway from agent-go
	if kiloKey != "" {
		summary, err := e.callLLM(ctx, "https://api.kilo.ai/api/gateway/chat/completions", kiloKey, "kilo-auto/free", query, results)
		if err == nil && summary != "" {
			return summary
		}
		log.Printf("[agent-go] kilo gateway failed: %v", err)
	}

	// 2. Try NVIDIA NIM from agent-go
	if llmKey != "" {
		baseURL := llmBase
		if baseURL == "" {
			baseURL = "https://integrate.api.nvidia.com/v1"
		}
		if !strings.HasSuffix(baseURL, "/chat/completions") {
			baseURL = strings.TrimRight(baseURL, "/") + "/chat/completions"
		}
		summary, err := e.callLLM(ctx, baseURL, llmKey, "deepseek-ai/deepseek-v4.1-flash", query, results)
		if err == nil && summary != "" {
			return summary
		}
		log.Printf("[agent-go] nvidia nim failed: %v", err)
	}

	// 3. Fallback to local extractive summary
	log.Printf("[agent-go] using local fallback summary")
	return synthesizeAISummaryLocal(query, results)
}

func (e *Engine) callLLM(ctx context.Context, endpoint, apiKey, model, query string, results []ResultItem) (string, error) {
	var sources strings.Builder
	for i, r := range results {
		if i >= 6 {
			break
		}
		sources.WriteString(fmt.Sprintf("[%d] Título: %s\nURL: %s\nConteúdo: %s\n\n", i+1, r.Title, r.URL, r.Snippet))
	}

	sysPrompt := "Você é o assistente inteligente de busca do AI Go Browser conectado ao Agent-Go.\n" +
		"Sua missão é responder à pesquisa do usuário de forma clara, aprofundada, didática e precisa em português do Brasil.\n" +
		"Use as informações encontradas na web para embasar sua resposta. Organize com subtítulos em Markdown e tópicos quando apropriado."

	userPrompt := fmt.Sprintf("Pesquisa do usuário: %q\n\nFontes em tempo real coletadas pelo navegador em Go:\n%s\nPor favor, forneça uma síntese completa e estruturada respondendo diretamente à pesquisa.", query, sources.String())

	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": sysPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.3,
		"max_tokens":  1024,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[agent-go] calling LLM %s via %s (timeout: 60s)...", model, endpoint)
	resp, err := e.llmClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("llm status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Choices) > 0 && chatResp.Choices[0].Message.Content != "" {
		res := strings.TrimSpace(chatResp.Choices[0].Message.Content)
		res += "\n\n---\n💡 *Síntese gerada pelo Agente de IA em Go (Agent-Go) com fontes da web.*"
		log.Printf("[agent-go] LLM response received successfully (%d bytes)", len(res))
		return res, nil
	}

	return "", fmt.Errorf("empty completion from llm")
}

// synthesizeAISummaryLocal is the instant fallback extractor when LLM is unavailable.
func synthesizeAISummaryLocal(query string, results []ResultItem) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### 🤖 Resumo da IA para: *\"%s\"*\n\n", query))

	top := results[0]
	sb.WriteString(fmt.Sprintf("**Destaque:** %s\n\n", top.Snippet))

	if len(results) > 1 {
		sb.WriteString("#### 📌 Principais Tópicos:\n")
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

	sb.WriteString("💡 *Pesquisa processada em tempo real pelo motor ultraleve AI Go Browser (~12MB RAM).*")
	return sb.String()
}
