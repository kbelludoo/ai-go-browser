package benchmark

import (
	"encoding/json"
	"fmt"
	"net/url"
	"runtime"
	"sort"
	"strings"
	"time"
)

type Report struct {
	GeneratedAt        string    `json:"generated_at_utc"`
	Mode               string    `json:"mode"`
	FixtureVersion     string    `json:"fixture_version"`
	BrowserPath        string    `json:"chromium_binary"`
	BaselineNavigation string    `json:"baseline_navigation"`
	Runs               int       `json:"runs_per_engine"`
	GOOS               string    `json:"goos"`
	GOARCH             string    `json:"goarch"`
	GoVersion          string    `json:"go_version"`
	CPUs               int       `json:"logical_cpus"`
	Samples            []Sample  `json:"samples"`
	Summaries          []Summary `json:"summaries"`
	Limitations        []string  `json:"limitations"`
}

type Summary struct {
	Engine                         string  `json:"engine"`
	Succeeded                      int     `json:"succeeded"`
	Attempted                      int     `json:"attempted"`
	MeanRunTotalMS                 float64 `json:"mean_run_total_ms_including_setup"`
	MeanSetupMS                    float64 `json:"mean_setup_ms"`
	MedianTaskMS                   float64 `json:"median_task_ms_mixed_corpus"`
	MedianSnapshotBytes            float64 `json:"median_snapshot_bytes"`
	MaxSampledGoHeapBytes          uint64  `json:"max_sampled_go_heap_bytes"`
	MaxSampledHarnessRSSBytes      *uint64 `json:"max_sampled_harness_rss_bytes"`
	MaxSampledChromiumTreeRSSBytes *uint64 `json:"max_sampled_chromium_tree_rss_bytes"`
}

func NewReport(browserPath string, runs int, public bool, samples []Sample) Report {
	mode := "local fixtures"
	if public {
		mode = "explicit public URLs (read-only smoke checks)"
	}
	r := Report{GeneratedAt: time.Now().UTC().Format(time.RFC3339), Mode: mode, FixtureVersion: "local-v1", BrowserPath: browserPath, BaselineNavigation: "NavigateReady: loader-matched lifecycle load event, without a fixed settling delay", Runs: runs, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, GoVersion: runtime.Version(), CPUs: runtime.NumCPU(), Samples: samples}
	r.Limitations = []string{
		"This corpus cannot prove access to every website or universal superiority. No external AI model is run and token counts are not measured.",
		"Each engine and repetition runs in a fresh isolated Go worker. The first case includes engine setup; subsequent cases reuse that engine. The first JS case may still include a lazy Chromium launch for AI Go.",
		"The baseline is real Chromium controlled through the repository's CDP/stealth launcher and NavigateReady. It waits for the navigation's lifecycle load event without a fixed settling delay, then polls task assertions; it is not a comparison against every agent framework.",
		"AI Go uses CompactSnapshots (CLI default), omitting automatic Markdown. Compact snapshot bytes are actual UTF-8 JSON lengths. Both adapters cap visible text at 5000 Unicode code points, links at 200 and headings at 100; schemas differ. Document bytes represent fetched/serialized HTML, not model input tokens or total network traffic.",
		"GoHeapAlloc and GoSys belong to the benchmark worker and include fixtures/adapter allocations. GoSys is runtime-reserved memory, not RSS. Harness RSS and Chromium tree RSS are separate OS samples after each task, not continuously measured peaks.",
		"Chromium tree RSS sums resident working sets of process descendants. Shared pages can be counted multiple times, processes can exit during sampling, and unavailable readings remain null rather than zero.",
		"Per-case contexts bound CDP operations and assertion polling; AI Go commands also use the browser's internal request/render timeouts and may exceed the requested case timeout.",
		"Results depend on hardware, OS, Chromium version, local caches and task order. The command alternates engine order across repetitions; aggregate medians across different task types describe this fixed corpus only.",
		"Very short local HTTP tasks can round to zero at the host timer resolution; these readings do not mean that the operation has no cost.",
	}
	if public {
		r.Limitations = append(r.Limitations, "Public checks require a nonempty document and reject reported HTTP errors and a small set of obvious challenge/error phrases. They do not prove semantic completion or distinguish every challenge from usable content; no forms or account actions are performed.")
	}
	for _, engine := range []string{EngineAI, EngineChromium} {
		s := Summary{Engine: engine}
		var durations, sizes []float64
		runSet := map[int]bool{}
		for _, row := range samples {
			if row.Engine != engine {
				continue
			}
			s.Attempted++
			if row.Success {
				s.Succeeded++
			}
			runSet[row.Run] = true
			s.MeanRunTotalMS += row.TotalMS
			s.MeanSetupMS += row.SetupMS
			durations = append(durations, row.TaskMS)
			if row.SnapshotBytes > 0 {
				sizes = append(sizes, float64(row.SnapshotBytes))
			}
			if row.GoHeapAllocBytes > s.MaxSampledGoHeapBytes {
				s.MaxSampledGoHeapBytes = row.GoHeapAllocBytes
			}
			s.MaxSampledHarnessRSSBytes = maxReading(s.MaxSampledHarnessRSSBytes, row.HarnessRSSBytes)
			s.MaxSampledChromiumTreeRSSBytes = maxReading(s.MaxSampledChromiumTreeRSSBytes, row.ChromiumTreeRSSBytes)
		}
		if len(runSet) > 0 {
			s.MeanRunTotalMS /= float64(len(runSet))
			s.MeanSetupMS /= float64(len(runSet))
		}
		s.MedianTaskMS = median(durations)
		s.MedianSnapshotBytes = median(sizes)
		r.Summaries = append(r.Summaries, s)
	}
	return r
}

