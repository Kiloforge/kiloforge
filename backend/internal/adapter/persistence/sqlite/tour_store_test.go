package sqlite

import (
	"testing"
	"time"
)

func TestTourStore_DefaultPending(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewTourStore(db)

	state, err := store.GetTourState()
	if err != nil {
		t.Fatalf("GetTourState: %v", err)
	}
	if state.Status != "pending" {
		t.Fatalf("expected pending, got %q", state.Status)
	}
	if state.CurrentStep != 0 {
		t.Fatalf("expected step 0, got %d", state.CurrentStep)
	}
}

func TestTourStore_UpdateTransitions(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewTourStore(db)

	now := time.Now().UTC().Truncate(time.Second)

	// Accept tour.
	if err := store.UpdateTourState(TourState{
		Status:      "active",
		CurrentStep: 0,
		StartedAt:   &now,
	}); err != nil {
		t.Fatalf("UpdateTourState active: %v", err)
	}

	state, err := store.GetTourState()
	if err != nil {
		t.Fatalf("GetTourState: %v", err)
	}
	if state.Status != "active" {
		t.Fatalf("expected active, got %q", state.Status)
	}
	if state.StartedAt == nil || state.StartedAt.Truncate(time.Second) != now {
		t.Fatalf("unexpected StartedAt: %v", state.StartedAt)
	}

	// Advance step.
	state.CurrentStep = 2
	if err := store.UpdateTourState(state); err != nil {
		t.Fatalf("UpdateTourState step: %v", err)
	}
	state, _ = store.GetTourState()
	if state.CurrentStep != 2 {
		t.Fatalf("expected step 2, got %d", state.CurrentStep)
	}

	// Complete.
	completed := time.Now().UTC().Truncate(time.Second)
	state.Status = "completed"
	state.CompletedAt = &completed
	if err := store.UpdateTourState(state); err != nil {
		t.Fatalf("UpdateTourState complete: %v", err)
	}
	state, _ = store.GetTourState()
	if state.Status != "completed" {
		t.Fatalf("expected completed, got %q", state.Status)
	}
	if state.CompletedAt == nil {
		t.Fatal("expected non-nil CompletedAt")
	}
}

func TestTourStore_Dismiss(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewTourStore(db)

	now := time.Now().UTC()
	if err := store.UpdateTourState(TourState{
		Status:      "dismissed",
		DismissedAt: &now,
	}); err != nil {
		t.Fatalf("UpdateTourState: %v", err)
	}

	state, _ := store.GetTourState()
	if state.Status != "dismissed" {
		t.Fatalf("expected dismissed, got %q", state.Status)
	}
	if state.DismissedAt == nil {
		t.Fatal("expected non-nil DismissedAt")
	}
}
