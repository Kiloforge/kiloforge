package service_test

import (
	"testing"
	"time"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
	"kiloforge/internal/core/service"
)

// mockNativeBoardStore implements service.NativeBoardStore for testing.
type mockNativeBoardStore struct {
	boards map[string]*domain.BoardState
}

func newMockNativeBoardStore() *mockNativeBoardStore {
	return &mockNativeBoardStore{boards: make(map[string]*domain.BoardState)}
}

func (m *mockNativeBoardStore) GetBoard(slug string) (*domain.BoardState, error) {
	return m.boards[slug], nil
}

func (m *mockNativeBoardStore) SaveBoard(slug string, board *domain.BoardState) error {
	m.boards[slug] = board
	return nil
}

func TestNativeBoardService_GetBoard_Empty(t *testing.T) {
	t.Parallel()
	store := newMockNativeBoardStore()
	svc := service.NewNativeBoardService(store)

	board, err := svc.GetBoard("myapp")
	if err != nil {
		t.Fatal(err)
	}
	if board == nil {
		t.Fatal("expected non-nil board")
	}
	if len(board.Columns) != 4 {
		t.Errorf("expected 4 columns, got %d", len(board.Columns))
	}
	if len(board.Cards) != 0 {
		t.Errorf("expected 0 cards, got %d", len(board.Cards))
	}
}

func TestNativeBoardService_MoveCard(t *testing.T) {
	t.Parallel()
	store := newMockNativeBoardStore()
	now := time.Now().Truncate(time.Second)
	store.boards["myapp"] = &domain.BoardState{
		Columns: domain.BoardColumns,
		Cards: map[string]domain.BoardCard{
			"track-1": {TrackID: "track-1", Column: domain.ColumnBacklog, MovedAt: now},
		},
	}
	svc := service.NewNativeBoardService(store)

	result, err := svc.MoveCard("myapp", "track-1", domain.ColumnApproved)
	if err != nil {
		t.Fatal(err)
	}
	if result.FromColumn != domain.ColumnBacklog {
		t.Errorf("FromColumn: want %q, got %q", domain.ColumnBacklog, result.FromColumn)
	}
	if result.ToColumn != domain.ColumnApproved {
		t.Errorf("ToColumn: want %q, got %q", domain.ColumnApproved, result.ToColumn)
	}

	// Verify persisted.
	board := store.boards["myapp"]
	if board.Cards["track-1"].Column != domain.ColumnApproved {
		t.Errorf("stored column: want %q, got %q", domain.ColumnApproved, board.Cards["track-1"].Column)
	}
}

func TestNativeBoardService_MoveCard_InvalidColumn(t *testing.T) {
	t.Parallel()
	store := newMockNativeBoardStore()
	store.boards["myapp"] = &domain.BoardState{
		Columns: domain.BoardColumns,
		Cards: map[string]domain.BoardCard{
			"track-1": {TrackID: "track-1", Column: domain.ColumnBacklog},
		},
	}
	svc := service.NewNativeBoardService(store)

	_, err := svc.MoveCard("myapp", "track-1", "invalid_column")
	if err == nil {
		t.Error("expected error for invalid column")
	}
}

func TestNativeBoardService_MoveCard_TrackNotFound(t *testing.T) {
	t.Parallel()
	store := newMockNativeBoardStore()
	store.boards["myapp"] = domain.NewBoardState()
	svc := service.NewNativeBoardService(store)

	_, err := svc.MoveCard("myapp", "nonexistent", domain.ColumnApproved)
	if err == nil {
		t.Error("expected error for missing track")
	}
}

func TestNativeBoardService_MoveCard_SameColumn(t *testing.T) {
	t.Parallel()
	store := newMockNativeBoardStore()
	store.boards["myapp"] = &domain.BoardState{
		Columns: domain.BoardColumns,
		Cards: map[string]domain.BoardCard{
			"track-1": {TrackID: "track-1", Column: domain.ColumnApproved},
		},
	}
	svc := service.NewNativeBoardService(store)

	result, err := svc.MoveCard("myapp", "track-1", domain.ColumnApproved)
	if err != nil {
		t.Fatal(err)
	}
	if result.FromColumn != result.ToColumn {
		t.Errorf("expected same column, got from=%q to=%q", result.FromColumn, result.ToColumn)
	}
}

