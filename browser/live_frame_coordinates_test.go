package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestFrameCoordinatePathRejectsUnsafeAncestry(t *testing.T) {
	l := &liveSession{frameID: "main", frames: newLiveFrameController()}
	l.frames.selected = &liveFrameScope{id: "child"}
	l.frames.states["child"] = &liveFrameState{parent: "outer"}
	l.frames.states["outer"] = &liveFrameState{parent: "main"}
	path, err := l.frameCoordinatePathLocked()
	want := []frameCoordinateHop{{child: "child", parent: "outer"}, {child: "outer", parent: "main"}}
	if err != nil || !reflect.DeepEqual(path, want) {
		t.Fatalf("nested path=%v err=%v", path, err)
	}
	for _, condition := range []string{"missing", "detached", "cycle", "invalid-context"} {
		t.Run(condition, func(t *testing.T) {
			l.frames.selected.invalid = condition == "invalid-context"
			l.frames.states["outer"] = &liveFrameState{parent: "main"}
			switch condition {
			case "missing":
				delete(l.frames.states, "outer")
			case "detached":
				l.frames.states["outer"].detached = true
			case "cycle":
				l.frames.states["outer"].parent = "child"
			}
			if _, err := l.frameCoordinatePathLocked(); err == nil {
				t.Fatal("unsafe frame ancestry accepted")
			}
		})
	}
	for _, point := range [][2]float64{{math.NaN(), 1}, {1, math.Inf(1)}} {
		if _, _, err := l.framePointToMain(context.Background(), point[0], point[1]); err == nil {
			t.Fatal("non-finite input coordinates accepted")
		}
	}
}

func TestLiveFrameCoordinatesScrollPaddingAndScale(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/child" {
			fmt.Fprint(w, `<!doctype html><style>body{margin:0;height:1800px}button{position:absolute;left:400px;top:1100px;width:130px;height:45px}</style><button id="action" onclick="this.textContent='trusted '+event.isTrusted+' count '+(window.count=(window.count||0)+1)">Child initial</button>`)
			return
		}
		fmt.Fprint(w, `<!doctype html><style>body{margin:0;width:2300px;height:2100px}iframe{position:absolute;left:1400px;top:1300px;width:620px;height:370px;border:7px solid navy;padding:11px;transform:scale(.75,.8);transform-origin:0 0}button{position:fixed;top:0;left:0;z-index:2}</style><button id="rotate" onclick="document.querySelector('iframe').style.transform='rotate(12deg)'">Rotate owner</button><iframe name="scaled" src="/child"></iframe>`)
	}))
	t.Cleanup(server.Close)
	b := newLiveTestBrowser(t)
	frameAPICommand(t, b, Command{Cmd: "open", URL: server.URL, RenderJS: true})
	child := frameAPINamed(t, frameAPIList(t, b), "scaled")
	frameAPISelect(t, b, child.Ref)
	frameAPICommand(t, b, Command{Cmd: "press", Ref: frameAPIElement(t, b, "#action").Ref})
	if got := frameAPIElement(t, b, "#action").Text; got != "trusted true count 1" {
		t.Fatalf("scaled/padded frame native click missed: %q", got)
	}
	frameAPISelect(t, b, "fr1")
	ctx, cancel := b.liveContext()
	raw, err := b.live.evaluate(ctx, `({x:scrollX,y:scrollY})`)
	cancel()
	var scroll struct{ X, Y float64 }
	if err == nil {
		err = json.Unmarshal(raw, &scroll)
	}
	if err != nil || scroll.X <= 0 || scroll.Y <= 0 {
		t.Fatalf("main document did not scroll in both directions: %s, %v", raw, err)
	}
	frameAPICommand(t, b, Command{Cmd: "press", Ref: frameAPIElement(t, b, "#rotate").Ref})
	frameAPISelect(t, b, child.Ref)
	blocked := b.Execute(Command{Cmd: "press", Ref: frameAPIElement(t, b, "#action").Ref})
	if blocked.OK || !strings.Contains(blocked.Error, "transform") {
		t.Fatalf("unsupported rotation did not fail clearly: %+v", blocked)
	}
	if got := frameAPIElement(t, b, "#action").Text; got != "trusted true count 1" {
		t.Fatalf("rotated frame received an unintended click: %q", got)
	}
}

