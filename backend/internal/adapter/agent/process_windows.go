//go:build windows

package agent

import (
	"os"
	"syscall"
)

const processQueryLimitedInformation = 0x1000

// ProcessAlive checks if a process with the given PID exists on Windows.
// Uses OpenProcess with PROCESS_QUERY_LIMITED_INFORMATION to test existence
// without sending any signal.
func ProcessAlive(pid int) bool {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	syscall.CloseHandle(h)
	return true
}

// interruptProcess sends an interrupt signal (CTRL_BREAK_EVENT) to the process.
func interruptProcess(pid int) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = proc.Signal(os.Interrupt)
}

// killProcess forcefully terminates the process (TerminateProcess).
func killProcess(pid int) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = proc.Kill()
}
