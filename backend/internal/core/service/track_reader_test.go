package service_test

import (
	"os"
	"path/filepath"
	"testing"

	"kiloforge/internal/core/service"
	"kiloforge/pkg/kf"
)

// setupKFProject creates a minimal kf project directory with tracks.yaml,
// deps.yaml, and conflicts.yaml. Returns the project root dir.
func setupKFProject(t *testing.T, entries []kf.TrackEntry, deps kf.DepsGraph, conflicts []kf.ConflictPair) string {
	t.Helper()
	projectDir := t.TempDir()
	kfDir := filepath.Join(projectDir, ".agent", "kf")
	tracksDir := filepath.Join(kfDir, "tracks")
	os.MkdirAll(tracksDir, 0o755)

	if err := kf.WriteRegistryFile(filepath.Join(kfDir, "tracks.yaml"), entries); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	if deps != nil {
		if err := kf.WriteDepsFile(filepath.Join(tracksDir, "deps.yaml"), deps); err != nil {
			t.Fatalf("write deps: %v", err)
		}
	}
	if conflicts != nil {
		if err := kf.WriteConflictsFile(filepath.Join(tracksDir, "conflicts.yaml"), conflicts); err != nil {
			t.Fatalf("write conflicts: %v", err)
		}
	}
	return projectDir
}

func TestTrackReaderImpl_DiscoverTracks_WithDeps(t *testing.T) {
	t.Parallel()
	entries := []kf.TrackEntry{
		{ID: "track-a", Title: "Track A", Status: kf.StatusPending, Type: "feature"},
		{ID: "track-b", Title: "Track B", Status: kf.StatusCompleted, Type: "feature"},
		{ID: "track-c", Title: "Track C", Status: kf.StatusInProgress, Type: "bug"},
	}
	deps := kf.DepsGraph{
		"track-a": {"track-b", "track-c"},
		"track-c": {"track-b"},
	}
	conflicts := []kf.ConflictPair{
		kf.NewConflictPair("track-a", "track-c", "high", "overlapping files"),
	}
	projectDir := setupKFProject(t, entries, deps, conflicts)

	reader := service.NewTrackReader()
	tracks, err := reader.DiscoverTracks(projectDir)
	if err != nil {
		t.Fatalf("DiscoverTracks: %v", err)
	}
	if len(tracks) != 3 {
		t.Fatalf("expected 3 tracks, got %d", len(tracks))
	}

	// Build a map for easier lookup.
	byID := map[string]int{}
	for i, tr := range tracks {
		byID[tr.ID] = i
	}

	// track-a: 2 deps, 1 met (track-b is completed), 1 conflict
	a := tracks[byID["track-a"]]
	if a.DepsCount != 2 {
		t.Errorf("track-a DepsCount = %d, want 2", a.DepsCount)
	}
	if a.DepsMet != 1 {
		t.Errorf("track-a DepsMet = %d, want 1", a.DepsMet)
	}
	if a.ConflictCount != 1 {
		t.Errorf("track-a ConflictCount = %d, want 1", a.ConflictCount)
	}

	// track-b: 0 deps, 0 conflicts
	b := tracks[byID["track-b"]]
	if b.DepsCount != 0 {
		t.Errorf("track-b DepsCount = %d, want 0", b.DepsCount)
	}
	if b.ConflictCount != 0 {
		t.Errorf("track-b ConflictCount = %d, want 0", b.ConflictCount)
	}

	// track-c: 1 dep, 1 met, 1 conflict
	c := tracks[byID["track-c"]]
	if c.DepsCount != 1 {
		t.Errorf("track-c DepsCount = %d, want 1", c.DepsCount)
	}
	if c.DepsMet != 1 {
		t.Errorf("track-c DepsMet = %d, want 1", c.DepsMet)
	}
	if c.ConflictCount != 1 {
		t.Errorf("track-c ConflictCount = %d, want 1", c.ConflictCount)
	}
}

func TestTrackReaderImpl_DiscoverTracks_NoDepsFile(t *testing.T) {
	t.Parallel()
	entries := []kf.TrackEntry{
		{ID: "track-x", Title: "Track X", Status: kf.StatusPending, Type: "feature"},
	}
	projectDir := setupKFProject(t, entries, nil, nil)

	reader := service.NewTrackReader()
	tracks, err := reader.DiscoverTracks(projectDir)
	if err != nil {
		t.Fatalf("DiscoverTracks: %v", err)
	}
	if len(tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(tracks))
	}
	if tracks[0].DepsCount != 0 || tracks[0].DepsMet != 0 || tracks[0].ConflictCount != 0 {
		t.Errorf("expected zero counts without deps/conflicts files, got deps=%d met=%d conflicts=%d",
			tracks[0].DepsCount, tracks[0].DepsMet, tracks[0].ConflictCount)
	}
}

