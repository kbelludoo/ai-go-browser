package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModernComparisonRejectsPublicInputs(t *testing.T) {
	for _, args := range [][]string{{"-public"}, {"https://example.com"}} {
		if err := run(args); err == nil {
			t.Fatalf("public input accepted: %v", args)
		}
	}
}

func TestModernComparisonValidatesBeforeDependencyProbe(t *testing.T) {
	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"-runs", "0"}, "-runs"},
		{[]string{"-runs", "101"}, "-runs"},
		{[]string{"-timeout", "0s"}, "-timeout"},
		{[]string{"-iteration", "0"}, "-iteration"},
		{[]string{"-worker", "unknown"}, "unknown worker"},
		{[]string{"-worker", "playwright_chromium", "-browser", "unused", "-dependency-json", "{}"}, "invalid internal dependency"},
	} {
		if err := run(tt.args); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("%v: expected %q, got %v", tt.args, tt.want, err)
		}
	}
}

func TestModernReportWriteCreatesIndependentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "modern", "report.json")
	if err := writeReport(path, []byte("modern report\n")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "modern report\n" {
		t.Fatalf("report write: %q, %v", data, err)
	}
}
