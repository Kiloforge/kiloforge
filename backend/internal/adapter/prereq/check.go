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

	// docker
	if _, err := exec.LookPath("docker"); err != nil {
		errs = append(errs, PrereqError{
			Tool:        "docker",
			Reason:      "required for running Gitea container",
			InstallHint: dockerHint(platform),
		})
	}

	// docker compose (v2 or v1)
	if !hasDockerCompose() {
		errs = append(errs, PrereqError{
			Tool:        "docker compose",
			Reason:      "required for container lifecycle management",
			InstallHint: composeHint(platform),
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

func hasDockerCompose() bool {
	if exec.Command("docker", "compose", "version").Run() == nil {
		return true
	}
	if exec.Command("docker-compose", "version").Run() == nil {
		return true
	}
	return false
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
	b.WriteString("\nInstall the missing tools and run 'kf init' again.")
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

func dockerHint(platform string) string {
	switch platform {
	case "darwin":
		return "Install Docker Desktop: https://docker.com/products/docker-desktop  (or: brew install --cask docker)"
	default:
		return "Install Docker Engine: https://docs.docker.com/engine/install/"
	}
}

func composeHint(platform string) string {
	switch platform {
	case "darwin":
		return "brew install docker-compose  (included with Docker Desktop)"
	default:
		return "sudo apt install docker-compose-plugin  (or install Docker Desktop)"
	}
}

func claudeHint() string {
	return "npm install -g @anthropic-ai/claude-code  (https://docs.anthropic.com/en/docs/claude-code)"
}
