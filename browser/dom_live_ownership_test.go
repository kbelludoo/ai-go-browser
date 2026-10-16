package browser

import (
	"reflect"
	"testing"
)

func TestDOMLiveFormOwnershipMarkers(t *testing.T) {
	raw := []byte(`<form id="duplicate" data-aigobrowser="first-form">
<input name="ancestor" value="a">
<input name="detached" form="duplicate" data-aigobrowser-form="" value="d">
<input name="second" form="duplicate" data-aigobrowser-form="second-form" value="s">
<input name="unknown" data-aigobrowser-form="missing-form" value="u">
<button id="detached-button" data-aigobrowser-form="">Detached</button>
<button id="second-button" data-aigobrowser-form="second-form">Second</button>
</form>
<form id="duplicate" data-aigobrowser="second-form">
<input name="first" data-aigobrowser-form="first-form" value="f">
<button id="first-button" data-aigobrowser-form="first-form">First</button>
</form>
<input name="http-duplicate" form="duplicate" value="h">
<input name="outside-second" data-aigobrowser-form="second-form" value="o">`)
	doc, err := ParseDocument(raw, "https://example.org", 200, len(raw))
	if err != nil {
		t.Fatal(err)
	}
	for engine, page := range map[string]Page{
		"DOM":    doc.Page(1000),
		"Parser": Parse(raw, "https://example.org", 200, len(raw), 1000),
	} {
		t.Run(engine, func(t *testing.T) {
			if len(page.Forms) != 2 {
				t.Fatalf("forms=%+v", page.Forms)
			}
			fieldNames := func(fields []Field) []string {
				names := make([]string, 0, len(fields))
				for _, field := range fields {
					names = append(names, field.Name)
				}
				return names
			}
			if got := fieldNames(page.Forms[0].Fields); !reflect.DeepEqual(got, []string{"ancestor", "first", "http-duplicate"}) {
				t.Fatalf("first form fields=%v", got)
			}
			if got := fieldNames(page.Forms[1].Fields); !reflect.DeepEqual(got, []string{"second", "outside-second"}) {
				t.Fatalf("second form fields=%v", got)
			}
			if got := fieldNames(page.Inputs); !reflect.DeepEqual(got, []string{"detached", "unknown"}) {
				t.Fatalf("standalone inputs=%v", got)
			}
			buttons := map[string]Button{}
			for _, button := range page.Buttons {
				buttons[button.ID] = button
			}
			if buttons["detached-button"].FormRef != "" || buttons["first-button"].FormRef != page.Forms[0].Ref || buttons["second-button"].FormRef != page.Forms[1].Ref {
				t.Fatalf("button owners=%+v", page.Buttons)
			}
			if page.Forms[0].SubmitRef != buttons["first-button"].Ref || page.Forms[1].SubmitRef != buttons["second-button"].Ref {
				t.Fatalf("default submitters=%+v", page.Forms)
			}
		})
	}
}

func TestDOMRoleTextboxDoesNotGrantEditing(t *testing.T) {
	raw := []byte(`<div id="role-only" role="textbox" aria-label="Read text">original</div>
<div contenteditable="true"><div id="inherited-role" role="textbox">editable</div></div>
<div id="false-role" role="textbox" contenteditable="false">fixed</div>
<input id="native" role="textbox" value="native">`)
	doc, err := ParseDocument(raw, "https://example.org", 200, len(raw))
	if err != nil {
		t.Fatal(err)
	}
	for engine, page := range map[string]Page{
		"DOM":    doc.Page(1000),
		"Parser": Parse(raw, "https://example.org", 200, len(raw), 1000),
	} {
		t.Run(engine, func(t *testing.T) {
			fields := map[string]Field{}
			for _, field := range page.Inputs {
				fields[field.ID] = field
			}
			if fields["role-only"].Ref == "" || fields["role-only"].Type != "contenteditable" || fields["role-only"].Value != "original" || !fields["role-only"].ReadOnly || !fields["false-role"].ReadOnly {
				t.Fatalf("non-editable textbox metadata=%+v", page.Inputs)
			}
			if fields["inherited-role"].ReadOnly || fields["native"].ReadOnly || fields["native"].Type != "text" {
				t.Fatalf("editable textbox metadata=%+v", page.Inputs)
			}
		})
	}
	matches, err := doc.Query("#role-only")
	if err != nil || len(matches) != 1 || !matches[0].ReadOnly || matches[0].Value != "original" {
		t.Fatalf("non-editable query=%+v, %v", matches, err)
	}
}

