package browser

import (
	"errors"
	"testing"
)

func TestNativeOnlyModeBlocksExternalBrowsers(t *testing.T) {
	b := New([]string{"*"}, true)
	b.NativeOnly = true

	// 1. RenderPageWithJS must fail immediately with ErrExternalBrowserDisabled
	err := b.RenderPageWithJS("https://example.com")
	if !errors.Is(err, ErrExternalBrowserDisabled) {
		t.Fatalf("expected ErrExternalBrowserDisabled, got %v", err)
	}

	// 2. openLive must fail immediately with ErrExternalBrowserDisabled
	err = b.openLive("https://example.com", false)
	if !errors.Is(err, ErrExternalBrowserDisabled) {
		t.Fatalf("expected ErrExternalBrowserDisabled, got %v", err)
	}

	// 3. promoteForm must fail immediately with ErrExternalBrowserDisabled
	_, _, err = b.promoteForm("f1", "b1")
	if !errors.Is(err, ErrExternalBrowserDisabled) {
		t.Fatalf("expected ErrExternalBrowserDisabled, got %v", err)
	}

	// 4. Execute with render_js must fail gracefully without launching Chromium
	res := b.Execute(Command{
		Cmd:      "open",
		URL:      "https://example.com",
		RenderJS: true,
	})
	if res.OK {
		t.Fatalf("expected command to fail in native-only mode with RenderJS")
	}
	if res.Error != ErrExternalBrowserDisabled.Error() {
		t.Errorf("expected error %q, got %q", ErrExternalBrowserDisabled.Error(), res.Error)
	}
}

func TestAntiSSRFProtection(t *testing.T) {
	b := New([]string{"*"}, true)
	b.SetBlockPrivateNetworks(true)

	// Block loopback
	res := b.Execute(Command{
		Cmd: "open",
		URL: "http://127.0.0.1:8080/admin",
	})
	if res.OK {
		t.Fatalf("expected loopback request to be blocked by SSRF defense")
	}

	// Block localhost
	res = b.Execute(Command{
		Cmd: "open",
		URL: "http://localhost/secret",
	})
	if res.OK {
		t.Fatalf("expected localhost request to be blocked by SSRF defense")
	}

	// Block Cloud Metadata IP (169.254.169.254)
	res = b.Execute(Command{
		Cmd: "open",
		URL: "http://169.254.169.254/latest/meta-data/",
	})
	if res.OK {
		t.Fatalf("expected AWS/Oracle metadata request to be blocked by SSRF defense")
	}
}
