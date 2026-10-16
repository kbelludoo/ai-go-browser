//go:build linux

package benchmark

import (
	"os"
	"strconv"
	"strings"
)

func processTree(root int) ([]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	parents := map[int]int{}
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		b, err := os.ReadFile("/proc/" + entry.Name() + "/stat")
		if err != nil {
			continue
		}
		text := string(b)
		end := strings.LastIndex(text, ")")
		if end < 0 {
			continue
		}
		fields := strings.Fields(text[end+1:])
		if len(fields) < 2 {
			continue
		}
		parent, err := strconv.Atoi(fields[1])
		if err == nil {
			parents[pid] = parent
		}
	}
	return descendants(root, parents), nil
}
