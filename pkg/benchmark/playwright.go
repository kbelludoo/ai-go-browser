package benchmark

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const EnginePlaywright = "playwright_chromium"

//go:embed playwright_worker.cjs
var playwrightWorker string

type PlaywrightDependencies struct {
	NodePath    string `json:"node_executable"`
	NodeVersion string `json:"node_version"`
	ModulePath  string `json:"playwright_module"`
	Version     string `json:"playwright_version"`
}

type FrameworkSample struct {
	Sample
	NodeHeapUsedBytes    *uint64 `json:"node_heap_used_bytes,omitempty"`
	NodeRSSBytes         *uint64 `json:"node_rss_bytes,omitempty"`
	NodeRSSSource        string  `json:"node_rss_source,omitempty"`
	NodeExternalBytes    *uint64 `json:"node_external_bytes,omitempty"`
	NodeArrayBufferBytes *uint64 `json:"node_array_buffer_bytes,omitempty"`
	Assertion            string  `json:"assertion"`
	NotRun               bool    `json:"not_run,omitempty"`
}

type FrameworkRun struct {
	Engine          string                  `json:"engine"`
	Run             int                     `json:"run"`
	WorkerWallMS    float64                 `json:"worker_wall_ms"`
	ChromiumVersion string                  `json:"chromium_version,omitempty"`
	Dependencies    *PlaywrightDependencies `json:"dependencies,omitempty"`
	Samples         []FrameworkSample       `json:"samples"`
}

// DetectPlaywright probes available packages without installing packages or
// downloading browsers. A bundled Node layout is supported in addition to normal
// project node_modules and an explicitly supplied package path.
func DetectPlaywright(ctx context.Context, nodePath, module string) (PlaywrightDependencies, error) {
	if nodePath == "" {
		var err error
		nodePath, err = exec.LookPath("node")
		if err != nil {
			return PlaywrightDependencies{}, fmt.Errorf("Node.js unavailable: %w", err)
		}
	}
	if resolved, err := exec.LookPath(nodePath); err == nil {
		nodePath = resolved
	}
	candidates := []string{module}
	if module == "" {
		candidates = []string{"playwright", "playwright-core", filepath.Join(filepath.Dir(nodePath), "..", "node_modules", "playwright"), filepath.Join(filepath.Dir(nodePath), "..", "node_modules", "playwright-core")}
	}
	arg, _ := json.Marshal(candidates)
	const probe = `const fs=require('node:fs'),path=require('node:path');const candidates=JSON.parse(process.argv[1]);let found=false;for(const candidate of candidates){try{const module=require.resolve(candidate);if(!require(module).chromium)continue;let dir=path.dirname(module);while(!fs.existsSync(path.join(dir,'package.json'))){const parent=path.dirname(dir);if(parent===dir)throw new Error('package metadata unavailable');dir=parent;}const pkg=require(path.join(dir,'package.json'));console.log(JSON.stringify({node_executable:process.execPath,node_version:process.version,playwright_module:module,playwright_version:pkg.version}));found=true;break;}catch{}}if(!found){console.error('Playwright unavailable: provide -playwright-module or install playwright locally; no automatic installation performed');process.exit(1)}`
	cmd := exec.CommandContext(ctx, nodePath, "-e", probe, string(arg))
	var errorOutput strings.Builder
	cmd.Stderr = &errorOutput
	b, err := cmd.Output()
	if err != nil {
		return PlaywrightDependencies{}, fmt.Errorf("Playwright dependency probe: %v: %s", err, strings.TrimSpace(errorOutput.String()))
	}
	var deps PlaywrightDependencies
	if err = json.Unmarshal(b, &deps); err != nil {
		return deps, fmt.Errorf("invalid dependency probe response: %w", err)
	}
	return deps, nil
}