func TestDOMInheritedARIAReadonly(t *testing.T) {
	raw := []byte(`<div aria-readonly="true"><div id="editor" contenteditable>locked</div>
<input id="native" value="locked"><div aria-readonly="false"><textarea id="nested">locked</textarea></div></div>
<div readonly><input id="writable" value="editable"></div>`)
	doc, err := ParseDocument(raw, "https://example.org", 200, len(raw))
	if err != nil {
		t.Fatal(err)
	}
	for engine, page := range map[string]Page{
		"DOM":    doc.Page(1000),
		"Parser": Parse(raw, "https://example.org", 200, len(raw), 1000),
	} {
		t.Run(engine, func(t *testing.T) {
			fields := map[string]Field{}
			for _, field := range page.Inputs {
				fields[field.ID] = field
			}
			for _, id := range []string{"editor", "native", "nested"} {
				if !fields[id].ReadOnly {
					t.Fatalf("field %q did not inherit ARIA readonly: %+v", id, page.Inputs)
				}
			}
			if fields["writable"].ReadOnly {
				t.Fatalf("HTML readonly was inherited: %+v", page.Inputs)
			}
		})
	}
	matches, err := doc.Query("#editor")
	if err != nil || len(matches) != 1 || !matches[0].ReadOnly {
		t.Fatalf("inherited ARIA readonly query=%+v, %v", matches, err)
	}
}

func TestDOMSnapshotLabelOverridesFlattenedIDs(t *testing.T) {
	raw := []byte(`<label for="shared">Light label</label><label for="shared">Shadow label</label>
<input id="shared" name="light" data-aigobrowser-label="  Light   label  ">
<input id="shared" name="shadow" data-aigobrowser-label="Shadow label">
<span id="caption">Light caption</span><span id="caption">Shadow caption</span>
<input name="caption" aria-labelledby="caption" data-aigobrowser-label="Shadow caption">
<input name="override" aria-label="Wrong label" data-aigobrowser-label="Native label">
<label for="empty">Wrong associated label</label>
<input id="empty" name="empty" aria-label="Wrong ARIA label" data-aigobrowser-label="" placeholder="Enter value">
<input id="shared" name="unnamed" data-aigobrowser-label="">
<label for="http">HTTP label</label><input id="http" name="http">
<input name="http-aria" aria-label="HTTP ARIA label">
<div role="button" data-aigobrowser-label="Native button">Arrow</div>`)
	doc, err := ParseDocument(raw, "https://example.org", 200, len(raw))
	if err != nil {
		t.Fatal(err)
	}
	page := doc.Page(1000)
	fields := map[string]Field{}
	for _, field := range page.Inputs {
		fields[field.Name] = field
	}
	labels := map[string]string{
		"light": "Light label", "shadow": "Shadow label", "caption": "Shadow caption",
		"override": "Native label", "empty": "Enter value", "unnamed": "",
		"http": "HTTP label", "http-aria": "HTTP ARIA label",
	}
	for name, want := range labels {
		if got := fields[name].Label; got != want {
			t.Errorf("field %q label=%q; want %q", name, got, want)
		}
		matches, err := doc.Query(`[name="` + name + `"]`)
		if err != nil || len(matches) != 1 {
			t.Fatalf("query %q=%+v, %v", name, matches, err)
		}
		// HTTP query retains its existing aria-label-only contract.
		if name == "http" {
			want = ""
		}
		if matches[0].Label != want {
			t.Errorf("query %q label=%q; want %q", name, matches[0].Label, want)
		}
	}
	if len(page.Buttons) != 1 || page.Buttons[0].Label != "Native button" {
		t.Fatalf("button label=%+v", page.Buttons)
	}
}
