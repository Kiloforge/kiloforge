package jsonfile_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"kiloforge/internal/adapter/persistence/jsonfile"
	"kiloforge/internal/core/domain"
)

func TestBoardStore_SaveAndLoad(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "projects", "myapp")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	store := jsonfile.NewBoardStore(dir)
	now := time.Now().Truncate(time.Second)

	board := domain.NewBoardState()
	board.Cards["track-1"] = domain.BoardCard{
		TrackID:   "track-1",
		Title:     "Test Track",
		Type:      "feature",
		Column:    domain.ColumnBacklog,
		Position:  0,
		MovedAt:   now,
		CreatedAt: now,
	}

	if err := store.SaveBoard("myapp", board); err != nil {
		t.Fatalf("SaveBoard: %v", err)
	}

	got, err := store.GetBoard("myapp")
	if err != nil {
		t.Fatalf("GetBoard: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil board")
	}
	if len(got.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(got.Cards))
	}
	card := got.Cards["track-1"]
	if card.Title != "Test Track" {
		t.Errorf("Title: want %q, got %q", "Test Track", card.Title)
	}
	if card.Column != domain.ColumnBacklog {
		t.Errorf("Column: want %q, got %q", domain.ColumnBacklog, card.Column)
	}
}

func TestBoardStore_GetBoard_Missing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	store := jsonfile.NewBoardStore(dir)
	board, err := store.GetBoard("nonexistent")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if board != nil {
		t.Errorf("expected nil board for missing project")
	}
}

func TestBoardStore_CorruptFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "projects", "myapp")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(projectDir, "board.json"), []byte("{invalid"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := jsonfile.NewBoardStore(dir)
	_, err := store.GetBoard("myapp")
	if err == nil {
		t.Error("expected error for corrupt file")
	}
}
