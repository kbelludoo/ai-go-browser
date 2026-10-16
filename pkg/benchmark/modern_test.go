package benchmark

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"ai-go-browser/browser"
)

func TestModernWorkerFailurePreservesFivePlannedTasks(t *testing.T) {
	run := ModernWorkerFailure(EngineAI, 2, errors.New("worker terminated"))
	if err := ValidateModernRun(run, EngineAI, 2); err != nil {
		t.Fatal(err)
	}
	for _, row := range run.Samples {
		if row.Success || !row.NotRun || row.Error != "worker terminated" || row.SnapshotBytes != 0 || row.ChromiumTreeRSSBytes != nil {
			t.Fatalf("failure manufactured a result or measurement: %+v", row)
		}
	}
	run.Samples = run.Samples[:4]
	if err := ValidateModernRun(run, EngineAI, 2); err == nil {
		t.Fatal("missing planned task accepted")
	}
}

func TestModernRejectsPublicURLsWithoutLaunchingEngines(t *testing.T) {
	opts := Options{Run: 2, PublicURLs: []string{"https://example.com"}}
	for _, result := range []ModernRun{
		RunModernAI(context.Background(), opts),
		RunModernPlaywright(context.Background(), opts, PlaywrightDependencies{}),
	} {
		if err := ValidateModernRun(result, result.Engine, 2); err != nil {
			t.Fatal(err)
		}
		for _, row := range result.Samples {
			if row.Success || !row.NotRun || !strings.Contains(row.Error, "local fixtures only") {
				t.Fatalf("public visit accepted: %+v", row)
			}
		}
	}
}

func TestModernPlaywrightMissingNodePreservesFiveTasks(t *testing.T) {
	result := RunModernPlaywright(context.Background(), Options{Run: 2}, PlaywrightDependencies{NodePath: "aigo-modern-node-does-not-exist"})
	if err := ValidateModernRun(result, EnginePlaywright, 2); err != nil {
		t.Fatal(err)
	}
	for _, row := range result.Samples {
		if row.Success || !row.NotRun || row.NodeRSSBytes != nil {
			t.Fatalf("startup error concealed or memory fabricated: %+v", row)
		}
	}
}

