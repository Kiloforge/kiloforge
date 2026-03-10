package sqlite

import (
	"errors"
	"testing"

	"kiloforge/internal/core/domain"
)

func TestPRTrackingStore_SaveAndLoad_AllFields(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewPRTrackingStore(db)

	pr := &domain.PRTracking{
		PRNumber:         10,
		ProjectSlug:      "myapp",
		TrackID:          "feat-login_2025",
		DeveloperAgentID: "dev-agent-1",
		DeveloperSession: "session-abc",
		DeveloperWorkDir: "/work/worker-1",
		ReviewerAgentID:  "rev-agent-1",
		ReviewerSession:  "session-def",
		ReviewCycleCount: 2,
		MaxReviewCycles:  5,
		Status:           "in-review",
	}

	if err := store.SavePRTracking("myapp", pr); err != nil {
		t.Fatalf("SavePRTracking: %v", err)
	}

	got, err := store.LoadPRTracking("myapp")
	if err != nil {
		t.Fatalf("LoadPRTracking: %v", err)
	}

	if got.PRNumber != 10 {
		t.Errorf("PRNumber = %d, want %d", got.PRNumber, 10)
	}
	if got.TrackID != "feat-login_2025" {
		t.Errorf("TrackID = %q, want %q", got.TrackID, "feat-login_2025")
	}
	if got.DeveloperAgentID != "dev-agent-1" {
		t.Errorf("DeveloperAgentID = %q, want %q", got.DeveloperAgentID, "dev-agent-1")
	}
	if got.ReviewerAgentID != "rev-agent-1" {
		t.Errorf("ReviewerAgentID = %q, want %q", got.ReviewerAgentID, "rev-agent-1")
	}
	if got.ReviewCycleCount != 2 {
		t.Errorf("ReviewCycleCount = %d, want %d", got.ReviewCycleCount, 2)
	}
	if got.MaxReviewCycles != 5 {
		t.Errorf("MaxReviewCycles = %d, want %d", got.MaxReviewCycles, 5)
	}
	if got.Status != "in-review" {
		t.Errorf("Status = %q, want %q", got.Status, "in-review")
	}
}

func TestPRTrackingStore_LoadNotFound(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewPRTrackingStore(db)

	_, err := store.LoadPRTracking("no-such-project")
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
	if !errors.Is(err, domain.ErrPRTrackingNotFound) {
		t.Errorf("expected ErrPRTrackingNotFound, got: %v", err)
	}
}

func TestPRTrackingStore_LoadReturnsLatestPR(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewPRTrackingStore(db)

	pr1 := &domain.PRTracking{
		PRNumber:    5,
		ProjectSlug: "myapp",
		TrackID:     "old-track",
		Status:      "merged",
	}
	pr2 := &domain.PRTracking{
		PRNumber:    15,
		ProjectSlug: "myapp",
		TrackID:     "new-track",
		Status:      "open",
	}

	store.SavePRTracking("myapp", pr1)
	store.SavePRTracking("myapp", pr2)

	got, err := store.LoadPRTracking("myapp")
	if err != nil {
		t.Fatalf("LoadPRTracking: %v", err)
	}
	if got.PRNumber != 15 {
		t.Errorf("expected latest PR (15), got %d", got.PRNumber)
	}
	if got.TrackID != "new-track" {
		t.Errorf("TrackID = %q, want %q", got.TrackID, "new-track")
	}
}

func TestPRTrackingStore_SaveOverwrites(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewPRTrackingStore(db)

	pr := &domain.PRTracking{
		PRNumber:    10,
		ProjectSlug: "myapp",
		TrackID:     "track-1",
		Status:      "open",
	}
	store.SavePRTracking("myapp", pr)

	// Overwrite with updated status.
	pr.Status = "merged"
	pr.ReviewCycleCount = 3
	if err := store.SavePRTracking("myapp", pr); err != nil {
		t.Fatalf("SavePRTracking (overwrite): %v", err)
	}

	got, _ := store.LoadPRTracking("myapp")
	if got.Status != "merged" {
		t.Errorf("Status = %q, want %q", got.Status, "merged")
	}
	if got.ReviewCycleCount != 3 {
		t.Errorf("ReviewCycleCount = %d, want %d", got.ReviewCycleCount, 3)
	}
}

func TestPRTrackingStore_IsolationByProject(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewPRTrackingStore(db)

	prA := &domain.PRTracking{PRNumber: 1, ProjectSlug: "alpha", TrackID: "a-track", Status: "open"}
	prB := &domain.PRTracking{PRNumber: 2, ProjectSlug: "beta", TrackID: "b-track", Status: "merged"}

	store.SavePRTracking("alpha", prA)
	store.SavePRTracking("beta", prB)

	gotA, _ := store.LoadPRTracking("alpha")
	gotB, _ := store.LoadPRTracking("beta")

	if gotA.TrackID != "a-track" {
		t.Errorf("alpha TrackID = %q, want %q", gotA.TrackID, "a-track")
	}
	if gotB.TrackID != "b-track" {
		t.Errorf("beta TrackID = %q, want %q", gotB.TrackID, "b-track")
	}
}
