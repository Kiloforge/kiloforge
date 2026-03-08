package ws

import (
	"testing"
)

func TestRingBuffer_Basic(t *testing.T) {
	t.Parallel()
	rb := NewRingBuffer(3)

	rb.Write([]byte("a"))
	rb.Write([]byte("b"))
	lines := rb.Lines()
	if len(lines) != 2 {
		t.Fatalf("len = %d, want 2", len(lines))
	}
	if string(lines[0]) != "a" || string(lines[1]) != "b" {
		t.Errorf("lines = %v, want [a, b]", lines)
	}
}

func TestRingBuffer_Wrap(t *testing.T) {
	t.Parallel()
	rb := NewRingBuffer(3)

	rb.Write([]byte("a"))
	rb.Write([]byte("b"))
	rb.Write([]byte("c"))
	rb.Write([]byte("d")) // wraps, evicts "a"

	lines := rb.Lines()
	if len(lines) != 3 {
		t.Fatalf("len = %d, want 3", len(lines))
	}
	if string(lines[0]) != "b" || string(lines[1]) != "c" || string(lines[2]) != "d" {
		t.Errorf("lines = [%s, %s, %s], want [b, c, d]", lines[0], lines[1], lines[2])
	}
}

func TestRingBuffer_Empty(t *testing.T) {
	t.Parallel()
	rb := NewRingBuffer(5)
	lines := rb.Lines()
	if len(lines) != 0 {
		t.Errorf("len = %d, want 0", len(lines))
	}
}

func TestRingBuffer_ExactFull(t *testing.T) {
	t.Parallel()
	rb := NewRingBuffer(2)
	rb.Write([]byte("x"))
	rb.Write([]byte("y"))

	lines := rb.Lines()
	if len(lines) != 2 {
		t.Fatalf("len = %d, want 2", len(lines))
	}
	if string(lines[0]) != "x" || string(lines[1]) != "y" {
		t.Errorf("lines = [%s, %s], want [x, y]", lines[0], lines[1])
	}
}
