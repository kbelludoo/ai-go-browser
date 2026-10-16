package browser

import (
	"syscall"
	"unsafe"
)

var processMemoryInfo = syscall.NewLazyDLL("kernel32.dll").NewProc("K32GetProcessMemoryInfo")

// PROCESS_MEMORY_COUNTERS uses SIZE_T fields, so uintptr preserves the ABI
// for both Windows 386 and amd64/arm64. WorkingSetSize includes shared pages.
type processMemoryCounters struct {
	Size               uint32
	PageFaultCount     uint32
	PeakWorkingSetSize uintptr
	WorkingSetSize     uintptr
	QuotaPeakPagedPool uintptr
	QuotaPagedPool     uintptr
	QuotaPeakNonPaged  uintptr
	QuotaNonPaged      uintptr
	PagefileUsage      uintptr
	PeakPagefileUsage  uintptr
}

func processRSS(pid int) (float64, string, bool) {
	if err := processMemoryInfo.Find(); err != nil {
		return 0, "unavailable", false
	}
	handle, err := syscall.OpenProcess(0x1000, false, uint32(pid)) // PROCESS_QUERY_LIMITED_INFORMATION
	if err != nil {
		return 0, "unavailable", false
	}
	defer syscall.CloseHandle(handle)
	var counters processMemoryCounters
	counters.Size = uint32(unsafe.Sizeof(counters))
	ok, _, _ := processMemoryInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&counters)), uintptr(counters.Size))
	if ok == 0 {
		return 0, "unavailable", false
	}
	return float64(counters.WorkingSetSize) / 1048576, "windows_working_set", true
}
