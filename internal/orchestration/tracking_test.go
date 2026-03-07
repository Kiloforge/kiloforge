package orchestration

import (
	"path/filepath"
	"testing"

	"crelay/internal/core/domain"
)

func TestPRTracking_SaveLoad_RoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	original := &domain.PRTracking{
		PRNumber:         42,
		TrackID:          "test-track_20260101Z",
		ProjectSlug:      "myproject",
		DeveloperAgentID: "dev-agent-123",
		DeveloperSession: "dev-session-456",
		ReviewerAgentID:  "rev-agent-789",
		ReviewerSession:  "rev-session-012",
		ReviewCycleCount: 2,
		MaxReviewCycles:  5,
		Status:           "review",
	}

	if err := SavePRTracking(original, dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadPRTracking(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.PRNumber != original.PRNumber {
		t.Errorf("PRNumber: want %d, got %d", original.PRNumber, loaded.PRNumber)
	}
	if loaded.TrackID != original.TrackID {
		t.Errorf("TrackID: want %q, got %q", original.TrackID, loaded.TrackID)
	}
	if loaded.DeveloperAgentID != original.DeveloperAgentID {
		t.Errorf("DeveloperAgentID: want %q, got %q", original.DeveloperAgentID, loaded.DeveloperAgentID)
	}
	if loaded.ReviewCycleCount != original.ReviewCycleCount {
		t.Errorf("ReviewCycleCount: want %d, got %d", original.ReviewCycleCount, loaded.ReviewCycleCount)
	}
	if loaded.Status != original.Status {
		t.Errorf("Status: want %q, got %q", original.Status, loaded.Status)
	}
}

func TestPRTracking_LoadMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, err := LoadPRTracking(dir)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestPRTrackingPath(t *testing.T) {
	t.Parallel()

	path := PRTrackingPath("/data", "myproject")
	expected := filepath.Join("/data", "projects", "myproject", "pr-tracking.json")
	if path != expected {
		t.Errorf("want %q, got %q", expected, path)
	}
}
