package sqlite

import (
	"testing"
	"time"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/core/domain"
)

func TestBoardStore_SaveAndGet(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewBoardStore(db)

	board := &domain.BoardState{
		Columns: domain.BoardColumns,
		Cards: map[string]domain.BoardCard{
			"track-1": {
				TrackID:   "track-1",
				Title:     "Test Track",
				Type:      "feature",
				Column:    "backlog",
				Position:  0,
				MovedAt:   time.Now().Truncate(time.Second),
				CreatedAt: time.Now().Truncate(time.Second),
			},
		},
	}

	if err := store.SaveBoard("proj1", board); err != nil {
		t.Fatalf("SaveBoard: %v", err)
	}

	got, err := store.GetBoard("proj1")
	if err != nil {
		t.Fatalf("GetBoard: %v", err)
	}
	if got == nil {
		t.Fatal("GetBoard: nil")
	}
	if len(got.Cards) != 1 {
		t.Errorf("Cards: want 1, got %d", len(got.Cards))
	}
	card := got.Cards["track-1"]
	if card.Title != "Test Track" {
		t.Errorf("Title: want 'Test Track', got %q", card.Title)
	}
}

func TestBoardStore_GetNonExistent(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewBoardStore(db)

	got, err := store.GetBoard("nonexistent")
	if err != nil {
		t.Fatalf("GetBoard: %v", err)
	}
	if got != nil {
		t.Error("GetBoard: want nil for nonexistent project")
	}
}

func TestPRTrackingStore_SaveAndLoad(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewPRTrackingStore(db)

	tracking := &domain.PRTracking{
		PRNumber:         42,
		ProjectSlug:      "myproj",
		TrackID:          "feature/my-track",
		DeveloperAgentID: "dev-1",
		Status:           "in-review",
		MaxReviewCycles:  3,
	}

	if err := store.SavePRTracking("myproj", tracking); err != nil {
		t.Fatalf("SavePRTracking: %v", err)
	}

	got, err := store.LoadPRTracking("myproj")
	if err != nil {
		t.Fatalf("LoadPRTracking: %v", err)
	}
	if got.PRNumber != 42 {
		t.Errorf("PRNumber: want 42, got %d", got.PRNumber)
	}
	if got.Status != "in-review" {
		t.Errorf("Status: want in-review, got %q", got.Status)
	}
}

func TestQuotaStore_RecordAndGet(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQuotaStore(db)

	store.RecordUsage(&agent.AgentUsage{
		AgentID:      "agent-1",
		TotalCostUSD: 0.05,
		InputTokens:  1000,
		OutputTokens: 500,
	})

	got := store.GetAgentUsage("agent-1")
	if got == nil {
		t.Fatal("GetAgentUsage: nil")
	}
	if got.TotalCostUSD != 0.05 {
		t.Errorf("TotalCostUSD: want 0.05, got %f", got.TotalCostUSD)
	}

	total := store.GetTotalUsage()
	if total.AgentCount != 1 {
		t.Errorf("AgentCount: want 1, got %d", total.AgentCount)
	}
	if total.InputTokens != 1000 {
		t.Errorf("InputTokens: want 1000, got %d", total.InputTokens)
	}
}