func TestTrackReaderImpl_GetTrackDetail_WithDepsAndConflicts(t *testing.T) {
	t.Parallel()
	entries := []kf.TrackEntry{
		{ID: "track-a", Title: "Track A", Status: kf.StatusPending, Type: "feature", Created: "2026-03-10", Updated: "2026-03-10"},
		{ID: "track-b", Title: "Track B", Status: kf.StatusCompleted, Type: "feature", Created: "2026-03-09", Updated: "2026-03-10"},
	}
	deps := kf.DepsGraph{
		"track-a": {"track-b"},
	}
	conflicts := []kf.ConflictPair{
		kf.NewConflictPair("track-a", "track-b", "medium", "shared config"),
	}
	projectDir := setupKFProject(t, entries, deps, conflicts)

	// Write track.yaml for track-a
	trackDir := filepath.Join(projectDir, ".agent", "kf", "tracks", "track-a")
	os.MkdirAll(trackDir, 0o755)
	trackYAML := `id: track-a
title: Track A
type: feature
status: pending
created: "2026-03-10"
updated: "2026-03-10"
spec:
  summary: "Test track"
plan:
  - phase: Setup
    tasks:
      - text: "Do something"
        done: false
`
	os.WriteFile(filepath.Join(trackDir, "track.yaml"), []byte(trackYAML), 0o644)

	reader := service.NewTrackReader()
	detail, err := reader.GetTrackDetail(projectDir, "track-a")
	if err != nil {
		t.Fatalf("GetTrackDetail: %v", err)
	}

	// Check dependencies
	if len(detail.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(detail.Dependencies))
	}
	dep := detail.Dependencies[0]
	if dep.ID != "track-b" {
		t.Errorf("dep ID = %q, want %q", dep.ID, "track-b")
	}
	if dep.Title != "Track B" {
		t.Errorf("dep Title = %q, want %q", dep.Title, "Track B")
	}
	if dep.Status != "complete" {
		t.Errorf("dep Status = %q, want %q", dep.Status, "complete")
	}

	// Check conflicts
	if len(detail.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(detail.Conflicts))
	}
	conf := detail.Conflicts[0]
	if conf.TrackID != "track-b" {
		t.Errorf("conflict TrackID = %q, want %q", conf.TrackID, "track-b")
	}
	if conf.TrackTitle != "Track B" {
		t.Errorf("conflict TrackTitle = %q, want %q", conf.TrackTitle, "Track B")
	}
	if conf.Risk != "medium" {
		t.Errorf("conflict Risk = %q, want %q", conf.Risk, "medium")
	}
	if conf.Note != "shared config" {
		t.Errorf("conflict Note = %q, want %q", conf.Note, "shared config")
	}
}

func TestTrackReaderImpl_GetTrackDetail_NoDeps(t *testing.T) {
	t.Parallel()
	entries := []kf.TrackEntry{
		{ID: "track-solo", Title: "Solo Track", Status: kf.StatusPending, Type: "chore", Created: "2026-03-10", Updated: "2026-03-10"},
	}
	projectDir := setupKFProject(t, entries, nil, nil)

	// Write track.yaml
	trackDir := filepath.Join(projectDir, ".agent", "kf", "tracks", "track-solo")
	os.MkdirAll(trackDir, 0o755)
	trackYAML := `id: track-solo
title: Solo Track
type: chore
status: pending
created: "2026-03-10"
updated: "2026-03-10"
spec:
  summary: "No deps"
plan: []
`
	os.WriteFile(filepath.Join(trackDir, "track.yaml"), []byte(trackYAML), 0o644)

	reader := service.NewTrackReader()
	detail, err := reader.GetTrackDetail(projectDir, "track-solo")
	if err != nil {
		t.Fatalf("GetTrackDetail: %v", err)
	}

	if len(detail.Dependencies) != 0 {
		t.Errorf("expected 0 dependencies, got %d", len(detail.Dependencies))
	}
	if len(detail.Conflicts) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(detail.Conflicts))
	}
}
