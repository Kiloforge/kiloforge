package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"kiloforge/internal/adapter/pidfile"
)

// startDaemon spawns `kf serve` as a detached background process.
// Returns the PID of the spawned process.
func startDaemon(dataDir string) (int, error) {
	pidMgr := pidfile.New(dataDir)

	// Check if already running.
	running, pid, err := pidMgr.IsRunning()
	if err != nil {
		return 0, fmt.Errorf("check pid: %w", err)
	}
	if running {
		return pid, nil
	}

	// Clean stale PID file if needed.
	if pid != 0 {
		_ = pidMgr.Remove()
	}

	// Find our own executable path.
	executable, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("find executable: %w", err)
	}

	// Open log file for daemon output.
	logPath := filepath.Join(dataDir, "relay.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, fmt.Errorf("open log file: %w", err)
	}
	defer logFile.Close()

	cmd := exec.Command(executable, "serve")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start daemon: %w", err)
	}

	daemonPID := cmd.Process.Pid

	// Detach — don't wait for the child.
	_ = cmd.Process.Release()

	// Wait briefly for PID file to appear.
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if running, _, _ := pidMgr.IsRunning(); running {
			return daemonPID, nil
		}
	}

	return daemonPID, nil
}

// stopDaemon sends SIGTERM to the relay daemon and waits for it to exit.
func stopDaemon(dataDir string) error {
	pidMgr := pidfile.New(dataDir)

	running, pid, err := pidMgr.IsRunning()
	if err != nil {
		return fmt.Errorf("check pid: %w", err)
	}
	if !running {
		if pid != 0 {
			// Stale PID file.
			_ = pidMgr.Remove()
		}
		return nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		_ = pidMgr.Remove()
		return nil
	}

	// Send SIGTERM.
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		_ = pidMgr.Remove()
		return nil
	}

	// Wait up to 5 seconds for clean exit.
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if running, _, _ := pidMgr.IsRunning(); !running {
			return nil
		}
	}

	// Force kill.
	_ = proc.Signal(syscall.SIGKILL)
	_ = pidMgr.Remove()
	return nil
}
