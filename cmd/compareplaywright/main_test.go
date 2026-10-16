package main

import (
	"strings"
	"testing"
)

func TestPlaywrightComparisonIsLocalOnly(t *testing.T) {
	if err := run([]string{"-public"}); err == nil {
		t.Fatal("public flag must not be accepted")
	}
	if err := run([]string{"https://example.com"}); err == nil || !strings.Contains(err.Error(), "unexpected arguments") {
		t.Fatalf("unexpected public argument accepted: %v", err)
	}
}

func TestPlaywrightComparisonValidatesRepetitions(t *testing.T) {
	if err := run([]string{"-runs", "0"}); err == nil || !strings.Contains(err.Error(), "-runs") {
		t.Fatalf("invalid repetition count accepted: %v", err)
	}
}
