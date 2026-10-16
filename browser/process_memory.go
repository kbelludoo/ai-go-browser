package browser

// ProcessMemory is an instantaneous OS sample, not a peak or a process-tree sum.
type ProcessMemory struct {
	PID       int     `json:"pid"`
	RSSMiB    float64 `json:"rss_mib"`
	Source    string  `json:"source"`
	Available bool    `json:"available"`
}

// ReadProcessMemory samples physical resident memory for a single process.
// Unavailable platforms or exited/inaccessible processes are explicit.
func ReadProcessMemory(pid int) ProcessMemory {
	m := ProcessMemory{PID: pid, Source: "unavailable"}
	if pid <= 0 {
		return m
	}
	m.RSSMiB, m.Source, m.Available = processRSS(pid)
	return m
}
