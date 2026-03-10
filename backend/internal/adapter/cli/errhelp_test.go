package cli

import (
	"fmt"
	"strings"
	"testing"
)

func TestNotInitializedError(t *testing.T) {
	t.Parallel()
	msg := notInitializedError()
	if msg == "" {
		t.Fatal("expected non-empty error message")
	}
	assertContains(t, msg, "not initialized")
	assertContains(t, msg, "kf init")
}

func TestNotInitializedErrorDistinctFromGiteaDown(t *testing.T) {
	t.Parallel()
	initMsg := notInitializedError()
	giteaMsg := giteaNotRunningError()
	if initMsg == giteaMsg {
		t.Error("not-initialized and gitea-not-running errors should be distinct")
	}
}

func TestGiteaNotRunningError(t *testing.T) {
	t.Parallel()
	msg := giteaNotRunningError()
	if msg == "" {
		t.Fatal("expected non-empty error message")
	}
	assertContains(t, msg, "not running")
	assertContains(t, msg, "kf up")
}

func TestConfigLoadError(t *testing.T) {
	t.Parallel()
	msg := configLoadError(fmt.Errorf("file not found"))
	assertContains(t, msg, "file not found")
	assertContains(t, msg, "kf init")
}

func TestEmptyStateMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resource string
		hint     string
		wantRes  string
		wantHint string
	}{
		{
			name:     "with hint",
			resource: "agents",
			hint:     "Spawn one with: kf implement <track-id>",
			wantRes:  "No agents",
			wantHint: "kf implement",
		},
		{
			name:     "without hint",
			resource: "cost data",
			hint:     "",
			wantRes:  "No cost data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := emptyState(tt.resource, tt.hint)
			assertContains(t, msg, tt.wantRes)
			if tt.wantHint != "" {
				assertContains(t, msg, tt.wantHint)
			}
		})
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}
