package domain_test

import (
	"errors"
	"testing"

	"kiloforge/internal/core/domain"
)

func TestSentinelErrors_AreDistinct(t *testing.T) {
	t.Parallel()

	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrProjectNotFound", domain.ErrProjectNotFound},
		{"ErrProjectExists", domain.ErrProjectExists},
		{"ErrAgentNotFound", domain.ErrAgentNotFound},
		{"ErrPRTrackingNotFound", domain.ErrPRTrackingNotFound},
		{"ErrPoolExhausted", domain.ErrPoolExhausted},

		{"ErrForbidden", domain.ErrForbidden},
	}

	for i, a := range sentinels {
		for j, b := range sentinels {
			if i != j && errors.Is(a.err, b.err) {
				t.Errorf("%s should not match %s", a.name, b.name)
			}
		}
	}
}

func TestSentinelErrors_MatchWithErrorsIs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{"ErrProjectNotFound", domain.ErrProjectNotFound},
		{"ErrProjectExists", domain.ErrProjectExists},
		{"ErrAgentNotFound", domain.ErrAgentNotFound},
		{"ErrPRTrackingNotFound", domain.ErrPRTrackingNotFound},
		{"ErrPoolExhausted", domain.ErrPoolExhausted},

		{"ErrForbidden", domain.ErrForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if !errors.Is(tt.err, tt.err) {
				t.Errorf("errors.Is(%s, %s) should be true", tt.name, tt.name)
			}
		})
	}
}

func TestProjectStatus_Constants(t *testing.T) {
	t.Parallel()

	if domain.ProjectActive != "active" {
		t.Errorf("ProjectActive = %q, want %q", domain.ProjectActive, "active")
	}
	if domain.ProjectInactive != "inactive" {
		t.Errorf("ProjectInactive = %q, want %q", domain.ProjectInactive, "inactive")
	}
}

func TestAgentRole_Constants(t *testing.T) {
	t.Parallel()

	if domain.AgentRoleDeveloper != "developer" {
		t.Errorf("AgentRoleDeveloper = %q, want %q", domain.AgentRoleDeveloper, "developer")
	}
}

func TestIsAdvisorRole(t *testing.T) {
	t.Parallel()
	tests := []struct {
		role string
		want bool
	}{
		{"advisor-product", true},
		{"advisor-reliability", true},
		{"advisor-unknown", true},
		{"developer", false},
		{"interactive", false},
		{"architect", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := domain.IsAdvisorRole(tt.role); got != tt.want {
			t.Errorf("IsAdvisorRole(%q) = %v, want %v", tt.role, got, tt.want)
		}
	}
}

func TestIsWorkerRole(t *testing.T) {
	t.Parallel()
	tests := []struct {
		role string
		want bool
	}{
		{"developer", true},
		{"reviewer", false},
		{"interactive", false},
		{"architect", false},
		{"advisor-product", false},
		{"advisor-reliability", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := domain.IsWorkerRole(tt.role); got != tt.want {
			t.Errorf("IsWorkerRole(%q) = %v, want %v", tt.role, got, tt.want)
		}
	}
}

func TestAgentStatus_Constants(t *testing.T) {
	t.Parallel()

	statuses := map[domain.AgentStatus]string{
		domain.AgentStatusRunning:   "running",
		domain.AgentStatusWaiting:   "waiting",
		domain.AgentStatusHalted:    "halted",
		domain.AgentStatusStopped:   "stopped",
		domain.AgentStatusCompleted: "completed",
		domain.AgentStatusFailed:    "failed",
	}

	for status, expected := range statuses {
		if string(status) != expected {
			t.Errorf("AgentStatus %q != %q", status, expected)
		}
	}
}

func TestProject_ZeroValue(t *testing.T) {
	t.Parallel()

	var p domain.Project
	if p.Slug != "" {
		t.Error("zero-value Project.Slug should be empty")
	}
	if p.Active {
		t.Error("zero-value Project.Active should be false")
	}
}

func TestPRTracking_ZeroValue(t *testing.T) {
	t.Parallel()

	var pr domain.PRTracking
	if pr.PRNumber != 0 {
		t.Error("zero-value PRTracking.PRNumber should be 0")
	}
	if pr.ReviewCycleCount != 0 {
		t.Error("zero-value PRTracking.ReviewCycleCount should be 0")
	}
}
