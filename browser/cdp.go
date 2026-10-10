package browser

// Minimal Chrome DevTools Protocol client over WebSocket.
// Launches a real Chromium instance (Chrome/Edge/Brave) with a stealth
// fingerprint and exposes navigate/evaluate/mouse primitives. No third-party
// automation framework — every message is hand-shaped CDP JSON.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type cdpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *cdpError) Error() string { return fmt.Sprintf("cdp error %d: %s", e.Code, e.Message) }

type cdpMessage struct {
	ID      int             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *cdpError       `json:"error,omitempty"`
	Session string          `json:"sessionId,omitempty"`
}

type cdpConn struct {
	ws      *websocket.Conn
	writeMu sync.Mutex
	mu      sync.Mutex
	nextID  int
	pending map[int]chan cdpMessage
	events  chan cdpMessage
	closed  chan struct{}
	readErr error
}

func dialCDP(ctx context.Context, wsURL string) (*cdpConn, error) {
	ws, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("cdp dial: %w", err)
	}
	c := &cdpConn{
		ws:      ws,
		pending: make(map[int]chan cdpMessage),
		events:  make(chan cdpMessage, 256),
		closed:  make(chan struct{}),
	}
	go c.readLoop()
	return c, nil
}

func (c *cdpConn) readLoop() {
	defer close(c.closed)
	for {
		_, data, err := c.ws.Read(context.Background())
		if err != nil {
			c.readErr = err
			return
		}
		var msg cdpMessage
		if json.Unmarshal(data, &msg) != nil {
			continue
		}
		if msg.ID != 0 {
			c.mu.Lock()
			ch := c.pending[msg.ID]
			delete(c.pending, msg.ID)
			c.mu.Unlock()
			if ch != nil {
				ch <- msg
			}
			continue
		}
		select {
		case c.events <- msg:
		default:
		}
	}
}

func (c *cdpConn) send(ctx context.Context, method string, params map[string]any) (json.RawMessage, error) {
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	ch := make(chan cdpMessage, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	payload := map[string]any{"id": id, "method": method}
	if params != nil {
		payload["params"] = params
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		c.dropPending(id)
		return nil, err
	}
	c.writeMu.Lock()
	err = c.ws.Write(ctx, websocket.MessageText, buf)
	c.writeMu.Unlock()
	if err != nil {
		c.dropPending(id)
		return nil, fmt.Errorf("cdp write %s: %w", method, err)
	}
	select {
	case msg := <-ch:
		if msg.Error != nil {
			return nil, msg.Error
		}
		return msg.Result, nil
	case <-ctx.Done():
		c.dropPending(id)
		return nil, ctx.Err()
	case <-c.closed:
		return nil, fmt.Errorf("cdp connection closed: %v", c.readErr)
	}
}

func (c *cdpConn) dropPending(id int) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

func (c *cdpConn) waitEvent(ctx context.Context, method string, timeout time.Duration) (cdpMessage, error) {
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		select {
		case msg := <-c.events:
			if msg.Method == method {
				return msg, nil
			}
		case <-tctx.Done():
			return cdpMessage{}, fmt.Errorf("timeout waiting for %s", method)
		case <-c.closed:
			return cdpMessage{}, fmt.Errorf("cdp connection closed: %v", c.readErr)
		}
	}
}

func (c *cdpConn) close() {
	if c == nil || c.ws == nil {
		return
	}
	_ = c.ws.Close(websocket.StatusNormalClosure, "")
	<-c.closed
}

// StealthConfig controls how the automation browser is launched.
type StealthConfig struct {
	BrowserPath string
	Headless    bool
	Width       int
	Height      int
	Lang        string
}

// StealthSession is a live CDP-controlled browser with stealth patches applied.
type StealthSession struct {
	proc        *exec.Cmd
	userDataDir string
	cleanupDir  string
	browserCDP  *cdpConn
	pageCDP     *cdpConn
	targetID    string
}

