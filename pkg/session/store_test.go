package session

import (
	"net/http"
	"os"
	"testing"
)

func TestEncryptedSessionStore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "session_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewStore(tempDir, "test-secret-key-1234")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// 1. Test generic session save & load
	type UserSession struct {
		Email string `json:"email"`
		Token string `json:"token"`
	}
	original := UserSession{Email: "user@example.com", Token: "secret-token-xyz"}

	if err := store.SaveSession("auth", original); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	var loaded UserSession
	if err := store.LoadSession("auth", &loaded); err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}
	if loaded != original {
		t.Fatalf("loaded session %+v != original %+v", loaded, original)
	}

	// 2. Test cookie persistence
	cookies := []*http.Cookie{
		{Name: "session_id", Value: "123456", Domain: "arena.ai", Path: "/"},
		{Name: "cf_clearance", Value: "abcdef", Domain: ".arena.ai", Path: "/"},
	}
	if err := store.SaveCookies("arena.ai", cookies); err != nil {
		t.Fatalf("SaveCookies failed: %v", err)
	}

	loadedCookies, err := store.LoadCookies("arena.ai")
	if err != nil {
		t.Fatalf("LoadCookies failed: %v", err)
	}
	if len(loadedCookies) != 2 || loadedCookies[0].Name != "session_id" {
		t.Fatalf("cookie mismatch: %+v", loadedCookies)
	}
}
