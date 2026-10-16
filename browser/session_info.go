package browser

// ProcessID identifies the launched Chromium root process for resource sampling.
// Attached sessions return zero because this package did not launch the process.
func (s *StealthSession) ProcessID() int {
	if s == nil || s.proc == nil || s.proc.Process == nil {
		return 0
	}
	return s.proc.Process.Pid
}
