package orchestration

import (
	"context"
	"testing"

	"crelay/internal/core/domain"
	"crelay/internal/state"
)

type mockMerger struct {
	mergedPR    int
	mergeMethod string
	commented   string
	deletedBranch string
}

func (m *mockMerger) MergePR(ctx context.Context, repo string, prNum int, method string) error {
	m.mergedPR = prNum
	m.mergeMethod = method
	return nil
}

func (m *mockMerger) CommentOnPR(ctx context.Context, repo string, prNum int, body string) error {
	m.commented = body
	return nil
}

func (m *mockMerger) DeleteBranch(ctx context.Context, repo, branch string) error {
	m.deletedBranch = branch
	return nil
}

type mockPoolReturner struct {
	returnedTrackID string
}

func (m *mockPoolReturner) ReturnByTrackID(trackID string) error {
	m.returnedTrackID = trackID
	return nil
}

func TestMergeAndCleanup_FullFlow(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	merger := &mockMerger{}
	poolRet := &mockPoolReturner{}

	store := &state.Store{
		Agents: []domain.AgentInfo{
			{ID: "dev-123", Role: "developer", Ref: "my-track", Status: "running", PID: 0, SessionID: "dev-sess"},
			{ID: "rev-456", Role: "reviewer", Ref: "PR #5", Status: "completed", SessionID: "rev-sess"},
		},
	}

	tracking := &PRTracking{
		PRNumber:         5,
		TrackID:          "my-track",
		ProjectSlug:      "myapp",
		DeveloperAgentID: "dev-123",
		DeveloperSession: "dev-sess",
		ReviewerAgentID:  "rev-456",
		ReviewerSession:  "rev-sess",
		Status:           "approved",
	}

	opts := CleanupOpts{
		Tracking:    tracking,
		Store:       store,
		Merger:      merger,
		PoolReturn:  poolRet,
		DataDir:     dir,
		MergeMethod: "merge",
	}

	err := MergeAndCleanup(context.Background(), opts)
	if err != nil {
		t.Fatalf("MergeAndCleanup: %v", err)
	}

	// Verify merge was called.
	if merger.mergedPR != 5 {
		t.Errorf("mergedPR: want 5, got %d", merger.mergedPR)
	}
	if merger.mergeMethod != "merge" {
		t.Errorf("mergeMethod: want %q, got %q", "merge", merger.mergeMethod)
	}

	// Verify comment posted.
	if merger.commented == "" {
		t.Error("expected comment to be posted")
	}

	// Verify remote branch deleted.
	if merger.deletedBranch != "my-track" {
		t.Errorf("deletedBranch: want %q, got %q", "my-track", merger.deletedBranch)
	}

	// Verify pool returned.
	if poolRet.returnedTrackID != "my-track" {
		t.Errorf("returnedTrackID: want %q, got %q", "my-track", poolRet.returnedTrackID)
	}

	// Verify agent status updated.
	dev, _ := store.FindAgent("dev-123")
	if dev.Status != "completed" {
		t.Errorf("developer status: want %q, got %q", "completed", dev.Status)
	}
	rev, _ := store.FindAgent("rev-456")
	if rev.Status != "completed" {
		t.Errorf("reviewer status: want %q, got %q", "completed", rev.Status)
	}

	// Verify tracking status.
	if tracking.Status != "merged" {
		t.Errorf("tracking status: want %q, got %q", "merged", tracking.Status)
	}
}

func TestMergeAndCleanup_DefaultMergeMethod(t *testing.T) {
	t.Parallel()

	merger := &mockMerger{}
	store := &state.Store{}

	opts := CleanupOpts{
		Tracking:   &PRTracking{PRNumber: 1, ProjectSlug: "app"},
		Store:      store,
		Merger:     merger,
		PoolReturn: nil,
	}

	err := MergeAndCleanup(context.Background(), opts)
	if err != nil {
		t.Fatalf("MergeAndCleanup: %v", err)
	}
	if merger.mergeMethod != "merge" {
		t.Errorf("default merge method: want %q, got %q", "merge", merger.mergeMethod)
	}
}
