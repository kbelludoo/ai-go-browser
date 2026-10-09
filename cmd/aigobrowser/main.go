package main

import (
	"ai-go-browser/browser"
	"bufio"
	"encoding/json"
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
func main() {
	hosts := flag.String("allow-hosts", "", "Comma-separated domain/IP allowlist. Recommended for agents")
	allowPost := flag.Bool("allow-post", false, "Allow POST form submissions with confirm:true")
	selftest := flag.Bool("demo", false, "Start internal test pages, print JSON results, and exit")
	demoEvents := flag.Bool("demo-events", false, "Demonstrate persistent JavaScript click events (needs -js-worker)")
	demoActions := flag.Bool("demo-actions", false, "Demonstrate Action Resolver discovering and fetching JSON without clicking")
	jsWorker := flag.String("js-worker", "", "Optional external Duktape JS worker path (classic inline scripts only)")
	scanScripts := flag.Bool("scan-scripts", false, "Discover literal endpoints in same-origin external JS (up to 4 files, 256 KiB each)")
	maxText := flag.Int("max-text", 5000, "Maximum visible text characters returned per page")
	flag.Parse()
	b := browser.New(strings.Split(*hosts, ","), *allowPost)
	b.MaxText = *maxText
	b.ScanScripts = *scanScripts
	b.JSWorker = *jsWorker
	defer b.Close()
	emit := func(r browser.Result) { _ = json.NewEncoder(os.Stdout).Encode(r) }
	if *selftest || *demoEvents || *demoActions {
		u, cleanup, e := demo()
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
		defer cleanup()
		// Internal demo pages are explicitly authorized; normal interactive
		// mode requires -allow-hosts and denies network access by default.
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
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		c, e := browser.Decode(scanner.Bytes())
		if e != nil {
			emit(browser.Result{Error: e.Error(), Memory: browser.Result{}.Memory})
			continue
		}
		r := b.Execute(c)
		emit(r)
		if c.Cmd == "quit" {
			break
		}
	}
	if e := scanner.Err(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