func TestModernLocalTasksIntegration(t *testing.T) {
	if os.Getenv("AI_BROWSER_MODERN_INTEGRATION") != "1" {
		t.Skip("set AI_BROWSER_MODERN_INTEGRATION=1 to exercise both modern adapters")
	}
	deps, err := DetectPlaywright(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	path, err := browser.FindSystemBrowser()
	if err != nil {
		t.Fatal(err)
	}
	opts := Options{Run: 1, BrowserPath: path, Timeout: 12 * time.Second}
	for _, result := range []ModernRun{RunModernAI(context.Background(), opts), RunModernPlaywright(context.Background(), opts, deps)} {
		if err := ValidateModernRun(result, result.Engine, 1); err != nil {
			t.Error(err)
		}
		if len(result.Evidence) != 5 {
			t.Errorf("%s recorded %d fixture evidence rows", result.Engine, len(result.Evidence))
		}
		for _, row := range result.Samples {
			if !row.Success {
				t.Errorf("%s %s: %s", result.Engine, row.Case, row.Error)
			}
		}
	}
}

func TestModernWorkerValidationRejectsMisattributedRows(t *testing.T) {
	for _, mutate := range []func(*ModernRun){
		func(r *ModernRun) { r.Engine = EnginePlaywright },
		func(r *ModernRun) { r.Samples[1].Case = r.Samples[0].Case },
		func(r *ModernRun) { r.Samples[1].Run++ },
		func(r *ModernRun) { r.Samples[1].Success = true },
	} {
		run := ModernWorkerFailure(EngineAI, 2, errors.New("failed"))
		mutate(&run)
		if err := ValidateModernRun(run, EngineAI, 2); err == nil {
			t.Fatal("invalid worker result accepted")
		}
	}
}

func TestModernObservedResultAlsoRequiresStrictHTTPOracle(t *testing.T) {
	c := modernFixtures()
	defer c.close()
	f := c.cases[2]
	c.record(f.Name, true)
	c.record(f.Name, true)
	row := modernRow(EngineAI, Options{Run: 1}, c, f, 2, 0)
	evidence := modernObserve(&row, snapshot{text: f.Expected, size: 123}, nil, time.Now(), c, f)
	if row.Success || evidence.Passed || !strings.Contains(row.Error, "fixture oracle") || row.SnapshotBytes != 123 {
		t.Fatalf("duplicate HTTP receipt passed displayed success: %+v %+v", row, evidence)
	}
}

func TestModernReportRetainsEvidenceAndPlannedDenominator(t *testing.T) {
	run := ModernWorkerFailure(EnginePlaywright, 1, errors.New("startup failure"))
	run.Evidence = []ModernEvidence{{Case: "native_validation_multipart", ExpectedRequests: 1, ObservedRequests: 2, ValidRequests: 2}}
	report := NewModernReport("same-chromium", 1, []ModernRun{run})
	if report.FixtureVersion != ModernFixtureVersion || report.Summaries[1].Cases != 5 || report.Summaries[1].Success != 0 {
		t.Fatalf("wrong corpus/denominator: %+v", report.Summaries)
	}
	data, err := report.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		FixtureVersion string      `json:"fixture_version"`
		Runs           []ModernRun `json:"runs"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.FixtureVersion != ModernFixtureVersion || len(parsed.Runs) != 1 || len(parsed.Runs[0].Samples) != 5 || len(parsed.Runs[0].Evidence) != 1 || parsed.Runs[0].Evidence[0].ObservedRequests != 2 {
		t.Fatalf("modern envelope lost evidence: %s", data)
	}
	if strings.Count(string(data), `"runs":`) != 1 || strings.Contains(string(data), `"node_rss_bytes": 0`) {
		t.Fatalf("ambiguous envelope or fabricated memory: %s", data)
	}
	markdown := report.Markdown()
	for _, want := range []string{ModernFixtureVersion, "0/5", "Independent fixture HTTP evidence", "startup failure", "Complete worker wall time includes both startup paths"} {
		if !strings.Contains(markdown, want) {
			t.Errorf("Markdown missing %q", want)
		}
	}
}

func modernTestFrameOracle(t *testing.T) (*modernFrameOracle, []browser.Frame) {
	t.Helper()
	frames := []browser.Frame{
		{Ref: "fr1", ID: "main-id", URL: "http://127.0.0.1:4000/frame-cross", Main: true, Allowed: true, Selected: true},
		{Ref: "fr4", ID: "child-id", ParentRef: "fr1", URL: "http://localhost:5000/frame-child?kind=cross", Name: "modern-cross", Allowed: true},
	}
	oracle, err := newModernFrameOracle(frames, "modern-cross", frames[0].URL)
	if err != nil {
		t.Fatal(err)
	}
	frames[0].Selected, frames[1].Selected = false, true
	return oracle, frames
}

func TestModernFrameOracleRejectsMainTabNavigationToChild(t *testing.T) {
	oracle, frames := modernTestFrameOracle(t)
	// The previous URL-only check accepted this visible result in the main tab.
	page := &browser.Page{FrameRef: "fr1", URL: oracle.child.URL, Text: "MODERN_FRAME_CROSS_OK"}
	if err := oracle.selectedPage(page); err == nil {
		t.Fatal("child URL observed in main scope accepted as a frame task")
	}
	// Even a falsely labeled page cannot pass when the main tab actually moved.
	page.FrameRef = oracle.child.Ref
	if err := oracle.selectedPage(page); err != nil {
		t.Fatal(err)
	}
	frames[0].URL = oracle.child.URL
	if err := oracle.unchangedTree(frames); err == nil {
		t.Fatal("main tab navigated to child URL without invalidating the task")
	}
}

func TestModernFrameOracleRequiresOriginalMainAndSelectedChild(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mutate func([]browser.Frame) []browser.Frame
	}{
		{"main removed", func(f []browser.Frame) []browser.Frame { return f[1:] }},
		{"main replaced", func(f []browser.Frame) []browser.Frame { f[0].ID = "replacement-main"; return f }},
		{"main changed URL", func(f []browser.Frame) []browser.Frame { f[0].URL += "?changed=1"; return f }},
		{"child replaced", func(f []browser.Frame) []browser.Frame { f[1].ID = "replacement-child"; return f }},
		{"child unselected", func(f []browser.Frame) []browser.Frame { f[1].Selected = false; return f }},
		{"child changed URL", func(f []browser.Frame) []browser.Frame { f[1].URL += "&changed=1"; return f }},
		{"two selected", func(f []browser.Frame) []browser.Frame { f[0].Selected = true; return f }},
		{"second main", func(f []browser.Frame) []browser.Frame { return append(f, browser.Frame{Main: true}) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			oracle, frames := modernTestFrameOracle(t)
			if err := oracle.unchangedTree(tt.mutate(frames)); err == nil {
				t.Fatal("invalid frame identity/scope accepted")
			}
		})
	}
	oracle, frames := modernTestFrameOracle(t)
	if err := oracle.selectedPage(&browser.Page{FrameRef: oracle.child.Ref, URL: oracle.child.URL}); err != nil {
		t.Fatal(err)
	}
	if err := oracle.unchangedTree(frames); err != nil {
		t.Fatal(err)
	}
}

func TestModernFrameDiscoveryRejectsAmbiguousOrMissingIdentity(t *testing.T) {
	for _, mutate := range []func([]browser.Frame) []browser.Frame{
		func(f []browser.Frame) []browser.Frame { f[0].URL = f[1].URL; return f },
		func(f []browser.Frame) []browser.Frame { f[0].ID = ""; return f },
		func(f []browser.Frame) []browser.Frame { f[1].Ref = ""; return f },
		func(f []browser.Frame) []browser.Frame { f[1].Allowed = false; return f },
		func(f []browser.Frame) []browser.Frame { return append(f, f[1]) },
	} {
		_, frames := modernTestFrameOracle(t)
		originalURL := frames[0].URL
		if _, err := newModernFrameOracle(mutate(frames), "modern-cross", originalURL); err == nil {
			t.Fatal("ambiguous/missing frame discovery accepted")
		}
	}
}
