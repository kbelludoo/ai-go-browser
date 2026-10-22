package browser

import (
	"bytes"
	"errors"
	"testing"
)

func TestMemoryGovernorBoundedReadAndPooling(t *testing.T) {
	gov := NewMemoryGovernor(128) // 128 MiB budget

	// 1. Normal read within limit
	data := []byte("Hello, memory governor! Testing sync.Pool and stream limits.")
	readBytes, err := gov.ReadBounded(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if string(readBytes) != string(data) {
		t.Errorf("expected %q, got %q", string(data), string(readBytes))
	}

	// 2. Stream exceeding MaxBody must be aborted proactively (heap protection)
	gov.maxBodyBytes = 1024 // 1 KB limit for testing
	overflowData := make([]byte, 2048)
	_, err = gov.ReadBounded(bytes.NewReader(overflowData))
	if !errors.Is(err, ErrBodyLimitExceeded) {
		t.Fatalf("expected ErrBodyLimitExceeded, got %v", err)
	}
}
