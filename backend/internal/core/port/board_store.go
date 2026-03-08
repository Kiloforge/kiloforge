package port

import "kiloforge/internal/core/domain"

// BoardStore persists board state per project.
type BoardStore interface {
	GetBoard(slug string) (*domain.BoardState, error)
	SaveBoard(slug string, board *domain.BoardState) error
}
