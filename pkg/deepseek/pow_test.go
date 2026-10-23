package deepseek

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
)

// TestPowSolverInstantiatesAndSolves exercises the embedded DeepSeekHashV1 wasm
// module offline. The vector is taken from a known solution: with salt
// "testsalt", expire_at 1700000000 and difficulty 1000, the nonce 42 hashes to
// the challenge below, so the solver must recover exactly that answer.
func TestPowSolverInstantiatesAndSolves(t *testing.T) {
	ctx := context.Background()
	solver, err := NewPowSolver(ctx)
	if err != nil {
		t.Fatalf("NewPowSolver: %v", err)
	}
	defer solver.Close()

	ch := PowChallenge{
		Algorithm:  "DeepSeekHashV1",
		Challenge:  "d4a2ea58c89e40887c933484868380c6f803eaa8dc53a3b9df8e431b921a4f09",
		Salt:       "testsalt",
		ExpireAt:   1700000000,
		Difficulty: 1000,
		Signature:  "sig",
		TargetPath: "/api/v0/chat/completion",
	}

	header, err := solver.MakeHeader(ctx, ch)
	if err != nil {
		t.Fatalf("MakeHeader: %v", err)
	}

	raw, err := base64.StdEncoding.DecodeString(header)
	if err != nil {
		t.Fatalf("header is not base64: %v", err)
	}
	var payload struct {
		Algorithm  string `json:"algorithm"`
		Challenge  string `json:"challenge"`
		Salt       string `json:"salt"`
		Answer     int64  `json:"answer"`
		Signature  string `json:"signature"`
		TargetPath string `json:"target_path"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("header payload is not JSON: %v", err)
	}
	if payload.Answer != 42 {
		t.Fatalf("answer = %d, want 42", payload.Answer)
	}
	if payload.Algorithm != ch.Algorithm || payload.Challenge != ch.Challenge ||
		payload.Salt != ch.Salt || payload.Signature != ch.Signature ||
		payload.TargetPath != ch.TargetPath {
		t.Fatalf("payload does not echo the challenge: %+v", payload)
	}
}
