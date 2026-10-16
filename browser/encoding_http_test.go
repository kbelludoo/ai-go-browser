package browser

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
)

func TestHTTPNavigationDecodesLegacyCharsetAndCountsWireBytes(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		encode     func(string) (string, error)
	}{
		{"shift_jis", "日本語のページ", japanese.ShiftJIS.NewEncoder().String},
		{"gbk", "中文页面", simplifiedchinese.GBK.NewEncoder().String},
	} {
		t.Run(tc.name, func(t *testing.T) {
			html, err := tc.encode("<!doctype html><title>" + tc.text + "</title><p>" + tc.text + "</p>")
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset="+tc.name)
				fmt.Fprint(w, html)
			}))
			defer server.Close()
			b := New([]string{"127.0.0.1"}, false)
			defer b.Close()
			result := b.Execute(Command{Cmd: "open", URL: server.URL})
			if !result.OK || result.Page.Title != tc.text || !strings.Contains(result.Page.Text, tc.text) || result.Page.Bytes != len(html) || result.Page.Stack.Encoding != tc.name {
				t.Fatalf("decoded navigation: %+v", result)
			}
		})
	}
}

func TestResourceDecodingRedactionAndExpansionBudget(t *testing.T) {
	jsonBody := []byte(`{"message":"ol`)
	jsonBody = append(jsonBody, 0xe1)
	jsonBody = append(jsonBody, []byte(`","token":"secret-value"}`)...)
	resource := resourceSnapshot(jsonBody, "http://example.test", 200, "application/json; charset=windows-1252", false)
	data, ok := resource.Data.(map[string]any)
	if !ok || data["message"] != "olá" || data["token"] != "[redacted]" || resource.Body != "" {
		t.Fatalf("decoded JSON: %+v", resource)
	}
	resource = resourceSnapshot(bytes.Repeat([]byte{0xe1}, maxResourceBytes), "http://example.test", 200, "text/plain; charset=windows-1252", false)
	if !resource.Truncated || !strings.Contains(resource.Body, "observation budget") || len(resource.Body) > maxResourceBytes {
		t.Fatalf("expanded text exceeded budget: %+v", resource)
	}
	resource = resourceSnapshot([]byte{0xff, 0xfe, 'o', 0, 'l', 0, 0xe1, 0}, "http://example.test", 200, "text/plain", false)
	if resource.Body != "olá" {
		t.Fatalf("UTF16 text: %+v", resource)
	}
	for _, body := range [][]byte{
		append([]byte{0xef, 0xbb, 0xbf}, []byte(`{"token":"secret-value"}`)...),
		[]byte{0xff, 0xfe, '{', 0, '"', 0, 't', 0, 'o', 0, 'k', 0, 'e', 0, 'n', 0, '"', 0, ':', 0, '"', 0, 'x', 0, '"', 0, '}', 0},
		[]byte(`{"token":"secret-value",`),
	} {
		resource := resourceSnapshot(body, "http://example.test", 200, "", false)
		if strings.Contains(resource.Body, "secret-value") || strings.Contains(resource.Body, `"x"`) {
			t.Fatalf("headerless JSON exposed a secret: %+v", resource)
		}
		if resource.Data != nil {
			data, ok := resource.Data.(map[string]any)
			if !ok || data["token"] != "[redacted]" {
				t.Fatalf("headerless JSON was not redacted: %+v", resource)
			}
		}
	}
}

func TestHTTPNavigationSniffsBOMEncodedHTML(t *testing.T) {
	for _, html := range []string{
		"\xef\xbb\xbf<!doctype html><title>BOM page</title><form><input name=q></form>",
		func() string {
			s, err := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder().String("<!doctype html><title>BOM page</title><form><input name=q></form>")
			if err != nil {
				t.Fatal(err)
			}
			return s
		}(),
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header()["Content-Type"] = nil // suppress net/http's automatic sniff
			fmt.Fprint(w, html)
		}))
		b := New([]string{"127.0.0.1"}, false)
		result := b.Execute(Command{Cmd: "open", URL: server.URL})
		b.Close()
		server.Close()
		if !result.OK || result.Page == nil || result.Resource != nil || result.Page.Title != "BOM page" || len(result.Page.Forms) != 1 || result.Page.Bytes != len(html) {
			t.Fatalf("headerless BOM HTML: %+v", result)
		}
	}
}
