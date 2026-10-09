//go:build linux && cgo

// Isolated JavaScript worker using the locally installed Duktape C engine.
// Not part of the pure-Go browser binary. Run only as a subprocess with a timeout.
package main

/*
#cgo LDFLAGS: -l:libduktape.so.207
#include <stddef.h>
#include <stdlib.h>
typedef struct duk_hthread duk_context;
extern duk_context *duk_create_heap(void *, void *, void *, void *, void *);
extern void duk_destroy_heap(duk_context *);
extern int duk_get_global_string(duk_context *, const char *);
extern const char *duk_push_lstring(duk_context *, const char *, size_t);
extern int duk_pcall(duk_context *, int);
extern const char *duk_safe_to_lstring(duk_context *, int, size_t *);
extern void duk_pop(duk_context *);

static int eval_str(duk_context *ctx, const char *src, size_t n, char **result) {
    duk_get_global_string(ctx, "eval");
    duk_push_lstring(ctx, src, n);
    int err = duk_pcall(ctx, 1);
    size_t l = 0;
    const char *s = duk_safe_to_lstring(ctx, -1, &l);
    *result = (char *)malloc(l + 1);
    if (!*result) { duk_pop(ctx); return -2; }
    for (size_t i=0; i<l; i++) (*result)[i] = s[i];
    (*result)[l] = 0;
    duk_pop(ctx);
    return err;
}
*/
import "C"
import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

type Node struct {
	TextContent string `json:"textContent"`
	Value       string `json:"value,omitempty"`
	OnClick     string `json:"onclick,omitempty"`
}
type Job struct {
	Nodes   map[string]Node `json:"nodes"`
	Title   string          `json:"title"`
	URL     string          `json:"url"`
	Scripts []string        `json:"scripts"`
}
type Result struct {
	Nodes            map[string]Node     `json:"nodes"`
	Title            string              `json:"title"`
	Writes           []string            `json:"writes"`
	Appends          map[string][]string `json:"appends"`
	Errors           []string            `json:"errors,omitempty"`
	Executed         int                 `json:"executed"`
	WorkerPeakRSSMiB float64             `json:"worker_peak_rss_mib"`
}

