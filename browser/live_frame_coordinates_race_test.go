package browser

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLiveFrameCoordinatesRejectTargetMovedOnParentScroll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/child" {
			fmt.Fprint(w, `<!doctype html><style>body{margin:0}button{position:absolute;left:20px;top:30px;width:140px;height:45px}#state{margin-top:100px}</style><button id="target" onclick="targetClicks++;update()">Target</button><button id="decoy" hidden onclick="decoyClicks++;update()">Decoy</button><p id="state">target=0 decoy=0 moved=false</p><script>let targetClicks=0,decoyClicks=0,moved=false;function update(){document.querySelector('#state').textContent='target='+targetClicks+' decoy='+decoyClicks+' moved='+moved}addEventListener('message',event=>{if(event.data==='move-target'){document.querySelector('#target').style.left='250px';document.querySelector('#decoy').hidden=false;moved=true;update()}})</script>`)
			return
		}
		fmt.Fprint(w, `<!doctype html><style>body{margin:0;height:2400px}iframe{position:absolute;left:110px;top:1600px;width:500px;height:300px;border:8px solid navy}</style><iframe name="moving" src="/child"></iframe><script>addEventListener('scroll',()=>requestAnimationFrame(()=>document.querySelector('iframe').contentWindow.postMessage('move-target','*')),{once:true})</script>`)
	}))
	t.Cleanup(server.Close)
	b := newLiveTestBrowser(t)
	frameAPICommand(t, b, Command{Cmd: "open", URL: server.URL, RenderJS: true})
	frameAPISelect(t, b, frameAPINamed(t, frameAPIList(t, b), "moving").Ref)
	result := b.Execute(Command{Cmd: "press", Ref: frameAPIElement(t, b, "#target").Ref})
	if result.OK || (!strings.Contains(result.Error, "changed") && !strings.Contains(result.Error, "covered") && !strings.Contains(result.Error, "moved")) {
		t.Fatalf("target moved during preparation without blocking native input: %+v", result)
	}
	state := frameAPICommand(t, b, Command{Cmd: "query", Selector: "#state"}).Matches
	if len(state) != 1 || state[0].Text != "target=0 decoy=0 moved=true" {
		t.Fatalf("stale point clicked another child control: %+v", state)
	}
}

func TestLiveFrameCoordinatesVerificationRejectsMovedOwnerAndOverlay(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/child" {
			fmt.Fprint(w, `<!doctype html><button>Child control</button>`)
			return
		}
		fmt.Fprint(w, `<!doctype html><style>body{margin:0}iframe{position:absolute;left:100px;top:100px;width:300px;height:200px;border:6px solid navy}#veil{position:absolute;left:0;top:0;width:100vw;height:100vh;z-index:2;background:#ddd}</style><iframe name="child" src="/child"></iframe><div id="veil" hidden>Overlay</div>`)
	}))
	t.Cleanup(server.Close)
	b := newLiveTestBrowser(t)
	frameAPICommand(t, b, Command{Cmd: "open", URL: server.URL, RenderJS: true})
	frameAPISelect(t, b, frameAPINamed(t, frameAPIList(t, b), "child").Ref)
	ctx, cancel := b.liveContext()
	defer cancel()
	x, y, err := b.live.framePointToMain(ctx, 90, 60)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.live.verifyFramePointToMain(ctx, 90, 60, x, y); err != nil {
		t.Fatalf("stable owner failed verification: %v", err)
	}
	b.live.mu.Lock()
	mainID := b.live.frameID
	b.live.mu.Unlock()
	endpoint, contextID, unique, err := b.live.frameWorld(ctx, mainID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := endpoint.EvaluateWorld(ctx, `document.querySelector('iframe').style.left='240px'`, contextID, unique); err != nil {
		t.Fatal(err)
	}
	if err := b.live.verifyFramePointToMain(ctx, 90, 60, x, y); err == nil || !strings.Contains(err.Error(), "moved") {
		t.Fatalf("moved owner accepted the stale main point: %v", err)
	}
	if _, err := endpoint.EvaluateWorld(ctx, `document.querySelector('iframe').style.left='100px';document.querySelector('#veil').hidden=false`, contextID, unique); err != nil {
		t.Fatal(err)
	}
	if err := b.live.verifyFramePointToMain(ctx, 90, 60, x, y); err == nil || !strings.Contains(err.Error(), "covered") {
		t.Fatalf("new parent overlay accepted the stale main point: %v", err)
	}
	if _, err := endpoint.EvaluateWorld(ctx, `document.querySelector('#veil').hidden=true;document.querySelector('iframe').style.offsetPath='path("M 0 0 L 100 100")';document.querySelector('iframe').style.offsetDistance='50%'`, contextID, unique); err != nil {
		t.Fatal(err)
	}
	if _, _, err := b.live.framePointToMain(ctx, 90, 60); err == nil || !strings.Contains(err.Error(), "motion-path") {
		t.Fatalf("motion-path frame did not fail clearly: %v", err)
	}
}
