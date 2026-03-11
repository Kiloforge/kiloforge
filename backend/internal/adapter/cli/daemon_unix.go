//go:build !windows

package cli

import (
	"os"
	"os/exec"
	"syscall"
)

// daemonSysProcAttr returns SysProcAttr for detaching the daemon process.
func daemonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

// terminateProcess sends SIGTERM to gracefully stop a process.
func terminateProcess(proc *os.Process) error {
	return proc.Signal(syscall.SIGTERM)
}

// forceKillProcess sends SIGKILL to forcefully stop a process.
func forceKillProcess(proc *os.Process) {
	_ = proc.Signal(syscall.SIGKILL)
}

// setDaemonAttrs configures the command for background daemon execution.
func setDaemonAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = daemonSysProcAttr()
}