// LaunchStealth starts a real Chromium instance patched against common
// automation detectors (webdriver flag, missing chrome runtime, empty plugin
// list, permissions probe). Returns a session with an about:blank page ready.
func LaunchStealth(ctx context.Context, cfg StealthConfig) (*StealthSession, error) {
	browserPath := cfg.BrowserPath
	if browserPath == "" {
		var err error
		browserPath, err = FindSystemBrowser()
		if err != nil {
			return nil, err
		}
	}
	if cfg.Width == 0 {
		cfg.Width = 1366
	}
	if cfg.Height == 0 {
		cfg.Height = 768
	}
	if cfg.Lang == "" {
		cfg.Lang = "en-US"
	}

	userDataDir, err := os.MkdirTemp("", "aigobrowser-stealth-*")
	if err != nil {
		return nil, err
	}

	args := []string{
		"--remote-debugging-port=0",
		"--user-data-dir=" + userDataDir,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-blink-features=AutomationControlled",
		"--disable-popup-blocking",
		"--mute-audio",
		"--lang=" + cfg.Lang,
		fmt.Sprintf("--window-size=%d,%d", cfg.Width, cfg.Height),
	}
	if cfg.Headless {
		args = append(args, "--headless=new")
	}
	args = append(args, "about:blank")

	proc := exec.Command(browserPath, args...)
	proc.Stdout = nil
	proc.Stderr = nil
	if os.Getenv("AIGOBROWSER_CDP_DEBUG") != "" {
		proc.Stdout = os.Stderr
		proc.Stderr = os.Stderr
	}
	if err := proc.Start(); err != nil {
		os.RemoveAll(userDataDir)
		return nil, fmt.Errorf("launch browser: %w", err)
	}

	port, wsPath, err := waitDevToolsPort(userDataDir, 15*time.Second)
	if err != nil {
		killTree(proc)
		os.RemoveAll(userDataDir)
		return nil, err
	}

	browserCDP, err := dialCDP(ctx, fmt.Sprintf("ws://127.0.0.1:%d%s", port, wsPath))
	if err != nil {
		killTree(proc)
		os.RemoveAll(userDataDir)
		return nil, err
	}

	createRes, err := browserCDP.send(ctx, "Target.createTarget", map[string]any{"url": "about:blank"})
	if err != nil {
		browserCDP.close()
		killTree(proc)
		os.RemoveAll(userDataDir)
		return nil, fmt.Errorf("create target: %w", err)
	}
	var created struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(createRes, &created); err != nil || created.TargetID == "" {
		browserCDP.close()
		killTree(proc)
		os.RemoveAll(userDataDir)
		return nil, errors.New("no targetId from Target.createTarget")
	}

	pageCDP, err := dialCDP(ctx, fmt.Sprintf("ws://127.0.0.1:%d/devtools/page/%s", port, created.TargetID))
	if err != nil {
		browserCDP.close()
		killTree(proc)
		os.RemoveAll(userDataDir)
		return nil, fmt.Errorf("dial page target: %w", err)
	}

	if _, err := pageCDP.send(ctx, "Page.enable", nil); err != nil {
		pageCDP.close()
		browserCDP.close()
		killTree(proc)
		os.RemoveAll(userDataDir)
		return nil, err
	}
	if _, err := pageCDP.send(ctx, "Runtime.enable", nil); err != nil {
		pageCDP.close()
		browserCDP.close()
		killTree(proc)
		os.RemoveAll(userDataDir)
		return nil, err
	}
	if _, err := pageCDP.send(ctx, "Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": stealthScript}); err != nil {
		pageCDP.close()
		browserCDP.close()
		killTree(proc)
		os.RemoveAll(userDataDir)
		return nil, fmt.Errorf("install stealth script: %w", err)
	}

	return &StealthSession{
		proc:        proc,
		userDataDir: userDataDir,
		cleanupDir:  userDataDir,
		browserCDP:  browserCDP,
		pageCDP:     pageCDP,
		targetID:    created.TargetID,
	}, nil
}

