package port

// PIDManager manages a PID file for daemon lifecycle.
type PIDManager interface {
	// Write writes the given PID to the PID file.
	Write(pid int) error
	// Read returns the PID from the PID file.
	Read() (int, error)
	// Remove deletes the PID file.
	Remove() error
	// IsRunning checks if the PID file exists and the process is alive.
	// Returns (running, pid, error).
	IsRunning() (bool, int, error)
}
