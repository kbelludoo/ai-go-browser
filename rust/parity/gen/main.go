// Command gen produces the golden snapshots that the Rust port is compared
// against. It runs the Go reference implementation over every fixture in
// rust/parity/fixtures and writes the agent-facing page fields to
// rust/parity/golden, so a differential test can detect drift instead of
// relying on hand-written expectations.
//
// Run from the repository root:
//
//	go run ./rust/parity/gen
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ai-go-browser/browser"
)

// golden keeps only the fields a static HTTP fetch exposes, so the file is
// stable across unrelated Go changes (JS stack detection, timing, and so on).
type golden struct {
	Fixture  string            `json:"fixture"`
	URL      string            `json:"url"`
	Title    string            `json:"title"`
	Headings []browser.Heading `json:"headings"`
	Links    []browser.Link    `json:"links"`
	Buttons  []browser.Button  `json:"buttons"`
	Inputs   []browser.Field   `json:"inputs"`
	Forms    []browser.Form    `json:"forms"`
}

func main() {
	fixtures := flag.String("fixtures", "rust/parity/fixtures", "directory with HTML fixtures")
	out := flag.String("out", "rust/parity/golden", "output directory for golden JSON")
	textLimit := flag.Int("text-limit", 120, "text limit passed to Document.Page")
	flag.Parse()

	entries, err := os.ReadDir(*fixtures)
	if err != nil {
		fail("read fixtures: %v", err)
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".html") {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		fail("no .html fixtures in %s", *fixtures)
	}
	sort.Strings(names)

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail("create output dir: %v", err)
	}

	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(*fixtures, name))
		if err != nil {
			fail("read %s: %v", name, err)
		}
		base := "https://example.test/" + strings.TrimSuffix(name, ".html")
		document, err := browser.ParseDocument(raw, base, 200, len(raw))
		if err != nil {
			fail("parse %s: %v", name, err)
		}
		page := document.Page(*textLimit)

		record := golden{
			Fixture:  name,
			URL:      page.URL,
			Title:    page.Title,
			Headings: nonNil(page.Headings),
			Links:    nonNil(page.Links),
			Buttons:  nonNil(page.Buttons),
			Inputs:   nonNil(page.Inputs),
			Forms:    nonNil(page.Forms),
		}
		encoded, err := json.MarshalIndent(record, "", "  ")
		if err != nil {
			fail("encode %s: %v", name, err)
		}
		target := filepath.Join(*out, strings.TrimSuffix(name, ".html")+".json")
		if err := os.WriteFile(target, append(encoded, '\n'), 0o644); err != nil {
			fail("write %s: %v", target, err)
		}
		fmt.Printf("%-16s links=%d buttons=%d inputs=%d headings=%d\n",
			name, len(page.Links), len(page.Buttons), len(page.Inputs), len(page.Headings))
	}
}

func nonNil[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gen: "+format+"\n", args...)
	os.Exit(1)
}
