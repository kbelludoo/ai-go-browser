package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"ai-go-browser/browser"
	"ai-go-browser/pkg/benchmark"
)

type urlsFlag []string

func (u *urlsFlag) String() string     { return fmt.Sprint([]string(*u)) }
func (u *urlsFlag) Set(v string) error { *u = append(*u, v); return nil }

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	f := flag.NewFlagSet("comparebrowser", flag.ContinueOnError)
	binary := f.String("browser", "", "Chromium/Chrome/Edge executable; the same binary is used for both engines")
	runs := f.Int("runs", 3, "repetitions per engine, each in a fresh worker")
	timeout := f.Duration("timeout", 12*time.Second, "per-case context deadline (AI Go also has internal command timeouts)")
	jsonPath := f.String("json", "benchmark-results/comparison.json", "JSON report path; - writes to stdout")
	markdownPath := f.String("markdown", "benchmark-results/comparison.md", "Markdown report path; empty disables")
	public := f.Bool("public", false, "enable read-only smoke checks on explicitly supplied -url values")
	worker := f.String("worker", "", "internal worker engine")
	iteration := f.Int("iteration", 1, "internal worker repetition")
	var urls urlsFlag
	f.Var(&urls, "url", "explicit public HTTP(S) URL; repeat to add another")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", f.Args())
	}
	if *runs < 1 || *runs > 100 {
		return errors.New("-runs must be between 1 and 100")
	}
	if *timeout <= 0 {
		return errors.New("-timeout must be positive")
	}
	if *public != (len(urls) > 0) {
		return errors.New("public mode requires both -public and at least one explicit -url")
	}
	if err := benchmark.ValidatePublicURLs(urls); err != nil {
		return err
	}
	if *binary == "" {
		var err error
		*binary, err = browser.FindSystemBrowser()
		if err != nil {
			return err
		}
	}
	opts := benchmark.Options{BrowserPath: *binary, Timeout: *timeout, PublicURLs: urls, Run: *iteration}
	if *worker != "" {
		if *worker != benchmark.EngineAI && *worker != benchmark.EngineChromium {
			return fmt.Errorf("unknown worker engine %q", *worker)
		}
		opts.Engine = *worker
		return json.NewEncoder(os.Stdout).Encode(benchmark.Run(context.Background(), opts))
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	var samples []benchmark.Sample
	for iteration := 1; iteration <= *runs; iteration++ {
		engines := []string{benchmark.EngineAI, benchmark.EngineChromium}
		if iteration%2 == 0 {
			engines[0], engines[1] = engines[1], engines[0]
		}
		for _, engine := range engines {
			fmt.Fprintf(os.Stderr, "Run %d/%d: %s\n", iteration, *runs, engine)
			workerArgs := []string{"-worker", engine, "-iteration", strconv.Itoa(iteration), "-browser", *binary, "-timeout", timeout.String()}
			if *public {
				workerArgs = append(workerArgs, "-public")
				for _, u := range urls {
					workerArgs = append(workerArgs, "-url", u)
				}
			}
			caseCount := 7
			if *public {
				caseCount = len(urls)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(caseCount)*(*timeout+30*time.Second)+30*time.Second)
			cmd := exec.CommandContext(ctx, executable, workerArgs...)
			cmd.Stderr = os.Stderr
			output, workerErr := cmd.Output()
			cancel()
			var rows []benchmark.Sample
			if workerErr == nil {
				workerErr = json.Unmarshal(output, &rows)
			}
			if workerErr != nil {
				rows = []benchmark.Sample{{Engine: engine, Run: iteration, Case: "worker_failure", Phase: "cold", Error: workerErr.Error(), HarnessRSSSource: "unavailable", ChromiumRSSSource: "unavailable"}}
			}
			samples = append(samples, rows...)
		}
	}
	report := benchmark.NewReport(*binary, *runs, *public, samples)
	data, err := report.JSON()
	if err != nil {
		return err
	}
	if err = writeReport(*jsonPath, append(data, '\n')); err != nil {
		return err
	}
	if *markdownPath != "" {
		if err = writeReport(*markdownPath, []byte(report.Markdown())); err != nil {
			return err
		}
	}
	for _, summary := range report.Summaries {
		fmt.Fprintf(os.Stderr, "%s: %d/%d assertions passed; corpus %.2f ms including startup\n", summary.Engine, summary.Succeeded, summary.Attempted, summary.MeanRunTotalMS)
	}
	for _, sample := range samples {
		if !sample.Success {
			return errors.New("comparison completed with failed tasks; see reports for case errors")
		}
	}
	return nil
}

func writeReport(path string, data []byte) error {
	if path == "" {
		return nil
	}
	if path == "-" {
		_, err := os.Stdout.Write(data)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