func waitDevToolsPort(profileDir string, timeout time.Duration) (int, string, error) {
	deadline := time.Now().Add(timeout)
	portFile := filepath.Join(profileDir, "DevToolsActivePort")
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(portFile)
		if err == nil {
			lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
			if len(lines) >= 2 {
				port, perr := strconv.Atoi(strings.TrimSpace(lines[0]))
				path := strings.TrimSpace(lines[1])
				if perr == nil && port > 0 && strings.HasPrefix(path, "/") {
					return port, path, nil
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return 0, "", errors.New("DevToolsActivePort never appeared")
}

// Navigate loads a URL and waits for the load event.
func (s *StealthSession) Navigate(ctx context.Context, targetURL string) error {
	res, err := s.pageCDP.send(ctx, "Page.navigate", map[string]any{"url": targetURL})
	if err != nil {
		return err
	}
	var nav struct {
		ErrorText string `json:"errorText"`
	}
	if err := json.Unmarshal(res, &nav); err == nil && nav.ErrorText != "" {
		return fmt.Errorf("navigate %s: %s", targetURL, nav.ErrorText)
	}
	if _, err := s.pageCDP.waitEvent(ctx, "Page.loadEventFired", 30*time.Second); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	return nil
}

// Evaluate runs an expression in the page context and returns the JSON value.
func (s *StealthSession) Evaluate(ctx context.Context, expr string) (json.RawMessage, error) {
	res, err := s.pageCDP.send(ctx, "Runtime.evaluate", map[string]any{
		"expression":    expr,
		"awaitPromise":  true,
		"returnByValue": true,
	})
	if err != nil {
		return nil, err
	}
	var out struct {
		Result struct {
			Type           string          `json:"type"`
			Subtype        string          `json:"subtype"`
			Value          json.RawMessage `json:"value"`
			Description    string          `json:"description"`
			Unserializable string          `json:"unserializableValue"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text        string `json:"text"`
			Description string `json:"exception"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(res, &out); err != nil {
		return nil, err
	}
	if out.ExceptionDetails != nil {
		d := out.ExceptionDetails.Description
		if d == "" {
			d = out.ExceptionDetails.Text
		}
		return nil, fmt.Errorf("evaluate: %s", d)
	}
	if out.Result.Subtype == "error" {
		return nil, fmt.Errorf("evaluate: %s", out.Result.Description)
	}
	return out.Result.Value, nil
}

// MouseJitter dispatches a few small mouse moves so pointer-driven scripts
// (reCAPTCHA telemetry) observe live input rather than a static cursor.
func (s *StealthSession) MouseJitter(ctx context.Context, steps int) {
	if steps <= 0 {
		steps = 3
	}
	base := time.Now().UnixNano()
	for i := 0; i < steps; i++ {
		x := 200 + int((base+int64(i*7919))%900)
		y := 150 + int((base+int64(i*104729))%500)
		_, _ = s.pageCDP.send(ctx, "Input.dispatchMouseEvent", map[string]any{
			"type":    "mouseMoved",
			"x":       x,
			"y":       y,
			"buttons": 0,
		})
		time.Sleep(time.Duration(80+i*40) * time.Millisecond)
	}
}

// Close tears down CDP connections, kills the browser tree, removes the profile.
func (s *StealthSession) Close() {
	if s == nil {
		return
	}
	if s.browserCDP != nil && s.targetID != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, _ = s.browserCDP.send(ctx, "Target.closeTarget", map[string]any{"targetId": s.targetID})
		cancel()
	}
	if s.pageCDP != nil {
		s.pageCDP.close()
	}
	if s.browserCDP != nil {
		s.browserCDP.close()
	}
	if s.proc != nil {
		killTree(s.proc)
	}
	if s.cleanupDir != "" {
		for i := 0; i < 5; i++ {
			if err := os.RemoveAll(s.cleanupDir); err == nil {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func killTree(proc *exec.Cmd) {
	if proc == nil || proc.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(proc.Process.Pid)).Run()
		return
	}
	_ = proc.Process.Kill()
}
