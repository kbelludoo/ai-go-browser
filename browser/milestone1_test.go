package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// TestMilestone1NativeGoBrowser proves the Marco 1 requirement from docs/PLANO_INDEPENDENCIA.md:
// "Rodar sem navegador externo instalado, visitar página local, manter cookie, interpretar HTML,
// identificar botão, executar JS necessário para disparar fetch, receber JSON e atualizar DOM."
func TestMilestone1NativeGoBrowser(t *testing.T) {
	// 1. Setup local test server with Cookie authentication and API endpoint
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			// Set session cookie
			http.SetCookie(w, &http.Cookie{
				Name:  "auth_token",
				Value: "secure_session_token_12345",
				Path:  "/",
			})
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><title>Milestone 1 Test Page</title></head>
<body>
    <h1 id="title">Demo Nativa</h1>
    <p id="estado">Aguardando acao</p>
    <button id="btn-fetch" type="button">Buscar Dados</button>
</body>
</html>`)

		case "/api/status":
			// Verify cookie
			cookie, err := r.Cookie("auth_token")
			if err != nil || cookie.Value != "secure_session_token_12345" {
				http.Error(w, "Unauthorized: missing cookie", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":  "ok",
				"message": "Dados recebidos em puro Go sem nenhum Chromium!",
				"memory":  "12MB",
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	// 2. Initialize Browser in strict NativeOnly mode
	b := New([]string{"*"}, true)
	b.NativeOnly = true
	// We allow localhost here because this is an internal httptest.Server
	b.SetBlockPrivateNetworks(false)

	// 3. Navigate to page (pure Go HTTP request, stores cookie in jar)
	res := b.Execute(Command{Cmd: "open", URL: ts.URL})
	if !res.OK {
		t.Fatalf("failed to open page: %s", res.Error)
	}

	// 4. Verify cookie was captured
	u, _ := url.Parse(ts.URL)
	cookies := b.client.Jar.Cookies(u)
	hasCookie := false
	for _, c := range cookies {
		if c.Name == "auth_token" && c.Value == "secure_session_token_12345" {
			hasCookie = true
			break
		}
	}
	if !hasCookie {
		t.Fatalf("session cookie was not retained in browser jar")
	}

	// 5. Verify initial DOM
	estadoNode := b.Document.Root.findNodeByID("estado")
	if estadoNode == nil || estadoNode.innerText(100) != "Aguardando acao" {
		t.Fatalf("unexpected initial estado text: %v", estadoNode)
	}

	// 6. Initialize in-process Native JS Runtime attached to the DOM
	rt, err := NewNativeJSRuntime(b, b.Document, ts.URL)
	if err != nil {
		t.Fatalf("failed to create native JS runtime: %v", err)
	}

	// 7. Execute JavaScript setting up the event listener with fetch
	script := `
	document.getElementById('btn-fetch').addEventListener('click', function() {
		fetch('/api/status')
			.then(function(res) {
				return res.json();
			})
			.then(function(data) {
				document.getElementById('estado').textContent = data.message;
			});
	});
	`
	if err := rt.ExecuteScript(script, 3*time.Second); err != nil {
		t.Fatalf("failed to execute script: %v", err)
	}

	// 8. Trigger click on 'btn-fetch' WITHOUT any external browser
	start := time.Now()
	if err := rt.TriggerEvent("btn-fetch", "click"); err != nil {
		t.Fatalf("click event execution failed: %v", err)
	}
	duration := time.Since(start)

	// 9. Verify DOM was updated with the message fetched from /api/status!
	updatedText := estadoNode.innerText(200)
	expectedMsg := "Dados recebidos em puro Go sem nenhum Chromium!"
	if !strings.Contains(updatedText, expectedMsg) {
		t.Fatalf("DOM text was not updated! Got: %q, Expected: %q", updatedText, expectedMsg)
	}

	// 10. Record memory RSS
	mem := ReadProcessMemory(os.Getpid())

	t.Logf("✅ MARCO 1 CONCLUÍDO COM SUCESSO!")
	t.Logf("   - Cookie retido e autenticado no /api/status")
	t.Logf("   - JS fetch executado e DOM atualizado em: %v", duration)
	t.Logf("   - Texto final no DOM: %q", updatedText)
	t.Logf("   - RSS real do processo: %.2f MiB", mem.RSSMiB)
}

func (n *Node) findNodeByID(id string) *Node {
	var found *Node
	controlWalk(n, func(node *Node) {
		if node.prop("id") == id {
			found = node
		}
	})
	return found
}