func TestNativeBoardService_SyncFromTracks(t *testing.T) {
	t.Parallel()
	store := newMockNativeBoardStore()
	svc := service.NewNativeBoardService(store)

	tracks := []port.TrackEntry{
		{ID: "track-1", Title: "New Track", Status: service.StatusPending},
		{ID: "track-2", Title: "In Progress", Status: service.StatusInProgress},
		{ID: "track-3", Title: "Done", Status: service.StatusComplete},
	}

	result, err := svc.SyncFromTracks("myapp", tracks, map[string]string{
		"track-1": "feature",
		"track-2": "bug",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Created != 3 {
		t.Errorf("Created: want 3, got %d", result.Created)
	}

	board := store.boards["myapp"]
	if board.Cards["track-1"].Column != domain.ColumnBacklog {
		t.Errorf("track-1 column: want %q, got %q", domain.ColumnBacklog, board.Cards["track-1"].Column)
	}
	if board.Cards["track-2"].Column != domain.ColumnInProgress {
		t.Errorf("track-2 column: want %q, got %q", domain.ColumnInProgress, board.Cards["track-2"].Column)
	}
	if board.Cards["track-3"].Column != domain.ColumnDone {
		t.Errorf("track-3 column: want %q, got %q", domain.ColumnDone, board.Cards["track-3"].Column)
	}
	if board.Cards["track-1"].Type != "feature" {
		t.Errorf("track-1 type: want %q, got %q", "feature", board.Cards["track-1"].Type)
	}
}

func TestNativeBoardService_SyncFromTracks_UpdateExisting(t *testing.T) {
	t.Parallel()
	store := newMockNativeBoardStore()
	now := time.Now().Truncate(time.Second)
	store.boards["myapp"] = &domain.BoardState{
		Columns: domain.BoardColumns,
		Cards: map[string]domain.BoardCard{
			"track-1": {TrackID: "track-1", Title: "Old Title", Column: domain.ColumnBacklog, MovedAt: now},
		},
	}
	svc := service.NewNativeBoardService(store)

	tracks := []port.TrackEntry{
		{ID: "track-1", Title: "Old Title", Status: service.StatusInProgress},
		{ID: "track-2", Title: "New", Status: service.StatusPending},
	}

	result, err := svc.SyncFromTracks("myapp", tracks, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Created != 1 {
		t.Errorf("Created: want 1, got %d", result.Created)
	}
	if result.Updated != 1 {
		t.Errorf("Updated: want 1, got %d", result.Updated)
	}

	board := store.boards["myapp"]
	if board.Cards["track-1"].Column != domain.ColumnInProgress {
		t.Errorf("track-1 column after sync: want %q, got %q", domain.ColumnInProgress, board.Cards["track-1"].Column)
	}
}

func TestNativeBoardService_StoreAndGetTraceID(t *testing.T) {
	t.Parallel()
	store := newMockNativeBoardStore()
	now := time.Now().Truncate(time.Second)
	store.boards["myapp"] = &domain.BoardState{
		Columns: domain.BoardColumns,
		Cards: map[string]domain.BoardCard{
			"track-1": {TrackID: "track-1", Column: domain.ColumnInProgress, MovedAt: now},
		},
	}
	svc := service.NewNativeBoardService(store)

	// Initially no trace ID.
	_, ok := svc.GetTraceID("myapp", "track-1")
	if ok {
		t.Error("expected no trace ID initially")
	}

	// Store a trace ID.
	if err := svc.StoreTraceID("myapp", "track-1", "abc123"); err != nil {
		t.Fatal(err)
	}

	// Retrieve it.
	traceID, ok := svc.GetTraceID("myapp", "track-1")
	if !ok {
		t.Fatal("expected trace ID to be stored")
	}
	if traceID != "abc123" {
		t.Errorf("trace ID: want %q, got %q", "abc123", traceID)
	}

	// Non-existent track returns false.
	_, ok = svc.GetTraceID("myapp", "nonexistent")
	if ok {
		t.Error("expected false for nonexistent track")
	}
}

func TestDomainBoard_IsValidColumn(t *testing.T) {
	t.Parallel()
	if !domain.IsValidColumn("backlog") {
		t.Error("backlog should be valid")
	}
	if domain.IsValidColumn("invalid") {
		t.Error("invalid should not be valid")
	}
}

func TestDomainBoard_ClampForwardMove(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		from    string
		to      string
		want    string
	}{
		// Forward moves that should be clamped
		{"backlog→approved (1 step, allowed)", domain.ColumnBacklog, domain.ColumnApproved, domain.ColumnApproved},
		{"backlog→in_progress (2 steps, clamp to approved)", domain.ColumnBacklog, domain.ColumnInProgress, domain.ColumnApproved},
		{"backlog→done (3 steps, clamp to approved)", domain.ColumnBacklog, domain.ColumnDone, domain.ColumnApproved},
		{"approved→in_progress (clamp: beyond approved)", domain.ColumnApproved, domain.ColumnInProgress, domain.ColumnApproved},
		{"approved→done (clamp: beyond approved)", domain.ColumnApproved, domain.ColumnDone, domain.ColumnApproved},
		{"in_progress→done (clamp: beyond approved)", domain.ColumnInProgress, domain.ColumnDone, domain.ColumnInProgress},

		// Backward moves — unrestricted (pass-through)
		{"approved→backlog (backward)", domain.ColumnApproved, domain.ColumnBacklog, domain.ColumnBacklog},
		{"in_progress→backlog (backward)", domain.ColumnInProgress, domain.ColumnBacklog, domain.ColumnBacklog},
		{"in_progress→approved (backward)", domain.ColumnInProgress, domain.ColumnApproved, domain.ColumnApproved},
		{"done→backlog (backward)", domain.ColumnDone, domain.ColumnBacklog, domain.ColumnBacklog},
		{"done→approved (backward)", domain.ColumnDone, domain.ColumnApproved, domain.ColumnApproved},

		// Same column — pass-through
		{"backlog→backlog (same)", domain.ColumnBacklog, domain.ColumnBacklog, domain.ColumnBacklog},
		{"done→done (same)", domain.ColumnDone, domain.ColumnDone, domain.ColumnDone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := domain.ClampForwardMove(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("ClampForwardMove(%q, %q) = %q, want %q", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestDomainBoard_IsBackwardForwardMove(t *testing.T) {
	t.Parallel()
	if !domain.IsBackwardMove("in_progress", "backlog") {
		t.Error("in_progress → backlog should be backward")
	}
	if domain.IsBackwardMove("backlog", "in_progress") {
		t.Error("backlog → in_progress should not be backward")
	}
	if !domain.IsForwardMove("backlog", "approved") {
		t.Error("backlog → approved should be forward")
	}
}
