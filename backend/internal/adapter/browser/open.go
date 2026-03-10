package browser

import (
	"os/exec"
	"runtime"
)

// Open opens the given URL in the user's default browser.
// It is cross-platform: macOS (open), Linux (xdg-open), Windows (start).
func Open(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default: // linux, freebsd, etc.
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}
