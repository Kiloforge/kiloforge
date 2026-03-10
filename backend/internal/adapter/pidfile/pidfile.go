package pidfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const FileName = "cortex.pid"

// oldFileName is the pre-rebrand PID file name, checked for backward compatibility.
const oldFileName = "orchestrator.pid"

// Manager implements port.PIDManager using a file on disk.
type Manager struct {
	path string
}

// New creates a PID file manager for the given data directory.
// If the old "orchestrator.pid" file exists, it is renamed to "cortex.pid".
func New(dataDir string) *Manager {
	newPath := filepath.Join(dataDir, FileName)
	oldPath := filepath.Join(dataDir, oldFileName)
	if _, err := os.Stat(oldPath); err == nil {
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			_ = os.Rename(oldPath, newPath)
		}
	}
	return &Manager{path: newPath}
}

func (m *Manager) Write(pid int) error {
	return os.WriteFile(m.path, []byte(strconv.Itoa(pid)), 0o644)
}

func (m *Manager) Read() (int, error) {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid pid file: %w", err)
	}
	return pid, nil
}

func (m *Manager) Remove() error {
	err := os.Remove(m.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// IsRunning checks if the PID file exists and the process is alive.
func (m *Manager) IsRunning() (bool, int, error) {
	pid, err := m.Read()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, 0, nil
		}
		return false, 0, err
	}

	if !processAlive(pid) {
		return false, pid, nil
	}
	return true, pid, nil
}

// processAlive checks if a process with the given PID exists.
func processAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks process existence without sending a signal.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
