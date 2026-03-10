package service

import (
	"fmt"
	"time"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

// Compile-time check that NativeBoardService satisfies port.BoardService.
var _ port.BoardService = (*NativeBoardService)(nil)

// NativeBoardService manages the native track board.
type NativeBoardService struct {
	store port.BoardStore
}

// NewNativeBoardService creates a new NativeBoardService.
func NewNativeBoardService(store port.BoardStore) *NativeBoardService {
	return &NativeBoardService{store: store}
}

// GetBoard returns the board state for a project.
// If no board exists yet, returns an empty board.
func (s *NativeBoardService) GetBoard(slug string) (*domain.BoardState, error) {
	board, err := s.store.GetBoard(slug)
	if err != nil {
		return nil, fmt.Errorf("get board: %w", err)
	}
	if board == nil {
		return domain.NewBoardState(), nil
	}
	return board, nil
}

// MoveCard moves a card to a new column.
func (s *NativeBoardService) MoveCard(slug, trackID, toColumn string) (*port.BoardMoveCardResult, error) {
	if !domain.IsValidColumn(toColumn) {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidColumn, toColumn)
	}

	board, err := s.store.GetBoard(slug)
	if err != nil {
		return nil, fmt.Errorf("get board: %w", err)
	}
	if board == nil {
		return nil, fmt.Errorf("%w: project %q", domain.ErrBoardNotFound, slug)
	}

	card, ok := board.Cards[trackID]
	if !ok {
		return nil, fmt.Errorf("%w: %q", domain.ErrCardNotFound, trackID)
	}

	fromColumn := card.Column
	// Clamp forward moves: users can only promote backlog→approved manually.
	toColumn = domain.ClampForwardMove(fromColumn, toColumn)
	if fromColumn == toColumn {
		return &port.BoardMoveCardResult{TrackID: trackID, FromColumn: fromColumn, ToColumn: toColumn}, nil
	}

	card.Column = toColumn
	card.MovedAt = time.Now().Truncate(time.Second)
	// Set position to end of column.
	card.Position = len(board.CardsByColumn(toColumn))
	board.Cards[trackID] = card

	if err := s.store.SaveBoard(slug, board); err != nil {
		return nil, fmt.Errorf("save board: %w", err)
	}

	return &port.BoardMoveCardResult{
		TrackID:    trackID,
		FromColumn: fromColumn,
		ToColumn:   toColumn,
	}, nil
}

// SyncFromTracks syncs the board from discovered tracks.
func (s *NativeBoardService) SyncFromTracks(slug string, tracks []port.TrackEntry, trackTypes map[string]string) (*port.BoardSyncResult, error) {
	board, err := s.store.GetBoard(slug)
	if err != nil {
		return nil, fmt.Errorf("get board: %w", err)
	}
	if board == nil {
		board = domain.NewBoardState()
	}

	result := &port.BoardSyncResult{}
	now := time.Now().Truncate(time.Second)

	for _, t := range tracks {
		existing, exists := board.Cards[t.ID]
		targetCol := statusToColumn(t.Status)

		if !exists {
			trackType := ""
			if trackTypes != nil {
				trackType = trackTypes[t.ID]
			}
			board.Cards[t.ID] = domain.BoardCard{
				TrackID:   t.ID,
				Title:     t.Title,
				Type:      trackType,
				Column:    targetCol,
				Position:  len(board.CardsByColumn(targetCol)),
				MovedAt:   now,
				CreatedAt: now,
			}
			result.Created++
			continue
		}

		if existing.Column != targetCol {
			existing.Column = targetCol
			existing.MovedAt = now
			existing.Position = len(board.CardsByColumn(targetCol))
			board.Cards[t.ID] = existing
			result.Updated++
		} else {
			// Update title if changed.
			if existing.Title != t.Title {
				existing.Title = t.Title
				board.Cards[t.ID] = existing
				result.Updated++
			} else {
				result.Unchanged++
			}
		}
	}

	if err := s.store.SaveBoard(slug, board); err != nil {
		return nil, fmt.Errorf("save board: %w", err)
	}
	return result, nil
}

// UpdateCardAgent updates the agent info on a board card.
func (s *NativeBoardService) UpdateCardAgent(slug, trackID, agentID, agentStatus string) error {
	board, err := s.store.GetBoard(slug)
	if err != nil || board == nil {
		return fmt.Errorf("get board: %w", err)
	}

	card, ok := board.Cards[trackID]
	if !ok {
		return nil // Track not on board, ignore.
	}

	card.AgentID = agentID
	card.AgentStatus = agentStatus
	board.Cards[trackID] = card

	return s.store.SaveBoard(slug, board)
}

// StoreTraceID persists the OTel trace ID on a board card.
func (s *NativeBoardService) StoreTraceID(slug, trackID, traceID string) error {
	board, err := s.store.GetBoard(slug)
	if err != nil || board == nil {
		return fmt.Errorf("get board: %w", err)
	}

	card, ok := board.Cards[trackID]
	if !ok {
		return nil
	}

	card.TraceID = traceID
	board.Cards[trackID] = card

	return s.store.SaveBoard(slug, board)
}

// GetTraceID returns the OTel trace ID stored on a board card.
func (s *NativeBoardService) GetTraceID(slug, trackID string) (string, bool) {
	board, err := s.store.GetBoard(slug)
	if err != nil || board == nil {
		return "", false
	}

	card, ok := board.Cards[trackID]
	if !ok || card.TraceID == "" {
		return "", false
	}

	return card.TraceID, true
}

// RemoveCard removes a card from the board. Returns true if the card existed.
func (s *NativeBoardService) RemoveCard(slug, trackID string) (bool, error) {
	board, err := s.store.GetBoard(slug)
	if err != nil {
		return false, fmt.Errorf("get board: %w", err)
	}
	if board == nil {
		return false, nil
	}

	if _, ok := board.Cards[trackID]; !ok {
		return false, nil
	}

	delete(board.Cards, trackID)
	if err := s.store.SaveBoard(slug, board); err != nil {
		return false, fmt.Errorf("save board: %w", err)
	}
	return true, nil
}

// statusToColumn maps a track status from tracks.md to a board column.
func statusToColumn(status string) string {
	switch status {
	case StatusPending:
		return domain.ColumnBacklog
	case StatusApproved:
		return domain.ColumnApproved
	case StatusInProgress:
		return domain.ColumnInProgress
	case StatusComplete:
		return domain.ColumnDone
	default:
		return domain.ColumnBacklog
	}
}
