package browser

import (
	"bytes"
	"os"
	"strconv"
)

func processRSS(pid int) (float64, string, bool) {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/status")
	if err != nil {
		return 0, "unavailable", false
	}
	for _, line := range bytes.Split(data, []byte("\n")) {
		if !bytes.HasPrefix(line, []byte("VmRSS:")) {
			continue
		}
		fields := bytes.Fields(line)
		if len(fields) < 2 {
			break
		}
		value, err := strconv.ParseFloat(string(fields[1]), 64)
		if err == nil {
			return value / 1024, "proc_status", true
		}
		break
	}
	return 0, "unavailable", false
}
