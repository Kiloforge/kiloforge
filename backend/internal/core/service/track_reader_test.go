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

func TestTrackReaderImpl_GetTrackDetail_WithAgentRegister(t *testing.T) {
	t.Parallel()
	entries := []kf.TrackEntry{
		{ID: "track-reg", Title: "Register Track", Status: kf.StatusInProgress, Type: "feature", Created: "2026-03-12", Updated: "2026-03-12"},
	}
	projectDir := setupKFProject(t, entries, nil, nil)

	trackDir := filepath.Join(projectDir, ".agent", "kf", "tracks", "track-reg")
	os.MkdirAll(trackDir, 0o755)
	trackYAML := `id: track-reg
title: Register Track
type: feature
status: in-progress
created: "2026-03-12"
updated: "2026-03-12"
spec:
  summary: "Test register"
plan: []
extra:
  created_by:
    agent_id: "arch-1"
    role: "architect"
    session_id: "sess-abc"
    created_at: "2026-03-12T14:00:00Z"
  claim:
    agent_id: "dev-1"
    role: "developer"
    session_id: "sess-xyz"
    worktree: "worker-3"
    branch: "feature/track-reg"
    model: "claude-opus-4-6"
    claimed_at: "2026-03-12T15:00:00Z"
`
	os.WriteFile(filepath.Join(trackDir, "track.yaml"), []byte(trackYAML), 0o644)

	reader := service.NewTrackReader()
	detail, err := reader.GetTrackDetail(projectDir, "track-reg")
	if err != nil {
		t.Fatalf("GetTrackDetail: %v", err)
	}

	if detail.AgentRegister == nil {
		t.Fatal("expected agent register")
	}
	if detail.AgentRegister.CreatedBy == nil {
		t.Fatal("expected created_by")
	}
	if detail.AgentRegister.CreatedBy.AgentID != "arch-1" {
		t.Errorf("created_by.agent_id = %q, want arch-1", detail.AgentRegister.CreatedBy.AgentID)
	}
	if detail.AgentRegister.CreatedBy.Role != "architect" {
		t.Errorf("created_by.role = %q, want architect", detail.AgentRegister.CreatedBy.Role)
	}
	if detail.AgentRegister.CreatedBy.Timestamp != "2026-03-12T14:00:00Z" {
		t.Errorf("created_by.timestamp = %q", detail.AgentRegister.CreatedBy.Timestamp)
	}
	if detail.AgentRegister.ClaimedBy == nil {
		t.Fatal("expected claimed_by")
	}
	if detail.AgentRegister.ClaimedBy.AgentID != "dev-1" {
		t.Errorf("claimed_by.agent_id = %q, want dev-1", detail.AgentRegister.ClaimedBy.AgentID)
	}
	if detail.AgentRegister.ClaimedBy.Worktree != "worker-3" {
		t.Errorf("claimed_by.worktree = %q, want worker-3", detail.AgentRegister.ClaimedBy.Worktree)
	}
	if detail.AgentRegister.ClaimedBy.Branch != "feature/track-reg" {
		t.Errorf("claimed_by.branch = %q", detail.AgentRegister.ClaimedBy.Branch)
	}
	if detail.AgentRegister.ClaimedBy.Model != "claude-opus-4-6" {
		t.Errorf("claimed_by.model = %q", detail.AgentRegister.ClaimedBy.Model)
	}
}

func TestTrackReaderImpl_GetTrackDetail_EmptyExtra(t *testing.T) {
	t.Parallel()
	entries := []kf.TrackEntry{
		{ID: "track-no-extra", Title: "No Extra", Status: kf.StatusPending, Type: "chore", Created: "2026-03-12", Updated: "2026-03-12"},
	}
	projectDir := setupKFProject(t, entries, nil, nil)

	trackDir := filepath.Join(projectDir, ".agent", "kf", "tracks", "track-no-extra")
	os.MkdirAll(trackDir, 0o755)
	trackYAML := `id: track-no-extra
title: No Extra
type: chore
status: pending
created: "2026-03-12"
updated: "2026-03-12"
spec:
  summary: "Empty extra"
plan: []
extra: {}
`
	os.WriteFile(filepath.Join(trackDir, "track.yaml"), []byte(trackYAML), 0o644)

	reader := service.NewTrackReader()
	detail, err := reader.GetTrackDetail(projectDir, "track-no-extra")
	if err != nil {
		t.Fatalf("GetTrackDetail: %v", err)
	}

	if detail.AgentRegister != nil {
		t.Error("expected nil agent register for empty extra")
	}
}
