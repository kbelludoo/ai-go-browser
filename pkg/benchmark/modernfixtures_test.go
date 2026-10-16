package benchmark

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestModernCorpusUsesIndependentFrameOrigins(t *testing.T) {
	c := modernFixtures()
	defer c.close()
	if len(c.cases) != 5 {
		t.Fatalf("modern corpus has %d tasks", len(c.cases))
	}
	response, err := http.Get(c.server.URL + "/frame-cross")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	crossURL := strings.Replace(c.cross.URL, "127.0.0.1", "localhost", 1)
	if !strings.Contains(string(body), crossURL+"/frame-child?kind=cross") {
		t.Fatalf("cross-origin task did not reference its second server: %s", body)
	}
	main, _ := url.Parse(c.server.URL)
	child, _ := url.Parse(crossURL)
	if main.Hostname() == child.Hostname() || main.Host == child.Host {
		t.Fatalf("frame was not cross-site and cross-origin: %s / %s", main, child)
	}
}

func modernMultipartRequest(t *testing.T, c *modernCorpus, overrides map[string]string) string {
	t.Helper()
	var data bytes.Buffer
	writer := multipart.NewWriter(&data)
	values := map[string]string{"email": "ana@example.test", "note": modernMultipartNote, "operation": "publish", "invalidCount": "1", "trustedClicks": "2", "untrustedClicks": "0", "submitEvents": "1"}
	for key, value := range overrides {
		values[key] = value
	}
	for key, value := range values {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	response, err := http.Post(c.server.URL+"/multipart-result", writer.FormDataContentType(), &data)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestModernMultipartOracleRejectsSyntheticAndDuplicateEvidence(t *testing.T) {
	t.Run("untrusted click", func(t *testing.T) {
		c := modernFixtures()
		defer c.close()
		body := modernMultipartRequest(t, c, map[string]string{"untrustedClicks": "1"})
		if strings.Contains(body, c.cases[2].Expected) || c.evidence(c.cases[2]).Passed {
			t.Fatal("untrusted click evidence passed native multipart task")
		}
	})
	t.Run("duplicate request", func(t *testing.T) {
		c := modernFixtures()
		defer c.close()
		body := modernMultipartRequest(t, c, nil)
		if !strings.Contains(body, c.cases[2].Expected) || !c.evidence(c.cases[2]).Passed {
			t.Fatal("valid fixture evidence rejected")
		}
		modernMultipartRequest(t, c, nil)
		if evidence := c.evidence(c.cases[2]); evidence.Passed || evidence.ObservedRequests != 2 {
			t.Fatalf("duplicate network submission hidden: %+v", evidence)
		}
	})
}

func TestModernFrameOracleRequiresValueAndNativeClick(t *testing.T) {
	for _, tt := range []struct {
		name      string
		trusted   int
		untrusted int
		value     string
		valid     bool
	}{
		{name: "valid", trusted: 1, value: modernFrameValue, valid: true},
		{name: "synthetic", untrusted: 1, value: modernFrameValue},
		{name: "wrong value", trusted: 1, value: "incorrect"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := modernFixtures()
			defer c.close()
			body, _ := json.Marshal(map[string]any{"value": tt.value, "trustedClicks": tt.trusted, "untrustedClicks": tt.untrusted})
			response, err := http.Post(c.cross.URL+"/frame-receipt?kind=cross", "application/json", bytes.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if evidence := c.evidence(c.cases[4]); evidence.Passed != tt.valid {
				t.Fatalf("wrong frame oracle: %+v", evidence)
			}
		})
	}
}
