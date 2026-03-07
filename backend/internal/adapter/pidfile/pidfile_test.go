package pidfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndRead(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	m := New(dir)

	if err := m.Write(12345); err != nil {
		t.Fatalf("Write: %v", err)
	}

	pid, err := m.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if pid != 12345 {
		t.Errorf("pid = %d, want 12345", pid)
	}
}

func TestRead_NoFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	m := New(dir)

	_, err := m.Read()
	if err == nil {
		t.Fatal("expected error for missing PID file")
	}
}

func TestRead_InvalidContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, FileName), []byte("not-a-number"), 0o644)

	m := New(dir)
	_, err := m.Read()
	if err == nil {
		t.Fatal("expected error for invalid PID content")
	}
}

func TestRemove(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	m := New(dir)

	_ = m.Write(1)
	if err := m.Remove(); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Removing again should not error.
	if err := m.Remove(); err != nil {
		t.Fatalf("Remove (idempotent): %v", err)
	}
}

func TestIsRunning_NoFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	m := New(dir)

	running, pid, err := m.IsRunning()
	if err != nil {
		t.Fatalf("IsRunning: %v", err)
	}
	if running {
		t.Error("expected not running")
	}
	if pid != 0 {
		t.Errorf("pid = %d, want 0", pid)
	}
}

func TestIsRunning_CurrentProcess(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	m := New(dir)

	myPID := os.Getpid()
	_ = m.Write(myPID)

	running, pid, err := m.IsRunning()
	if err != nil {
		t.Fatalf("IsRunning: %v", err)
	}
	if !running {
		t.Error("expected running for current process")
	}
	if pid != myPID {
		t.Errorf("pid = %d, want %d", pid, myPID)
	}
}

func TestIsRunning_StalePID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	m := New(dir)

	// PID 99999999 should not exist.
	_ = m.Write(99999999)

	running, pid, err := m.IsRunning()
	if err != nil {
		t.Fatalf("IsRunning: %v", err)
	}
	if running {
		t.Error("expected not running for stale PID")
	}
	if pid != 99999999 {
		t.Errorf("pid = %d, want 99999999", pid)
	}
}
