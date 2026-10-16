package browser

import "testing"

func TestDOMARIAControlsAndEditableHosts(t *testing.T) {
	raw := []byte(`<div id="compose" contenteditable="true" role="textbox" aria-label="Message"><b>old &amp; text</b></div>
<div id="plain" contenteditable="plaintext-only" aria-label="Notes">line1
line2</div><div id="readonly" role="textbox" contenteditable aria-readonly="true">locked</div>
<div id="send" role="button" aria-label="Send"><span>arrow</span></div><a id="expand" aria-label="Expand">Open</a><a id="real" href="/next">Next</a>
<div aria-disabled="true"><div id="disabledButton" role="button">Unavailable</div><div id="disabledEditor" role="textbox" contenteditable>Unavailable</div></div>
<div hidden><div id="hiddenButton" role="button">Secret</div><div id="hiddenEditor" contenteditable>Secret</div></div>
<span id="caption">Editor label</span><div id="rolebox" role="textbox" aria-labelledby="caption">role value</div>
<div id="container" contenteditable>outer <span id="inherited">span</span><div id="stop" contenteditable="false">fixed<div id="reopened" contenteditable="true">nested</div></div></div>
<form><input name="q" required pattern="[a-z]+"><button id="nativeSubmit">Search</button><div id="insideForm" contenteditable aria-label="Extra notes">extra</div></form>`)
	doc, err := ParseDocument(raw, "https://example.org", 200, len(raw))
	if err != nil {
		t.Fatal(err)
	}
	for engine, page := range map[string]Page{
		"DOM":    doc.Page(1000),
		"Parser": Parse(raw, "https://example.org", 200, len(raw), 1000),
	} {
		t.Run(engine, func(t *testing.T) {
			inputs := map[string]Field{}
			for _, input := range page.Inputs {
				inputs[input.ID] = input
			}
			if inputs["compose"].Type != "contenteditable" || inputs["compose"].Label != "Message" || inputs["compose"].Value != "old & text" || inputs["plain"].Value != "line1\nline2" || inputs["rolebox"].Label != "Editor label" || !inputs["readonly"].ReadOnly || !inputs["disabledEditor"].Disabled {
				t.Fatalf("editable metadata=%+v", page.Inputs)
			}
			for _, id := range []string{"hiddenEditor", "inherited", "stop"} {
				if _, exists := inputs[id]; exists {
					t.Fatalf("non-host editor %q exposed: %+v", id, page.Inputs)
				}
			}
			for _, id := range []string{"container", "reopened", "insideForm"} {
				if inputs[id].Ref == "" {
					t.Fatalf("editor %q missing: %+v", id, page.Inputs)
				}
			}
			buttons := map[string]Button{}
			for _, button := range page.Buttons {
				buttons[button.ID] = button
			}
			if buttons["send"].Type != "button" || buttons["send"].Label != "Send" || buttons["expand"].Type != "button" || !buttons["disabledButton"].Disabled || buttons["nativeSubmit"].Type != "submit" {
				t.Fatalf("button metadata=%+v", page.Buttons)
			}
			if _, exists := buttons["hiddenButton"]; exists {
				t.Fatal("hidden ARIA button exposed")
			}
			if len(page.Links) != 1 || page.Links[0].URL != "https://example.org/next" || len(page.Forms) != 1 || len(page.Forms[0].Fields) != 1 || !page.Forms[0].Fields[0].Required || page.Forms[0].Fields[0].Pattern != "[a-z]+" || page.Forms[0].SubmitRef != buttons["nativeSubmit"].Ref {
				t.Fatalf("native controls changed: links=%+v forms=%+v", page.Links, page.Forms)
			}
		})
	}
	for selector, typ := range map[string]string{"#compose": "contenteditable", "#send": "button", "#expand": "button"} {
		matches, err := doc.Query(selector)
		if err != nil || len(matches) != 1 || matches[0].Ref == "" || matches[0].Type != typ || matches[0].Label == "" {
			t.Fatalf("query %q=%+v, %v", selector, matches, err)
		}
		if selector == "#expand" && matches[0].Href != "" {
			t.Fatalf("anchor without href became link: %+v", matches)
		}
	}
}

func TestDOMContenteditableSetValueIsLiteralText(t *testing.T) {
	raw := []byte(`<div id="editor" contenteditable="true" aria-label="Message"><b>initial</b></div>`)
	doc, err := ParseDocument(raw, "https://example.org", 200, len(raw))
	if err != nil {
		t.Fatal(err)
	}
	page := doc.Page(1000)
	if len(page.Inputs) != 1 {
		t.Fatalf("inputs=%+v", page.Inputs)
	}
	node := doc.fields[page.Inputs[0].Ref]
	value := `<img src="x" onerror="alert(1)"> &amp; literal`
	node.setValue(value)
	if len(node.Children) != 1 || node.Children[0].Tag != "#text" || node.Children[0].Text != value || node.Children[0].Parent != node {
		t.Fatalf("editor children=%+v", node.Children)
	}
	if got := doc.Page(1000).Inputs[0].Value; got != value {
		t.Fatalf("editable value=%q; want %q", got, value)
	}
	matches, err := doc.Query("#editor")
	if err != nil || len(matches) != 1 || matches[0].Value != value {
		t.Fatalf("editable query=%+v, %v", matches, err)
	}
	images, err := doc.Query("img")
	if err != nil || len(images) != 0 {
		t.Fatalf("literal text created HTML elements: %+v, %v", images, err)
	}
}
