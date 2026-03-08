package cli

import (
	"testing"
	"time"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/persistence/jsonfile"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/service"
)

func TestRunDryRun_MovesCardToDone(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	cfg := &config.Config{DataDir: dataDir}

	slug := "test-proj"
	trackID := "test-track_123Z"

	// Set up a board with a card in backlog.
	boardStore := jsonfile.NewBoardStore(dataDir)
	board := domain.NewBoardState()
	board.Cards[trackID] = domain.BoardCard{
		TrackID:   trackID,
		Title:     "Test Track",
		Column:    domain.ColumnBacklog,
		CreatedAt: time.Now(),
		MovedAt:   time.Now(),
	}
	if err := boardStore.SaveBoard(slug, board); err != nil {
		t.Fatalf("save board: %v", err)
	}

	proj := domain.Project{Slug: slug, ProjectDir: t.TempDir()}

	// Run dry-run.
	if err := runDryRun(cfg, proj, trackID); err != nil {
		t.Fatalf("runDryRun: %v", err)
	}

	// Verify card moved to Done.
	svc := service.NewNativeBoardService(boardStore)
	updatedBoard, err := svc.GetBoard(slug)
	if err != nil {
		t.Fatalf("get board: %v", err)
	}

	card, ok := updatedBoard.Cards[trackID]
	if !ok {
		t.Fatal("card not found on board after dry-run")
	}
	if card.Column != domain.ColumnDone {
		t.Errorf("card column = %q, want %q", card.Column, domain.ColumnDone)
	}
}

func TestRunDryRun_NoBoard(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	cfg := &config.Config{DataDir: dataDir}

	proj := domain.Project{Slug: "no-board-proj", ProjectDir: t.TempDir()}

	// Should not error even if track is not on the board.
	if err := runDryRun(cfg, proj, "nonexistent-track"); err != nil {
		t.Fatalf("runDryRun should not error for missing board card: %v", err)
	}
}
