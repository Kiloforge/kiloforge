package prereq

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// PrereqError describes a missing prerequisite tool.
type PrereqError struct {
	Tool        string
	Reason      string
	InstallHint string
}

// Check verifies all required tools are available. Returns a slice of
// errors for any missing tools (empty slice means all present).
func Check() []PrereqError {
	var errs []PrereqError
	platform := runtime.GOOS

	// git
	if _, err := exec.LookPath("git"); err != nil {
		errs = append(errs, PrereqError{
			Tool:        "git",
			Reason:      "required for repository management and worktrees",
			InstallHint: gitHint(platform),
		})
	}

	// claude
	if _, err := exec.LookPath("claude"); err != nil {
		errs = append(errs, PrereqError{
			Tool:        "claude",
			Reason:      "required for agent spawning (kf implement)",
			InstallHint: claudeHint(),
		})
	}

	return errs
}

// FormatErrors formats prerequisite errors into a user-friendly message.
func FormatErrors(errs []PrereqError) string {
	if len(errs) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Missing prerequisites:\n")
	for _, e := range errs {
		fmt.Fprintf(&b, "\n  %s — %s\n", e.Tool, e.Reason)
		fmt.Fprintf(&b, "    Install: %s\n", e.InstallHint)
	}
	b.WriteString("\nInstall the missing tools and run 'kf up' again.")
	return b.String()
}

func gitHint(platform string) string {
	switch platform {
	case "darwin":
		return "xcode-select --install  (or: brew install git)"
	default:
		return "sudo apt install git  (or: sudo dnf install git)"
	}
}

func claudeHint() string {
	return "npm install -g @anthropic-ai/claude-code  (https://docs.anthropic.com/en/docs/claude-code)"
}
