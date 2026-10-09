package browser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

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
	if res.Page.Stack.RenderMode != "Headless Rendered (JS Executed)" {
		t.Fatalf("expected Headless Rendered, got: %s", res.Page.Stack.RenderMode)
	}
}
