package main

// Deterministic HTTP fixture server shared by the Go golden generator and the
// Rust replay. Every endpoint is stateless (or echoes request headers), so the
// same process can answer the Go reference and the Rust port with byte-equal
// responses. Redirects use relative Location headers so the golden stays valid
// across ephemeral ports.

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
)

const bigHTMLBytes = 5<<20 + 1

// recordedHeaders is the wire surface the Rust port must reproduce. It is the
// Go header set from browser.request plus whatever Go's transport adds
// (Accept-Encoding, Cookie, Content-Length).
var recordedHeaders = []string{
	"user-agent",
	"accept",
	"accept-language",
	"accept-encoding",
	"content-type",
	"content-length",
	"cookie",
	"sec-ch-ua",
	"sec-ch-ua-mobile",
	"sec-ch-ua-platform",
	"sec-fetch-dest",
	"sec-fetch-mode",
	"sec-fetch-site",
	"sec-fetch-user",
	"upgrade-insecure-requests",
}

type requestRecord struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Query   string            `json:"query"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body,omitempty"`
}

type recorder struct {
	mu         sync.Mutex
	byScenario map[string][]requestRecord
}

func newRecorder() *recorder {
	return &recorder{byScenario: map[string][]requestRecord{}}
}

func (r *recorder) add(scenario string, request *http.Request) {
	headers := map[string]string{}
	for _, name := range recordedHeaders {
		if value := request.Header.Get(name); value != "" {
			headers[name] = value
		}
	}
	body, _ := io.ReadAll(io.LimitReader(request.Body, 4096))
	// Hand the payload back to the handler: reading the body in the middleware
	// must not turn a POST fixture into an empty echo.
	request.Body = io.NopCloser(bytes.NewReader(body))
	record := requestRecord{
		Method:  request.Method,
		Path:    request.URL.Path,
		Query:   request.URL.RawQuery,
		Headers: headers,
		Body:    string(body),
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byScenario[scenario] = append(r.byScenario[scenario], record)
}

func (r *recorder) take(scenario string) []requestRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.byScenario[scenario]
	delete(r.byScenario, scenario)
	return out
}

func newServer() (*httptest.Server, *recorder) {
	rec := newRecorder()
	mux := http.NewServeMux()

	mux.HandleFunc("/plain", func(w http.ResponseWriter, _ *http.Request) {
		writeHTML(w, 200, "<title>Plain page</title><h1>Plain</h1><a href=\"/plain\">self</a>")
	})

	mux.HandleFunc("/echo", func(w http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(request.Body, 4096))
		writeHTML(w, 200, fmt.Sprintf("<title>echo %s</title><p>%s</p>", request.Method, body))
	})

	mux.HandleFunc("/gzip", func(w http.ResponseWriter, request *http.Request) {
		body := "<title>Compressed</title><p>compressed payload</p>"
		if !strings.Contains(request.Header.Get("Accept-Encoding"), "gzip") {
			// Go's custom RoundTripper never advertises gzip, so a real server
			// would answer in the clear. Serving plain bytes is what the fixture
			// must do to keep the wire comparison honest.
			writeHTML(w, 200, body)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Encoding", "gzip")
		writer := gzip.NewWriter(w)
		_, _ = writer.Write([]byte(body))
		_ = writer.Close()
	})

	mux.HandleFunc("/latin1", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=iso-8859-1")
		_, _ = w.Write([]byte("<title>Caf\xe9</title><p>cr\xe8me br\xfbl\xe9e</p>"))
	})

	mux.HandleFunc("/meta-charset", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<meta charset=\"windows-1252\"><title>Caf\xe9</title><p>d\xe9j\xe0 vu</p>"))
	})

	mux.HandleFunc("/r/", func(w http.ResponseWriter, request *http.Request) {
		depth, err := strconv.Atoi(strings.TrimPrefix(request.URL.Path, "/r/"))
		if err != nil {
			http.Error(w, "bad depth", http.StatusBadRequest)
			return
		}
		if depth <= 0 {
			writeHTML(w, 200, "<title>Redirect target</title><p>arrived</p>")
			return
		}
		http.Redirect(w, request, fmt.Sprintf("/r/%d?s=%s", depth-1, request.URL.Query().Get("s")), http.StatusFound)
	})

	mux.HandleFunc("/form", func(w http.ResponseWriter, _ *http.Request) {
		writeHTML(w, 200, `<title>Form</title>
<form method="post" action="/echo?s=post-form">
  <input name="q" value="caf&#233; au lait">
  <input name="tag" value="b">
  <input name="tag" value="a">
  <button type="submit" name="send" value="1">Send</button>
</form>`)
	})

	mux.HandleFunc("/big", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		chunk := strings.Repeat("<p>padding</p>", 256)
		written := 0
		for written < bigHTMLBytes {
			size := min(len(chunk), bigHTMLBytes-written)
			if _, err := w.Write([]byte(chunk[:size])); err != nil {
				return
			}
			written += size
		}
	})

	mux.HandleFunc("/missing", func(w http.ResponseWriter, _ *http.Request) {
		writeHTML(w, 404, "<title>Gone</title><p>not here</p>")
	})

	mux.HandleFunc("/no-type", func(w http.ResponseWriter, _ *http.Request) {
		// A nil entry suppresses net/http's automatic Content-Type sniffing, so
		// the response arrives without the header and Go must sniff it.
		w.Header()["Content-Type"] = nil
		_, _ = w.Write([]byte("<!doctype html><title>Sniffed</title><p>sniffed html</p>"))
	})

	mux.HandleFunc("/cookies/set", func(w http.ResponseWriter, request *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "sid", Value: "abc123", Path: "/"})
		http.Redirect(w, request, "/cookies/check?s="+request.URL.Query().Get("s"), http.StatusFound)
	})

	mux.HandleFunc("/cookies/check", func(w http.ResponseWriter, request *http.Request) {
		writeHTML(w, 200, fmt.Sprintf("<title>cookie</title><p>sid=%s</p>", request.Header.Get("Cookie")))
	})

	mux.HandleFunc("/__requests", func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rec.take(request.URL.Query().Get("scenario")))
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if scenario := request.URL.Query().Get("s"); scenario != "" {
			rec.add(scenario, request)
		}
		mux.ServeHTTP(w, request)
	})
	return httptest.NewServer(handler), rec
}

func writeHTML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}
