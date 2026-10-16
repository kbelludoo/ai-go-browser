package benchmark

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"ai-go-browser/browser"
)

//go:embed modern_playwright_worker.cjs
var modernPlaywrightWorker string

type ModernRun struct {
	FrameworkRun
	Evidence []ModernEvidence `json:"fixture_evidence"`
}

func modernFailureSamples(engine string, run int, cases []modernFixture, cause error) []FrameworkSample {
	var samples []FrameworkSample
	for i, f := range cases {
		phase := "warm"
		if i == 0 {
			phase = "cold"
		}
		samples = append(samples, FrameworkSample{Sample: Sample{Engine: engine, Run: run, Case: f.Name, Phase: phase, RequiresJS: true, Error: cause.Error(), HarnessRSSSource: "unavailable", ChromiumRSSSource: "unavailable"}, Assertion: f.Expected + "; strict fixture HTTP count", NotRun: true})
	}
	return samples
}

func ModernWorkerFailure(engine string, run int, cause error) ModernRun {
	return ModernRun{FrameworkRun: FrameworkRun{Engine: engine, Run: run, Samples: modernFailureSamples(engine, run, modernCases(), cause)}}
}

// ValidateModernRun rejects incomplete/misattributed worker output before the
// parent summarizes it. An omitted planned task must never become a 4/4 pass.
func ValidateModernRun(result ModernRun, engine string, iteration int) error {
	cases := modernCases()
	if result.Engine != engine || result.Run != iteration {
		return fmt.Errorf("worker identity mismatch: expected %s run %d, got %s run %d", engine, iteration, result.Engine, result.Run)
	}
	if len(result.Samples) != len(cases) {
		return fmt.Errorf("worker returned %d tasks; expected %d", len(result.Samples), len(cases))
	}
	for i, f := range cases {
		row := result.Samples[i]
		if row.Case != f.Name || row.Engine != engine || row.Run != iteration {
			return fmt.Errorf("worker task %d identity mismatch", i+1)
		}
		if row.NotRun && row.Success {
			return fmt.Errorf("worker task %s cannot pass without running", f.Name)
		}
	}
	return nil
}

func modernOptions(opts Options) Options {
	if opts.Run == 0 {
		opts.Run = 1
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 12 * time.Second
	}
	return opts
}

func modernRow(engine string, opts Options, c *modernCorpus, f modernFixture, index int, setupMS float64) FrameworkSample {
	row := FrameworkSample{Sample: Sample{Engine: engine, Run: opts.Run, Case: f.Name, URL: c.server.URL + f.Path, Phase: "warm", RequiresJS: true}, Assertion: f.Expected + fmt.Sprintf("; HTTP requests=%d, all valid", f.ExpectedRequests)}
	if index == 0 {
		row.Phase, row.SetupMS = "cold", setupMS
	}
	return row
}

func modernObserve(row *FrameworkSample, observed snapshot, err error, start time.Time, c *modernCorpus, f modernFixture) ModernEvidence {
	evidence := c.evidence(f)
	if err == nil && !evidence.Passed {
		err = fmt.Errorf("fixture oracle: expected %d valid requests, observed %d requests (%d valid)", evidence.ExpectedRequests, evidence.ObservedRequests, evidence.ValidRequests)
	}
	row.TaskMS = milliseconds(time.Since(start))
	row.TotalMS = row.TaskMS + row.SetupMS
	row.Success = err == nil
	if err != nil {
		row.Error = err.Error()
	}
	row.SnapshotBytes, row.DocumentBytes = observed.size, observed.document
	row.FinalURL, row.ObservedTitle, row.HTTPStatus = observed.url, observed.title, observed.status
	text := []rune(observed.text)
	if len(text) > 400 {
		text = text[:400]
	}
	row.ObservedTextExcerpt = string(text)
	return evidence
}

func modernMemory(row *FrameworkSample, pid int) {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	row.GoHeapAllocBytes, row.GoSysBytes = memory.HeapAlloc, memory.Sys
	row.HarnessRSSBytes, row.HarnessRSSSource = sampleRSS(os.Getpid(), false)
	if pid > 0 {
		row.ChromiumTreeRSSBytes, row.ChromiumRSSSource = sampleRSS(pid, true)
	} else {
		row.ChromiumRSSSource = "unavailable: no launched Chromium process identified"
	}
}

