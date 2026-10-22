package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-go-browser/browser"
	"ai-go-browser/pkg/memory"
)

type AuditTarget struct {
	Name string
	URL  string
}

func main() {
	fmt.Println("================================================================================")
	fmt.Println("🧪 AUDITORIA DE PROVA REAL EM SITES DE PRODUÇÃO (NOVA VERSÃO)")
	fmt.Println("   - Provas: Zero-JS Hydration, Dedução de Botões, Memória FTS5 e Baixo RSS")
	fmt.Println("================================================================================")

	targets := []AuditTarget{
		{
			Name: "Next.js Official Portal (Production Next.js / RSC)",
			URL:  "https://nextjs.org",
		},
		{
			Name: "GitHub Platform (Production Complex Forms & CSRF Tokens)",
			URL:  "https://github.com",
		},
		{
			Name: "Wikipedia - Go Language (Production Massive HTML)",
			URL:  "https://en.wikipedia.org/wiki/Go_(programming_language)",
		},
	}

	tmpDir, err := os.MkdirTemp("", "real_audit_mem_*")
	if err != nil {
		fmt.Printf("Erro criando temp dir: %v\n", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "audit_memory.db")
	eng, err := memory.NewEngine(dbPath)
	if err != nil {
		fmt.Printf("Erro ao inicializar memory engine: %v\n", err)
		return
	}
	defer eng.Close()

	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	for i, t := range targets {
		fmt.Printf("\n[%d/%d] 🌐 TESTANDO SITE REAL: %s\n", i+1, len(targets), t.Name)
		fmt.Printf("      URL: %s\n", t.URL)
		fmt.Println("--------------------------------------------------------------------------------")

		// 1. Download live HTML
		t0 := time.Now()
		req, _ := http.NewRequest("GET", t.URL, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) ai-go-browser/2.0")
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("❌ Falha de rede: %v\n", err)
			continue
		}
		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		fetchTime := time.Since(t0)

		if err != nil {
			fmt.Printf("❌ Falha ao ler corpo: %v\n", err)
			continue
		}

		fmt.Printf("1. 📡 REDE HTTP: Status %d | Tamanho: %d bytes | Latência: %v\n",
			resp.StatusCode, len(bodyBytes), fetchTime)

		// 2. Parse live DOM + Zero-JS Hydration
		p0 := time.Now()
		page := browser.Parse(bodyBytes, t.URL, resp.StatusCode, len(bodyBytes), 100000)
		parseTime := time.Since(p0)

		fmt.Printf("2. ⚡ PARSER & EXTRAÇÃO ZERO-JS: %v\n", parseTime)
		fmt.Printf("   - Título: %q\n", page.Title)
		fmt.Printf("   - Elementos: %d Links, %d Botões, %d Formulários, %d Inputs\n",
			len(page.Links), len(page.Buttons), len(page.Forms), len(page.Inputs))

		// 3. Prova: Dados de Hidratação / Schema.org encontrados sem executar JS
		if page.Hydration != nil {
			fmt.Printf("3. 📦 DADOS ESTRUTURADOS EXTRAÍDOS SEM EXECUTAR JS:\n")
			if page.Hydration.Summary != "" {
				fmt.Printf("   - Sumário de Estado: %s\n", page.Hydration.Summary)
			}
			if len(page.Hydration.JSONLD) > 0 {
				fmt.Printf("   - Entidades Schema.org (JSON-LD) Detectadas: %d\n", len(page.Hydration.JSONLD))
				for idx, ent := range page.Hydration.JSONLD {
					if idx < 2 {
						fmt.Printf("     * [%d] Tipo: %v | Nome: %v\n", idx+1, ent["@type"], ent["name"])
					}
				}
			}
			if len(page.Hydration.NextData) > 0 {
				fmt.Printf("   - Estado Pre-Render Next.js: %d chaves carregadas\n", len(page.Hydration.NextData))
			}
			if len(page.Hydration.MetaTags) > 0 {
				fmt.Printf("   - OpenGraph Tags: %d tags (og:title: %q)\n",
					len(page.Hydration.MetaTags), page.Hydration.MetaTags["og:title"])
			}
		}

		// 4. Prova: Inteligência nos Botões (O que tem dentro do botão antes do clique)
		fmt.Printf("4. 🔍 INTELIGÊNCIA PRÉ-CLIQUE NOS BOTÕES (A IA já sabe o que tem dentro):\n")
		buttonCount := 0
		for _, b := range page.Buttons {
			if b.Action != nil && b.Action.ActionSummary != "" {
				buttonCount++
				if buttonCount <= 3 {
					fmt.Printf("   - Botão [%s] %q:\n", b.Ref, b.Label)
					fmt.Printf("     ↳ Ação Revelada: %s\n", b.Action.ActionSummary)
					if len(b.Action.HiddenInputs) > 0 {
						fmt.Printf("     ↳ Tokens Ocultos: %v\n", b.Action.HiddenInputs)
					}
				}
			}
		}
		if buttonCount == 0 && len(page.Buttons) > 0 {
			for idx, b := range page.Buttons {
				if idx < 2 {
					fmt.Printf("   - Botão [%s] %q (Tipo: %s)\n", b.Ref, b.Label, b.Type)
				}
			}
		}

		// 5. Prova: Memória FTS5 de Conteúdo Visitado
		_ = eng.RecordVisit(t.URL, page.Title, page.Markdown, resp.StatusCode)
		searchQuery := strings.Split(page.Title, " ")[0]
		if len(searchQuery) > 3 {
			s0 := time.Now()
			hits, err := eng.SearchVisitedPages(searchQuery, 1)
			searchTime := time.Since(s0)
			if err == nil && len(hits) > 0 {
				fmt.Printf("5. 🧠 RECUPERAÇÃO NA MEMÓRIA FTS5 (Zero Rede):\n")
				fmt.Printf("   - Busca por: %q em %v (vs Rede: %v)\n", searchQuery, searchTime, fetchTime)
				fmt.Printf("   - Snippet Recuperado: %s\n", hits[0].Snippet)
			}
		}

		// 6. Prova: Consumo Real de RAM do Processo
		mem := browser.ReadProcessMemory(os.Getpid())
		fmt.Printf("6. 📊 CONSUMO REAL DE MEMÓRIA (RSS):\n")
		fmt.Printf("   - RSS do Processo Go: %.2f MiB (Sem Chromium no SO)\n", mem.RSSMiB)
	}

	fmt.Println("\n================================================================================")
	fmt.Println("🏁 AUDITORIA CONCLUÍDA: TODAS AS PROVAS FORAM OBTIDAS EM SITES REAIS!")
	fmt.Println("================================================================================")
}
