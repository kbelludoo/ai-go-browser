package browser

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// FindSystemBrowser searches for an installed Chromium-based browser (Edge, Chrome, Brave, Chromium).
func FindSystemBrowser() (string, error) {
	var candidates []string

	switch runtime.GOOS {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		progFiles := os.Getenv("ProgramFiles")
		progFilesX86 := os.Getenv("ProgramFiles(x86)")

		candidates = []string{
			filepath.Join(progFilesX86, "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(progFiles, "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(progFiles, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(progFilesX86, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(localAppData, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(progFiles, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
		}
	case "darwin":
		candidates = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
	default: // linux, freebsd
		candidates = []string{
			"google-chrome-stable",
			"google-chrome",
			"chromium-browser",
			"chromium",
			"microsoft-edge-stable",
			"microsoft-edge",
			"brave-browser",
		}
		// Check PATH first on Linux
		for _, name := range candidates {
			if path, err := exec.LookPath(name); err == nil {
				return path, nil
			}
		}
	}

	for _, p := range candidates {
		if p == "" {
			continue
		}
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, nil
		}
	}

	return "", errors.New("no system Chromium/Edge browser found for headless rendering")
}

// RenderHeadless fetches the fully rendered DOM of a URL using an ephemeral headless browser.
// It executes the browser with resource-constrained flags, captures the rendered HTML,
// and immediately terminates the process, keeping memory overhead minimal.
func RenderHeadless(browserPath, targetURL string, timeout time.Duration) ([]byte, error) {
	if browserPath == "" {
		var err error
		browserPath, err = FindSystemBrowser()
		if err != nil {
			return nil, err
		}
	}

	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Temporary profile directory to ensure clean isolated execution
	tempProfile, err := os.MkdirTemp("", "aigobrowser-ephemeral-*")
	if err == nil {
		defer os.RemoveAll(tempProfile)
	}

	args := []string{
		"--headless=new",
		"--dump-dom",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--disable-extensions",
		"--disable-background-networking",
		"--disable-sync",
		"--disable-default-apps",
		"--mute-audio",
		"--virtual-time-budget=5000",
	}

	if tempProfile != "" {
		args = append(args, "--user-data-dir="+tempProfile)
	}

	args = append(args, targetURL)

	cmd := exec.CommandContext(ctx, browserPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("headless rendering timed out after %v", timeout)
		}
		return nil, fmt.Errorf("headless browser failed: %w", err)
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil, errors.New("headless browser returned empty DOM")
	}

	return output, nil
}
