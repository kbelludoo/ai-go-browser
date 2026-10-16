package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// readView runs a read observation and returns its page.
func readView(t *testing.T, b *Browser) *Page {
	t.Helper()
	result := b.Execute(Command{Cmd: "snapshot", View: "read"})
	if !result.OK {
		t.Fatalf("read snapshot failed: %s", result.Error)
	}
	if result.Page == nil {
		t.Fatal("read snapshot returned no page")
	}
	return result.Page
}

func TestLiveReadViewMatchesFullSnapshotText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<title>Read page</title><main><h1>Heading visible</h1><p>Body café 日本語 😀</p>
<a id="go" href="#next">Go</a><form action="/x" method="get"><input name="q"><button>Send</button></form></main>`)
	}))
	t.Cleanup(server.Close)
	b := newLiveTestBrowser(t)
	if result := b.Execute(Command{Cmd: "open", URL: server.URL, RenderJS: true}); !result.OK {
		t.Fatalf("open: %s", result.Error)
	}
	full := b.Current
	if full == nil {
		t.Fatal("no full snapshot after open")
	}
	read := readView(t, b)
	if read.URL != full.URL || read.Title != full.Title {
		t.Fatalf("read identity differs: read=%q/%q full=%q/%q", read.URL, read.Title, full.URL, full.Title)
	}
	if read.Text != full.Text {
		t.Fatalf("read text differs from full observation:\nread=%q\nfull=%q", read.Text, full.Text)
	}
	if !strings.Contains(read.Text, "Body café 日本語 😀") {
		t.Fatalf("read text lost Unicode content: %q", read.Text)
	}
}

// A read must never look action-ready: an agent that mistook it for a full
// observation would send a stale or empty reference.
func TestLiveReadViewOmitsReferences(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<title>Refs</title><h1>Ref page</h1><a href="#a">A</a><button id="b">B</button>
<form action="/x" method="get"><input name="q"><button type="submit">Send</button></form>`)
	}))
	t.Cleanup(server.Close)
	b := newLiveTestBrowser(t)
	if result := b.Execute(Command{Cmd: "open", URL: server.URL, RenderJS: true}); !result.OK {
		t.Fatalf("open: %s", result.Error)
	}
	full := b.Execute(Command{Cmd: "snapshot"})
	if !full.OK || len(full.Page.Links) == 0 || len(full.Page.Forms) == 0 {
		t.Fatalf("full snapshot should carry references: %+v", full.Page)
	}
	read := readView(t, b)
	if len(read.Links) != 0 || len(read.Forms) != 0 || len(read.Headings) != 0 {
		t.Fatalf("read returned reference lists: links=%d forms=%d headings=%d", len(read.Links), len(read.Forms), len(read.Headings))
	}
	if read.Buttons != nil || read.Inputs != nil || read.Markdown != "" {
		t.Fatalf("read leaked optional reference data: buttons=%v inputs=%v markdown=%q", read.Buttons, read.Inputs, read.Markdown)
	}
	if read.Stack.RenderMode != "Chromium Live (read)" {
		t.Fatalf("read did not identify its engine: %+v", read.Stack)
	}
}