func maxReading(a, b *uint64) *uint64 {
	if b == nil {
		return a
	}
	if a == nil || *b > *a {
		v := *b
		return &v
	}
	return a
}
func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sort.Float64s(values)
	m := len(values) / 2
	if len(values)%2 == 0 {
		return (values[m-1] + values[m]) / 2
	}
	return values[m]
}

func (r Report) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }

func (r Report) Markdown() string {
	var out strings.Builder
	fmt.Fprintf(&out, "# Browser task comparison\n\nGenerated: %s. Mode: %s. Fixture version: %s.\n\nChromium: `%s`\n\nBaseline navigation: %s.\n\nEnvironment: %s/%s, %s, %d logical CPUs. %d repetitions per engine.\n\n", r.GeneratedAt, r.Mode, r.FixtureVersion, r.BrowserPath, r.BaselineNavigation, r.GOOS, r.GOARCH, r.GoVersion, r.CPUs, r.Runs)
	out.WriteString("| Engine | Task assertions passed | Mean corpus time including startup (ms) | Mean startup (ms) | Median compact JSON bytes | Max sampled Go heap (MiB) | Max sampled harness RSS (MiB) | Max sampled Chromium tree RSS (MiB) |\n|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, s := range r.Summaries {
		fmt.Fprintf(&out, "| %s | %d/%d | %.2f | %.2f | %.0f | %.2f | %s | %s |\n", s.Engine, s.Succeeded, s.Attempted, s.MeanRunTotalMS, s.MeanSetupMS, s.MedianSnapshotBytes, float64(s.MaxSampledGoHeapBytes)/(1<<20), formatMiB(s.MaxSampledHarnessRSSBytes), formatMiB(s.MaxSampledChromiumTreeRSSBytes))
	}
	out.WriteString("\n## Individual tasks\n\nCold rows include startup in total time. Warm rows reuse the engine; AI Go may lazily start Chromium on its first JS case. Snapshot extraction and assertion polling are included in task time. Memory sampling runs afterwards.\n\n| Engine | Run | Case | Phase | Success | Task ms | Startup ms | Total ms | Compact JSON bytes | Document bytes |\n|---|---:|---|---|---|---:|---:|---:|---:|---:|\n")
	for _, row := range r.Samples {
		fmt.Fprintf(&out, "| %s | %d | %s | %s | %t | %.2f | %.2f | %.2f | %d | %d |\n", row.Engine, row.Run, row.Case, row.Phase, row.Success, row.TaskMS, row.SetupMS, row.TotalMS, row.SnapshotBytes, row.DocumentBytes)
	}
	var errors bool
	for _, row := range r.Samples {
		if row.Error != "" {
			if !errors {
				out.WriteString("\n## Errors\n\n")
				errors = true
			}
			fmt.Fprintf(&out, "- %s, run %d, %s: %s\n", row.Engine, row.Run, row.Case, strings.ReplaceAll(row.Error, "\n", " "))
		}
	}
	out.WriteString("\n## Interpretation limits\n\n")
	for _, note := range r.Limitations {
		fmt.Fprintf(&out, "- %s\n", note)
	}
	return out.String()
}

func formatMiB(value *uint64) string {
	if value == nil {
		return "unavailable"
	}
	return fmt.Sprintf("%.2f", float64(*value)/(1<<20))
}

func ValidatePublicURLs(urls []string) error {
	for _, value := range urls {
		u, err := url.Parse(value)
		if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
			return fmt.Errorf("public URL must be an explicit HTTP(S) URL without embedded credentials: %q", value)
		}
	}
	return nil
}
