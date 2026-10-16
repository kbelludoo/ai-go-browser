package browser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestDOMFormControlStates(t *testing.T) {
	raw := []byte(`<label for="query">Search terms</label><input id="query" name="q" form="main" value="outside">
<form id="main" action="/search">
<fieldset disabled><legend><input name="legend" value="allowed"></legend><input name="blocked" value="no"><legend><input name="secondlegend" value="no"></legend></fieldset>
<input name="unchecked" type="CHECKBOX" value="yes"><input name="empty" type="checkbox" value="" checked><input name="default" type="checkbox" checked>
<input name="choice" type="radio" value="first" checked><input name="choice" type="radio" value="last" checked>
<input name="locked" readonly value="fixed"><input name="unknown" type="invented" value="plain">
<textarea name="note" value="wrong">
One &amp;
 two</textarea>
<button disabled>Unavailable</button><input type="SUBMIT" name="submitter" value="Go"><button>Later</button>
<input name="moved" form="other" value="elsewhere"><input name="orphan" form="missing" value="alone">
</form><form id="other"></form>`)
	doc, err := ParseDocument(raw, "https://example.org/root", 200, len(raw))
	if err != nil {
		t.Fatal(err)
	}
	for engine, page := range map[string]Page{
		"DOM":    doc.Page(5000),
		"Parser": Parse(raw, "https://example.org/root", 200, len(raw), 5000),
	} {
		t.Run(engine, func(t *testing.T) {
			if len(page.Forms) != 2 {
				t.Fatalf("forms=%+v", page.Forms)
			}
			form := page.Forms[0]
			byName := map[string]Field{}
			for _, field := range form.Fields {
				byName[field.Name] = field
			}
			if byName["q"].Label != "Search terms" || byName["q"].Value != "outside" || byName["q"].Ref != "i1" {
				t.Fatalf("external labeled field=%+v", byName["q"])
			}
			if byName["legend"].Disabled || !byName["blocked"].Disabled || !byName["secondlegend"].Disabled {
				t.Fatalf("fieldset inheritance=%+v", form.Fields)
			}
			if byName["unchecked"].Checked || byName["unchecked"].Value != "yes" || !byName["empty"].Checked || byName["empty"].Value != "" || byName["default"].Value != "on" {
				t.Fatalf("checkbox states=%+v", form.Fields)
			}
			for _, field := range form.Fields {
				if field.Type == "radio" && field.Checked != (field.Value == "last") {
					t.Fatalf("radio group=%+v", form.Fields)
				}
			}
			if !byName["locked"].ReadOnly || byName["unknown"].Type != "text" || byName["note"].Value != "One &\n two" {
				t.Fatalf("text control states=%+v", form.Fields)
			}
			if len(page.Forms[1].Fields) != 1 || page.Forms[1].Fields[0].Name != "moved" || len(page.Inputs) != 1 || page.Inputs[0].Name != "orphan" {
				t.Fatalf("form associations: forms=%+v inputs=%+v", page.Forms, page.Inputs)
			}
			if len(page.Buttons) != 3 || !page.Buttons[0].Disabled || page.Buttons[1].Type != "submit" || page.Buttons[2].Type != "submit" || form.SubmitRef != page.Buttons[1].Ref || form.SubmitLabel != "Go" || page.Buttons[1].FormRef != form.Ref {
				t.Fatalf("submit buttons: form=%+v buttons=%+v", form, page.Buttons)
			}
		})
	}
}