func RunModernAI(ctx context.Context, options Options) (result ModernRun) {
	opts := modernOptions(options)
	startWall := time.Now()
	defer func() { result.WorkerWallMS = milliseconds(time.Since(startWall)) }()
	result.FrameworkRun = FrameworkRun{Engine: EngineAI, Run: opts.Run}
	c := modernFixtures()
	defer c.close()
	if len(opts.PublicURLs) > 0 {
		result.Samples = modernFailureSamples(EngineAI, opts.Run, c.cases, fmt.Errorf("modern comparison supports local fixtures only"))
		return result
	}
	setupStart := time.Now()
	b := browser.New([]string{"127.0.0.1", "localhost"}, true)
	b.CompactSnapshots, b.AutoRenderJS, b.SystemBrowser = true, true, opts.BrowserPath
	b.OperationTimeout = opts.Timeout
	defer b.Close()
	setupMS := milliseconds(time.Since(setupStart))
	d := modernAIDriver{b: b}
	for i, f := range c.cases {
		row := modernRow(EngineAI, opts, c, f, i, setupMS)
		caseStart := time.Now()
		caseCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
		priorPID := b.RenderProcessID()
		observed, err := d.task(caseCtx, row.URL, f)
		cancel()
		result.Evidence = append(result.Evidence, modernObserve(&row, observed, err, caseStart, c, f))
		row.ChromiumStartedInTask = priorPID == 0 && b.RenderProcessID() > 0
		modernMemory(&row, b.RenderProcessID())
		result.Samples = append(result.Samples, row)
	}
	return result
}

type modernAIDriver struct {
	b          *browser.Browser
	mainRef    string
	frameCheck *modernFrameOracle
}

// A child URL alone is insufficient: navigating the main tab to that URL
// would produce the same visible result. Retain the actual frame identities.
type modernFrameOracle struct {
	main, child browser.Frame
}

func newModernFrameOracle(frames []browser.Frame, name, mainURL string) (*modernFrameOracle, error) {
	var result modernFrameOracle
	mainCount, childCount := 0, 0
	for _, frame := range frames {
		if frame.Main {
			result.main = frame
			mainCount++
		}
		if frame.Name == name && frame.Allowed && !frame.Main {
			result.child = frame
			childCount++
		}
	}
	if mainCount != 1 || result.main.ID == "" || result.main.Ref == "" || result.main.URL != mainURL {
		return nil, fmt.Errorf("frame oracle: original main document %q was not discovered", mainURL)
	}
	if childCount != 1 || result.child.ID == "" || result.child.Ref == "" || result.child.ID == result.main.ID || result.child.Ref == result.main.Ref {
		return nil, fmt.Errorf("frame oracle: exactly one allowed named child %q must be discovered", name)
	}
	return &result, nil
}

func (o *modernFrameOracle) selectedPage(page *browser.Page) error {
	if page == nil || page.FrameRef != o.child.Ref {
		return fmt.Errorf("frame oracle: snapshot must use selected child reference %q", o.child.Ref)
	}
	if page.URL != o.child.URL {
		return fmt.Errorf("frame oracle: selected child URL changed from %q to %q", o.child.URL, page.URL)
	}
	return nil
}

func (o *modernFrameOracle) unchangedTree(frames []browser.Frame) error {
	mainCount, childCount, selectedCount := 0, 0, 0
	for _, frame := range frames {
		if frame.Selected {
			selectedCount++
		}
		if frame.Main {
			mainCount++
			if frame.ID != o.main.ID || frame.Ref != o.main.Ref || frame.URL != o.main.URL || frame.Selected {
				return fmt.Errorf("frame oracle: original main frame identity or URL changed")
			}
		}
		if frame.ID == o.child.ID {
			childCount++
			if frame.Main || !frame.Selected || !frame.Allowed || frame.Ref != o.child.Ref || frame.Name != o.child.Name || frame.URL != o.child.URL || frame.ParentRef != o.child.ParentRef {
				return fmt.Errorf("frame oracle: original child frame is no longer the selected document")
			}
		}
	}
	if mainCount != 1 || childCount != 1 || selectedCount != 1 {
		return fmt.Errorf("frame oracle: original main and selected child must remain in the frame tree")
	}
	return nil
}

