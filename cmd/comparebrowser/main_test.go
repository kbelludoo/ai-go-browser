package main

import (
	"strings"
	"testing"
)

func TestCompareRejectsUnrequestedPublicTargets(t *testing.T) {
	for _, args := range [][]string{{"-public"}, {"-url", "https://example.com"}} {
		if err := run(args); err == nil || !strings.Contains(err.Error(), "both -public") {
			t.Fatalf("expected explicit public opt-in error, got %v", err)
		}
	}
}

func TestCompareRejectsInvalidRunCount(t *testing.T) {
	if err := run([]string{"-runs", "0"}); err == nil || !strings.Contains(err.Error(), "-runs") {
		t.Fatalf("invalid count accepted: %v", err)
	}
}
