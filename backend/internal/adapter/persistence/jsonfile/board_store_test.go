package jsonfile_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"crelay/internal/adapter/persistence/jsonfile"
	"crelay/internal/core/domain"
)

func TestBoardStore_SaveAndLoadConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "projects", "myapp")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	store := jsonfile.NewBoardStore(dir)

	cfg := &domain.BoardConfig{
		ProjectBoardID: 42,
		Columns: map[string]int{
			"suggested": 1,
			"approved":  2,
			"completed": 5,
		},
		Labels: map[string]int{
			"type:feature": 10,
		},
	}

	if err := store.SaveBoardConfig("myapp", cfg); err != nil {
		t.Fatalf("SaveBoardConfig: %v", err)
	}

	got, err := store.GetBoardConfig("myapp")
	if err != nil {
		t.Fatalf("GetBoardConfig: %v", err)
	}
	if got.ProjectBoardID != 42 {
		t.Errorf("ProjectBoardID: want 42, got %d", got.ProjectBoardID)
	}
	if got.Columns["suggested"] != 1 {
		t.Errorf("Columns[suggested]: want 1, got %d", got.Columns["suggested"])
	}
	if got.Labels["type:feature"] != 10 {
		t.Errorf("Labels[type:feature]: want 10, got %d", got.Labels["type:feature"])
	}
}

func TestBoardStore_GetBoardConfig_Missing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	store := jsonfile.NewBoardStore(dir)
	cfg, err := store.GetBoardConfig("nonexistent")
	if err != nil {
		t.Fatalf("expected nil error for missing, got: %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config for missing project, got %+v", cfg)
	}
}

func TestBoardStore_SaveAndLoadTrackIssue(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "projects", "myapp")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	store := jsonfile.NewBoardStore(dir)
	now := time.Now().Truncate(time.Second)

	ti := domain.TrackIssue{
		TrackID:     "impl-foo_20260308Z",
		IssueNumber: 42,
		CardID:      15,
		Column:      "suggested",
		LastSynced:  now,
	}

	if err := store.SaveTrackIssue("myapp", ti); err != nil {
		t.Fatalf("SaveTrackIssue: %v", err)
	}

	got, err := store.GetTrackIssue("myapp", "impl-foo_20260308Z")
	if err != nil {
		t.Fatalf("GetTrackIssue: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil TrackIssue")
	}
	if got.IssueNumber != 42 {
		t.Errorf("IssueNumber: want 42, got %d", got.IssueNumber)
	}
	if got.CardID != 15 {
		t.Errorf("CardID: want 15, got %d", got.CardID)
	}
}

func TestBoardStore_GetTrackIssue_Missing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "projects", "myapp")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	store := jsonfile.NewBoardStore(dir)
	got, err := store.GetTrackIssue("myapp", "nonexistent")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing track issue")
	}
}

func TestBoardStore_ListTrackIssues(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "projects", "myapp")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	store := jsonfile.NewBoardStore(dir)
	now := time.Now().Truncate(time.Second)

	for _, id := range []string{"track-a", "track-b"} {
		if err := store.SaveTrackIssue("myapp", domain.TrackIssue{
			TrackID:    id,
			Column:     "suggested",
			LastSynced: now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	list, err := store.ListTrackIssues("myapp")
	if err != nil {
		t.Fatalf("ListTrackIssues: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 track issues, got %d", len(list))
	}
}

func TestBoardStore_ListTrackIssues_NoFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	store := jsonfile.NewBoardStore(dir)
	list, err := store.ListTrackIssues("nonexistent")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d", len(list))
	}
}

func TestBoardStore_CorruptFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "projects", "myapp")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write corrupt JSON
	if err := os.WriteFile(filepath.Join(projectDir, "board.json"), []byte("{invalid"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := jsonfile.NewBoardStore(dir)
	_, err := store.GetBoardConfig("myapp")
	if err == nil {
		t.Error("expected error for corrupt file")
	}
}
