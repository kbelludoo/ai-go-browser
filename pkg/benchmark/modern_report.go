package benchmark

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ModernReport preserves the comparison envelope and summary calculations,
// while its runs include independently observed fixture HTTP evidence.
type ModernReport struct {
	FrameworkReport
	Runs []ModernRun `json:"runs"`
}

func NewModernReport(browserPath string, repetitions int, runs []ModernRun) ModernReport {
	frameworkRuns := make([]FrameworkRun, 0, len(runs))
	for _, run := range runs {
		frameworkRuns = append(frameworkRuns, run.FrameworkRun)
	}
	base := NewFrameworkReport(browserPath, repetitions, frameworkRuns)
	base.FixtureVersion = ModernFixtureVersion
	base.Limitations = []string{
		"Results describe five local modern DOM tasks, not universal website compatibility or superiority. The original local-v1 corpus and its reports are separate and unchanged. No public websites, credentials, LLMs or authenticated APIs are used.",
		"Both engines receive the same Chromium executable, selectors, input values, ordered actions and result assertions. AI Go uses its stealth launch configuration; Playwright uses normal headless launch defaults. The flags differ between stacks. Playwright uses native locator fill/click, waits for navigation load, and polls results every 50 ms. AI Go uses query references, explicit RenderJS and a 50 ms snapshot poll. Its input/textarea/select fill uses property setters with synthetic input/change events; contenteditable uses native CDP Input.insertText and press uses native mouse input. The shadow task requires trusted clicks and the contenteditable task requires trusted input; ordinary form-field input events are not asserted as trusted.",
		"The shadow task covers an open shadow root, a projected slot and trusted clicks. Contenteditable requires a trusted input event and a trusted click. The multipart task first observes native invalid/zero-submit evidence, then requires exactly one valid multipart POST with the submitter value and no files. Uploading files, closed shadow roots, nested frames and arbitrary custom element frameworks are outside this corpus.",
		"Frame tasks discover the frame by name and interact in its selected document. The cross-origin task uses localhost on a second local server while the main document uses 127.0.0.1, making host and port distinct. The snapshot/result must be from the child document, not text aggregated into the main page.",
		"Visible result text contains exact event counts and values. Independently counted local HTTP submission receipts must match the expected valid request count; a duplicate or malformed submission fails the task even if the displayed result matches. Counts cover designated receipt endpoints, excluding page/frame/asset loads. HTTP evidence is additional to the visible result assertion, not evidence that every task generated a submission.",
		"Playwright's custom executable option may not be compatible with every Chromium build. Actual dependency and browser versions are recorded. This command installs no packages and downloads no browser binaries.",
		"Every repetition starts each engine in a fresh Go worker; engine order alternates. Both workers host the same two local fixture servers. The Playwright worker additionally owns Node. Complete worker wall time includes process startup, fixtures, engine setup, tasks, memory sampling, cleanup and serialization. Task time uses the Go controller wall clock for both engines, including Playwright IPC, and excludes post-task memory sampling and cleanup.",
		"Cold is the first task. Playwright engine setup launches Node and Chromium. AI Go engine setup constructs Browser.New; Chromium starts lazily during the first task because all five tasks explicitly request rendering. Complete worker wall time is the separate cold end-to-end comparison. Corpus totals include engine setup once per repetition.",
		"Playwright's compact observation traverses visible composed content, open shadow roots and projected slots once, within the selected document. Text is capped at 5000 Unicode code points, headings at 100, and links/buttons/forms at 200. AI Go uses its compact page schema, which contains different fields. UTF-8 JSON byte counts are observed sizes, not equivalent schemas or model tokens. Playwright document_bytes is serialized light-DOM HTML, excluding shadow roots; AI Go page Bytes measures its serialized composed observation, including shadow content and reference metadata. These document-byte metrics describe different representations.",
		"Go heap and GoSys include fixture/controller overhead; GoSys is runtime memory reservation, not RSS. Node heap is V8 heapUsed and Node RSS is process.memoryUsage().rss. Go worker, Node and Chromium tree RSS are separate. Values are maxima of post-task samples, not continuously measured peaks. Chromium working-set sums can double-count shared pages; do not add separate maxima and present them as simultaneous physical-memory consumption. Unavailable readings remain absent/null.",
		"Startup/transport failures keep all five planned tasks in the denominator; skipped tasks are explicitly marked not_run. HTTP evidence exists only for tasks attempted in a live worker. Very short durations can round to zero; hardware, caches, browser build and concurrent workload affect results.",
	}
	base.Sources = append(base.Sources, "https://playwright.dev/docs/locators#locate-in-shadow-dom", "https://playwright.dev/docs/api/class-framelocator")
	return ModernReport{FrameworkReport: base, Runs: runs}
}

func (r ModernReport) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }

func (r ModernReport) Markdown() string {
	base := r.FrameworkReport.Markdown()
	base = strings.Replace(base, "# AI Go and Playwright task comparison", "# AI Go and Playwright modern DOM comparison", 1)
	base = strings.Replace(base, "Cold is the first task after engine construction. AI Go may still start Chromium lazily on a later dynamic task.", "Cold is the first rendered task after engine construction. AI Go launches Chromium lazily inside that task; Playwright launches it during engine setup. Complete worker wall time includes both startup paths.", 1)
	var evidence strings.Builder
	evidence.WriteString("\n## Independent fixture HTTP evidence\n\nCounts cover designated submission/receipt endpoints, excluding page/frame/asset loads. A passed HTTP count supplements the visible event/value assertion. Tasks not attempted have no evidence row.\n\n| Engine | Run | Task | Expected receipts | Observed receipts | Valid receipts | HTTP count passed |\n|---|---:|---|---:|---:|---:|---|\n")
	for _, run := range r.Runs {
		for _, row := range run.Evidence {
			fmt.Fprintf(&evidence, "| %s | %d | %s | %d | %d | %d | %t |\n", run.Engine, run.Run, row.Case, row.ExpectedRequests, row.ObservedRequests, row.ValidRequests, row.Passed)
		}
	}
	return strings.Replace(base, "\n## Interpretation limits\n", evidence.String()+"\n## Interpretation limits\n", 1)
}
