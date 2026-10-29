package main

// Generates HTTP behavior goldens from the Go reference and serves the same
// fixture endpoints for the Rust comparison.
//
//	go run ./rust/parity/httpgen              # write rust/parity/http-golden.json
//	go run ./rust/parity/httpgen -serve       # serve only (Rust replay mode)
//
// The Rust side spawns `-serve` itself, so a single `cargo run --example
// http_parity` performs the whole comparison.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ai-go-browser/browser"
)

type step struct {
	Cmd     string `json:"cmd"`
	URL     string `json:"url,omitempty"`
	Ref     string `json:"ref,omitempty"`
	Confirm bool   `json:"confirm,omitempty"`
}

type scenario struct {
	Name  string
	Start string
	Steps []step
}

type outcome struct {
	OK        bool   `json:"ok"`
	Error     string `json:"error,omitempty"`
	URL       string `json:"url,omitempty"`
	Status    int    `json:"status,omitempty"`
	Title     string `json:"title,omitempty"`
	Text      string `json:"text,omitempty"`
	TextSHA   string `json:"text_sha256,omitempty"`
	TextLen   int    `json:"text_len,omitempty"`
	Bytes     int    `json:"bytes,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
	Links     int    `json:"links,omitempty"`
	Buttons   int    `json:"buttons,omitempty"`
	Inputs    int    `json:"inputs,omitempty"`
	Forms     int    `json:"forms,omitempty"`
}

type scenarioGolden struct {
	Name     string          `json:"name"`
	Start    string          `json:"start"`
	Steps    []step          `json:"steps"`
	Requests []requestRecord `json:"requests"`
	Outcome  outcome         `json:"outcome"`
}

type goldenFile struct {
	Generator string           `json:"generator"`
	Scenarios []scenarioGolden `json:"scenarios"`
}

func scenarios() []scenario {
	open := func(name, path string) scenario {
		return scenario{
			Name:  name,
			Start: path + "?s=" + name,
			Steps: []step{{Cmd: "open", URL: path + "?s=" + name}},
		}
	}
	return []scenario{
		open("headers", "/plain"),
		open("gzip", "/gzip"),
		open("latin1", "/latin1"),
		open("meta-charset", "/meta-charset"),
		open("redirect", "/r/3"),
		open("redirect-limit", "/r/20"),
		open("not-found", "/missing"),
		open("no-type", "/no-type"),
		open("big-html", "/big"),
		{
			Name:  "cookies",
			Start: "/cookies/set?s=cookies",
			Steps: []step{{Cmd: "open", URL: "/cookies/set?s=cookies"}},
		},
		{
			Name:  "post-form",
			Start: "/form?s=post-form",
			Steps: []step{
				{Cmd: "open", URL: "/form?s=post-form"},
				{Cmd: "submit", Ref: "f1", Confirm: true},
			},
		},
	}
}

func main() {
	serve := flag.Bool("serve", false, "serve fixtures without writing goldens")
	out := flag.String("out", filepath.Join("rust", "parity", "http-golden.json"), "golden output path")
	serverInfo := flag.String("server-info", filepath.Join("rust", "parity", "http-server.json"), "live server descriptor")
	flag.Parse()

	server, recorder := newServer()
	defer server.Close()
	if *serve {
		if err := writeServerInfo(*serverInfo, server.URL); err != nil {
			fail(err)
		}
		fmt.Printf("listening %s\n", server.URL)
		select {}
	}

	file := goldenFile{Generator: "go", Scenarios: []scenarioGolden{}}
	for _, item := range scenarios() {
		result, err := runScenario(server.URL, item)
		if err != nil {
			fail(err)
		}
		result.Requests = recorder.take(item.Name)
		file.Scenarios = append(file.Scenarios, result)
		fmt.Printf("%-16s requests=%d ok=%v\n", item.Name, len(result.Requests), result.Outcome.OK)
	}
	if err := writeJSON(*out, file); err != nil {
		fail(err)
	}
	fmt.Printf("wrote %s\n", *out)
}

func runScenario(base string, item scenario) (scenarioGolden, error) {
	client := browser.New([]string{"127.0.0.1"}, true)
	defer client.Close()
	result := browser.Result{}
	for _, action := range item.Steps {
		result = client.Execute(browser.Command{Cmd: action.Cmd, URL: absolute(base, action.URL), Ref: action.Ref, Confirm: action.Confirm})
		if !result.OK {
			break
		}
	}
	return scenarioGolden{
		Name:    item.Name,
		Start:   item.Start,
		Steps:   item.Steps,
		Outcome: observe(base, result),
	}, nil
}

func observe(base string, result browser.Result) outcome {
	out := outcome{OK: result.OK, Error: normalizeError(base, result.Error)}
	page := result.Page
	if page == nil {
		return out
	}
	if os.Getenv("AIGB_PARITY_DEBUG") != "" {
		fmt.Printf("debug title=%q text=%q bytes=%d\n", page.Title, page.Text, page.Bytes)
	}
	text := page.Text
	sum := sha256.Sum256([]byte(text))
	out.URL = strings.TrimPrefix(page.URL, base)
	out.Status = page.Status
	out.Title = page.Title
	out.Text = truncate(text, 2000)
	out.TextSHA = hex.EncodeToString(sum[:])
	out.TextLen = len([]rune(text))
	out.Bytes = page.Bytes
	out.Truncated = page.Truncated
	out.Links = len(page.Links)
	out.Buttons = len(page.Buttons)
	out.Inputs = len(page.Inputs)
	out.Forms = len(page.Forms)
	return out
}

func normalizeError(base, message string) string {
	return strings.ReplaceAll(message, base, "")
}

func absolute(base, target string) string {
	if target == "" || strings.HasPrefix(target, "http") {
		return target
	}
	return base + target
}

func truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func writeServerInfo(path, base string) error {
	return writeJSON(path, map[string]string{"base_url": base})
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, errors.Unwrap(err))
	os.Exit(1)
}
