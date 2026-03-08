package ws

import "sync"

// RingBuffer stores the last N lines of output for reconnecting clients.
type RingBuffer struct {
	mu    sync.Mutex
	lines [][]byte
	size  int
	pos   int
	full  bool
}

// NewRingBuffer creates a ring buffer that holds up to size entries.
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		lines: make([][]byte, size),
		size:  size,
	}
}

// Write adds a line to the buffer.
func (rb *RingBuffer) Write(data []byte) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	// Copy data so callers can reuse their slice.
	cp := make([]byte, len(data))
	copy(cp, data)

	rb.lines[rb.pos] = cp
	rb.pos = (rb.pos + 1) % rb.size
	if rb.pos == 0 {
		rb.full = true
	}
}

// Lines returns all buffered lines in order (oldest first).
func (rb *RingBuffer) Lines() [][]byte {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if !rb.full {
		result := make([][]byte, rb.pos)
		copy(result, rb.lines[:rb.pos])
		return result
	}

	result := make([][]byte, rb.size)
	copy(result, rb.lines[rb.pos:])
	copy(result[rb.size-rb.pos:], rb.lines[:rb.pos])
	return result
}
