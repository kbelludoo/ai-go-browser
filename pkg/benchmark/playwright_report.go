package benchmark

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"
)

type FrameworkSummary struct {
	Engine                         string  `json:"engine"`
	Success                        int     `json:"success"`
	Cases                          int     `json:"cases"`
	MeanCorpusMS                   float64 `json:"mean_corpus_ms_including_engine_setup"`
	MeanEngineSetupMS              float64 `json:"mean_engine_setup_ms"`
	MeanWorkerWallMS               float64 `json:"mean_worker_wall_ms"`
	MedianSnapshotBytes            float64 `json:"median_snapshot_bytes"`
	MaxSampledGoHeapBytes          uint64  `json:"max_sampled_go_heap_bytes"`
	MaxSampledGoRSSBytes           *uint64 `json:"max_sampled_go_worker_rss_bytes"`
	MaxSampledNodeHeapBytes        *uint64 `json:"max_sampled_node_heap_bytes"`
	MaxSampledNodeRSSBytes         *uint64 `json:"max_sampled_node_rss_bytes"`
	MaxSampledChromiumTreeRSSBytes *uint64 `json:"max_sampled_chromium_tree_rss_bytes"`
}

type FrameworkReport struct {
	GeneratedAt    string             `json:"generated_at_utc"`
	BrowserPath    string             `json:"chromium_executable"`
	FixtureVersion string             `json:"fixture_version"`
	RunCount       int                `json:"repetitions_per_engine"`
	GOOS           string             `json:"goos"`
	GOARCH         string             `json:"goarch"`
	GoVersion      string             `json:"go_version"`
	Runs           []FrameworkRun     `json:"runs"`
	Summaries      []FrameworkSummary `json:"summaries"`
	Limitations    []string           `json:"limitations"`
	Sources        []string           `json:"official_api_sources"`
}

func NewFrameworkReport(browserPath string, repetitions int, runs []FrameworkRun) FrameworkReport {
	r := FrameworkReport{GeneratedAt: time.Now().UTC().Format(time.RFC3339), BrowserPath: browserPath, FixtureVersion: "local-v1 unchanged", RunCount: repetitions, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, GoVersion: runtime.Version(), Runs: runs}
	r.Limitations = []string{
		"Results describe these seven local tasks, not universal website access or superiority. No public sites, credentials, LLMs, model tokens or authenticated APIs are used.",
		"The same Chromium executable path is passed to both engines. AI Go uses its existing stealth launch configuration; Playwright uses its normal launch defaults. Launch flags therefore differ between the stacks. Playwright uses native locator fill/click actions, page.goto(waitUntil: load), and a 50 ms assertion poll without arbitrary settling delays. AI Go enables AutoRenderJS and explicit RenderJS for dynamic fixtures.",
		"Playwright's official documentation cautions that a custom executable may not be compatible with every Playwright version. The report records the actual Node, Playwright and Chromium versions tested; this command does not download browsers or install packages.",
		"Playwright and AI Go each run in a fresh Go worker per repetition; engine order alternates. Playwright's worker additionally owns one Node process. The Go fixture server is present in both workers. Worker wall time includes process startup, fixture setup, tasks, post-task memory sampling, cleanup and serialization; engine/task time excludes post-task memory sampling and cleanup.",
		"Playwright engine setup includes launching Node, loading Playwright and launching Chromium. AI Go engine setup is Browser.New in an already running worker, with Chromium launched lazily on its first dynamic task. Worker wall time is therefore the separate cold end-to-end comparison.",
		"The Playwright compact snapshot uses exactly the existing CDP baseline extraction expression, with visible text capped at 5000 Unicode code points and links at 200. AI Go's page schema contains different fields; UTF-8 JSON byte counts are observed sizes, not a claim of equivalent schemas or model token counts.",
		"Go heap and GoSys include the Go fixture/controller overhead; for AI Go they also include its adapter. Node heap is V8 heapUsed; Node RSS is process.memoryUsage().rss. GoSys is not RSS. Maximum values are maxima of post-task samples, not continuously observed peaks.",
		"Go worker RSS, Node RSS and Chromium tree RSS are reported separately. The tree sums resident working sets and can double-count shared pages. Separate sampled maxima must not be added and presented as a simultaneous physical-memory peak. Unavailable metrics remain null.",
		"Very short local tasks may round to zero at host timer resolution. Timing varies with hardware, caches, Chromium build and concurrent workload. Multiple repetitions improve reproducibility but do not eliminate these factors.",
		"Not-run tasks remain in the planned assertion denominator and are identified separately. Startup and transport failures preserve elapsed time when measurable; absent Node memory readings are omitted rather than recorded as zero.",
	}
	r.Sources = []string{"https://playwright.dev/docs/api/class-browsertype#browser-type-launch", "https://playwright.dev/docs/api/class-page#page-goto", "https://playwright.dev/docs/api/class-locator#locator-fill", "https://playwright.dev/docs/api/class-browser#browser-new-browser-cdp-session", "https://nodejs.org/api/process.html#processmemoryusage"}
	for _, engine := range []string{EngineAI, EnginePlaywright} {
		s := FrameworkSummary{Engine: engine}
		count := 0
		var sizes []float64
		for _, run := range runs {
			if run.Engine != engine {
				continue
			}
			count++
			s.MeanWorkerWallMS += run.WorkerWallMS
			for _, row := range run.Samples {
				s.Cases++
				if row.Success {
					s.Success++
				}
				s.MeanCorpusMS += row.TotalMS
				s.MeanEngineSetupMS += row.SetupMS
				if row.SnapshotBytes > 0 {
					sizes = append(sizes, float64(row.SnapshotBytes))
				}
				if row.GoHeapAllocBytes > s.MaxSampledGoHeapBytes {
					s.MaxSampledGoHeapBytes = row.GoHeapAllocBytes
				}
				s.MaxSampledGoRSSBytes = maxReading(s.MaxSampledGoRSSBytes, row.HarnessRSSBytes)
				s.MaxSampledNodeHeapBytes = maxReading(s.MaxSampledNodeHeapBytes, row.NodeHeapUsedBytes)
				s.MaxSampledNodeRSSBytes = maxReading(s.MaxSampledNodeRSSBytes, row.NodeRSSBytes)
				s.MaxSampledChromiumTreeRSSBytes = maxReading(s.MaxSampledChromiumTreeRSSBytes, row.ChromiumTreeRSSBytes)
			}
		}
		if count > 0 {
			s.MeanWorkerWallMS /= float64(count)
			s.MeanCorpusMS /= float64(count)
			s.MeanEngineSetupMS /= float64(count)
		}
		s.MedianSnapshotBytes = median(sizes)
		r.Summaries = append(r.Summaries, s)
	}
	return r
}

