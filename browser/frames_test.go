package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFramesMetadataPolicyAndParents(t *testing.T) {
	b := New([]string{"example.org"}, false)
	defer b.Close()
	frames, err := b.framesFromMetadata(json.RawMessage(`{
		"frameTree":{"frame":{"id":"main","url":"https://example.org/"},"childFrames":[
			{"frame":{"id":"child","url":"https://assets.example.org/embed","name":"content"},"childFrames":[
				{"frame":{"id":"nested","url":"https://other.org/"}}
			]},
			{"frame":{"id":"blank","url":"about:blank"}}
		]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 4 {
		t.Fatalf("frames = %+v", frames)
	}
	if !frames[0].Main || frames[0].Ref != "fr1" || !frames[0].Allowed {
		t.Fatalf("main frame = %+v", frames[0])
	}
	if frames[1].ParentRef != "fr1" || frames[1].Name != "content" || !frames[1].Allowed {
		t.Fatalf("child frame = %+v", frames[1])
	}
	if frames[2].ParentRef != "fr2" || frames[2].Main || frames[2].Allowed {
		t.Fatalf("nested frame = %+v", frames[2])
	}
	if frames[3].ParentRef != "fr1" || !frames[3].Allowed {
		t.Fatalf("about:blank policy = %+v", frames[3])
	}
}

func TestFramesMetadataBoundsAndValidation(t *testing.T) {
	b := New([]string{"example.org"}, false)
	defer b.Close()
	for _, count := range []int{maxLiveFrames, maxLiveFrames + 1} {
		root := liveFrameTree{}
		root.Frame.ID = "root"
		root.Frame.URL = "https://example.org/"
		for i := 1; i < count; i++ {
			child := liveFrameTree{}
			child.Frame.ID = fmt.Sprint(i)
			child.Frame.URL = "https://example.org/frame"
			root.Children = append(root.Children, child)
		}
		raw, err := json.Marshal(map[string]any{"frameTree": root})
		if err != nil {
			t.Fatal(err)
		}
		frames, err := b.framesFromMetadata(raw)
		if count == maxLiveFrames {
			if err != nil || len(frames) != count {
				t.Fatalf("exact limit: frames=%d err=%v", len(frames), err)
			}
		} else if err == nil || frames != nil || !strings.Contains(err.Error(), "incomplete") {
			t.Fatalf("over limit: frames=%d err=%v", len(frames), err)
		}
	}
	for _, raw := range []string{`{}`, `{"frameTree":{"frame":{"id":"same"},"childFrames":[{"frame":{"id":"same"}}]}}`, `{`} {
		if _, err := b.framesFromMetadata(json.RawMessage(raw)); err == nil {
			t.Fatalf("invalid metadata accepted: %s", raw)
		}
	}
	if _, err := b.framesFromMetadata(json.RawMessage(strings.Repeat(" ", maxFrameMetadataBytes+1))); err == nil {
		t.Fatal("oversized frame metadata accepted")
	}
	if _, err := b.liveFrames(); err == nil {
		t.Fatal("frames accepted without live session")
	}
}

func TestLiveFrameDiscoveryKeepsCurrentObservation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/frame" {
			fmt.Fprint(w, "<title>Child frame</title><p>Frame content</p>")
			return
		}
		fmt.Fprint(w, `<title>Main document</title><a href="/next">Next</a><iframe name="content" src="/frame"></iframe>`)
	}))
	defer srv.Close()
	b := newLiveTestBrowser(t)
	if err := b.openLive(srv.URL, true); err != nil {
		t.Fatal(err)
	}
	current, document, pid := b.Current, b.Document, b.RenderProcessID()
	frames, err := b.liveFrames()
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 || !frames[0].Main || frames[1].Name != "content" || frames[1].URL != srv.URL+"/frame" || !frames[1].Allowed {
		t.Fatalf("live frame tree = %+v", frames)
	}
	if b.Current != current || b.Document != document || b.RenderProcessID() != pid {
		t.Fatal("frame discovery changed the selected document or renderer")
	}
}