func TestLiveFrameCoordinatesCrossOriginShadowOwnerOverlay(t *testing.T) {
	var otherOrigin string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/child":
			fmt.Fprint(w, `<!doctype html><style>body{margin:0}button{margin:30px 25px;width:160px;height:45px}</style><button id="action" onclick="this.textContent='trusted '+event.isTrusted+' count '+(window.count=(window.count||0)+1)">Child initial</button>`)
		case "/outer":
			fmt.Fprintf(w, `<!doctype html><style>body{margin:0}#host{display:block;margin:35px}</style><button id="cover" onclick="document.querySelector('#host').shadowRoot.querySelector('#veil').hidden=false">Cover child</button><button id="uncover" onclick="document.querySelector('#host').shadowRoot.querySelector('#veil').hidden=true">Uncover child</button><div id="host"></div><script>document.querySelector('#host').attachShadow({mode:'open'}).innerHTML=%s</script>`, jsLiteral(`<style>iframe{width:330px;height:170px;border:9px solid green}#veil{position:absolute;inset:0;background:#ddd;z-index:2}</style><div style="position:relative;width:348px;height:188px"><iframe name="remote" src="`+otherOrigin+`/child"></iframe><div id="veil" hidden>Parent overlay</div></div>`))
		default:
			fmt.Fprint(w, `<!doctype html><style>body{margin:0}iframe{margin:100px 130px;width:500px;height:350px;border:6px solid navy}</style><iframe name="outer" src="/outer"></iframe>`)
		}
	}))
	t.Cleanup(server.Close)
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	u.Host = "localhost:" + u.Port()
	otherOrigin = u.String()
	b := newLiveTestBrowser(t)
	b.AllowAll = true
	frameAPICommand(t, b, Command{Cmd: "open", URL: server.URL, RenderJS: true})
	frames := frameAPIList(t, b)
	outer, child := frameAPINamed(t, frames, "outer"), frameAPINamed(t, frames, "remote")
	if child.ParentRef != outer.Ref {
		t.Fatalf("cross-origin frame ancestry=%+v", frames)
	}
	frameAPISelect(t, b, child.Ref)
	frameAPICommand(t, b, Command{Cmd: "press", Ref: frameAPIElement(t, b, "#action").Ref})
	if got := frameAPIElement(t, b, "#action").Text; got != "trusted true count 1" {
		t.Fatalf("cross-origin shadow owner click missed: %q", got)
	}
	frameAPISelect(t, b, outer.Ref)
	frameAPICommand(t, b, Command{Cmd: "press", Ref: frameAPIElement(t, b, "#cover").Ref})
	frameAPISelect(t, b, child.Ref)
	blocked := b.Execute(Command{Cmd: "press", Ref: frameAPIElement(t, b, "#action").Ref})
	if blocked.OK || !strings.Contains(blocked.Error, "covered") {
		t.Fatalf("shadow parent overlay was not rejected: %+v", blocked)
	}
	if got := frameAPIElement(t, b, "#action").Text; got != "trusted true count 1" {
		t.Fatalf("parent overlay allowed an unintended child click: %q", got)
	}
	frameAPISelect(t, b, outer.Ref)
	frameAPICommand(t, b, Command{Cmd: "press", Ref: frameAPIElement(t, b, "#uncover").Ref})
	frameAPISelect(t, b, child.Ref)
	frameAPICommand(t, b, Command{Cmd: "press", Ref: frameAPIElement(t, b, "#action").Ref})
	if got := frameAPIElement(t, b, "#action").Text; got != "trusted true count 2" {
		t.Fatalf("uncovered child click failed: %q", got)
	}
}