type playwrightResponse struct {
	OK                bool    `json:"ok"`
	Error             string  `json:"error"`
	NodePID           int     `json:"node_pid"`
	ChromiumPID       int     `json:"chromium_pid"`
	ChromiumPIDSource string  `json:"chromium_pid_source"`
	ChromiumVersion   string  `json:"chromium_version"`
	TaskMS            float64 `json:"task_ms"`
	Snapshot          struct {
		Text          string `json:"text"`
		Title         string `json:"title"`
		URL           string `json:"url"`
		Status        int    `json:"status"`
		DocumentBytes int    `json:"document_bytes"`
	} `json:"snapshot"`
	SnapshotBytes        int     `json:"snapshot_bytes"`
	NodeHeapUsedBytes    *uint64 `json:"node_heap_used_bytes"`
	NodeRSSBytes         *uint64 `json:"node_rss_bytes"`
	NodeExternalBytes    *uint64 `json:"node_external_bytes"`
	NodeArrayBufferBytes *uint64 `json:"node_array_buffer_bytes"`
}

type playwrightProcess struct {
	cmd       *exec.Cmd
	input     io.WriteCloser
	responses chan playwrightResponse
	errors    chan error
}

func startPlaywright(ctx context.Context, deps PlaywrightDependencies) (*playwrightProcess, error) {
	cmd := exec.CommandContext(ctx, deps.NodePath, "-e", playwrightWorker)
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
	if err = cmd.Start(); err != nil {
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

func (p *playwrightProcess) exchange(ctx context.Context, message any) (playwrightResponse, error) {
	if err := json.NewEncoder(p.input).Encode(message); err != nil {
		return playwrightResponse{}, err
	}
	select {
	case response := <-p.responses:
		return response, nil
	case err := <-p.errors:
		return playwrightResponse{}, fmt.Errorf("Playwright worker stopped: %w", err)
	case <-ctx.Done():
		return playwrightResponse{}, ctx.Err()
	}
}

func (p *playwrightProcess) close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := p.exchange(ctx, map[string]any{"cmd": "close"}); err != nil {
		_ = p.cmd.Process.Kill()
	}
	_ = p.input.Close()
	_ = p.cmd.Wait()
}

func frameworkFailureSamples(engine string, run int, cases []fixture, cause error) []FrameworkSample {
	rows := make([]FrameworkSample, 0, len(cases))
	for i, f := range cases {
		phase := "warm"
		if i == 0 {
			phase = "cold"
		}
		rows = append(rows, FrameworkSample{Sample: Sample{Engine: engine, Run: run, Case: f.name, Phase: phase, RequiresJS: f.javascript, Error: cause.Error(), HarnessRSSSource: "unavailable", ChromiumRSSSource: "unavailable"}, Assertion: "visible text contains " + f.expected, NotRun: true})
	}
	return rows
}

// FrameworkWorkerFailure preserves the planned denominator when a worker exits
// before returning a report. These tasks are recorded as not run, not measured.
func FrameworkWorkerFailure(engine string, run int, cause error) FrameworkRun {
	server, cases := localFixtures()
	server.Close()
	return FrameworkRun{Engine: engine, Run: run, Samples: frameworkFailureSamples(engine, run, cases, cause)}
}

// RunPlaywright uses the unchanged local-v1 fixtures and the exact compact DOM
// extraction expression used by the CDP baseline. It never visits public sites.
func RunPlaywright(ctx context.Context, opts Options, deps PlaywrightDependencies) (result FrameworkRun) {
	wallStart := time.Now()
	defer func() { result.WorkerWallMS = milliseconds(time.Since(wallStart)) }()
	if opts.Run == 0 {
		opts.Run = 1
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 12 * time.Second
	}
	result = FrameworkRun{Engine: EnginePlaywright, Run: opts.Run, Dependencies: &deps}
	if len(opts.PublicURLs) > 0 {
		result.Samples = frameworkFailureSamples(EnginePlaywright, opts.Run, []fixture{{name: "configuration"}}, fmt.Errorf("Playwright comparison currently supports local fixtures only"))
		return result
	}
	server, cases := localFixtures()
	defer server.Close()
	start := time.Now()
	p, err := startPlaywright(ctx, deps)
	if err != nil {
		result.Samples = frameworkFailureSamples(EnginePlaywright, opts.Run, cases, err)
		result.Samples[0].SetupMS = milliseconds(time.Since(start))
		result.Samples[0].TotalMS = result.Samples[0].SetupMS
		return result
	}
	defer p.close()
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	init, err := p.exchange(initCtx, map[string]any{"cmd": "init", "module": deps.ModulePath, "browser": opts.BrowserPath, "snapshot_script": compactDOM})
	cancel()
	if err == nil && !init.OK {
		err = fmt.Errorf("Playwright initialization: %s", init.Error)
	}
	if err != nil {
		result.Samples = frameworkFailureSamples(EnginePlaywright, opts.Run, cases, err)
		result.Samples[0].SetupMS = milliseconds(time.Since(start))
		result.Samples[0].TotalMS = result.Samples[0].SetupMS
		return result
	}
	setupMS := milliseconds(time.Since(start))
	result.ChromiumVersion = init.ChromiumVersion
	for i, f := range cases {
		row := FrameworkSample{Sample: Sample{Engine: EnginePlaywright, Run: opts.Run, Case: f.name, URL: server.URL + f.path, Phase: "warm", RequiresJS: f.javascript}, Assertion: "visible text contains " + f.expected}
		if i == 0 {
			row.Phase = "cold"
			row.SetupMS = setupMS
		}
		caseCtx, cancel := context.WithTimeout(ctx, opts.Timeout+5*time.Second)
		caseStart := time.Now()
		response, err := p.exchange(caseCtx, map[string]any{"cmd": "task", "url": row.URL, "action": f.action, "value": f.value, "expected": f.expected, "timeout_ms": opts.Timeout.Milliseconds()})
		cancel()
		row.TaskMS = response.TaskMS
		if err != nil {
			row.TaskMS = milliseconds(time.Since(caseStart))
		}
		row.TotalMS = row.SetupMS + row.TaskMS
		row.Success = err == nil && response.OK
		if err != nil {
			row.Error = err.Error()
		} else {
			row.Error = response.Error
		}
		row.SnapshotBytes = response.SnapshotBytes
		row.DocumentBytes = response.Snapshot.DocumentBytes
		row.FinalURL, row.ObservedTitle, row.HTTPStatus = response.Snapshot.URL, response.Snapshot.Title, response.Snapshot.Status
		excerpt := []rune(response.Snapshot.Text)
		if len(excerpt) > 400 {
			excerpt = excerpt[:400]
		}
		row.ObservedTextExcerpt = string(excerpt)
		if err == nil {
			row.NodeHeapUsedBytes = response.NodeHeapUsedBytes
			row.NodeExternalBytes = response.NodeExternalBytes
			row.NodeArrayBufferBytes = response.NodeArrayBufferBytes
			row.NodeRSSBytes = response.NodeRSSBytes
			if response.NodeRSSBytes != nil {
				row.NodeRSSSource = "Node process.memoryUsage().rss, single Node process OS resident memory"
			}
		}
		var memory runtime.MemStats
		runtime.ReadMemStats(&memory)
		row.GoHeapAllocBytes, row.GoSysBytes = memory.HeapAlloc, memory.Sys
		row.HarnessRSSBytes, row.HarnessRSSSource = sampleRSS(os.Getpid(), false)
		if init.ChromiumPID > 0 {
			row.ChromiumTreeRSSBytes, row.ChromiumRSSSource = sampleRSS(init.ChromiumPID, true)
		} else {
			row.ChromiumRSSSource = "unavailable: Chromium process ID: " + init.ChromiumPIDSource
		}
		result.Samples = append(result.Samples, row)
		if err != nil {
			remaining := frameworkFailureSamples(EnginePlaywright, opts.Run, cases[i+1:], fmt.Errorf("not run after worker failure: %w", err))
			for n := range remaining {
				remaining[n].Phase = "warm"
			}
			result.Samples = append(result.Samples, remaining...)
			break
		}
	}
	return result
}

func RunAIFramework(ctx context.Context, opts Options) FrameworkRun {
	if opts.Run == 0 {
		opts.Run = 1
	}
	start := time.Now()
	opts.Engine = EngineAI
	result := FrameworkRun{Engine: EngineAI, Run: opts.Run}
	for _, row := range Run(ctx, opts) {
		result.Samples = append(result.Samples, FrameworkSample{Sample: row, Assertion: "unchanged local-v1 visible text assertion from fixtures.go"})
	}
	result.WorkerWallMS = milliseconds(time.Since(start))
	return result
}
