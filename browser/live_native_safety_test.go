package browser

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLiveNativeFillPreservesFailureDuringNavigation(t *testing.T) {
	refusal := errors.New("field did not accept the requested text; observe the native result")
	errorsToPreserve := []struct {
		name string
		err  error
	}{
		{"preparation-context-destroyed", errors.New("evaluate: execution context was destroyed")},
		{"preparation-target-navigated", errors.New("evaluate: inspected target navigated")},
		{"refusal-and-context-destroyed", errors.Join(refusal, fmt.Errorf("could not observe native fill outcome: %w", errors.New("execution context was destroyed")))},
		{"refusal-and-recovery-navigation", errors.Join(refusal, fmt.Errorf("could not observe native fill outcome: %w", errors.New("inspected target navigated")))},
	}
	for _, controlType := range []string{"text", "textarea", "search", "tel", "url", "email", "password", "number", "contenteditable"} {
		t.Run(controlType, func(t *testing.T) {
			for _, failure := range errorsToPreserve {
				t.Run(failure.name, func(t *testing.T) {
					if liveActionTransient(failure.err, "fill", controlType) {
						t.Fatalf("native fill failure would be converted to success: %v", failure.err)
					}
				})
			}
		})
	}
}

func TestLiveActionTransientKeepsNavigationAndContextPolicy(t *testing.T) {
	cases := []struct {
		name, command, controlType string
		err                        error
		allowRecovery              bool
	}{
		{"click-context-destroyed", "click", "button", errors.New("execution context was destroyed"), true},
		{"click-target-navigated", "click", "button", fmt.Errorf("evaluate: %w", errors.New("inspected target navigated")), true},
		{"press-navigation", "press", "submit", errors.New("inspected target navigated"), true},
		{"click-no-error", "click", "button", nil, false},
		{"click-refused", "click", "button", errors.New("element is covered"), false},
		{"click-numeric-context-missing", "click", "button", errors.New("cannot find context with specified id"), false},
		{"click-unique-context-missing", "click", "button", errors.New("uniqueContextId not found"), false},
		{"navigation-with-wrong-context", "click", "button", errors.Join(errors.New("execution context was destroyed"), errors.New("cannot find context")), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := liveActionTransient(tc.err, tc.command, tc.controlType); got != tc.allowRecovery {
				t.Fatalf("navigation recovery=%v, want %v for %v", got, tc.allowRecovery, tc.err)
			}
		})
	}
}

func TestLiveNativeEditableSnapshotBudget(t *testing.T) {
	// The page itself contains only about 400 KiB of UTF-8 text. Annotating
	// every nested textbox without an aggregate budget would duplicate it
	// about 500 times before the final serialized HTML budget is checked.
	const page = `<!doctype html><meta charset="utf-8"><title>Nested editor budget</title><script>
		const fragment = document.createDocumentFragment();
		let parent = fragment;
		for (let i = 0; i < 500; i++) {
			const textbox = document.createElement('div');
			textbox.setAttribute('role', 'textbox');
			parent.appendChild(textbox);
			parent = textbox;
		}
		parent.textContent = 'café 🙂'.repeat(40960);
		document.documentElement.appendChild(fragment);
	</script>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, page)
	}))
	t.Cleanup(srv.Close)
	b := newLiveTestBrowser(t)
	result := b.Execute(Command{Cmd: "open", URL: srv.URL, RenderJS: true})
	if result.OK || !strings.Contains(result.Error, "live editable values exceed 5242880 bytes") {
		t.Fatalf("expected annotation budget failure before HTML serialization, got OK=%v error=%s", result.OK, result.Error)
	}
	if result.Page != nil || b.Current != nil || b.Document != nil {
		t.Fatal("oversized editable annotations published a partial page")
	}
	if b.RenderProcessID() != 0 {
		t.Fatal("failed initial snapshot retained its renderer")
	}
}
