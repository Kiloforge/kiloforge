//go:build !windows

package agent

import (
	"os"
	"syscall"
)

// ProcessAlive checks if a process with the given PID exists.
func ProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// interruptProcess sends SIGINT to the process.
func interruptProcess(pid int) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = proc.Signal(os.Interrupt)
}

// killProcess forcefully terminates the process (SIGKILL).
func killProcess(pid int) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = proc.Signal(syscall.SIGKILL)
}
