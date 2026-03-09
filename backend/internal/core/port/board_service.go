package port

import "kiloforge/internal/core/domain"

// BoardService abstracts native board operations for adapters.
type BoardService interface {
	GetBoard(slug string) (*domain.BoardState, error)
	MoveCard(slug, trackID, toColumn string) (*BoardMoveCardResult, error)
	SyncFromTracks(slug string, tracks []TrackEntry, trackTypes map[string]string) (*BoardSyncResult, error)
	UpdateCardAgent(slug, trackID, agentID, agentStatus string) error
	StoreTraceID(slug, trackID, traceID string) error
	GetTraceID(slug, trackID string) (string, bool)
	RemoveCard(slug, trackID string) (bool, error)
}

// BoardMoveCardResult holds the outcome of a card move.
type BoardMoveCardResult struct {
	TrackID    string
	FromColumn string
	ToColumn   string
}

// BoardSyncResult holds the results of a track sync operation.
type BoardSyncResult struct {
	Created   int
	Updated   int
	Unchanged int
}
