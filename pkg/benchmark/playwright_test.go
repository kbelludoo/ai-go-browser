package benchmark

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"ai-go-browser/browser"
)

func TestPlaywrightDependenciesNeverInstallMissingModule(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node not installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := DetectPlaywright(ctx, node, "aigo-benchmark-nonexistent-module"); err == nil || !strings.Contains(err.Error(), "no automatic installation") {
		t.Fatalf("expected missing dependency error, got %v", err)
	}
}

func TestPlaywrightDependencyEntryPointIsReusable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	deps, err := DetectPlaywright(ctx, "", "")
	if err != nil {
		t.Skipf("Playwright not installed: %v", err)
	}
	loaded, err := DetectPlaywright(ctx, deps.NodePath, deps.ModulePath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ModulePath != deps.ModulePath || loaded.Version != deps.Version {
		t.Fatalf("dependency changed after passing its resolved entry point: %+v/%+v", deps, loaded)
	}
}

func TestPlaywrightLaunchFailureRecordsAllTasks(t *testing.T) {
	r := RunPlaywright(context.Background(), Options{Run: 2}, PlaywrightDependencies{NodePath: "aigo-node-does-not-exist"})
	if len(r.Samples) != 7 {
		t.Fatalf("expected seven failed tasks, got %d", len(r.Samples))
	}
	for _, row := range r.Samples {
		if row.Success || row.Error == "" || row.NodeHeapUsedBytes != nil || !row.NotRun {
			t.Fatalf("invalid failure evidence: %+v", row)
		}
	}
}

func TestFrameworkWorkerFailurePreservesPlannedDenominator(t *testing.T) {
	run := FrameworkWorkerFailure(EngineAI, 3, errors.New("worker exited before JSON report"))
	if len(run.Samples) != 7 {
		t.Fatalf("worker failure changed the planned task denominator: %d", len(run.Samples))
	}
	report := NewFrameworkReport("same.exe", 1, []FrameworkRun{run})
	if report.Summaries[0].Cases != 7 || report.Summaries[0].Success != 0 {
		t.Fatalf("failed tasks were hidden: %+v", report.Summaries[0])
	}
	for _, row := range run.Samples {
		if !row.NotRun || row.NodeRSSBytes != nil || row.Assertion == "" {
			t.Fatalf("invalid not-run evidence: %+v", row)
		}
	}
}

func TestPlaywrightMissingMemoryIsUnavailable(t *testing.T) {
	var response playwrightResponse
	if err := json.Unmarshal([]byte(`{"ok":false,"error":"worker failure"}`), &response); err != nil {
		t.Fatal(err)
	}
	if response.NodeRSSBytes != nil || response.NodeHeapUsedBytes != nil {
		t.Fatal("absent Node readings were converted to measured zero values")
	}
}

func TestFrameworkReportSeparatesNodeAndChromiumMemory(t *testing.T) {
	nodeHeap := uint64(4 << 20)
	nodeRSS := uint64(12 << 20)
	chromeRSS := uint64(80 << 20)
	r := NewFrameworkReport("same-chromium.exe", 1, []FrameworkRun{{Engine: EnginePlaywright, Run: 1, WorkerWallMS: 30, Samples: []FrameworkSample{{Sample: Sample{Engine: EnginePlaywright, Case: "spa", Success: true, SetupMS: 5, TaskMS: 10, TotalMS: 15, SnapshotBytes: 123, ChromiumTreeRSSBytes: &chromeRSS}, NodeHeapUsedBytes: &nodeHeap, NodeRSSBytes: &nodeRSS}}}})
	got := r.Summaries[1]
	if got.Success != 1 || got.Cases != 1 || got.MeanWorkerWallMS != 30 || got.MeanCorpusMS != 15 {
		t.Fatalf("wrong comparison summary: %+v", got)
	}
	if *got.MaxSampledNodeRSSBytes != nodeRSS || *got.MaxSampledChromiumTreeRSSBytes != chromeRSS {
		t.Fatal("Node and Chromium RSS were not preserved separately")
	}
	for _, marker := range []string{"Node RSS", "Chromium tree RSS", "GoSys is not RSS", "not universal", "do not sum separate maxima"} {
		if !strings.Contains(r.Markdown(), marker) {
			t.Errorf("report omitted %q", marker)
		}
	}
}

func TestPlaywrightLocalTasksIntegration(t *testing.T) {
	if os.Getenv("AI_BROWSER_PLAYWRIGHT_INTEGRATION") != "1" {
		t.Skip("set AI_BROWSER_PLAYWRIGHT_INTEGRATION=1 to exercise real Playwright")
	}
	deps, err := DetectPlaywright(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	path, err := browser.FindSystemBrowser()
	if err != nil {
		t.Fatal(err)
	}
	r := RunPlaywright(context.Background(), Options{Run: 1, BrowserPath: path, Timeout: 12 * time.Second}, deps)
	for _, row := range r.Samples {
		if !row.Success {
			t.Errorf("%s: %s", row.Case, row.Error)
		}
		if row.Success && row.NodeRSSBytes == nil {
			t.Errorf("%s missing Node RSS", row.Case)
		}
	}
}
