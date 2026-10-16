package browser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPersistentButtonEvents(t *testing.T) {
	worker := os.Getenv("AI_JS_WORKER")
	if worker == "" {
		t.Skip("set AI_JS_WORKER for integration")
	}
	visits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visits++
		fmt.Fprint(w, `<title>Actions</title><p id="count">0</p><button id="add">Increase</button><button id="inline" onclick="document.getElementById('count').textContent='inline'">Inline</button><script>
    var count=0;
    document.getElementById('add').addEventListener('click',function(){
      count++;document.getElementById('count').textContent=String(count);
    });
   </script>`)
	}))
	defer srv.Close()
	b := New([]string{"127.0.0.1"}, false)
	b.JSWorker = worker
	b.JSTimeout = 2 * time.Second
	defer b.closeJSSession()
	opened := b.Execute(Command{Cmd: "open", URL: srv.URL})
	if !opened.OK || len(opened.Page.Buttons) != 2 {
		t.Fatalf("open: %+v", opened)
	}
	for i := 1; i <= 3; i++ {
		clicked := b.Execute(Command{Cmd: "press", Ref: "b1"})
		if !clicked.OK || !strings.Contains(clicked.Page.Text, fmt.Sprint(i)) {
			t.Fatalf("press %d: %+v", i, clicked)
		}
	}
	if visits != 1 {
		t.Fatalf("press unexpectedly reloaded %d times", visits)
	}
	clicked := b.Execute(Command{Cmd: "press", Ref: "b2"})
	if !clicked.OK || !strings.Contains(clicked.Page.Text, "inline") {
		t.Fatalf("inline: %+v", clicked)
	}
	snapshot := b.Execute(Command{Cmd: "query", Selector: "#count"})
	if !snapshot.OK || len(snapshot.Matches) != 1 || snapshot.Matches[0].Text != "inline" {
		t.Fatalf("query: %+v", snapshot)
	}
}

func TestButtonTimeoutKeepsGoAlive(t *testing.T) {
	worker := os.Getenv("AI_JS_WORKER")
	if worker == "" {
		t.Skip("JS worker required")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<title>Safe</title><p id="count">0</p><button id="loop">Bad</button><script>document.getElementById('loop').addEventListener('click',function(){while(true) {}});</script>`)
	}))
	defer srv.Close()
	b := New([]string{"127.0.0.1"}, false)
	b.JSWorker = worker
	b.JSTimeout = 180 * time.Millisecond
	defer b.closeJSSession()
	opened := b.Execute(Command{Cmd: "open", URL: srv.URL})
	if !opened.OK {
		t.Fatal(opened.Error)
	}
	clicked := b.Execute(Command{Cmd: "press", Ref: "b1"})
	if clicked.OK || !strings.Contains(clicked.Error, "deadline") {
		t.Fatalf("expected timeout: %+v", clicked)
	}
	still := b.Execute(Command{Cmd: "snapshot"})
	if !still.OK || !strings.Contains(still.Page.Text, "0") {
		t.Fatalf("static fallback: %+v", still)
	}
}

func TestFillThenClickJSReadsValue(t *testing.T) {
	worker := os.Getenv("AI_JS_WORKER")
	if worker == "" {
		t.Skip("JS worker required")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<title>Inputs</title><form><input id="query" name="q"><button id="run" type="button">Go</button></form><p id="out">empty</p><script>
    document.getElementById('run').addEventListener('click',function(){
      document.getElementById('out').textContent=document.getElementById('query').value;
    });
   </script>`)
	}))
	defer srv.Close()
	b := New([]string{"127.0.0.1"}, false)
	b.JSWorker = worker
	b.JSTimeout = 2 * time.Second
	defer b.closeJSSession()
	opened := b.Execute(Command{Cmd: "open", URL: srv.URL})
	if !opened.OK {
		t.Fatal(opened.Error)
	}
	filled := b.Execute(Command{Cmd: "fill", Ref: "i1", Value: "IA em Go"})
	if !filled.OK {
		t.Fatal(filled.Error)
	}
	clicked := b.Execute(Command{Cmd: "press", Ref: "b1"})
	if !clicked.OK || !strings.Contains(clicked.Page.Text, "IA em Go") {
		t.Fatalf("click after fill: %+v", clicked)
	}
	if clicked.Page.Forms[0].Fields[0].Value != "IA em Go" {
		t.Fatalf("field value lost: %+v", clicked.Page.Forms)
	}
}

func TestButtonInlineHandlerWithoutScriptTag(t *testing.T) {
	worker := os.Getenv("AI_JS_WORKER")
	if worker == "" {
		t.Skip("JS worker required")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<title>No external JS</title><p id="state">before</p><button id="press" onclick="document.getElementById('state').textContent='after'">Go</button>`)
	}))
	defer srv.Close()
	b := New([]string{"127.0.0.1"}, false)
	b.JSWorker = worker
	b.JSTimeout = 2 * time.Second
	defer b.closeJSSession()
	open := b.Execute(Command{Cmd: "open", URL: srv.URL})
	if !open.OK {
		t.Fatal(open.Error)
	}
	press := b.Execute(Command{Cmd: "press", Ref: "b1"})
	if !press.OK || !strings.Contains(press.Page.Text, "after") {
		t.Fatalf("onclick event: %+v", press)
	}
}
