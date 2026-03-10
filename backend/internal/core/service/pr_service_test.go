package service_test

import (
	"context"
	"testing"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/service"
	"kiloforge/internal/core/testutil"
)

func TestCreateTracking_FindsDeveloperAgent(t *testing.T) {
	t.Parallel()

	svc := service.NewPRService(&testutil.MockAgentSpawner{}, &testutil.MockLogger{})

	agents := []domain.AgentInfo{
		{ID: "dev-1", Role: "developer", Ref: "feature/auth", SessionID: "sess-1", WorktreeDir: "/wt/dev-1"},
		{ID: "rev-1", Role: "reviewer", Ref: "PR #1", SessionID: "sess-2"},
	}

	tracking := svc.CreateTracking(5, "feature/auth", "myapp", agents, 3)

	if tracking.PRNumber != 5 {
		t.Errorf("PRNumber = %d, want 5", tracking.PRNumber)
	}
	if tracking.DeveloperAgentID != "dev-1" {
		t.Errorf("DeveloperAgentID = %q, want %q", tracking.DeveloperAgentID, "dev-1")
	}
	if tracking.DeveloperSession != "sess-1" {
		t.Errorf("DeveloperSession = %q, want %q", tracking.DeveloperSession, "sess-1")
	}
	if tracking.DeveloperWorkDir != "/wt/dev-1" {
		t.Errorf("DeveloperWorkDir = %q, want %q", tracking.DeveloperWorkDir, "/wt/dev-1")
	}
	if tracking.MaxReviewCycles != 3 {
		t.Errorf("MaxReviewCycles = %d, want 3", tracking.MaxReviewCycles)
	}
	if tracking.Status != "waiting-review" {
		t.Errorf("Status = %q, want %q", tracking.Status, "waiting-review")
	}
}

func TestCreateTracking_NoDeveloperAgent(t *testing.T) {
	t.Parallel()

	svc := service.NewPRService(&testutil.MockAgentSpawner{}, &testutil.MockLogger{})

	tracking := svc.CreateTracking(5, "feature/auth", "myapp", nil, 3)

	if tracking.DeveloperAgentID != "" {
		t.Errorf("DeveloperAgentID = %q, want empty", tracking.DeveloperAgentID)
	}
}

func TestHandleApproval_SetsApprovedStatus(t *testing.T) {
	t.Parallel()

	svc := service.NewPRService(&testutil.MockAgentSpawner{}, &testutil.MockLogger{})
	tracking := &domain.PRTracking{PRNumber: 5, Status: "in-review"}

	err := svc.HandleApproval(context.Background(), tracking)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tracking.Status != "approved" {
		t.Errorf("Status = %q, want %q", tracking.Status, "approved")
	}
}

func TestHandleChangesRequested_IncrementsCycle(t *testing.T) {
	t.Parallel()

	svc := service.NewPRService(&testutil.MockAgentSpawner{}, &testutil.MockLogger{})

	tests := []struct {
		name       string
		cycleCount int
		maxCycles  int
		wantResume bool
		wantStatus string
	}{
		{"first cycle", 0, 3, true, "changes-requested"},
		{"second cycle", 1, 3, true, "changes-requested"},
		{"at limit", 2, 3, false, "in-review"}, // status unchanged when limit hit
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tracking := &domain.PRTracking{
				ReviewCycleCount: tt.cycleCount,
				MaxReviewCycles:  tt.maxCycles,
				Status:           "in-review",
			}

			resume := svc.HandleChangesRequested(tracking)

			if resume != tt.wantResume {
				t.Errorf("resume = %v, want %v", resume, tt.wantResume)
			}
			if tt.wantResume && tracking.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", tracking.Status, tt.wantStatus)
			}
		})
	}
}
