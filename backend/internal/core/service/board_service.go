package service

import (
	"fmt"
	"time"

	"crelay/internal/core/domain"
)

// NativeBoardStore abstracts persistence for the native board.
type NativeBoardStore interface {
	GetBoard(slug string) (*domain.BoardState, error)
	SaveBoard(slug string, board *domain.BoardState) error
}

// NativeBoardService manages the native track board.
type NativeBoardService struct {
	store NativeBoardStore
}

// NewNativeBoardService creates a new NativeBoardService.
func NewNativeBoardService(store NativeBoardStore) *NativeBoardService {
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

// MoveCardResult holds the outcome of a card move.
type MoveCardResult struct {
	TrackID    string `json:"track_id"`
	FromColumn string `json:"from_column"`
	ToColumn   string `json:"to_column"`
}

// MoveCard moves a card to a new column.
func (s *NativeBoardService) MoveCard(slug, trackID, toColumn string) (*MoveCardResult, error) {
	if !domain.IsValidColumn(toColumn) {
		return nil, fmt.Errorf("invalid column: %s", toColumn)
	}

	board, err := s.store.GetBoard(slug)
	if err != nil {
		return nil, fmt.Errorf("get board: %w", err)
	}
	if board == nil {
		return nil, fmt.Errorf("no board for project %q", slug)
	}

	card, ok := board.Cards[trackID]
	if !ok {
		return nil, fmt.Errorf("track %q not on board", trackID)
	}

	fromColumn := card.Column
	if fromColumn == toColumn {
		return &MoveCardResult{TrackID: trackID, FromColumn: fromColumn, ToColumn: toColumn}, nil
	}

	card.Column = toColumn
	card.MovedAt = time.Now().Truncate(time.Second)
	// Set position to end of column.
	card.Position = len(board.CardsByColumn(toColumn))
	board.Cards[trackID] = card

	if err := s.store.SaveBoard(slug, board); err != nil {
		return nil, fmt.Errorf("save board: %w", err)
	}

	return &MoveCardResult{
		TrackID:    trackID,
		FromColumn: fromColumn,
		ToColumn:   toColumn,
	}, nil
}

// SyncResult holds the results of a track sync operation.
type SyncResult struct {
	Created   int
	Updated   int
	Unchanged int
}

// SyncFromTracks syncs the board from discovered tracks.
func (s *NativeBoardService) SyncFromTracks(slug string, tracks []TrackEntry, trackTypes map[string]string) (*SyncResult, error) {
	board, err := s.store.GetBoard(slug)
	if err != nil {
		return nil, fmt.Errorf("get board: %w", err)
	}
	if board == nil {
		board = domain.NewBoardState()
	}

	result := &SyncResult{}
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

// statusToColumn maps a track status from tracks.md to a board column.
func statusToColumn(status string) string {
	switch status {
	case StatusPending:
		return domain.ColumnBacklog
	case StatusApproved:
		return domain.ColumnApproved
	case StatusInProgress:
		return domain.ColumnInProgress
	case StatusInReview:
		return domain.ColumnInReview
	case StatusComplete:
		return domain.ColumnDone
	default:
		return domain.ColumnBacklog
	}
}
