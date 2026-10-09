package main

import (
	"ai-go-browser/browser"
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func demo() (string, func(), error) {
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		return "", nil, e
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "go-demo", Path: "/", HttpOnly: true})
		fmt.Fprint(w, `<!doctype html><html><head><title>AI Browser Demo</title><style>.x {color:red}</style></head><body><main><p id="estado">Sem JavaScript</p><button id="contar" type="button">Contar</button><button id="produtos" type="button">Ver produtos</button><h1>Página inicial</h1><p>Navegação para agente IA.</p><a href="/artigo"><span>Ler</span> artigo</a><form action="/busca" method="GET"><label>Busca</label><input name="q" aria-label="Pesquisa"><button>Pesquisar</button></form><script>document.getElementById('estado').textContent='JavaScript ativo';document.write('<a href="/artigo">Link dinâmico</a>');var n=0;document.getElementById('contar').addEventListener('click',function(){n++;document.getElementById('estado').textContent='Cliques: '+n;});document.getElementById('produtos').addEventListener('click',function(){fetch('/api/produtos')})</script></main></body></html>`)
	})
	mux.HandleFunc("/api/produtos", func(w http.ResponseWriter, r *http.Request) {
		if _, e := r.Cookie("session"); e != nil {
			http.Error(w, "session missing", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"products":[{"name":"Caderno","price":12.5},{"name":"Caneta","price":2.9}],"api_token":"masked"}`)
	})
	mux.HandleFunc("/artigo", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<title>Artigo local</title><h1>Artigo</h1><p>Funcionou sem Chromium.</p><a href="/">Voltar</a>`)
	})
	mux.HandleFunc("/busca", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<title>Busca</title><h1>Resultado</h1><p>Pesquisa: %s</p>`, r.URL.Query().Get("q"))
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go srv.Serve(l)
	return "http://" + l.Addr().String(), func() { _ = srv.Close() }, nil
}

func parseCommand(line string) (browser.Command, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return browser.Command{}, errors.New("empty command")
	}
	if strings.HasPrefix(line, "{") {
		return browser.Decode([]byte(line))
	}
	parts := strings.Fields(line)
	cmd := strings.ToLower(parts[0])
	switch cmd {
	case "open":
		if len(parts) < 2 {
			return browser.Command{}, errors.New("uso: open <url>")
		}
		return browser.Command{Cmd: "open", URL: parts[1]}, nil
	case "click":
		if len(parts) < 2 {
			return browser.Command{}, errors.New("uso: click <ref>")
		}
		return browser.Command{Cmd: "click", Ref: parts[1]}, nil
	case "press":
		if len(parts) < 2 {
			return browser.Command{}, errors.New("uso: press <ref>")
		}
		return browser.Command{Cmd: "press", Ref: parts[1]}, nil
	case "fill":
		if len(parts) < 3 {
			return browser.Command{}, errors.New("uso: fill <ref> <valor>")
		}
		return browser.Command{Cmd: "fill", Ref: parts[1], Value: strings.Join(parts[2:], " ")}, nil
	case "submit":
		if len(parts) < 2 {
			return browser.Command{}, errors.New("uso: submit <ref>")
		}
		return browser.Command{Cmd: "submit", Ref: parts[1], Confirm: true}, nil
	case "query":
		if len(parts) < 2 {
			return browser.Command{}, errors.New("uso: query <seletor>")
		}
		return browser.Command{Cmd: "query", Selector: strings.Join(parts[1:], " ")}, nil
	case "inspect":
		if len(parts) < 2 {
			return browser.Command{}, errors.New("uso: inspect <ref>")
		}
		return browser.Command{Cmd: "inspect", Ref: parts[1]}, nil
	case "run":
		if len(parts) < 2 {
			return browser.Command{}, errors.New("uso: run <ref>")
		}
		return browser.Command{Cmd: "run", Ref: parts[1], Confirm: true}, nil
	case "snapshot", "discover", "back", "markdown", "text", "help", "quit", "exit":
		if cmd == "exit" {
			cmd = "quit"
		}
		return browser.Command{Cmd: cmd}, nil
	default:
		// Se for uma URL digitada diretamente
		if strings.HasPrefix(cmd, "http://") || strings.HasPrefix(cmd, "https://") {
			return browser.Command{Cmd: "open", URL: parts[0]}, nil
		}
		return browser.Command{Cmd: cmd}, nil
	}
}

func printPrettyPage(p *browser.Page, showMarkdown bool) {
	if p == nil {
		return
	}
	fmt.Printf("\n=== [%d] %s ===\n", p.Status, p.Title)
	fmt.Printf("URL: %s (%d bytes)\n", p.URL, p.Bytes)
	if p.Description != "" {
		fmt.Printf("Descrição: %s\n", p.Description)
	}
	if p.Stack.Framework != "" {
		fmt.Printf("Tecnologia: %s (%s) | Encoding: %s | Rendimento de Texto: %.1f%%\n",
			p.Stack.Framework, p.Stack.RenderMode, p.Stack.Encoding, p.Stack.ContentYield)
	}
	if p.Stack.Notice != "" {
		fmt.Printf("⚠️  Aviso: %s\n", p.Stack.Notice)
	}
	if len(p.Headings) > 0 {
		fmt.Print("Seções: ")
		var heads []string
		for _, h := range p.Headings {
			heads = append(heads, h.Text)
		}
		if len(heads) > 5 {
			heads = append(heads[:5], fmt.Sprintf("... (+%d mais)", len(heads)-5))
		}
		fmt.Println(strings.Join(heads, " | "))
	}
	if len(p.Links) > 0 {
		fmt.Printf("Links (%d): ", len(p.Links))
		var lks []string
		for i, l := range p.Links {
			if i >= 6 {
				lks = append(lks, fmt.Sprintf("... (+%d mais)", len(p.Links)-6))
				break
			}
			lks = append(lks, fmt.Sprintf("[%s] %s", l.Ref, l.Label))
		}
		fmt.Println(strings.Join(lks, "  "))
	}
	if len(p.Forms) > 0 {
		fmt.Printf("Formulários (%d):\n", len(p.Forms))
		for _, f := range p.Forms {
			var fieldNames []string
			for _, fld := range f.Fields {
				fieldNames = append(fieldNames, fmt.Sprintf("[%s: %s (%s)]", fld.Ref, fld.Name, fld.Type))
			}
			fmt.Printf("  - %s: %s %s -> %s\n", f.Ref, f.Method, f.Action, strings.Join(fieldNames, " "))
		}
	}
	if showMarkdown && p.Markdown != "" {
		fmt.Println("\n--- Conteúdo (Markdown) ---")
		fmt.Println(p.Markdown)
		fmt.Println("---------------------------")
	} else if p.Text != "" {
		fmt.Println("\n--- Texto ---")
		fmt.Println(p.Text)
		fmt.Println("-------------")
	}
}

func main() {
	hosts := flag.String("allow-hosts", "", "Comma-separated domain/IP allowlist")
	allowAll := flag.Bool("allow-all", true, "Allow navigation to any public domain/IP (default: true)")
	allowPost := flag.Bool("allow-post", false, "Allow POST form submissions with confirm:true")
	selftest := flag.Bool("demo", false, "Start internal test pages, print JSON results, and exit")
	demoEvents := flag.Bool("demo-events", false, "Demonstrate persistent JavaScript click events (needs -js-worker)")
	demoActions := flag.Bool("demo-actions", false, "Demonstrate Action Resolver discovering and fetching JSON without clicking")
	jsWorker := flag.String("js-worker", "", "Optional external Duktape JS worker path (classic inline scripts only)")
	scanScripts := flag.Bool("scan-scripts", false, "Discover literal endpoints in same-origin external JS")
	maxText := flag.Int("max-text", 10000, "Maximum visible text characters returned per page")
	userAgent := flag.String("user-agent", "", "Custom User-Agent header")
	format := flag.String("format", "json", "Output format: json, markdown, or pretty")
	flag.Parse()

	var allowedHosts []string
	if *hosts != "" {
		allowedHosts = strings.Split(*hosts, ",")
	}
	b := browser.New(allowedHosts, *allowPost)
	b.MaxText = *maxText
	b.ScanScripts = *scanScripts
	b.JSWorker = *jsWorker
	b.UserAgent = *userAgent
	if *hosts == "" && *allowAll {
		b.AllowAll = true
	}
	defer b.Close()

	emit := func(r browser.Result) {
		if *format == "json" {
			_ = json.NewEncoder(os.Stdout).Encode(r)
			return
		}
		if !r.OK {
			fmt.Fprintf(os.Stderr, "Erro: %s\n", r.Error)
			return
		}
		if *format == "markdown" && r.Page != nil && r.Page.Markdown != "" {
			fmt.Println(r.Page.Markdown)
			return
		}
		if r.Page != nil {
			printPrettyPage(r.Page, *format == "markdown")
		} else if len(r.Actions) > 0 {
			fmt.Printf("Ações encontradas (%d):\n", len(r.Actions))
			for _, a := range r.Actions {
				fmt.Printf("  - [%s] %s (%s %s) [%s]\n", a.Ref, a.Label, a.Method, a.URL, a.Kind)
			}
		}
	}

	if *selftest || *demoEvents || *demoActions {
		u, cleanup, e := demo()
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
		defer cleanup()
		b.Allowed["127.0.0.1"] = true
		if *demoActions {
			opening := b.Execute(browser.Command{Cmd: "open", URL: u})
			emit(opening)
			if !opening.OK {
				os.Exit(1)
			}
			discovered := b.Execute(browser.Command{Cmd: "discover"})
			emit(discovered)
			if !discovered.OK {
				os.Exit(1)
			}
			ref := ""
			for _, a := range discovered.Actions {
				if a.Label == "Ver produtos" && a.Kind == "api" {
					ref = a.Ref
					break
				}
			}
			if ref == "" {
				fmt.Fprintln(os.Stderr, "API products action not found")
				os.Exit(1)
			}
			result := b.Execute(browser.Command{Cmd: "run", Ref: ref, Confirm: true})
			emit(result)
			if !result.OK {
				os.Exit(1)
			}
			return
		}
		commands := []browser.Command{{Cmd: "open", URL: u}, {Cmd: "click", Ref: "l1"}, {Cmd: "back"}, {Cmd: "fill", Ref: "i1", Value: "navegador em Go"}, {Cmd: "submit", Ref: "f1"}}
		if *demoEvents {
			if b.JSWorker == "" {
				fmt.Fprintln(os.Stderr, "-demo-events requires -js-worker")
				os.Exit(2)
			}
			commands = []browser.Command{{Cmd: "open", URL: u}, {Cmd: "press", Ref: "b1"}, {Cmd: "press", Ref: "b1"}, {Cmd: "query", Selector: "#estado"}}
		}
		for _, c := range commands {
			r := b.Execute(c)
			emit(r)
			if !r.OK {
				os.Exit(1)
			}
		}
		return
	}

	// Se uma URL foi passada como argumento posicional: ex: ./aigobrowser https://site.com
	args := flag.Args()
	if len(args) > 0 {
		targetURL := args[0]
		if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
			targetURL = "https://" + targetURL
		}
		r := b.Execute(browser.Command{Cmd: "open", URL: targetURL})
		emit(r)
		if !r.OK {
			os.Exit(1)
		}
		return
	}

	fi, _ := os.Stdin.Stat()
	isInteractive := (fi.Mode() & os.ModeCharDevice) != 0

	if isInteractive {
		fmt.Println("AI Go Browser v0.6 — Navegador Headless Ultra-Leve (<30 MB RAM)")
		fmt.Println("Digite 'open <url>', 'snapshot', 'markdown', 'text', 'links', 'help' ou comandos JSON.")
		fmt.Println("Para sair, digite 'quit'.")
		fmt.Print("ai-browser> ")
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4096), 2*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			if isInteractive {
				fmt.Print("ai-browser> ")
			}
			continue
		}
		c, e := parseCommand(line)
		if e != nil {
			if isInteractive {
				fmt.Fprintf(os.Stderr, "Comando inválido: %s\n", e)
				fmt.Print("ai-browser> ")
			} else {
				emit(browser.Result{Error: e.Error()})
			}
			continue
		}
		r := b.Execute(c)
		if isInteractive {
			if !r.OK {
				fmt.Fprintf(os.Stderr, "Erro: %s\n", r.Error)
			} else if c.Cmd == "markdown" && r.Page != nil {
				fmt.Println(r.Page.Markdown)
			} else if c.Cmd == "text" && r.Page != nil {
				fmt.Println(r.Page.Text)
			} else if r.Page != nil {
				printPrettyPage(r.Page, false)
			} else if len(r.Actions) > 0 {
				fmt.Printf("Ações encontradas (%d):\n", len(r.Actions))
				for _, a := range r.Actions {
					fmt.Printf("  - [%s] %s (%s %s) [%s]\n", a.Ref, a.Label, a.Method, a.URL, a.Kind)
				}
			} else {
				_ = json.NewEncoder(os.Stdout).Encode(r)
			}
			fmt.Printf("[RAM: Heap %.2f MiB | Sys %.2f MiB | Duração: %d ms]\n", r.Memory.HeapMiB, r.Memory.RSSMiB, r.DurationMs)
		} else {
			emit(r)
		}
		if c.Cmd == "quit" {
			break
		}
		if isInteractive {
			fmt.Print("ai-browser> ")
		}
	}
	if e := scanner.Err(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