func TestDOMSelectValuesAndMutation(t *testing.T) {
	raw := []byte(`<form><select name="empty"><option value="" selected>Choose</option><option value="other">Other</option></select>
<select name="multiple" multiple><option value="" selected>Empty</option><option value="blue" selected>Blue</option><optgroup disabled><option value="blocked" selected>Blocked</option></optgroup></select>
<select name="none" multiple><option value="first">First</option></select>
<select name="list" size="3"><option value="first">First</option></select>
<select name="default"><option disabled value="blocked">Blocked</option><option value="">Empty</option><option>Last label</option></select>
<select name="disabledselected"><option value="ready">Ready</option><option value="blocked" selected disabled>Blocked</option></select>
<select name="last"><option value="a" selected>A</option><option value="b" selected>B</option></select>
<textarea name="note" value="ignored">initial</textarea></form>`)
	doc, err := ParseDocument(raw, "https://example.org", 200, len(raw))
	if err != nil {
		t.Fatal(err)
	}
	page := doc.Page(1000)
	fields := map[string]Field{}
	for _, field := range page.Forms[0].Fields {
		fields[field.Name] = field
	}
	for name, want := range map[string][]string{
		"empty": {""}, "multiple": {"", "blue"}, "none": {}, "list": {}, "default": {""}, "disabledselected": {}, "last": {"b"},
	} {
		if !reflect.DeepEqual(fields[name].Values, want) {
			t.Errorf("select %q values=%q; want %q", name, fields[name].Values, want)
		}
	}
	if fields["empty"].Value != "" || fields["disabledselected"].Value != "blocked" || !fields["multiple"].Multiple || !fields["multiple"].Options[2].Disabled {
		t.Fatalf("select state=%+v", fields)
	}
	doc.fields[fields["multiple"].Ref].setValue("blue")
	doc.fields[fields["empty"].Ref].setValue("missing")
	doc.fields[fields["note"].Ref].setValue("new\nvalue")
	page = doc.Page(1000)
	for _, field := range page.Forms[0].Fields {
		switch field.Name {
		case "multiple":
			if !reflect.DeepEqual(field.Values, []string{"blue"}) || field.Options[0].Selected || !field.Options[1].Selected || field.Options[2].Selected {
				t.Fatalf("select mutation=%+v", field)
			}
		case "empty":
			if len(field.Values) != 0 || field.Value != "" || field.Options[0].Selected {
				t.Fatalf("unmatched select value=%+v", field)
			}
		case "note":
			if field.Value != "new\nvalue" {
				t.Fatalf("textarea mutation=%+v", field)
			}
		}
	}
}

func TestDOMStandaloneInputsAndQueryStates(t *testing.T) {
	raw := []byte(`<span id="caption">Message</span><textarea id="chat" aria-labelledby="caption">hello</textarea>
<label>Accept <input id="accept" type="checkbox"></label><input type="radio" name="group" value="a" checked><input type="radio" name="group" value="b" checked>
<form><input type="radio" name="group" value="form" checked></form><input id="secret" type="PASSWORD" value="private">
<fieldset disabled><legend><fieldset disabled><legend><input id="nested"></legend><input id="blocked"></fieldset></legend></fieldset>`)
	doc, err := ParseDocument(raw, "https://example.org", 200, len(raw))
	if err != nil {
		t.Fatal(err)
	}
	page := doc.Page(1000)
	if len(page.Inputs) != 7 || page.Inputs[0].Label != "Message" || page.Inputs[1].Label != "Accept" || page.Inputs[2].Checked || !page.Inputs[3].Checked || !page.Forms[0].Fields[0].Checked {
		t.Fatalf("standalone fields: inputs=%+v forms=%+v", page.Inputs, page.Forms)
	}
	for selector, expected := range map[string]struct {
		disabled bool
		value    string
	}{
		"#nested": {false, ""}, "#blocked": {true, ""}, "#secret": {false, "[redacted]"},
	} {
		matches, err := doc.Query(selector)
		if err != nil || len(matches) != 1 || matches[0].Disabled != expected.disabled || matches[0].Value != expected.value {
			t.Fatalf("query %q=%+v, %v", selector, matches, err)
		}
	}
	matches, err := doc.Query("#accept")
	if err != nil || len(matches) != 1 || matches[0].Checked == nil || *matches[0].Checked || matches[0].Value != "on" {
		t.Fatalf("checkbox query=%+v, %v", matches, err)
	}
	field := page.Inputs[0]
	doc.fields[field.Ref].setValue("new message")
	if got := doc.Page(1000).Inputs[0].Value; got != "new message" {
		t.Fatalf("standalone textarea value=%q", got)
	}
}