func (r FrameworkReport) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }
func (r FrameworkReport) Markdown() string {
	var out strings.Builder
	fmt.Fprintf(&out, "# AI Go and Playwright task comparison\n\nGenerated: %s. Fixture corpus: %s. Repetitions per engine: %d.\n\nSame Chromium executable: `%s`. Environment: %s/%s, %s.\n\n", r.GeneratedAt, r.FixtureVersion, r.RunCount, r.BrowserPath, r.GOOS, r.GOARCH, r.GoVersion)
	for _, run := range r.Runs {
		if run.Dependencies != nil {
			fmt.Fprintf(&out, "Playwright: %s; Node: %s; Chromium: %s.\n\n", run.Dependencies.Version, run.Dependencies.NodeVersion, run.ChromiumVersion)
			break
		}
	}
	out.WriteString("| Engine | Assertions passed | Mean task corpus incl. engine setup (ms) | Mean complete worker wall time (ms) | Mean engine setup (ms) | Median compact JSON bytes |\n|---|---:|---:|---:|---:|---:|\n")
	for _, s := range r.Summaries {
		fmt.Fprintf(&out, "| %s | %d/%d | %.2f | %.2f | %.2f | %.0f |\n", s.Engine, s.Success, s.Cases, s.MeanCorpusMS, s.MeanWorkerWallMS, s.MeanEngineSetupMS, s.MedianSnapshotBytes)
	}
	out.WriteString("\n## Separate post-task memory samples\n\nThe Go worker includes the fixture server. The Playwright worker additionally owns Node. Values are maximum sampled readings, not continuous peaks; do not sum separate maxima. Chromium sums can include shared pages more than once.\n\n| Engine | Go worker heap (MiB) | Go worker RSS (MiB) | Node V8 heap (MiB) | Node RSS (MiB) | Chromium tree RSS (MiB) |\n|---|---:|---:|---:|---:|---:|\n")
	for _, s := range r.Summaries {
		fmt.Fprintf(&out, "| %s | %.2f | %s | %s | %s | %s |\n", s.Engine, float64(s.MaxSampledGoHeapBytes)/(1<<20), formatMiB(s.MaxSampledGoRSSBytes), formatMiB(s.MaxSampledNodeHeapBytes), formatMiB(s.MaxSampledNodeRSSBytes), formatMiB(s.MaxSampledChromiumTreeRSSBytes))
	}
	out.WriteString("\n## Task results\n\nCold is the first task after engine construction. AI Go may still start Chromium lazily on a later dynamic task.\n\n| Engine | Run | Task | Phase | Passed | Not run | Task ms | Engine setup ms | JSON bytes |\n|---|---:|---|---|---|---|---:|---:|---:|\n")
	for _, run := range r.Runs {
		for _, row := range run.Samples {
			fmt.Fprintf(&out, "| %s | %d | %s | %s | %t | %t | %.2f | %.2f | %d |\n", run.Engine, run.Run, row.Case, row.Phase, row.Success, row.NotRun, row.TaskMS, row.SetupMS, row.SnapshotBytes)
		}
	}
	errors := false
	for _, run := range r.Runs {
		for _, row := range run.Samples {
			if row.Error != "" {
				if !errors {
					out.WriteString("\n## Recorded errors\n\n")
					errors = true
				}
				fmt.Fprintf(&out, "- %s run %d, %s: %s\n", run.Engine, run.Run, row.Case, strings.ReplaceAll(row.Error, "\n", " "))
			}
		}
	}
	out.WriteString("\n## Interpretation limits\n\n")
	for _, note := range r.Limitations {
		fmt.Fprintf(&out, "- %s\n", note)
	}
	out.WriteString("\n## Official API references\n\n")
	for _, source := range r.Sources {
		fmt.Fprintf(&out, "- [API reference](%s)\n", source)
	}
	return out.String()
}
