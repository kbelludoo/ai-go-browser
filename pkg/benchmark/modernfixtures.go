package benchmark

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

const ModernFixtureVersion = "local-modern-v1"

const (
	modernShadowValue   = "Shadow café 😀"
	modernEditableValue = "Texto café 😀"
	modernMultipartNote = "Resumo café 😀"
	modernFrameValue    = "Frame café 😀"
	modernBlocked       = "VALIDATION_BLOCKED:invalid=1;trusted_clicks=1;submits=0"
)

type modernStep struct {
	Kind     string `json:"kind"`
	Selector string `json:"selector,omitempty"`
	Value    string `json:"value,omitempty"`
	Expected string `json:"expected,omitempty"`
}

type modernFixture struct {
	Name             string       `json:"name"`
	Path             string       `json:"path"`
	Expected         string       `json:"expected"`
	FrameName        string       `json:"frame_name,omitempty"`
	ExpectedRequests int          `json:"expected_requests"`
	Steps            []modernStep `json:"steps"`
}

// ModernEvidence is fixture-side evidence, separate from engine snapshots.
// Counts come from the local HTTP handlers, not a browser-provided estimate.
type ModernEvidence struct {
	Case             string `json:"case"`
	ExpectedRequests int    `json:"expected_requests"`
	ObservedRequests int    `json:"observed_requests"`
	ValidRequests    int    `json:"valid_requests"`
	Passed           bool   `json:"passed"`
}

type modernCorpus struct {
	server, cross *httptest.Server
	cases         []modernFixture
	mu            sync.Mutex
	requests      map[string]int
	valid         map[string]int
}

func modernCases() []modernFixture {
	return []modernFixture{
		{Name: "open_shadow_slot", Path: "/shadow", Expected: "MODERN_SHADOW_OK:shadow=1;slot=1;untrusted=0;projected=true;value=" + modernShadowValue, Steps: []modernStep{{Kind: "fill", Selector: "#shadow-input", Value: modernShadowValue}, {Kind: "press", Selector: "#shadow-check"}, {Kind: "press", Selector: "#slotted-action"}}},
		{Name: "trusted_contenteditable", Path: "/editable", Expected: "MODERN_EDITABLE_OK:trusted_input=true;trusted_clicks=1;untrusted_clicks=0;value=" + modernEditableValue, Steps: []modernStep{{Kind: "fill", Selector: "#editor", Value: modernEditableValue}, {Kind: "press", Selector: "#editor-send"}}},
		{Name: "native_validation_multipart", Path: "/multipart", Expected: "MODERN_MULTIPART_OK:invalid=1;trusted_clicks=2;submits=1;files=0;value=" + modernMultipartNote, ExpectedRequests: 1, Steps: []modernStep{{Kind: "press", Selector: "#multipart-send"}, {Kind: "assert", Expected: modernBlocked}, {Kind: "fill", Selector: "#email", Value: "ana@example.test"}, {Kind: "fill", Selector: "#note", Value: modernMultipartNote}, {Kind: "press", Selector: "#multipart-send"}}},
		{Name: "frame_same_origin", Path: "/frame-same", FrameName: "modern-same", Expected: "MODERN_FRAME_SAME_OK:trusted_clicks=1;requests=1;value=" + modernFrameValue, ExpectedRequests: 1, Steps: []modernStep{{Kind: "fill", Selector: "#frame-note", Value: modernFrameValue}, {Kind: "press", Selector: "#frame-send"}}},
		{Name: "frame_cross_origin", Path: "/frame-cross", FrameName: "modern-cross", Expected: "MODERN_FRAME_CROSS_OK:trusted_clicks=1;requests=1;value=" + modernFrameValue, ExpectedRequests: 1, Steps: []modernStep{{Kind: "fill", Selector: "#frame-note", Value: modernFrameValue}, {Kind: "press", Selector: "#frame-send"}}},
	}
}