func TestDOMFormSubmissionSuccessfulControls(t *testing.T) {
	received := make(chan url.Values, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/submit" {
			received <- r.URL.Query()
			fmt.Fprint(w, "<h1>Submitted</h1>")
			return
		}
		fmt.Fprint(w, `<input name="external" value="outside" form="main"><input name="standalone" value="alone">
<form id="main" action="/submit"><fieldset disabled><legend><input name="legend" value="allowed"></legend><input name="blocked" value="no"></fieldset>
<input name="unchecked" type="checkbox" value="yes"><input name="empty" type="checkbox" value="" checked><input name="default" type="checkbox" checked>
<input name="choice" type="radio" value="no"><input name="choice" type="radio" value="yes" checked>
<select name="multiple" multiple><option value="a" selected>A</option><option value="b" selected>B</option><option value="skip" disabled selected>Skip</option></select>
<select name="emptyselect"><option value="" selected>Choose</option></select><input name="readonly" readonly value="fixed"><input type="hidden" name="hidden" value="opaque">
<textarea name="note" value="ignored">line1
line2</textarea><input name="duplicate" value="x"><input name="duplicate" value="y"><input disabled="false" name="booleanDisabled" value="no">
<button name="commit" value="go">Go</button><button name="other" value="no">Other</button></form>`)
	}))
	defer server.Close()
	browser := New([]string{"127.0.0.1"}, false)
	defer browser.Close()
	if opened := browser.Execute(Command{Cmd: "open", URL: server.URL}); !opened.OK {
		t.Fatal(opened.Error)
	}
	if submitted := browser.Execute(Command{Cmd: "submit", Ref: "f1"}); !submitted.OK {
		t.Fatal(submitted.Error)
	}
	want := url.Values{
		"external": {"outside"}, "legend": {"allowed"}, "empty": {""}, "default": {"on"}, "choice": {"yes"},
		"multiple": {"a", "b"}, "emptyselect": {""}, "readonly": {"fixed"}, "hidden": {"opaque"}, "note": {"line1\r\nline2"},
		"duplicate": {"x", "y"}, "commit": {"go"},
	}
	select {
	case got := <-received:
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("submitted values=%#v; want %#v", got, want)
		}
	default:
		t.Fatal("submission did not reach server")
	}
}

func TestDOMFormEncodingAndValidationMetadata(t *testing.T) {
	raw := []byte(`<form id="upload" method="post" enctype="MULTIPART/FORM-DATA" novalidate><input id="number" name="amount" type="number" required min="1" max="10" step="0.5"><input name="code" pattern="[A-Z]{3}"></form>
<form method="dialog"></form><form enctype="text/plain"></form><form enctype="invalid"></form><form></form>`)
	doc, err := ParseDocument(raw, "https://example.org", 200, len(raw))
	if err != nil {
		t.Fatal(err)
	}
	for engine, page := range map[string]Page{
		"DOM":    doc.Page(1000),
		"Parser": Parse(raw, "https://example.org", 200, len(raw), 1000),
	} {
		t.Run(engine, func(t *testing.T) {
			if len(page.Forms) != 5 || page.Forms[0].Method != "POST" || page.Forms[0].Enctype != "multipart/form-data" || !page.Forms[0].NoValidate || page.Forms[1].Method != "DIALOG" || page.Forms[2].Enctype != "text/plain" || page.Forms[3].Enctype != "application/x-www-form-urlencoded" || page.Forms[4].Enctype != "application/x-www-form-urlencoded" {
				t.Fatalf("form metadata=%+v", page.Forms)
			}
			field := page.Forms[0].Fields[0]
			if !field.Required || field.Min != "1" || field.Max != "10" || field.Step != "0.5" || page.Forms[0].Fields[1].Pattern != "[A-Z]{3}" {
				t.Fatalf("constraints=%+v", page.Forms[0].Fields)
			}
		})
	}
	matches, err := doc.Query("#number")
	if err != nil || len(matches) != 1 || !matches[0].Required || matches[0].Min != "1" || matches[0].Max != "10" || matches[0].Step != "0.5" {
		t.Fatalf("query constraints=%+v, %v", matches, err)
	}
}
