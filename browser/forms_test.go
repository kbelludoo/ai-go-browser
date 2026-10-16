package browser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestBrowserControlCommandsAndQueryRefs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<textarea id="chat" aria-label="Message">initial</textarea><form id="main">
<input id="disabled" disabled value="fixed"><input id="readonly" readonly value="fixed">
<input id="accept" type="checkbox"><input id="radio1" type="radio" name="choice" value="a" checked><input id="radio2" type="radio" name="choice" value="b">
<select id="color"><option value="a">A</option><option value="b" disabled>B</option><option value="c">C</option><option value="">Empty</option></select><button id="submit">Go</button></form><a id="link" href="/next">Next</a>`)
	}))
	defer server.Close()
	browser := New([]string{"127.0.0.1"}, false)
	defer browser.Close()
	opened := browser.Execute(Command{Cmd: "open", URL: server.URL})
	if !opened.OK {
		t.Fatal(opened.Error)
	}
	refs := map[string]string{}
	for _, input := range browser.Current.Inputs {
		refs[input.ID] = input.Ref
	}
	for _, field := range browser.Current.Forms[0].Fields {
		refs[field.ID] = field.Ref
	}
	refs["main"] = browser.Current.Forms[0].Ref
	refs["submit"] = browser.Current.Buttons[0].Ref
	refs["link"] = browser.Current.Links[0].Ref
	for _, id := range []string{"chat", "accept", "main", "submit", "link"} {
		query := browser.Execute(Command{Cmd: "query", Selector: "#" + id})
		if !query.OK || len(query.Matches) != 1 || query.Matches[0].Ref != refs[id] {
			t.Fatalf("query %q=%+v; expected ref %q", id, query, refs[id])
		}
	}
	if result := browser.Execute(Command{Cmd: "fill", Ref: refs["chat"], Value: "new message"}); !result.OK || result.Page.Inputs[0].Value != "new message" {
		t.Fatalf("standalone fill=%+v", result)
	}
	for _, id := range []string{"readonly", "disabled", "accept"} {
		if result := browser.Execute(Command{Cmd: "fill", Ref: refs[id], Value: "changed"}); result.OK {
			t.Fatalf("fill %q unexpectedly succeeded", id)
		}
	}
	checked := true
	if result := browser.Execute(Command{Cmd: "check", Ref: refs["accept"], Checked: &checked}); !result.OK {
		t.Fatal(result.Error)
	}
	query := browser.Execute(Command{Cmd: "query", Selector: "#accept"})
	if !query.OK || len(query.Matches) != 1 || query.Matches[0].Checked == nil || !*query.Matches[0].Checked || query.Matches[0].Value != "on" {
		t.Fatalf("checkbox after check=%+v", query)
	}
	if result := browser.Execute(Command{Cmd: "check", Ref: refs["radio2"], Checked: &checked}); !result.OK {
		t.Fatal(result.Error)
	}
	for _, field := range browser.Current.Forms[0].Fields {
		if field.Type == "radio" && field.Checked != (field.ID == "radio2") {
			t.Fatalf("radio state after check=%+v", browser.Current.Forms[0].Fields)
		}
	}
	if result := browser.Execute(Command{Cmd: "fill", Ref: refs["color"], Value: "c"}); !result.OK {
		t.Fatal(result.Error)
	}
	for _, value := range []string{"b", "missing"} {
		if result := browser.Execute(Command{Cmd: "fill", Ref: refs["color"], Value: value}); result.OK {
			t.Fatalf("invalid select fill %q succeeded", value)
		}
	}
	query = browser.Execute(Command{Cmd: "query", Selector: "#color"})
	if !query.OK || len(query.Matches) != 1 || !reflect.DeepEqual(query.Matches[0].Values, []string{"c"}) {
		t.Fatalf("select state=%+v", query)
	}
}

func TestBrowserSubmitterOverrideAndPostGuard(t *testing.T) {
	type submission struct {
		path, method string
		values       url.Values
	}
	received := make(chan submission, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path != "/" {
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			received <- submission{path: r.URL.Path, method: r.Method, values: r.Form}
			fmt.Fprint(w, "<h1>Submitted</h1>")
			return
		}
		fmt.Fprint(w, `<form id="main" action="/fallback"><input name="q" value="test">
<button id="disabled" disabled>Disabled</button><button id="get" name="which" value="get" formaction="/override">Get</button></form>
<button id="post" form="main" name="which" value="post" formaction="/write" formmethod="POST">Post</button>`)
	}))
	defer server.Close()
	browser := New([]string{"127.0.0.1"}, false)
	defer browser.Close()
	open := func() map[string]string {
		if result := browser.Execute(Command{Cmd: "open", URL: server.URL}); !result.OK {
			t.Fatal(result.Error)
		}
		refs := map[string]string{}
		for _, button := range browser.Current.Buttons {
			refs[button.ID] = button.Ref
		}
		return refs
	}
	refs := open()
	if result := browser.Execute(Command{Cmd: "press", Ref: refs["disabled"]}); result.OK {
		t.Fatal("disabled button press succeeded")
	}
	if result := browser.Execute(Command{Cmd: "submit", Ref: "f1"}); !result.OK {
		t.Fatal(result.Error)
	}
	first := <-received
	if first.path != "/override" || first.method != "GET" || !reflect.DeepEqual(first.values, url.Values{"q": {"test"}, "which": {"get"}}) {
		t.Fatalf("default submitter=%+v", first)
	}
	refs = open()
	for _, confirm := range []bool{false, true} {
		if result := browser.Execute(Command{Cmd: "press", Ref: refs["post"], Confirm: confirm}); result.OK {
			t.Fatal("POST was permitted without AllowPost")
		}
	}
	browser.AllowPost = true
	if result := browser.Execute(Command{Cmd: "press", Ref: refs["post"]}); result.OK {
		t.Fatal("POST was permitted without confirmation")
	}
	if result := browser.Execute(Command{Cmd: "press", Ref: refs["post"], Confirm: true}); !result.OK {
		t.Fatal(result.Error)
	}
	second := <-received
	if second.path != "/write" || second.method != "POST" || !reflect.DeepEqual(second.values, url.Values{"q": {"test"}, "which": {"post"}}) {
		t.Fatalf("external POST submitter=%+v", second)
	}
}