func modernFixtures() *modernCorpus {
	c := &modernCorpus{cases: modernCases(), requests: make(map[string]int), valid: make(map[string]int)}
	mux := http.NewServeMux()
	write := func(w http.ResponseWriter, title, body string) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><title>%s</title></head><body>%s</body></html>`, title, body)
	}
	mux.HandleFunc("/shadow", func(w http.ResponseWriter, r *http.Request) {
		write(w, "Open shadow and slot", `<main><h1>Shadow controls</h1><div id="host"><button id="slotted-action" slot="finish">Finish through slot</button></div><output id="result">Waiting for shadow actions</output></main><script>
const host=document.getElementById('host'),root=host.attachShadow({mode:'open'});
root.innerHTML='<label for="shadow-input">Name</label><input id="shadow-input"><button id="shadow-check">Confirm shadow</button><slot id="finish-slot" name="finish"></slot>';
let shadow=0,slot=0,untrusted=0,value='';
root.getElementById('shadow-check').addEventListener('click',event=>{if(event.isTrusted)shadow++;else untrusted++;value=root.getElementById('shadow-input').value});
document.getElementById('slotted-action').addEventListener('click',event=>{if(event.isTrusted)slot++;else untrusted++;const projected=event.composedPath().includes(root.getElementById('finish-slot'));document.getElementById('result').textContent='MODERN_SHADOW_OK:shadow='+shadow+';slot='+slot+';untrusted='+untrusted+';projected='+projected+';value='+value});
</script>`)
	})
	mux.HandleFunc("/editable", func(w http.ResponseWriter, r *http.Request) {
		write(w, "Trusted contenteditable", `<main><h1>Compose</h1><div id="editor" role="textbox" aria-label="Draft" contenteditable="true" style="min-height:40px;border:1px solid">`+`</div><button id="editor-send" type="button">Send draft</button><output id="result">Waiting for edit</output></main><script>
const editor=document.getElementById('editor');let trustedInput=0,trustedClicks=0,untrustedClicks=0;
editor.addEventListener('input',event=>{if(event.isTrusted)trustedInput++});
document.getElementById('editor-send').addEventListener('click',event=>{if(event.isTrusted)trustedClicks++;else untrustedClicks++;document.getElementById('result').textContent='MODERN_EDITABLE_OK:trusted_input='+(trustedInput>0)+';trusted_clicks='+trustedClicks+';untrusted_clicks='+untrustedClicks+';value='+editor.innerText});
</script>`)
	})
	mux.HandleFunc("/multipart", func(w http.ResponseWriter, r *http.Request) {
		write(w, "Native validation and multipart", `<form id="upload" method="post" action="/multipart-result" enctype="multipart/form-data"><label for="email">Email</label><input id="email" name="email" type="email" required><label for="note">Note</label><textarea id="note" name="note"></textarea><input type="hidden" name="invalidCount" value="0"><input type="hidden" name="trustedClicks" value="0"><input type="hidden" name="untrustedClicks" value="0"><input type="hidden" name="submitEvents" value="0"><button id="multipart-send" name="operation" value="publish" type="submit">Publish</button></form><output id="result">Waiting for validation</output><script>
const form=document.getElementById('upload');let invalid=0,trustedClicks=0,untrustedClicks=0,submits=0;
const field=(name,value)=>{form.elements.namedItem(name).value=value};
document.getElementById('multipart-send').addEventListener('click',event=>{if(event.isTrusted)trustedClicks++;else untrustedClicks++;field('trustedClicks',trustedClicks);field('untrustedClicks',untrustedClicks)});
form.addEventListener('invalid',()=>{invalid++;field('invalidCount',invalid);document.getElementById('result').textContent='VALIDATION_BLOCKED:invalid='+invalid+';trusted_clicks='+trustedClicks+';submits='+submits},true);
form.addEventListener('submit',event=>{if(event.isTrusted)submits++;field('submitEvents',submits)});
</script>`)
	})
	mux.HandleFunc("/multipart-result", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(1 << 20)
		files := 0
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
			for _, entries := range r.MultipartForm.File {
				files += len(entries)
			}
		}
		valid := err == nil && r.Method == http.MethodPost && strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") && files == 0 && r.FormValue("email") == "ana@example.test" && r.FormValue("note") == modernMultipartNote && r.FormValue("operation") == "publish" && r.FormValue("invalidCount") == "1" && r.FormValue("trustedClicks") == "2" && r.FormValue("untrustedClicks") == "0" && r.FormValue("submitEvents") == "1"
		c.record("native_validation_multipart", valid)
		if !valid {
			write(w, "Multipart rejected", `<h1>MODERN_MULTIPART_FAILED:body or native event evidence mismatch</h1>`)
			return
		}
		write(w, "Multipart accepted", "<h1>"+html.EscapeString(c.cases[2].Expected)+"</h1>")
	})
	mux.HandleFunc("/frame-same", func(w http.ResponseWriter, r *http.Request) {
		write(w, "Same origin frame host", `<h1>Main document</h1><iframe name="modern-same" title="Same origin work" src="/frame-child?kind=same" style="width:700px;height:260px"></iframe>`)
	})
	mux.HandleFunc("/frame-cross", func(w http.ResponseWriter, r *http.Request) {
		child := strings.Replace(c.cross.URL, "127.0.0.1", "localhost", 1) + "/frame-child?kind=cross"
		write(w, "Cross origin frame host", `<h1>Main document</h1><iframe name="modern-cross" title="Cross origin work" src="`+html.EscapeString(child)+`" style="width:700px;height:260px"></iframe>`)
	})
	mux.HandleFunc("/frame-child", func(w http.ResponseWriter, r *http.Request) {
		kind := r.URL.Query().Get("kind")
		if kind != "same" && kind != "cross" {
			http.Error(w, "unknown frame", http.StatusBadRequest)
			return
		}
		kindJSON, _ := json.Marshal(kind)
		write(w, "Child frame "+kind, `<h1>Frame controls</h1><label for="frame-note">Frame note</label><input id="frame-note"><button id="frame-send" type="button">Send frame note</button><output id="result">Waiting inside child frame</output><script>
let trustedClicks=0,untrustedClicks=0;document.getElementById('frame-send').addEventListener('click',async event=>{
if(event.isTrusted)trustedClicks++;else untrustedClicks++;
const response=await fetch('/frame-receipt?kind='+`+string(kindJSON)+`,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({value:document.getElementById('frame-note').value,trustedClicks,untrustedClicks})});
const result=await response.json();document.getElementById('result').textContent=result.text;
});</script>`)
	})
	mux.HandleFunc("/frame-receipt", func(w http.ResponseWriter, r *http.Request) {
		kind := r.URL.Query().Get("kind")
		name, index := "frame_same_origin", 3
		if kind == "cross" {
			name, index = "frame_cross_origin", 4
		}
		var data struct {
			Value           string `json:"value"`
			TrustedClicks   int    `json:"trustedClicks"`
			UntrustedClicks int    `json:"untrustedClicks"`
		}
		err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&data)
		valid := err == nil && r.Method == http.MethodPost && (kind == "same" || kind == "cross") && data.Value == modernFrameValue && data.TrustedClicks == 1 && data.UntrustedClicks == 0
		count := c.record(name, valid)
		text := "MODERN_FRAME_FAILED:request or native click evidence mismatch"
		if valid && count == 1 {
			text = c.cases[index].Expected
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"text": text, "requests": count})
	})
	c.cross = httptest.NewServer(mux)
	c.server = httptest.NewServer(mux)
	return c
}

func (c *modernCorpus) close() {
	c.server.Close()
	c.cross.Close()
}

func (c *modernCorpus) record(name string, valid bool) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests[name]++
	if valid {
		c.valid[name]++
	}
	return c.requests[name]
}

func (c *modernCorpus) evidence(f modernFixture) ModernEvidence {
	c.mu.Lock()
	defer c.mu.Unlock()
	observed, valid := c.requests[f.Name], c.valid[f.Name]
	return ModernEvidence{Case: f.Name, ExpectedRequests: f.ExpectedRequests, ObservedRequests: observed, ValidRequests: valid, Passed: observed == f.ExpectedRequests && valid == f.ExpectedRequests}
}
