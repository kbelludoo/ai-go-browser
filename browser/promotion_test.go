package browser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func TestFormPromotionPreservesValuesAndUsesFreshHiddenToken(t *testing.T) {
	var sourceHits atomic.Int32
	var submitted url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/result" {
			submitted = r.URL.Query()
			fmt.Fprint(w, `<title>Submitted</title><p>Submission succeeded</p>`)
			return
		}
		token := sourceHits.Add(1)
		fmt.Fprintf(w, `<title>Form promotion</title><form id="f" action="/result?discard=yes"><input name="token" type="hidden" value="token-%d"><input id="q" name="q" required pattern="[a-z]+" value="initial"><textarea name="note">original</textarea><input name="check" type="checkbox" value="yes"><select name="tags" multiple><option value="a" selected>A</option><option value="b">B</option><option value="c" selected>C</option></select></form><button id="send" form="f" name="intent" value="search">Send</button>`, token)
	}))
	defer server.Close()
	b := newLiveTestBrowser(t)
	b.AutoRenderJS = true
	if result := b.Execute(Command{Cmd: "open", URL: server.URL}); !result.OK || result.Engine != "http" {
		t.Fatalf("initial static navigation: %+v", result)
	}
	for name, value := range map[string]string{"q": "edited", "note": "line1\nline2"} {
		if result := b.Execute(Command{Cmd: "fill", Ref: liveTestField(t, b, name), Value: value}); !result.OK {
			t.Fatal(result.Error)
		}
	}
	checked := true
	if result := b.Execute(Command{Cmd: "check", Ref: liveTestField(t, b, "check"), Checked: &checked}); !result.OK {
		t.Fatal(result.Error)
	}
	result := b.Execute(Command{Cmd: "submit", Ref: "f1"})
	if !result.OK || result.Engine != "chromium" || result.Page.Title != "Submitted" {
		t.Fatalf("promoted submission: %+v", result)
	}
	if submitted.Get("q") != "edited" || submitted.Get("note") != "line1\r\nline2" || submitted.Get("check") != "yes" || submitted.Get("intent") != "search" || submitted.Get("token") != "token-2" || !reflect.DeepEqual(submitted["tags"], []string{"a", "c"}) || submitted.Has("discard") {
		t.Fatalf("controls after promotion: %v", submitted)
	}
}

func TestFormPromotionNativeValidationAndMultipart(t *testing.T) {
	var received atomic.Int32
	var contentType, query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/result" {
			received.Add(1)
			contentType = r.Header.Get("Content-Type")
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Errorf("multipart parse: %v", err)
			}
			query = r.FormValue("q")
			fmt.Fprint(w, `<title>Multipart received</title><p>Done</p>`)
			return
		}
		fmt.Fprint(w, `<title>Validated form</title><form method="POST" enctype="multipart/form-data" action="/result"><input name="q" required pattern="[a-z]+"><button>Send</button></form>`)
	}))
	defer server.Close()
	b := newLiveTestBrowser(t)
	b.AutoRenderJS, b.AllowPost = true, true
	if result := b.Execute(Command{Cmd: "open", URL: server.URL}); !result.OK {
		t.Fatal(result.Error)
	}
	if result := b.Execute(Command{Cmd: "submit", Ref: "f1"}); result.OK || b.RenderProcessID() != 0 {
		t.Fatal("POST without confirmation promoted or submitted")
	}
	result := b.Execute(Command{Cmd: "submit", Ref: "f1", Confirm: true})
	if result.OK || !strings.Contains(result.Error, "form validation failed") || received.Load() != 0 {
		t.Fatalf("native required validation: %+v", result)
	}
	if result := b.Execute(Command{Cmd: "fill", Ref: liveTestField(t, b, "q"), Value: "valid"}); !result.OK {
		t.Fatal(result.Error)
	}
	if result := b.Execute(Command{Cmd: "submit", Ref: "f1", Confirm: true}); !result.OK || result.Page.Title != "Multipart received" {
		t.Fatalf("multipart submission: %+v", result)
	}
	if received.Load() != 1 || query != "valid" || !strings.HasPrefix(contentType, "multipart/form-data; boundary=") {
		t.Fatalf("native encoding: count=%d type=%q q=%q", received.Load(), contentType, query)
	}
}

