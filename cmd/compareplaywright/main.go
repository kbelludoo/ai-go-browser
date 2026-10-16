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

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	f := flag.NewFlagSet("compareplaywright", flag.ContinueOnError)
	binary := f.String("browser", "", "same Chromium/Chrome/Edge executable for both engines")
	node := f.String("node", "", "Node executable; defaults to PATH")
	module := f.String("playwright-module", "", "installed Playwright package path or module name; no installation or browser download")
	runs := f.Int("runs", 3, "fresh worker repetitions per engine")
	timeout := f.Duration("timeout", 12*time.Second, "per-task timeout")
	jsonPath := f.String("json", "benchmark-results/playwright-comparison.json", "JSON report path; - writes to stdout")
	markdownPath := f.String("markdown", "benchmark-results/playwright-comparison.md", "Markdown report path; empty disables")
	worker := f.String("worker", "", "internal worker engine")
	iteration := f.Int("iteration", 1, "internal worker repetition")
	dependencyJSON := f.String("dependency-json", "", "internal preflight dependency metadata")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", f.Args())
	}
	if *runs < 1 || *runs > 100 {
		return errors.New("-runs must be between 1 and 100")
	}
	if *timeout <= 0 {
		return errors.New("-timeout must be positive")
	}
	if *binary == "" {
		var err error
		*binary, err = browser.FindSystemBrowser()
		if err != nil {
			return err
		}
	}
	opts := benchmark.Options{BrowserPath: *binary, Run: *iteration, Timeout: *timeout}
	if *worker == benchmark.EngineAI {
		return json.NewEncoder(os.Stdout).Encode(benchmark.RunAIFramework(context.Background(), opts))
	}
	var deps benchmark.PlaywrightDependencies
	var err error
	if *worker != "" && *dependencyJSON != "" {
		err = json.Unmarshal([]byte(*dependencyJSON), &deps)
		if err != nil || deps.NodePath == "" || deps.ModulePath == "" {
			return fmt.Errorf("invalid internal dependency metadata: %v", err)
		}
	} else {
		probeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		deps, err = benchmark.DetectPlaywright(probeCtx, *node, *module)
		cancel()
		if err != nil {
			return err
		}
	}
	if *worker != "" {
		if *worker != benchmark.EnginePlaywright {
			return fmt.Errorf("unknown worker engine %q", *worker)
		}
		return json.NewEncoder(os.Stdout).Encode(benchmark.RunPlaywright(context.Background(), opts, deps))
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	var results []benchmark.FrameworkRun
	encodedDependencies, _ := json.Marshal(deps)
	for n := 1; n <= *runs; n++ {
		engines := []string{benchmark.EngineAI, benchmark.EnginePlaywright}
		if n%2 == 0 {
			engines[0], engines[1] = engines[1], engines[0]
		}
		for _, engine := range engines {
			fmt.Fprintf(os.Stderr, "Run %d/%d: %s\n", n, *runs, engine)
			workerArgs := []string{"-worker", engine, "-iteration", strconv.Itoa(n), "-browser", *binary, "-node", deps.NodePath, "-playwright-module", deps.ModulePath, "-dependency-json", string(encodedDependencies), "-timeout", timeout.String()}
			ctx, cancel := context.WithTimeout(context.Background(), 7*(*timeout+30*time.Second)+60*time.Second)
			command := exec.CommandContext(ctx, executable, workerArgs...)
			command.Stderr = os.Stderr
			start := time.Now()
			output, workerErr := command.Output()
			wallMS := float64(time.Since(start)) / float64(time.Millisecond)
			cancel()
			var result benchmark.FrameworkRun
			if workerErr == nil {
				workerErr = json.Unmarshal(output, &result)
			}
			if workerErr != nil {
				result = benchmark.FrameworkWorkerFailure(engine, n, workerErr)
			}
			result.WorkerWallMS = wallMS
			results = append(results, result)
		}
	}
	report := benchmark.NewFrameworkReport(*binary, *runs, results)
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
		fmt.Fprintf(os.Stderr, "%s: %d/%d passed, corpus %.2f ms, worker wall %.2f ms\n", summary.Engine, summary.Success, summary.Cases, summary.MeanCorpusMS, summary.MeanWorkerWallMS)
	}
	for _, result := range results {
		for _, sample := range result.Samples {
			if !sample.Success {
				return errors.New("comparison finished with failed task assertions; see written reports")
			}
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
