//go:build !windows && !linux

package benchmark

import "fmt"

func processTree(pid int) ([]int, error) {
	return nil, fmt.Errorf("process tree sampler supported on Windows and Linux only")
}
