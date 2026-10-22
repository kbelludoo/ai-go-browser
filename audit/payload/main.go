// Command payload measures what an agent actually pays for an observation.
//
// It serves two local fixtures (a text-heavy "realistic" page and a 2.000-link
// synthetic page), opens them with the pure-Go engine and reports:
//
//	raw HTML bytes | full snapshot JSON | compact snapshot JSON | view:"read" JSON
//	text chars, reference counts, and the latency of each observation
//
// Run: go run ./audit/payload
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"ai-go-browser/browser"
)

func realisticPage() string {
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Realistic Fixture</title><style>`)
	for i := 0; i < 600; i++ {
		fmt.Fprintf(&b, ".c%d{color:#%06x;margin:%dpx;padding:%dpx;display:flex;gap:%dpx}\n", i, i*2654435761%0xffffff, i%20, i%13, i%9)
	}
	b.WriteString("</style><script>")
	for i := 0; i < 400; i++ {
		fmt.Fprintf(&b, "function f%d(a,b){var x=a*%d+b;return x%%7===0?'ok':'bad';}\n", i, i+3)
	}
	b.WriteString("</script></head><body><header><nav>")
	for i := 0; i < 120; i++ {
		fmt.Fprintf(&b, "<a href=\"/p/%d\" class=\"c%d\">Nav %d</a>", i, i%600, i)
	}
	b.WriteString("</nav></header><main>")
	for i := 0; i < 400; i++ {
		fmt.Fprintf(&b, "<article class=\"c%d\"><h2>Section %d</h2><p>%s</p></article>", i%600, i,
			strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 12))
	}
	b.WriteString("</main><footer>")
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&b, "<a href=\"/f/%d\">Footer link %d</a>", i, i)
	}
	b.WriteString("</footer></body></html>")
	return b.String()
}

func linkyPage(n int) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><title>2000 links</title></head><body><h1>Link farm</h1><ul>")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "<li><a href=\"/item/%d\">Item number %d with some label text</a> <button>Open %d</button></li>\n", i, i, i)
	}
	b.WriteString("</ul></body></html>")
	return b.String()
}

func serve(pages map[string]string) (string, func()) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	for path, body := range pages {
		body := body
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, body)
		})
	}
	srv := &http.Server{Handler: mux}
	go srv.Serve(l)
	return "http://" + l.Addr().String(), func() { srv.Close() }
}

func size(v any) int {
	b, _ := json.Marshal(v)
	return len(b)
}

func median(d []time.Duration) time.Duration {
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
	return d[len(d)/2]
}

func newBrowser(compact bool, maxText int) *browser.Browser {
	b := browser.New(nil, false)
	b.AllowAll = true
	b.Engine = "http"
	b.AutoRenderJS = false
	b.NativeOnly = true
	b.SetBlockPrivateNetworks(false)
	b.CompactSnapshots = compact
	b.MaxText = maxText
	return b
}

func run(name, url, page string) {
	fmt.Printf("\n== %s\n", name)
	fmt.Printf("raw HTML bytes ............ %d\n", len(page))

	// Library defaults: MaxText 5.000, compact off.
	b := newBrowser(false, 5000)
	defer b.Close()
	if open := b.Execute(browser.Command{Cmd: "open", URL: url}); !open.OK {
		fmt.Println("open error:", open.Error)
		return
	}
	full := b.Execute(browser.Command{Cmd: "snapshot"})
	read := b.Execute(browser.Command{Cmd: "snapshot", View: "read"})
	fmt.Printf("full snapshot JSON ........ %d bytes", size(full))
	if full.Page != nil {
		fmt.Printf("  (links=%d headings=%d buttons=%d markdown=%d chars)",
			len(full.Page.Links), len(full.Page.Headings), len(full.Page.Buttons), len(full.Page.Markdown))
	}
	fmt.Println()
	fmt.Printf("view:\"read\" JSON .......... %d bytes", size(read))
	if read.Page != nil {
		fmt.Printf("  (links=%d headings=%d buttons=%d forms=%d text=%d chars)",
			len(read.Page.Links), len(read.Page.Headings), len(read.Page.Buttons),
			len(read.Page.Forms), len(read.Page.Text))
	}
	fmt.Println()
	if full.Page != nil && read.Page != nil {
		fmt.Printf("ratio full/read .......... %.1fx\n", float64(size(full))/float64(size(read)))
		fmt.Printf("ratio HTML/read .......... %.1fx\n", float64(len(page))/float64(size(read)))
	}

	// CLI default: -compact=true, -max-text=10000.
	c := newBrowser(true, 10000)
	defer c.Close()
	c.Execute(browser.Command{Cmd: "open", URL: url})
	comp := c.Execute(browser.Command{Cmd: "snapshot"})
	if comp.Page != nil {
		fmt.Printf("CLI default snapshot ..... %d bytes (links=%d omitted=%d)  ratio HTML/compact %.1fx\n",
			size(comp), len(comp.Page.Links), comp.Page.OmittedLinks,
			float64(len(page))/float64(size(comp)))
	}

	var tf, tr []time.Duration
	for i := 0; i < 25; i++ {
		t0 := time.Now()
		b.Execute(browser.Command{Cmd: "snapshot"})
		tf = append(tf, time.Since(t0))
		t0 = time.Now()
		b.Execute(browser.Command{Cmd: "snapshot", View: "read"})
		tr = append(tr, time.Since(t0))
	}
	fmt.Printf("latency median ........... full=%v read=%v (in-process HTTP engine)\n", median(tf), median(tr))
}

func peakRSS() int64 {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmHWM:") {
			f := strings.Fields(line)
			if len(f) >= 2 {
				kb, _ := strconv.ParseInt(f[1], 10, 64)
				return kb * 1024
			}
		}
	}
	return 0
}

func main() {
	real := realisticPage()
	links := linkyPage(2000)
	url, stop := serve(map[string]string{"/real": real, "/links": links})
	defer stop()
	run("realistic text-heavy page (367 KB)", url+"/real", real)
	run("synthetic 2.000-link page (195 KB)", url+"/links", links)
	if rss := peakRSS(); rss > 0 {
		fmt.Printf("\npeak RSS of this harness (VmHWM): %.1f MiB\n", float64(rss)/1048576)
	}
}