// The whole point of a read is polling. It must observe a late mutation, and
// it must not invalidate the references taken before it.
func TestLiveReadViewSeesLateMutationAndKeepsReferences(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<title>Late</title><p id="state">Waiting</p><input id="name" value="start">
<script>setTimeout(function(){document.getElementById('state').textContent='LATE_CONTENT_READY'},150)</script>`)
	}))
	t.Cleanup(server.Close)
	b := newLiveTestBrowser(t)
	if result := b.Execute(Command{Cmd: "open", URL: server.URL, RenderJS: true}); !result.OK {
		t.Fatalf("open: %s", result.Error)
	}
	fieldRef := ""
	for _, field := range b.Current.Inputs {
		if field.Name == "name" || field.ID == "name" {
			fieldRef = field.Ref
		}
	}
	if fieldRef == "" {
		t.Fatalf("input not discovered: %+v", b.Current.Inputs)
	}
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(readView(t, b).Text, "LATE_CONTENT_READY") {
		if time.Now().After(deadline) {
			t.Fatal("read view never observed the late mutation")
		}
		time.Sleep(20 * time.Millisecond)
	}
	// References taken before the read stay usable: a read produces no new
	// document, so the action path must still accept them.
	if result := b.Execute(Command{Cmd: "fill", Ref: fieldRef, Value: "edited"}); !result.OK {
		t.Fatalf("reference went stale after read polling: %s", result.Error)
	}
	if got := b.Execute(Command{Cmd: "query", Selector: "#name"}); !got.OK || len(got.Matches) != 1 || got.Matches[0].Value != "edited" {
		t.Fatalf("fill after read did not land: %+v", got.Matches)
	}
}

// A read inside a selected frame must report that frame, not the parent.
func TestLiveReadViewUsesSelectedFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/child":
			fmt.Fprint(w, `<title>Child title</title><p>CHILD_ONLY_CONTENT</p>`)
		default:
			fmt.Fprint(w, `<title>Parent title</title><p>PARENT_ONLY_CONTENT</p><iframe name="kid" src="/child"></iframe>`)
		}
	}))
	t.Cleanup(server.Close)
	b := newLiveTestBrowser(t)
	if result := b.Execute(Command{Cmd: "open", URL: server.URL, RenderJS: true}); !result.OK {
		t.Fatalf("open: %s", result.Error)
	}
	var child string
	for _, frame := range b.Execute(Command{Cmd: "frames"}).Frames {
		if frame.Name == "kid" {
			child = frame.Ref
		}
	}
	if child == "" {
		t.Fatal("child frame not listed")
	}
	if result := b.Execute(Command{Cmd: "frame", Ref: child}); !result.OK {
		t.Fatalf("select frame: %s", result.Error)
	}
	read := readView(t, b)
	if read.Title != "Child title" || !strings.Contains(read.Text, "CHILD_ONLY_CONTENT") {
		t.Fatalf("read escaped the selected frame: %+v", read)
	}
	if strings.Contains(read.Text, "PARENT_ONLY_CONTENT") {
		t.Fatalf("read aggregated parent text into the child observation: %q", read.Text)
	}
	if read.FrameRef != child {
		t.Fatalf("read reported frame %q, want %q", read.FrameRef, child)
	}
}

// The HTTP path already holds a parsed document, so a read is only a
// projection. It must still refuse to hand back references.
func TestHTTPReadViewOmitsReferences(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<title>Static read</title><h1>STATIC_READ_TEXT</h1><a href="#x">X</a>
<form action="/x" method="get"><input name="q"><button>Send</button></form>`)
	}))
	t.Cleanup(server.Close)
	b := New([]string{"127.0.0.1"}, false)
	t.Cleanup(b.Close)
	if result := b.Execute(Command{Cmd: "open", URL: server.URL}); !result.OK {
		t.Fatalf("open: %s", result.Error)
	}
	if result := b.Execute(Command{Cmd: "discover"}); !result.OK {
		t.Fatalf("discover: %s", result.Error)
	}
	read := readView(t, b)
	if read.Title != "Static read" || !strings.Contains(read.Text, "STATIC_READ_TEXT") {
		t.Fatalf("http read lost the page: %+v", read)
	}
	if len(read.Links) != 0 || len(read.Forms) != 0 || len(read.Headings) != 0 {
		t.Fatalf("http read returned reference lists: %+v", read)
	}
	if len(b.Current.Links) == 0 || len(b.Current.Forms) == 0 {
		t.Fatal("http read replaced the reference-bearing page")
	}
}

// The read payload is what an agent pays per poll, so it must stay far below
// the full observation on a page with many references.
func TestLiveReadViewPayloadIsSmallerThanFullSnapshot(t *testing.T) {
	var body strings.Builder
	body.WriteString(`<title>Wide</title><p>WIDE_PAGE_TEXT</p>`)
	for i := 0; i < 200; i++ {
		fmt.Fprintf(&body, `<a href="#%d">Link %d</a><button id="b%d">Button %d</button>`, i, i, i, i)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, body.String())
	}))
	t.Cleanup(server.Close)
	b := newLiveTestBrowser(t)
	if result := b.Execute(Command{Cmd: "open", URL: server.URL, RenderJS: true}); !result.OK {
		t.Fatalf("open: %s", result.Error)
	}
	fullJSON, err := json.Marshal(b.Execute(Command{Cmd: "snapshot"}).Page)
	if err != nil {
		t.Fatal(err)
	}
	readJSON, err := json.Marshal(readView(t, b))
	if err != nil {
		t.Fatal(err)
	}
	if len(readJSON) >= len(fullJSON) {
		t.Fatalf("read payload %d was not smaller than the full snapshot %d", len(readJSON), len(fullJSON))
	}
	if !strings.Contains(string(readJSON), "WIDE_PAGE_TEXT") {
		t.Fatalf("read payload lost visible text: %s", readJSON)
	}
}

func TestReadViewUnknownValueKeepsFullObservation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<title>Other view</title><h1>OTHER_VIEW_TEXT</h1><a href="#x">X</a>`)
	}))
	t.Cleanup(server.Close)
	b := New([]string{"127.0.0.1"}, false)
	t.Cleanup(b.Close)
	if result := b.Execute(Command{Cmd: "open", URL: server.URL}); !result.OK {
		t.Fatalf("open: %s", result.Error)
	}
	result := b.Execute(Command{Cmd: "snapshot", View: "everything"})
	if !result.OK || result.Page == nil {
		t.Fatalf("unknown view must fall back to the full snapshot: %+v", result)
	}
	if len(result.Page.Links) == 0 {
		t.Fatal("unknown view unexpectedly dropped references")
	}
}
