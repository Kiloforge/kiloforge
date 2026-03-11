//go:build windows

package cli

import (
	"os"
	"os/exec"
	"syscall"
)

// setDaemonAttrs configures the command for background daemon execution on Windows.
// CREATE_NEW_PROCESS_GROUP detaches the child from the parent's console.
func setDaemonAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

// terminateProcess sends an interrupt to gracefully stop a process on Windows.
func terminateProcess(proc *os.Process) error {
	return proc.Signal(os.Interrupt)
}

// forceKillProcess forcefully terminates a process on Windows (TerminateProcess).
func forceKillProcess(proc *os.Process) {
	_ = proc.Kill()
}
