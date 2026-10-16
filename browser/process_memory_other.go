//go:build !windows && !linux

package browser

func processRSS(pid int) (float64, string, bool) {
	return 0, "unavailable", false
}
