package browser

import (
	"os"
	"runtime"
	"testing"
)

func TestReadProcessMemory(t *testing.T) {
	m := ReadProcessMemory(os.Getpid())
	if runtime.GOOS == "windows" || runtime.GOOS == "linux" {
		if !m.Available || m.RSSMiB <= 0 || m.Source == "unavailable" {
			t.Fatalf("process memory unavailable on supported OS: %+v", m)
		}
	}
	for _, pid := range []int{0, -1, 2147483647} {
		m := ReadProcessMemory(pid)
		if m.Available || m.RSSMiB != 0 {
			t.Fatalf("invalid PID produced memory sample: %+v", m)
		}
	}
}
