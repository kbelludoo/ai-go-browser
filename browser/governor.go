package browser

import (
	"errors"
	"io"
	"runtime/debug"
	"sync"
)

var (
	// ErrBodyLimitExceeded is returned when a remote server attempts to stream more than MaxBody bytes.
	ErrBodyLimitExceeded = errors.New("response body exceeded memory limit; aborting to preserve heap")
)

// MemoryGovernor enforces memory discipline inspired by the MIT Biscuit research:
// 1. GOMEMLIMIT soft budget to prevent kernel OOM kills on 1GB VPS.
// 2. Buffer pooling to reduce GC pause times to sub-millisecond levels.
// 3. Bounded streaming readers.
type MemoryGovernor struct {
	maxBodyBytes int64
	bufPool      sync.Pool
}

// NewMemoryGovernor initializes memory governance and configures the Go runtime soft limit.
func NewMemoryGovernor(memoryLimitMiB int64) *MemoryGovernor {
	if memoryLimitMiB > 0 {
		// Set GOMEMLIMIT so Go GC triggers proactively before the VPS Linux OOM killer strikes
		debug.SetMemoryLimit(memoryLimitMiB << 20)
	}

	return &MemoryGovernor{
		maxBodyBytes: MaxBody, // 5 MiB default
		bufPool: sync.Pool{
			New: func() any {
				// 64 KB reusable read buffers
				b := make([]byte, 64*1024)
				return &b
			},
		},
	}
}

// GetBuffer retrieves a pooled buffer from the sync.Pool to minimize heap allocations.
func (g *MemoryGovernor) GetBuffer() *[]byte {
	return g.bufPool.Get().(*[]byte)
}

// PutBuffer returns a buffer to the sync.Pool.
func (g *MemoryGovernor) PutBuffer(b *[]byte) {
	if b != nil {
		g.bufPool.Put(b)
	}
}

// ReadBounded reads from r up to maxBodyBytes. If the remote stream exceeds the limit,
// it aborts immediately, protecting the VPS heap from unbounded streaming attacks.
func (g *MemoryGovernor) ReadBounded(r io.Reader) ([]byte, error) {
	buf := g.GetBuffer()
	defer g.PutBuffer(buf)

	var result []byte
	var totalRead int64

	for {
		n, err := r.Read(*buf)
		if n > 0 {
			totalRead += int64(n)
			if totalRead > g.maxBodyBytes {
				return nil, ErrBodyLimitExceeded
			}
			result = append(result, (*buf)[:n]...)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
	}

	return result, nil
}
