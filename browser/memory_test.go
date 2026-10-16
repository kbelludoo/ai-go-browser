package browser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNavigationReleasesUnusedSourceHTML(t *testing.T) {
	raw := `<title>Article</title><p>Read this article.</p><button onclick="fetch('/api/items')">Items</button><script>` + strings.Repeat("x", 1<<20) + `</script>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, raw)
	}))
	defer srv.Close()
	for _, worker := range []string{"", t.TempDir() + "/missing-worker"} {
		t.Run(fmt.Sprintf("worker=%q", worker), func(t *testing.T) {
			b := New([]string{"127.0.0.1"}, false)
			defer b.Close()
			b.JSWorker = worker
			r := b.Execute(Command{Cmd: "open", URL: srv.URL})
			if !r.OK || r.Page.Title != "Article" {
				t.Fatalf("navigation: %+v", r)
			}
			if len(b.rawHTML) != 0 {
				t.Fatalf("retained %d unused source bytes", len(b.rawHTML))
			}
			// Endpoint discovery must still work from the DOM without source HTML.
			found := false
			for _, a := range b.Discover() {
				found = found || a.URL == srv.URL+"/api/items"
			}
			if !found {
				t.Fatal("endpoint discovery lost after releasing source HTML")
			}
		})
	}
}

func TestBoundedHistoryAndFailedBack(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<title>History</title><p>Page</p>")
	}))
	defer srv.Close()
	b := New([]string{"127.0.0.1"}, false)
	defer b.Close()
	b.MaxHistory = 2
	for i := 0; i < 5; i++ {
		r := b.Execute(Command{Cmd: "open", URL: fmt.Sprintf("%s/%d", srv.URL, i)})
		if !r.OK {
			t.Fatal(r.Error)
		}
	}
	if len(b.History) != 2 || b.History[0] != srv.URL+"/2" || b.History[1] != srv.URL+"/3" {
		t.Fatalf("bounded history: %v", b.History)
	}
	b.Allowed = nil
	if r := b.Execute(Command{Cmd: "back"}); r.OK {
		t.Fatal("expected blocked navigation")
	}
	if len(b.History) != 2 || b.History[1] != srv.URL+"/3" || b.Current.URL != srv.URL+"/4" {
		t.Fatal("failed back changed history or current page")
	}
	b.Allowed = map[string]bool{"127.0.0.1": true}
	if r := b.Execute(Command{Cmd: "back"}); !r.OK || r.Page.URL != srv.URL+"/3" {
		t.Fatalf("back retry: %+v", r)
	}
	if len(b.History) != 1 {
		t.Fatalf("history after back: %v", b.History)
	}
	// The popped URL must not remain alive in the backing array.
	if tail := b.History[:cap(b.History)][1]; tail != "" {
		t.Fatalf("popped URL retained: %q", tail)
	}
}

func TestHistoryModesAndClose(t *testing.T) {
	for _, limit := range []int{0, 1, -1} {
		b := New(nil, false)
		b.MaxHistory = limit
		for i := 0; i < 4; i++ {
			b.Current = &Page{URL: fmt.Sprint(i)}
			b.rememberCurrent()
		}
		want := limit
		if limit < 0 {
			want = 4
		}
		if len(b.History) != want {
			t.Fatalf("limit=%d: history=%v", limit, b.History)
		}
		b.Document = &Document{}
		b.rawHTML = []byte("cached HTML")
		b.scannedScriptText = "cached JavaScript"
		b.Close()
		b.Close()
		if b.Current != nil || b.Document != nil || b.History != nil || b.rawHTML != nil || b.scannedScriptText != "" {
			t.Fatal("Close retained page data")
		}
	}
}

func TestMemoryStatsSource(t *testing.T) {
	m := mem()
	if m.HeapMiB <= 0 || m.SysMiB <= 0 || m.RSSMiB <= 0 {
		t.Fatalf("invalid memory metrics: %+v", m)
	}
	if m.RSSSource != "go_sys_estimate" && m.RSSSource != "proc_status" && m.RSSSource != "windows_working_set" {
		t.Fatalf("unidentified RSS source: %+v", m)
	}
}

func TestHistoryLimitCanBeLowered(t *testing.T) {
	b := New(nil, false)
	defer b.Close()
	b.History = []string{"oldest", "older", "previous"}
	b.Current = &Page{URL: "current"}
	b.MaxHistory = 2
	b.rememberCurrent()
	if len(b.History) != 2 || b.History[0] != "previous" || b.History[1] != "current" {
		t.Fatalf("lowered limit: %v", b.History)
	}
	b.MaxHistory = 0
	b.rememberCurrent()
	if b.History != nil {
		t.Fatalf("disabled history retained URLs: %v", b.History)
	}
}
