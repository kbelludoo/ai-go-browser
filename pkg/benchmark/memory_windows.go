//go:build windows

package benchmark

import (
	"syscall"
	"unsafe"
)

func processTree(root int) ([]int, error) {
	h, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(h)
	var entry syscall.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err = syscall.Process32First(h, &entry); err != nil {
		return nil, err
	}
	parents := map[int]int{}
	for {
		parents[int(entry.ProcessID)] = int(entry.ParentProcessID)
		if err = syscall.Process32Next(h, &entry); err != nil {
			break
		}
	}
	return descendants(root, parents), nil
}