func main() {
	// CPU-time guard. The parent additionally imposes a wall-clock deadline.
	// RAM is MEASURED, not hard-capped: use Linux cgroups for strict memory caps.
	_ = syscall.Setrlimit(syscall.RLIMIT_CPU, &syscall.Rlimit{Cur: 2, Max: 3})
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4096), 256*1024)
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, "missing JS job")
		os.Exit(2)
	}
	data := append([]byte(nil), scanner.Bytes()...)
	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if len(job.Scripts) > 16 || len(job.Nodes) > 256 {
		fmt.Fprintln(os.Stderr, "script/node limit exceeded")
		os.Exit(2)
	}
	ctx := C.duk_create_heap(nil, nil, nil, nil, nil)
	if ctx == nil {
		fmt.Fprintln(os.Stderr, "JS VM allocation failed")
		os.Exit(2)
	}
	defer C.duk_destroy_heap(ctx)
	evaluate := func(src string) (string, error) {
		c := C.CString(src)
		defer C.free(unsafe.Pointer(c))
		var out *C.char
		rc := C.eval_str(ctx, c, C.size_t(len(src)), &out)
		if out == nil {
			return "", fmt.Errorf("JS allocation error")
		}
		res := C.GoString(out)
		C.free(unsafe.Pointer(out))
		if rc != 0 {
			return "", fmt.Errorf("%s", res)
		}
		return res, nil
	}
	initial, _ := json.Marshal(map[string]interface{}{"nodes": job.Nodes, "title": job.Title, "writes": []string{}, "appends": map[string][]string{}})
	bootstrap := `var __ai = ` + string(initial) + `;
// Minimal DOM facade, not a full implementation of browser DOM APIs.
function __escape(s) { return String(s).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g,"&quot;"); }
function __serialize(el) {
 var tag = el.tagName;
 if (!/^(a|p|span|div|button|input|strong|em|ul|li|h1|h2|h3)$/.test(tag)) return "";
 var attr = "";
 var whitelist = ['href','name','type','value','title','id'];
 for (var k=0; k<whitelist.length; k++) {
   var key=whitelist[k];
   if (el.attrs[key] !== undefined) attr+=' '+key+'="'+__escape(el.attrs[key])+'"';
 }
 if (tag==='input') return '<input'+attr+'>';
 return '<'+tag+attr+'>'+__escape(el.textContent || '')+'</'+tag+'>';
}
var document = {
 getElementById: function(id) { return __ai.nodes[String(id)] || null; },
 createElement: function(tag) {
  var e={tagName:String(tag).toLowerCase(),textContent:'',attrs:{}};
  e.setAttribute=function(k,v){this.attrs[String(k).toLowerCase()]=String(v);};
  Object.defineProperty(e,'href',{get:function(){return this.attrs.href||'';},set:function(v){this.attrs.href=String(v);}});
  return e;
 },
 querySelector: function(sel) { if (typeof sel === 'string' && sel[0] === '#') return this.getElementById(sel.slice(1)); return null; },
 write: function(html) { __ai.writes.push(String(html)); },
};
Object.defineProperty(document, 'title', {
 get: function() { return __ai.title; }, set: function(v) { __ai.title = String(v); }
});
document.body={ appendChild:function(el){__ai.writes.push(__serialize(el));} };
Object.keys(__ai.nodes).forEach(function(id) {
 var n = __ai.nodes[id];
 n.appendChild = function(el) {if(!__ai.appends[id]) __ai.appends[id]=[]; __ai.appends[id].push(__serialize(el));};
 n.__listeners={};
 n.addEventListener=function(type,handler) {
   type=String(type).toLowerCase();
   if (typeof handler!== 'function') return;
   if (!this.__listeners[type]) this.__listeners[type]=[];
   if (this.__listeners[type].length < 16) this.__listeners[type].push(handler);
 };
 n.dispatchEvent=function(event) {
   var typ=String(event.type).toLowerCase();
   event.target=this; event.currentTarget=this;
   if (typeof this['on'+typ]==='function') this['on'+typ].call(this,event);
   var fns=this.__listeners[typ]||[];
   for (var j=0;j<fns.length;j++) fns[j].call(this,event);
   return !event.defaultPrevented;
 };
 if (n.onclick) { try { n.onclick=new Function('event',n.onclick); } catch(e) {n.onclick=null;} }

 Object.defineProperty(n, 'innerText', {
  get: function() { return n.textContent; },
  set: function(v) { n.textContent = String(v); }
 });
});
var window = {document:document};
var location = {href:` + string(mustJSON(job.URL)) + `};
window.location=location;
`
	if _, err := evaluate(bootstrap); err != nil {
		fmt.Fprintln(os.Stderr, "bootstrap:", err)
		os.Exit(2)
	}
	var errors []string
	count := 0
	for _, script := range job.Scripts {
		if len(script) > 64*1024 {
			errors = append(errors, "script too large")
			continue
		}
		_, e := evaluate(script)
		if e != nil {
			errors = append(errors, e.Error())
		} else {
			count++
		}
	}
	respond := func(errors []string, count int) {
		answer, e := evaluate(`JSON.stringify(__ai)`)
		if e != nil {
			fmt.Fprintln(os.Stderr, "extract:", e)
			return
		}
		var result Result
		if e := json.Unmarshal([]byte(answer), &result); e != nil {
			fmt.Fprintln(os.Stderr, "extract JSON:", e)
			return
		}
		result.Errors = errors
		result.Executed = count
		result.WorkerPeakRSSMiB = peakRSS()
		_ = json.NewEncoder(os.Stdout).Encode(result)
	}
	respond(errors, count)
	_, _ = evaluate(`__ai.writes=[];__ai.appends={};`)
	// The VM lives for one page and preserves listeners and script variables.
	// Each event gets a bounded JSON command; Go imposes a wall-clock limit.
	for scanner.Scan() {
		var action struct {
			Cmd   string `json:"cmd"`
			ID    string `json:"id"`
			Event string `json:"event"`
			Value string `json:"value"`
		}
		if e := json.Unmarshal(scanner.Bytes(), &action); e != nil {
			respond([]string{"invalid event request"}, 0)
			continue
		}
		if action.Cmd == "setvalue" && len(action.ID) <= 128 {
			id, _ := json.Marshal(action.ID)
			val, _ := json.Marshal(action.Value)
			_, e := evaluate(`(function(){var n=document.getElementById(` + string(id) + `);if(n)n.value=` + string(val) + `;})()`)
			if e != nil {
				respond([]string{e.Error()}, 0)
			} else {
				respond(nil, 0)
			}
			_, _ = evaluate(`__ai.writes=[];__ai.appends={};`)
			continue
		}
		if action.Cmd != "event" || action.Event != "click" || len(action.ID) > 128 {
			respond([]string{"unsupported event"}, 0)
			continue
		}
		id, _ := json.Marshal(action.ID)
		src := `(function() { var node=document.getElementById(` + string(id) + `);
          if (!node) { throw new Error('event target not found'); }
          node.dispatchEvent({type:'click',defaultPrevented:false,
            preventDefault:function(){this.defaultPrevented=true;}});
        })()`
		_, e := evaluate(src)
		errs := []string{}
		if e != nil {
			errs = append(errs, e.Error())
		}
		respond(errs, 1)
		_, _ = evaluate(`__ai.writes=[];__ai.appends={};`)
	}
}
func mustJSON(s string) []byte { b, _ := json.Marshal(s); return b }

// Linux high-water RSS of the worker (MiB).
func peakRSS() float64 {
	status, e := os.ReadFile("/proc/self/status")
	if e != nil {
		return 0
	}
	for _, l := range strings.Split(string(status), "\n") {
		if strings.HasPrefix(l, "VmHWM:") {
			p := strings.Fields(l)
			if len(p) >= 2 {
				n, _ := strconv.ParseFloat(p[1], 64)
				return n / 1024
			}
		}
	}
	return 0
}
