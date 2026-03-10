package cli

import (
	"fmt"
	"strings"
	"testing"
)

func TestNotInitializedError(t *testing.T) {
	t.Parallel()
	msg := notInitializedError()
	if !strings.Contains(msg, "not initialized") {
		t.Errorf("expected 'not initialized' in message, got: %q", msg)
	}
	if !strings.Contains(msg, "kf up") {
		t.Errorf("expected 'kf up' guidance in message, got: %q", msg)
	}
}

func TestGiteaNotRunningError(t *testing.T) {
	t.Parallel()
	msg := giteaNotRunningError()
	if !strings.Contains(msg, "not running") {
		t.Errorf("expected 'not running' in message, got: %q", msg)
	}
	if !strings.Contains(msg, "kf up") {
		t.Errorf("expected 'kf up' guidance in message, got: %q", msg)
	}
}

func TestNotInitializedDistinctFromGiteaDown(t *testing.T) {
	t.Parallel()
	if notInitializedError() == giteaNotRunningError() {
		t.Error("not-initialized and gitea-not-running errors should be distinct")
	}
}

func TestConfigLoadError(t *testing.T) {
	t.Parallel()
	msg := configLoadError(fmt.Errorf("file not found"))
	if !strings.Contains(msg, "file not found") {
		t.Errorf("expected wrapped error in message, got: %q", msg)
	}
	if !strings.Contains(msg, "kf up") {
		t.Errorf("expected 'kf up' guidance in message, got: %q", msg)
	}
}

func TestEmptyState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resource string
		hint     string
		wantSub  []string
	}{
		{
			name:     "with hint",
			resource: "agents tracked",
			hint:     "Spawn one with: kf implement <track-id>",
			wantSub:  []string{"No agents tracked.", "Spawn one"},
		},
		{
			name:     "without hint",
			resource: "projects",
			hint:     "",
			wantSub:  []string{"No projects."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := emptyState(tt.resource, tt.hint)
			for _, sub := range tt.wantSub {
				if !strings.Contains(got, sub) {
					t.Errorf("emptyState(%q, %q) = %q, want substring %q", tt.resource, tt.hint, got, sub)
				}
			}
		})
	}
}

func TestEmptyState_NoHintNoNewline(t *testing.T) {
	t.Parallel()
	got := emptyState("items", "")
	if strings.Contains(got, "\n") {
		t.Errorf("emptyState with empty hint should not contain newline, got: %q", got)
	}
}

func TestPrereqErrorContext_NoPanic(t *testing.T) {
	t.Parallel()
	// Should not panic regardless of installed tools on test machine.
	_ = prereqErrorContext()
}
