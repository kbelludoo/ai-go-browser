package browser

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNavigateReadyUsesCurrentLoaderAndRecoversFromTimeout(t *testing.T) {
	path, err := FindSystemBrowser()
	if err != nil {
		t.Skip("navigation integration requires installed Chromium/Edge")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/blocked-script" {
			<-r.Context().Done()
			return
		}
		if r.URL.Path == "/slow" {
			fmt.Fprint(w, `<title>Waiting</title><script src="/blocked-script"></script>`)
			return
		}
		fmt.Fprintf(w, `<title>Before load</title><p>Document %s</p><script>window.addEventListener('load',()=>document.title='Loaded '+location.pathname)</script>`, r.URL.Path)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s, err := LaunchStealth(ctx, StealthConfig{BrowserPath: path, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		s.Close()
		if s.proc != nil {
			_ = s.proc.Wait()
		}
	}()
	for _, target := range []string{srv.URL + "/first", srv.URL + "/second", srv.URL + "/second#section"} {
		op, end := context.WithTimeout(context.Background(), 3*time.Second)
		err := s.NavigateReady(op, target)
		if err == nil {
			raw, evaluateErr := s.Evaluate(op, "JSON.stringify({url:location.href,title:document.title})")
			err = evaluateErr
			if err == nil && (!strings.Contains(string(raw), "Loaded") || !strings.Contains(string(raw), target)) {
				t.Fatalf("returned before current document load: %s", raw)
			}
		}
		end()
		if err != nil {
			t.Fatalf("navigate %s: %v", target, err)
		}
	}
	op, end := context.WithTimeout(context.Background(), 150*time.Millisecond)
	err = s.NavigateReady(op, srv.URL+"/slow")
	end()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("slow navigation: %v", err)
	}
	op, end = context.WithTimeout(context.Background(), 3*time.Second)
	err = s.NavigateReady(op, srv.URL+"/recovered")
	end()
	if err != nil {
		t.Fatalf("recovery after timed out loader: %v", err)
	}
}

func TestNavigateReadyRequiresSession(t *testing.T) {
	var session *StealthSession
	if err := session.NavigateReady(context.Background(), "http://127.0.0.1/"); err == nil {
		t.Fatal("nil session accepted")
	}
}
