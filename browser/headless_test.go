package browser

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// Run the current test executable as a renderer so output limits and process
// cancellation are reproducible on every host, without Chromium or networking.
func TestHeadlessProcessHelper(t *testing.T) {
	if os.Getenv("AIGOBROWSER_HEADLESS_HELPER") != "1" {
		return
	}
	switch os.Getenv("AIGOBROWSER_HEADLESS_MODE") {
	case "dom":
		fmt.Fprint(os.Stdout, "\n<!doctype html><p>Rendered</p>\n")
	case "empty":
		fmt.Fprint(os.Stdout, "\t\n \u00a0")
	case "limit", "overflow":
		chunk := bytes.Repeat([]byte("x"), 32*1024)
		for remaining := MaxBody; remaining > 0; remaining -= len(chunk) {
			if _, err := os.Stdout.Write(chunk[:min(remaining, len(chunk))]); err != nil {
				os.Exit(2)
			}
		}
		if os.Getenv("AIGOBROWSER_HEADLESS_MODE") == "overflow" {
			fmt.Fprint(os.Stdout, "x")
			// An output overflow must kill the helper, not wait for normal exit.
			time.Sleep(30 * time.Second)
		}
	case "sleep":
		time.Sleep(30 * time.Second)
	case "exit":
		os.Exit(7)
	default:
		os.Exit(3)
	}
	os.Exit(0)
}

func captureHeadlessHelper(t *testing.T, ctx context.Context, mode string) ([]byte, error) {
	t.Helper()
	t.Setenv("AIGOBROWSER_HEADLESS_HELPER", "1")
	t.Setenv("AIGOBROWSER_HEADLESS_MODE", mode)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return captureHeadlessDOM(ctx, executable, []string{"-test.run=^TestHeadlessProcessHelper$"})
}

func TestCaptureHeadlessDOM(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := captureHeadlessHelper(t, ctx, "dom")
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "\n<!doctype html><p>Rendered</p>\n" {
		t.Fatalf("rendered DOM was changed: %q", output)
	}
}

func TestCaptureHeadlessDOMAtLimit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := captureHeadlessHelper(t, ctx, "limit")
	if err != nil {
		t.Fatal(err)
	}
	if len(output) != MaxBody {
		t.Fatalf("got %d DOM bytes, want %d", len(output), MaxBody)
	}
	if cap(output) > MaxBody {
		t.Fatalf("capture allocated capacity %d above limit %d", cap(output), MaxBody)
	}
}

func TestCaptureHeadlessDOMOverflowKillsRenderer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := captureHeadlessHelper(t, ctx, "overflow")
	if !errors.Is(err, errHeadlessDOMTooLarge) {
		t.Fatalf("got error %v, want output limit error", err)
	}
	if output != nil {
		t.Fatalf("overflow returned %d partial DOM bytes", len(output))
	}
	if ctx.Err() != nil {
		t.Fatalf("renderer only stopped at the caller deadline: %v", ctx.Err())
	}
}

func TestCaptureHeadlessDOMEmpty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := captureHeadlessHelper(t, ctx, "empty")
	if err == nil || err.Error() != "headless browser returned empty DOM" {
		t.Fatalf("got output %q and error %v, want empty DOM error", output, err)
	}
}

func TestCaptureHeadlessDOMProcessFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := captureHeadlessHelper(t, ctx, "exit")
	if err == nil || !strings.Contains(err.Error(), "headless browser failed:") {
		t.Fatalf("got error %v, want process failure", err)
	}
}

func TestCaptureHeadlessDOMDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := captureHeadlessHelper(t, ctx, "sleep")
	if err == nil || ctx.Err() != context.DeadlineExceeded {
		t.Fatalf("got error %v and context error %v, want deadline cancellation", err, ctx.Err())
	}
}

func TestFindSystemBrowser(t *testing.T) {
	path, err := FindSystemBrowser()
	if err != nil {
		t.Logf("system browser not found on host: %v", err)
		return
	}
	t.Logf("Found system browser at: %s", path)
}

func TestRenderHeadlessLocal(t *testing.T) {
	browserPath, err := FindSystemBrowser()
	if err != nil {
		t.Skip("skipping headless test: no browser installed")
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><head><title>JS Test</title></head><body>
		<div id="target">Initial Server Text</div>
		<script>
			document.getElementById('target').textContent = 'Rendered By Headless JS';
		</script></body></html>`)
	}))
	defer ts.Close()

	output, err := RenderHeadless(browserPath, ts.URL, 15*time.Second)
	if err != nil {
		t.Fatalf("RenderHeadless failed: %v", err)
	}

	htmlStr := string(output)
	if !strings.Contains(htmlStr, "Rendered By Headless JS") {
		t.Fatalf("expected dynamically rendered content, got: %s", htmlStr)
	}
}

func TestBrowserRenderPageWithJS(t *testing.T) {
	_, err := FindSystemBrowser()
	if err != nil {
		t.Skip("skipping test: no browser installed")
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><head><title>SPA App</title></head><body>
		<div id="root"></div>
		<script>
			var div = document.getElementById('root');
			div.innerHTML = '<h1>Dynamic React SPA Title</h1><p>Client side content loaded successfully.</p><a href="/contact">Fale Conosco</a>';
		</script></body></html>`)
	}))
	defer ts.Close()

	b := New([]string{"*"}, false)
	b.AllowAll = true
	defer b.Close()

	res := b.Execute(Command{Cmd: "open", URL: ts.URL, RenderJS: true})
	if !res.OK {
		t.Fatalf("execute open with js failed: %s", res.Error)
	}
	if res.Page == nil {
		t.Fatal("expected page, got nil")
	}
	if !strings.Contains(res.Page.Text, "Dynamic React SPA Title") {
		t.Fatalf("expected dynamic title in text, got: %q", res.Page.Text)
	}
	if len(res.Page.Links) == 0 || res.Page.Links[0].Label != "Fale Conosco" {
		t.Fatalf("expected rendered link, got: %+v", res.Page.Links)
	}
	if res.Page.Stack.RenderMode != "Chromium Live (JS Executed)" {
		t.Fatalf("expected Headless Rendered, got: %s", res.Page.Stack.RenderMode)
	}
}