func (d *modernAIDriver) execute(ctx context.Context, command browser.Command) (browser.Result, error) {
	if err := ctx.Err(); err != nil {
		return browser.Result{}, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		d.b.OperationTimeout = max(time.Millisecond, time.Until(deadline))
	}
	result := d.b.Execute(command)
	if !result.OK {
		return result, fmt.Errorf("%s: %s", command.Cmd, result.Error)
	}
	return result, nil
}

func (d *modernAIDriver) observe(ctx context.Context) (snapshot, error) {
	result, err := d.execute(ctx, browser.Command{Cmd: "snapshot"})
	if err != nil {
		return snapshot{}, err
	}
	if result.Page == nil {
		return snapshot{}, fmt.Errorf("snapshot returned no page")
	}
	if d.frameCheck != nil {
		if err := d.frameCheck.selectedPage(result.Page); err != nil {
			return snapshot{}, err
		}
	}
	data, err := json.Marshal(result.Page)
	return snapshot{text: result.Page.Text, title: result.Page.Title, url: result.Page.URL, size: len(data), document: result.Page.Bytes, status: result.Page.Status}, err
}

func (d *modernAIDriver) waitText(ctx context.Context, expected string) (snapshot, error) {
	for {
		observed, err := d.observe(ctx)
		if err != nil {
			return observed, err
		}
		if strings.Contains(observed.text, expected) {
			return observed, nil
		}
		select {
		case <-ctx.Done():
			return observed, fmt.Errorf("visible result must contain %q: %w", expected, ctx.Err())
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (d *modernAIDriver) task(ctx context.Context, target string, f modernFixture) (snapshot, error) {
	d.frameCheck = nil
	if d.mainRef != "" {
		if _, err := d.execute(ctx, browser.Command{Cmd: "frame", Ref: d.mainRef}); err != nil {
			return snapshot{}, err
		}
		d.mainRef = ""
	}
	if _, err := d.execute(ctx, browser.Command{Cmd: "open", URL: target, RenderJS: true}); err != nil {
		return snapshot{}, err
	}
	if f.FrameName != "" {
		frames, err := d.execute(ctx, browser.Command{Cmd: "frames"})
		if err != nil {
			return snapshot{}, err
		}
		d.frameCheck, err = newModernFrameOracle(frames.Frames, f.FrameName, target)
		if err != nil {
			return snapshot{}, err
		}
		d.mainRef = d.frameCheck.main.Ref
		selected, err := d.execute(ctx, browser.Command{Cmd: "frame", Ref: d.frameCheck.child.Ref})
		if err != nil {
			return snapshot{}, err
		}
		if err := d.frameCheck.selectedPage(selected.Page); err != nil {
			return snapshot{}, err
		}
	}
	for _, step := range f.Steps {
		if step.Kind == "assert" {
			if _, err := d.waitText(ctx, step.Expected); err != nil {
				return snapshot{}, err
			}
			continue
		}
		found, err := d.execute(ctx, browser.Command{Cmd: "query", Selector: step.Selector})
		if err != nil {
			return snapshot{}, err
		}
		if len(found.Matches) != 1 || found.Matches[0].Ref == "" {
			return snapshot{}, fmt.Errorf("selector %q must discover exactly one actionable reference", step.Selector)
		}
		if _, err := d.execute(ctx, browser.Command{Cmd: step.Kind, Ref: found.Matches[0].Ref, Value: step.Value, Confirm: true}); err != nil {
			return snapshot{}, err
		}
	}
	observed, err := d.waitText(ctx, f.Expected)
	if err == nil && d.frameCheck != nil {
		frames, treeErr := d.execute(ctx, browser.Command{Cmd: "frames"})
		if treeErr != nil {
			err = treeErr
		} else {
			err = d.frameCheck.unchangedTree(frames.Frames)
		}
	}
	return observed, err
}

func startModernPlaywright(ctx context.Context, deps PlaywrightDependencies) (*playwrightProcess, error) {
	cmd := exec.CommandContext(ctx, deps.NodePath, "-e", modernPlaywrightWorker)
	cmd.Stderr = os.Stderr
	input, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		input.Close()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		input.Close()
		return nil, err
	}
	p := &playwrightProcess{cmd: cmd, input: input, responses: make(chan playwrightResponse, 1), errors: make(chan error, 1)}
	go func() {
		decoder := json.NewDecoder(output)
		for {
			var response playwrightResponse
			if err := decoder.Decode(&response); err != nil {
				p.errors <- err
				return
			}
			select {
			case p.responses <- response:
			case <-ctx.Done():
				return
			}
		}
	}()
	return p, nil
}

func RunModernPlaywright(ctx context.Context, options Options, deps PlaywrightDependencies) (result ModernRun) {
	opts := modernOptions(options)
	wallStart := time.Now()
	defer func() { result.WorkerWallMS = milliseconds(time.Since(wallStart)) }()
	result.FrameworkRun = FrameworkRun{Engine: EnginePlaywright, Run: opts.Run, Dependencies: &deps}
	c := modernFixtures()
	defer c.close()
	if len(opts.PublicURLs) > 0 {
		result.Samples = modernFailureSamples(EnginePlaywright, opts.Run, c.cases, fmt.Errorf("modern comparison supports local fixtures only"))
		return result
	}
	setupStart := time.Now()
	p, err := startModernPlaywright(ctx, deps)
	if err != nil {
		result.Samples = modernFailureSamples(EnginePlaywright, opts.Run, c.cases, err)
		result.Samples[0].SetupMS = milliseconds(time.Since(setupStart))
		result.Samples[0].TotalMS = result.Samples[0].SetupMS
		return result
	}
	defer p.close()
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	init, err := p.exchange(initCtx, map[string]any{"cmd": "init", "module": deps.ModulePath, "browser": opts.BrowserPath, "snapshot_script": modernCompactDOM})
	cancel()
	if err == nil && !init.OK {
		err = fmt.Errorf("Playwright initialization: %s", init.Error)
	}
	if err != nil {
		result.Samples = modernFailureSamples(EnginePlaywright, opts.Run, c.cases, err)
		result.Samples[0].SetupMS = milliseconds(time.Since(setupStart))
		result.Samples[0].TotalMS = result.Samples[0].SetupMS
		return result
	}
	setupMS := milliseconds(time.Since(setupStart))
	result.ChromiumVersion = init.ChromiumVersion
	for i, f := range c.cases {
		row := modernRow(EnginePlaywright, opts, c, f, i, setupMS)
		caseStart := time.Now()
		caseCtx, cancel := context.WithTimeout(ctx, opts.Timeout+5*time.Second)
		response, wireErr := p.exchange(caseCtx, map[string]any{"cmd": "task", "url": row.URL, "frame_name": f.FrameName, "steps": f.Steps, "expected": f.Expected, "timeout_ms": opts.Timeout.Milliseconds()})
		cancel()
		err := wireErr
		if err == nil && !response.OK {
			err = fmt.Errorf("Playwright task: %s", response.Error)
		}
		observed := snapshot{text: response.Snapshot.Text, title: response.Snapshot.Title, url: response.Snapshot.URL, status: response.Snapshot.Status, document: response.Snapshot.DocumentBytes, size: response.SnapshotBytes}
		if err == nil && f.FrameName != "" && !strings.Contains(observed.url, "/frame-child?kind=") {
			err = fmt.Errorf("frame snapshot remained in main document: %s", observed.url)
		}
		result.Evidence = append(result.Evidence, modernObserve(&row, observed, err, caseStart, c, f))
		row.NodeHeapUsedBytes, row.NodeRSSBytes = response.NodeHeapUsedBytes, response.NodeRSSBytes
		row.NodeExternalBytes, row.NodeArrayBufferBytes = response.NodeExternalBytes, response.NodeArrayBufferBytes
		if response.NodeRSSBytes != nil {
			row.NodeRSSSource = "Node process.memoryUsage().rss, single Node process OS resident memory"
		}
		modernMemory(&row, init.ChromiumPID)
		result.Samples = append(result.Samples, row)
		if wireErr != nil {
			remaining := modernFailureSamples(EnginePlaywright, opts.Run, c.cases[i+1:], fmt.Errorf("not run after worker failure: %w", wireErr))
			for n := range remaining {
				remaining[n].Phase = "warm"
			}
			result.Samples = append(result.Samples, remaining...)
			break
		}
	}
	return result
}
