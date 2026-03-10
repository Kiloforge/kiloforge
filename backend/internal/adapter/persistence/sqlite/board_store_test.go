package sqlite

import (
	"testing"
	"time"

	"kiloforge/internal/core/domain"
)

func TestBoardStore_SaveAndGetBoard_AllFields(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewBoardStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	board := &domain.BoardState{
		Columns: domain.BoardColumns,
		Cards: map[string]domain.BoardCard{
			"track-1": {
				TrackID:        "track-1",
				Title:          "First Track",
				Type:           "feature",
				Column:         "in-progress",
				Position:       0,
				AgentID:        "agent-abc",
				AgentStatus:    "running",
				AssignedWorker: "worker-1",
				PRNumber:       42,
				TraceID:        "trace-xyz",
				MovedAt:        now,
				CreatedAt:      now,
			},
			"track-2": {
				TrackID:   "track-2",
				Title:     "Second Track",
				Type:      "bug",
				Column:    "pending",
				Position:  1,
				CreatedAt: now,
			},
		},
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
	if len(got.Cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(got.Cards))
	}

	card1 := got.Cards["track-1"]
	if card1.Title != "First Track" {
		t.Errorf("Title = %q, want %q", card1.Title, "First Track")
	}
	if card1.Column != "in-progress" {
		t.Errorf("Column = %q, want %q", card1.Column, "in-progress")
	}
	if card1.AgentID != "agent-abc" {
		t.Errorf("AgentID = %q, want %q", card1.AgentID, "agent-abc")
	}
	if card1.PRNumber != 42 {
		t.Errorf("PRNumber = %d, want %d", card1.PRNumber, 42)
	}
	if card1.TraceID != "trace-xyz" {
		t.Errorf("TraceID = %q, want %q", card1.TraceID, "trace-xyz")
	}

	card2 := got.Cards["track-2"]
	if card2.Type != "bug" {
		t.Errorf("Type = %q, want %q", card2.Type, "bug")
	}
}

func TestBoardStore_SaveBoard_ReplacesExisting(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewBoardStore(db)

	now := time.Now().UTC().Truncate(time.Second)

	// Save initial board with 2 cards.
	board1 := &domain.BoardState{
		Columns: domain.BoardColumns,
		Cards: map[string]domain.BoardCard{
			"track-a": {TrackID: "track-a", Title: "A", Column: "pending", CreatedAt: now},
			"track-b": {TrackID: "track-b", Title: "B", Column: "pending", CreatedAt: now},
		},
	}
	if err := store.SaveBoard("proj", board1); err != nil {
		t.Fatalf("SaveBoard(1): %v", err)
	}

	// Save replacement board with 1 card.
	board2 := &domain.BoardState{
		Columns: domain.BoardColumns,
		Cards: map[string]domain.BoardCard{
			"track-c": {TrackID: "track-c", Title: "C", Column: "done", CreatedAt: now},
		},
	}
	if err := store.SaveBoard("proj", board2); err != nil {
		t.Fatalf("SaveBoard(2): %v", err)
	}

	got, err := store.GetBoard("proj")
	if err != nil {
		t.Fatalf("GetBoard: %v", err)
	}
	if len(got.Cards) != 1 {
		t.Fatalf("expected 1 card after replace, got %d", len(got.Cards))
	}
	if _, ok := got.Cards["track-c"]; !ok {
		t.Error("expected track-c in board after replace")
	}
}

func TestBoardStore_IsolationBySlug(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewBoardStore(db)

	now := time.Now().UTC().Truncate(time.Second)

	boardA := &domain.BoardState{
		Columns: domain.BoardColumns,
		Cards: map[string]domain.BoardCard{
			"a-1": {TrackID: "a-1", Title: "A1", Column: "pending", CreatedAt: now},
		},
	}
	boardB := &domain.BoardState{
		Columns: domain.BoardColumns,
		Cards: map[string]domain.BoardCard{
			"b-1": {TrackID: "b-1", Title: "B1", Column: "done", CreatedAt: now},
			"b-2": {TrackID: "b-2", Title: "B2", Column: "done", CreatedAt: now},
		},
	}

	if err := store.SaveBoard("alpha", boardA); err != nil {
		t.Fatalf("SaveBoard(alpha): %v", err)
	}
	if err := store.SaveBoard("beta", boardB); err != nil {
		t.Fatalf("SaveBoard(beta): %v", err)
	}

	gotA, _ := store.GetBoard("alpha")
	gotB, _ := store.GetBoard("beta")

	if len(gotA.Cards) != 1 {
		t.Errorf("alpha: expected 1 card, got %d", len(gotA.Cards))
	}
	if len(gotB.Cards) != 2 {
		t.Errorf("beta: expected 2 cards, got %d", len(gotB.Cards))
	}
}
