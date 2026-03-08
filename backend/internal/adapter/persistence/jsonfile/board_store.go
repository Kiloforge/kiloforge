package jsonfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

var _ port.BoardStore = (*BoardStore)(nil)

// BoardStore persists native board state per project.
type BoardStore struct {
	dataDir string
}

// NewBoardStore creates a new BoardStore rooted at dataDir.
func NewBoardStore(dataDir string) *BoardStore {
	return &BoardStore{dataDir: dataDir}
}

func (s *BoardStore) boardPath(slug string) string {
	return filepath.Join(s.dataDir, "projects", slug, "board.json")
}

// GetBoard returns the board state for a project, or nil if not found.
func (s *BoardStore) GetBoard(slug string) (*domain.BoardState, error) {
	data, err := os.ReadFile(s.boardPath(slug))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read board.json: %w", err)
	}

	var board domain.BoardState
	if err := json.Unmarshal(data, &board); err != nil {
		return nil, fmt.Errorf("parse board.json: %w", err)
	}
	if board.Cards == nil {
		board.Cards = make(map[string]domain.BoardCard)
	}
	return &board, nil
}

// SaveBoard persists the board state for a project.
func (s *BoardStore) SaveBoard(slug string, board *domain.BoardState) error {
	dir := filepath.Dir(s.boardPath(slug))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}

	data, err := json.MarshalIndent(board, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal board.json: %w", err)
	}
	return os.WriteFile(s.boardPath(slug), append(data, '\n'), 0o644)
}
