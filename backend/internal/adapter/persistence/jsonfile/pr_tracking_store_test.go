package jsonfile

import (
	"os"
	"path/filepath"
	"testing"

	"kiloforge/internal/core/domain"
)

func TestPRTracking_SaveAndLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tracking := &domain.PRTracking{
		PRNumber:         5,
		TrackID:          "my-track",
		ProjectSlug:      "myapp",
		DeveloperAgentID: "dev-1",
		DeveloperSession: "sess-1",
		ReviewCycleCount: 2,
		MaxReviewCycles:  3,
		Status:           "in-review",
	}

	if err := SavePRTracking(tracking, dir); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := LoadPRTracking(dir)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if loaded.PRNumber != 5 {
		t.Errorf("PRNumber = %d, want 5", loaded.PRNumber)
	}
	if loaded.TrackID != "my-track" {
		t.Errorf("TrackID = %q, want %q", loaded.TrackID, "my-track")
	}
	if loaded.ReviewCycleCount != 2 {
		t.Errorf("ReviewCycleCount = %d, want 2", loaded.ReviewCycleCount)
	}
	if loaded.Status != "in-review" {
		t.Errorf("Status = %q, want %q", loaded.Status, "in-review")
	}
}

func TestPRTracking_LoadNotFound(t *testing.T) {
	t.Parallel()

	_, err := LoadPRTracking(t.TempDir())
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestPRTracking_CorruptJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, prTrackingFile), []byte("not json"), 0o644)

	_, err := LoadPRTracking(dir)
	if err == nil {
		t.Error("expected error for corrupt JSON")
	}
}

func TestPRTrackingPath(t *testing.T) {
	t.Parallel()

	path := PRTrackingPath("/data", "myapp")
	expected := filepath.Join("/data", "projects", "myapp", prTrackingFile)
	if path != expected {
		t.Errorf("path = %q, want %q", path, expected)
	}
}

func TestPRTracking_Overwrite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	first := &domain.PRTracking{PRNumber: 1, Status: "waiting-review"}
	SavePRTracking(first, dir)

	second := &domain.PRTracking{PRNumber: 2, Status: "merged"}
	SavePRTracking(second, dir)

	loaded, err := LoadPRTracking(dir)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if loaded.PRNumber != 2 {
		t.Errorf("PRNumber = %d, want 2 (overwritten)", loaded.PRNumber)
	}
}
