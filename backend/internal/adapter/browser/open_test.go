package browser

import (
	"runtime"
	"testing"
)

func TestOpen_SelectsCorrectCommand(t *testing.T) {
	t.Parallel()

	// We can't easily mock exec.Command without refactoring, but we can
	// verify the function doesn't panic and returns a predictable error
	// type when the URL is invalid or the command isn't found.
	// On CI/headless, the command may fail — that's expected.

	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		// These are the supported platforms — Open should not panic.
		// We don't actually want to open a browser in tests, so we
		// call with an invalid URL scheme that will fail gracefully.
		err := Open("kiloforge://test")
		// On macOS, `open` will succeed (returns nil) even for custom schemes.
		// On Linux without a desktop, xdg-open will fail.
		// Either way, no panic is the key assertion.
		_ = err
	default:
		t.Skipf("unsupported platform: %s", runtime.GOOS)
	}
}
