package service_test

import (
	"context"
	"errors"
	"testing"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/service"
	"kiloforge/internal/core/testutil"
)

func TestMergeAndCleanup_HappyPath(t *testing.T) {
	t.Parallel()

	merger := &testutil.MockMerger{}
	poolRet := &testutil.MockPoolReturner{}
	agentStore := &testutil.MockAgentStore{}
	agentStore.AddAgent(domain.AgentInfo{ID: "dev-1", Status: "waiting-review"})
	agentStore.AddAgent(domain.AgentInfo{ID: "rev-1", Status: "running"})

	tracking := &domain.PRTracking{
		PRNumber:         5,
		TrackID:          "my-track",
		ProjectSlug:      "myapp",
		DeveloperAgentID: "dev-1",
		DeveloperSession: "sess-dev",
		ReviewerAgentID:  "rev-1",
		ReviewerSession:  "sess-rev",
		Status:           "approved",
	}

	err := service.MergeAndCleanup(context.Background(), service.CleanupOpts{
		Tracking:    tracking,
		AgentStore:  agentStore,
		Merger:      merger,
		PoolReturn:  poolRet,
		MergeMethod: "merge",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tracking.Status != "merged" {
		t.Errorf("Status = %q, want %q", tracking.Status, "merged")
	}

	// Check merge was called.
	var hasMerge bool
	for _, c := range merger.Calls {
		if c.Method == "MergePR" {
			hasMerge = true
		}
	}
	if !hasMerge {
		t.Error("expected MergePR call")
	}

	// Check pool return was called.
	if len(poolRet.Calls) != 1 || poolRet.Calls[0] != "my-track" {
		t.Errorf("pool return calls = %v, want [my-track]", poolRet.Calls)
	}

	// Check agents marked completed.
	dev, _ := agentStore.FindAgent("dev-1")
	if dev.Status != "completed" {
		t.Errorf("dev status = %q, want %q", dev.Status, "completed")
	}
	rev, _ := agentStore.FindAgent("rev-1")
	if rev.Status != "completed" {
		t.Errorf("rev status = %q, want %q", rev.Status, "completed")
	}
}

func TestMergeAndCleanup_MergeFailure(t *testing.T) {
	t.Parallel()

	mergeErr := errors.New("merge conflict")
	merger := &testutil.MockMerger{MergeErr: mergeErr}

	tracking := &domain.PRTracking{
		PRNumber:    5,
		ProjectSlug: "myapp",
		Status:      "approved",
	}

	err := service.MergeAndCleanup(context.Background(), service.CleanupOpts{
		Tracking: tracking,
		Merger:   merger,
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, mergeErr) {
		t.Errorf("error = %v, want wrapped %v", err, mergeErr)
	}
	// Status should NOT be merged on failure.
	if tracking.Status == "merged" {
		t.Error("status should not be merged after failure")
	}
}

func TestMergeAndCleanup_DefaultMergeMethod(t *testing.T) {
	t.Parallel()

	merger := &testutil.MockMerger{}
	tracking := &domain.PRTracking{
		PRNumber:    5,
		ProjectSlug: "myapp",
		Status:      "approved",
	}

	err := service.MergeAndCleanup(context.Background(), service.CleanupOpts{
		Tracking: tracking,
		Merger:   merger,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use default "merge" method.
	if len(merger.Calls) == 0 {
		t.Fatal("expected merge call")
	}
	method := merger.Calls[0].Args[1].(string)
	if method != "merge" {
		t.Errorf("method = %q, want %q", method, "merge")
	}
}

func TestMergeAndCleanup_NilPoolReturn(t *testing.T) {
	t.Parallel()

	merger := &testutil.MockMerger{}
	tracking := &domain.PRTracking{
		PRNumber:    5,
		ProjectSlug: "myapp",
		TrackID:     "my-track",
		Status:      "approved",
	}

	// Should not panic with nil PoolReturn.
	err := service.MergeAndCleanup(context.Background(), service.CleanupOpts{
		Tracking:   tracking,
		Merger:     merger,
		PoolReturn: nil,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMergeAndCleanup_NilAgentStore(t *testing.T) {
	t.Parallel()

	merger := &testutil.MockMerger{}
	tracking := &domain.PRTracking{
		PRNumber:         5,
		ProjectSlug:      "myapp",
		DeveloperAgentID: "dev-1",
		Status:           "approved",
	}

	// Should not panic with nil AgentStore.
	err := service.MergeAndCleanup(context.Background(), service.CleanupOpts{
		Tracking:   tracking,
		Merger:     merger,
		AgentStore: nil,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMergeAndCleanup_PoolReturnError(t *testing.T) {
	t.Parallel()

	merger := &testutil.MockMerger{}
	poolRet := &testutil.MockPoolReturner{ReturnErr: errors.New("pool error")}

	tracking := &domain.PRTracking{
		PRNumber:    5,
		ProjectSlug: "myapp",
		TrackID:     "my-track",
		Status:      "approved",
	}

	// Pool return error is logged, not returned.
	err := service.MergeAndCleanup(context.Background(), service.CleanupOpts{
		Tracking:   tracking,
		Merger:     merger,
		PoolReturn: poolRet,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tracking.Status != "merged" {
		t.Errorf("Status = %q, want %q", tracking.Status, "merged")
	}
}
