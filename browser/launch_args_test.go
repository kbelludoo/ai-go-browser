package browser

import (
	"context"
	"encoding/json"
	"runtime"
	"slices"
	"testing"
	"time"
)

func TestStealthLaunchArgsPreservePersistentAndVisibleProfiles(t *testing.T) {
	for _, tt := range []struct {
		name      string
		headless  bool
		persisted bool
	}{
		{name: "headless temporary", headless: true},
		{name: "headless persistent", headless: true, persisted: true},
		{name: "visible temporary"},
		{name: "visible persistent", persisted: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			args := stealthLaunchArgs(StealthConfig{Headless: tt.headless, Width: 1366, Height: 768, Lang: "pt-BR"}, "owned-profile", tt.persisted)
			for _, flag := range []string{"--user-data-dir=owned-profile", "--lang=pt-BR", "--window-size=1366,768"} {
				if !slices.Contains(args, flag) {
					t.Errorf("missing existing browser setting %q", flag)
				}
			}
			if slices.Contains(args, "--no-startup-window") != tt.headless || slices.Contains(args, "about:blank") == tt.headless {
				t.Errorf("incorrect startup page behavior: %v", args)
			}
			// Software rendering is a headless-only choice: a visible profile
			// must keep the GPU path for normal presentation.
			if slices.Contains(args, "--disable-gpu") != tt.headless {
				t.Errorf("headless GPU setting changed profile behavior: %v", args)
			}
			for _, flag := range []string{"--disable-background-networking", "--disable-background-timer-throttling", "--disable-backgrounding-occluded-windows", "--disable-breakpad", "--disable-client-side-phishing-detection", "--disable-component-extensions-with-background-pages", "--disable-component-update", "--disable-default-apps", "--disable-dev-shm-usage", "--disable-extensions", "--disable-field-trial-config", "--disable-hang-monitor", "--disable-ipc-flooding-protection", "--disable-lcd-text", "--disable-prompt-on-repost", "--disable-speech-api", "--disable-sync", "--force-color-profile=srgb", "--hide-scrollbars", "--metrics-recording-only", "--no-pings", "--no-service-autorun", "--password-store=basic", "--use-mock-keychain", "--disable-features=" + conservativeSwitches} {
				if slices.Contains(args, flag) != (tt.headless && !tt.persisted) {
					t.Errorf("background setting %q changed profile behavior: %v", flag, args)
				}
			}
		})
	}
}

// TestStealthLaunchArgsKeepBackForwardCache guards the one feature the
// conservative switch list must not disable: this project restores documents
// from the back/forward cache on history navigation.
func TestStealthLaunchArgsKeepBackForwardCache(t *testing.T) {
	args := stealthLaunchArgs(StealthConfig{Headless: true, Width: 1366, Height: 768, Lang: "en-US"}, "owned-profile", false)
	for _, arg := range args {
		if len(arg) > len("--disable-features=") && arg[:len("--disable-features=")] == "--disable-features=" {
			for _, feature := range splitFeatureList(arg[len("--disable-features="):]) {
				if feature == "BackForwardCache" {
					t.Fatalf("BackForwardCache must stay enabled, got %q", arg)
				}
			}
		}
	}
}

func splitFeatureList(list string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(list); i++ {
		if i == len(list) || list[i] == ',' {
			if i > start {
				out = append(out, list[start:i])
			}
			start = i + 1
		}
	}
	return out
}

func TestHeadlessStealthLaunchOwnsOnePage(t *testing.T) {
	path, err := FindSystemBrowser()
	if err != nil {
		t.Skip("launch integration requires installed Chromium/Edge")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	session, err := LaunchStealth(ctx, StealthConfig{BrowserPath: path, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		session.Close()
		if session.proc != nil {
			_ = session.proc.Wait()
		}
	}()
	if session.ProcessID() <= 0 {
		t.Fatal("launched browser has no process ID")
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "linux" {
		if memory := ReadProcessMemory(session.ProcessID()); !memory.Available {
			t.Fatalf("launched process is unavailable: %+v", memory)
		}
	}
	raw, err := session.browserCDP.send(ctx, "Target.getTargets", nil)
	if err != nil {
		t.Fatal(err)
	}
	var targets struct {
		Targets []struct {
			ID   string `json:"targetId"`
			Type string `json:"type"`
			URL  string `json:"url"`
		} `json:"targetInfos"`
	}
	if err := json.Unmarshal(raw, &targets); err != nil {
		t.Fatal(err)
	}
	pages := 0
	for _, target := range targets.Targets {
		if target.Type == "page" {
			pages++
			if target.ID != session.targetID || target.URL != "about:blank" {
				t.Errorf("unexpected startup page: %+v", target)
			}
		}
	}
	if pages != 1 {
		t.Fatalf("headless launch left %d pages alive: %s", pages, raw)
	}
}
