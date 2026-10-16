package benchmark

import (
	"ai-go-browser/browser"
	"fmt"
)

func residentBytes(pid int) (uint64, error) {
	m := browser.ReadProcessMemory(pid)
	if !m.Available {
		return 0, fmt.Errorf("resident memory unavailable (%s)", m.Source)
	}
	return uint64(m.RSSMiB * (1 << 20)), nil
}

func sampleRSS(pid int, tree bool) (*uint64, string) {
	pids := []int{pid}
	source := "OS resident working set, single process"
	if tree {
		var err error
		pids, err = processTree(pid)
		if err != nil {
			return nil, "unavailable: " + err.Error()
		}
		source = "sum of OS resident working sets for Chromium process and descendants; shared pages may be counted more than once"
	}
	var total uint64
	for _, id := range pids {
		value, err := residentBytes(id)
		if err != nil {
			return nil, fmt.Sprintf("unavailable: PID %d: %v", id, err)
		}
		total += value
	}
	return &total, source
}

func descendants(root int, parents map[int]int) []int {
	out := []int{root}
	seen := map[int]bool{root: true}
	for i := 0; i < len(out); i++ {
		for pid, parent := range parents {
			if parent == out[i] && !seen[pid] {
				seen[pid] = true
				out = append(out, pid)
			}
		}
	}
	return out
}