func TestFormPromotionRejectsChangedControls(t *testing.T) {
	var sourceHits, resultHits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/result" {
			resultHits.Add(1)
		}
		name := "q"
		if sourceHits.Add(1) > 1 {
			name = "different"
		}
		fmt.Fprintf(w, `<title>Changed controls</title><form action="/result"><input id="q" name="%s" required value="initial"><button>Send</button></form>`, name)
	}))
	defer server.Close()
	b := newLiveTestBrowser(t)
	b.AutoRenderJS = true
	if result := b.Execute(Command{Cmd: "open", URL: server.URL}); !result.OK {
		t.Fatal(result.Error)
	}
	if result := b.Execute(Command{Cmd: "submit", Ref: "f1"}); result.OK || !strings.Contains(result.Error, "controls changed") || resultHits.Load() != 0 {
		t.Fatalf("mismatched control submitted: %+v", result)
	}
}

func TestFormPromotionIgnoresUnrelatedEditorAndOtherForm(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/result" {
			if r.URL.Query().Get("q") != "valid" {
				t.Error("promoted form lost its value")
			}
			fmt.Fprint(w, `<title>Submitted</title><p>Done</p>`)
			return
		}
		unrelatedName := "initial"
		if hits.Add(1) > 1 {
			unrelatedName = "changed"
		}
		fmt.Fprintf(w, `<title>Independent forms</title><div id="chat" role="textbox" contenteditable="true">Editor</div><form id="target" action="/result"><input id="q" name="q" required value="valid"><button>Submit</button></form><form><input name="%s"></form>`, unrelatedName)
	}))
	defer server.Close()
	b := newLiveTestBrowser(t)
	b.AutoRenderJS = true
	if r := b.Execute(Command{Cmd: "open", URL: server.URL}); !r.OK {
		t.Fatal(r.Error)
	}
	if r := b.Execute(Command{Cmd: "submit", Ref: "f1"}); !r.OK || r.Page.Title != "Submitted" {
		t.Fatalf("unrelated controls blocked promotion: %+v", r)
	}
}

func TestFormPromotionRejectsChangedAssociationBeforeAndAfterEvents(t *testing.T) {
	for _, afterEvent := range []bool{false, true} {
		t.Run(fmt.Sprintf("after-event-%t", afterEvent), func(t *testing.T) {
			var hits, submitted atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				if r.URL.Path == "/result" {
					submitted.Add(1)
					fmt.Fprint(w, `<title>Unexpected submission</title>`)
					return
				}
				changed := ""
				if hits.Add(1) > 1 {
					if afterEvent {
						changed = `oninput="this.setAttribute('form','other')"`
					} else {
						changed = `form="other"`
					}
				}
				fmt.Fprintf(w, `<title>Changed owner</title><form id="f" action="/result"><input id="q" name="q" required value="original" %s><button>Send</button></form><form id="other"></form>`, changed)
			}))
			defer server.Close()
			b := newLiveTestBrowser(t)
			b.AutoRenderJS = true
			if r := b.Execute(Command{Cmd: "open", URL: server.URL}); !r.OK {
				t.Fatal(r.Error)
			}
			if r := b.Execute(Command{Cmd: "fill", Ref: liveTestField(t, b, "q"), Value: "edited"}); !r.OK {
				t.Fatal(r.Error)
			}
			if r := b.Execute(Command{Cmd: "submit", Ref: "f1"}); r.OK || submitted.Load() != 0 {
				t.Fatalf("changed owner submitted: %+v count=%d", r, submitted.Load())
			}
		})
	}
}
