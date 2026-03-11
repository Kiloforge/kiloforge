package prereq

import (
	"strings"
	"testing"
)

func TestCheck_AllPresent(t *testing.T) {
	// On a dev machine, all tools should be present.
	errs := Check()
	if len(errs) > 0 {
		var tools []string
		for _, e := range errs {
			tools = append(tools, e.Tool)
		}
		t.Skipf("missing tools on this machine (expected in CI): %s", strings.Join(tools, ", "))
	}
}

func TestFormatErrors_Empty(t *testing.T) {
	result := FormatErrors(nil)
	if result != "" {
		t.Errorf("expected empty string for no errors, got: %q", result)
	}
}

func TestFormatErrors_Single(t *testing.T) {
	errs := []PrereqError{
		{Tool: "git", Reason: "needed for repos", InstallHint: "brew install git"},
	}
	result := FormatErrors(errs)
	if !strings.Contains(result, "git") {
		t.Error("expected 'git' in output")
	}
	if !strings.Contains(result, "brew install git") {
		t.Error("expected install hint in output")
	}
	if !strings.Contains(result, "Missing prerequisites") {
		t.Error("expected header in output")
	}
}

func TestFormatErrors_Multiple(t *testing.T) {
	errs := []PrereqError{
		{Tool: "git", Reason: "repos", InstallHint: "install git"},
		{Tool: "docker", Reason: "containers", InstallHint: "install docker"},
		{Tool: "claude", Reason: "agents", InstallHint: "install claude"},
	}
	result := FormatErrors(errs)
	if strings.Count(result, "Install:") != 3 {
		t.Errorf("expected 3 install hints, got output:\n%s", result)
	}
	if !strings.Contains(result, "run 'kf up' again") {
		t.Error("expected retry instruction")
	}
}

func TestGitHint_Darwin(t *testing.T) {
	hint := gitHint("darwin")
	if !strings.Contains(hint, "xcode-select") {
		t.Errorf("expected xcode-select for darwin, got: %s", hint)
	}
}

func TestGitHint_Windows(t *testing.T) {
	hint := gitHint("windows")
	if !strings.Contains(hint, "winget") {
		t.Errorf("expected winget for windows, got: %s", hint)
	}
}

func TestGitHint_Linux(t *testing.T) {
	hint := gitHint("linux")
	if !strings.Contains(hint, "apt") {
		t.Errorf("expected apt for linux, got: %s", hint)
	}
}

func TestClaudeHint(t *testing.T) {
	hint := claudeHint()
	if !strings.Contains(hint, "npm install") {
		t.Errorf("expected npm install in hint, got: %s", hint)
	}
}
