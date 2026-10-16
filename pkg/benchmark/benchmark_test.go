package benchmark

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"ai-go-browser/browser"
)

func TestLocalHTTPTasks(t *testing.T) {
	server, cases := localFixtures()
	defer server.Close()
	d, err := newDriver(context.Background(), Options{Engine: EngineAI}, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer d.close()
	for _, f := range cases {
		if f.javascript {
			continue
		}
		t.Run(f.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := d.open(ctx, server.URL+f.path, false); err != nil {
				t.Fatal(err)
			}
			if err := d.action(ctx, f); err != nil {
				t.Fatal(err)
			}
			observed, err := d.snapshot(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(observed.text, f.expected) {
				t.Fatalf("assertion missing %q in %q", f.expected, observed.text)
			}
			if observed.size <= 0 || observed.document <= 0 {
				t.Fatalf("invalid measured byte counts: %d/%d", observed.size, observed.document)
			}
		})
	}
}

func TestUnknownEngineRecordsFailures(t *testing.T) {
	rows := Run(context.Background(), Options{Engine: "unknown"})
	if len(rows) != 7 {
		t.Fatalf("expected all seven local cases, got %d", len(rows))
	}
	for i, row := range rows {
		if row.Success || !strings.Contains(row.Error, "unknown engine") {
			t.Fatalf("invalid failure result: %+v", row)
		}
		if i == 0 && row.Phase != "cold" {
			t.Fatal("first case must include cold setup")
		}
	}
}

func TestReportPreservesErrorsAndMemoryLabels(t *testing.T) {
	rss := uint64(8 << 20)
	report := NewReport("browser", 1, false, []Sample{
		{Engine: EngineAI, Run: 1, Case: "static", Phase: "cold", Success: true, SetupMS: 1, TaskMS: 2, TotalMS: 3, SnapshotBytes: 100, GoHeapAllocBytes: 2048, HarnessRSSBytes: &rss},
		{Engine: EngineAI, Run: 1, Case: "async", Phase: "warm", Error: "assertion failed", TaskMS: 5, TotalMS: 5, SnapshotBytes: 200},
	})
	if got := report.Summaries[0]; got.Succeeded != 1 || got.Attempted != 2 || got.MeanRunTotalMS != 8 || got.MedianSnapshotBytes != 150 {
		t.Fatalf("incorrect summary: %+v", got)
	}
	md := report.Markdown()
	for _, required := range []string{"1/2", "assertion failed", "GoSys", "not RSS", "unavailable", "universal superiority"} {
		if !strings.Contains(md, required) {
			t.Fatalf("report omits %q", required)
		}
	}
	if _, err := report.JSON(); err != nil {
		t.Fatal(err)
	}
}

func TestPublicURLsMustBeExplicitHTTP(t *testing.T) {
	for _, u := range []string{"example.com", "file:///etc/passwd", "http://user:secret@example.com", "javascript:alert(1)", "https://"} {
		if err := ValidatePublicURLs([]string{u}); err == nil {
			t.Errorf("accepted invalid public URL %q", u)
		}
	}
	if err := ValidatePublicURLs([]string{"https://example.com/path?q=test", "http://127.0.0.1:1234/"}); err != nil {
		t.Fatal(err)
	}
}

func TestPublicSmokeRejectsObviousChallenges(t *testing.T) {
	for _, p := range []snapshot{{status: 403}, {title: "Just a moment...", url: "https://example.com"}, {text: "Verify you are human", url: "https://example.com"}, {url: "chrome-error://chromewebdata/"}} {
		if err := publicSmokeProblem(p); err == nil {
			t.Fatalf("challenge accepted as smoke success: %+v", p)
		}
	}
	if err := publicSmokeProblem(snapshot{status: 200, title: "Example Domain", text: "This domain is for use in illustrative examples.", url: "https://example.com/"}); err != nil {
		t.Fatal(err)
	}
}

func TestDescendantTreeHandlesCycles(t *testing.T) {
	got := descendants(10, map[int]int{11: 10, 12: 11, 10: 12, 13: 99})
	if len(got) != 3 {
		t.Fatalf("unexpected descendants: %v", got)
	}
}

func TestRSSSamplerUsesOperatingSystem(t *testing.T) {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		t.Skip("RSS sampling unsupported here")
	}
	value, source := sampleRSS(os.Getpid(), false)
	if value == nil || *value == 0 {
		t.Fatalf("current process RSS unavailable: %s", source)
	}
}

func TestBrowserTaskComparisonIntegration(t *testing.T) {
	if os.Getenv("AI_BROWSER_INTEGRATION") != "1" {
		t.Skip("set AI_BROWSER_INTEGRATION=1 to run real Chromium task assertions")
	}
	path, err := browser.FindSystemBrowser()
	if err != nil {
		t.Fatal(err)
	}
	for _, engine := range []string{EngineAI, EngineChromium} {
		t.Run(engine, func(t *testing.T) {
			rows := Run(context.Background(), Options{Engine: engine, BrowserPath: path, Timeout: 12 * time.Second})
			for _, row := range rows {
				if !row.Success {
					t.Errorf("%s: %s", row.Case, row.Error)
				}
			}
		})
	}
}
