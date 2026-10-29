// Go reference for a reproducible native Go vs Odin *static* page benchmark.
// It imports the actual browser package, not an imitation of its parser.
// Scope: parse HTML and compact CSO. This DOES NOT benchmark JS, CDP, HTTP or TLS.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"ai-go-browser/browser"
)

type Heading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}
type Link struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}
type Projection struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Text        string    `json:"text"`
	Headings    []Heading `json:"headings"`
	Links       []Link    `json:"links"`
	CSO         string    `json:"cso"`
}

func project(p browser.Page) Projection {
	result := Projection{
		Title:       p.Title,
		Description: p.Description,
		Text:        p.Text,
		CSO:         p.ToCSO(15000),
		Headings:    make([]Heading, 0, len(p.Headings)),
		Links:       make([]Link, 0, len(p.Links)),
	}
	for _, h := range p.Headings {
		result.Headings = append(result.Headings, Heading{h.Level, h.Text})
	}
	for _, l := range p.Links {
		result.Links = append(result.Links, Link{l.Label, l.URL})
	}
	return result
}

// Live baseline uses the repository's real Browser.Execute on a localhost
// HTTP server; the sandboxed Go browser is not replaced with net/http code.
func agentMode() {
	b := browser.New([]string{"127.0.0.1"}, false)
	b.Engine = "http"
	b.NativeOnly = true
	b.MaxText = 200000
	b.CompactSnapshots = true
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 65536), 1048576)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var req struct {
			Cmd      string `json:"cmd"`
			URL      string `json:"url"`
			Selector string `json:"selector"`
			Ref      string `json:"ref"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			_ = encoder.Encode(map[string]interface{}{"ok": false, "error": "invalid JSON"})
			continue
		}
		if req.Cmd == "quit" {
			_ = encoder.Encode(map[string]interface{}{"ok": true})
			return
		}
		if req.Cmd == "query" {
			if b.Document == nil {
				_ = encoder.Encode(map[string]interface{}{"ok": false, "error": "no page open"})
				continue
			}
			matches, err := b.Document.Query(req.Selector)
			if err != nil {
				_ = encoder.Encode(map[string]interface{}{"ok": false, "error": err.Error()})
				continue
			}
			_ = encoder.Encode(map[string]interface{}{"ok": true, "matches": matches})
			continue
		}
		if req.Cmd == "follow" || req.Cmd == "click" {
			r := b.Execute(browser.Command{Cmd: "click", Ref: req.Ref})
			if !r.OK {
				_ = encoder.Encode(map[string]interface{}{"ok": false, "error": r.Error})
				continue
			}
			p := b.Current
			if p == nil {
				p = r.Page
			}
			if p == nil {
				_ = encoder.Encode(map[string]interface{}{"ok": false, "error": "no parsed page"})
				continue
			}
			_ = encoder.Encode(map[string]interface{}{"ok": true, "page": project(*p)})
			continue
		}
		if req.Cmd == "back" {
			r := b.Execute(browser.Command{Cmd: "back"})
			if !r.OK {
				_ = encoder.Encode(map[string]interface{}{"ok": false, "error": r.Error})
				continue
			}
			p := b.Current
			if p == nil {
				p = r.Page
			}
			if p == nil {
				_ = encoder.Encode(map[string]interface{}{"ok": false, "error": "no parsed page"})
				continue
			}
			_ = encoder.Encode(map[string]interface{}{"ok": true, "page": project(*p)})
			continue
		}
		if req.Cmd != "open" && req.Cmd != "snapshot" {
			_ = encoder.Encode(map[string]interface{}{"ok": false, "error": "unsupported Go benchmark command"})
			continue
		}
		r := b.Execute(browser.Command{Cmd: req.Cmd, URL: req.URL})
		if !r.OK {
			_ = encoder.Encode(map[string]interface{}{"ok": false, "error": r.Error})
			continue
		}
		p := b.Current
		if p == nil {
			p = r.Page
		}
		if p == nil {
			_ = encoder.Encode(map[string]interface{}{"ok": false, "error": "no parsed page"})
			continue
		}
		_ = encoder.Encode(map[string]interface{}{"ok": true, "page": project(*p)})
	}
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "agent" {
		agentMode()
		return
	}
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: go-reference parse-file|bench-file|query-file path url [iterations|selector]")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[2])
	if err != nil {
		panic(err)
	}
	action, url := os.Args[1], os.Args[3]
	switch action {
	case "parse-file":
		p := browser.Parse(data, url, 200, len(data), 0)
		if err := json.NewEncoder(os.Stdout).Encode(project(p)); err != nil {
			panic(err)
		}
	case "query-file":
		if len(os.Args) != 5 {
			panic("selector expected")
		}
		doc, err := browser.ParseDocument(data, url, 200, len(data))
		if err != nil {
			panic(err)
		}
		matches, err := doc.Query(os.Args[4])
		if err != nil {
			if err := json.NewEncoder(os.Stdout).Encode(map[string]interface{}{"ok": false, "error": err.Error()}); err != nil {
				panic(err)
			}
			return
		}
		if err := json.NewEncoder(os.Stdout).Encode(map[string]interface{}{"ok": true, "matches": matches}); err != nil {
			panic(err)
		}
	case "bench-file":
		if len(os.Args) != 5 {
			panic("iterations expected")
		}
		n, err := strconv.Atoi(os.Args[4])
		if err != nil || n < 1 || n > 200000 {
			panic("bad iterations")
		}
		sum := 0
		for i := 0; i < n; i++ {
			p := browser.Parse(data, url, 200, len(data), 0)
			s := p.ToCSO(15000)
			sum += len(p.Text) + len(p.Title) + len(s)
		}
		fmt.Printf("checksum=%d iterations=%d runtime=Go\n", sum, n)
	case "tokens":
		fmt.Println(browser.EstimateTokens(string(data)))
	case "markdown":
		fmt.Println(strings.TrimSpace(browser.MarkdownToCSO(string(data), 10000)))
	default:
		panic("unknown action")
	}
}
