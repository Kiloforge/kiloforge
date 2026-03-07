package jsonfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"crelay/internal/core/domain"
)

// boardData is the on-disk format for board.json.
type boardData struct {
	Config      *domain.BoardConfig         `json:"config,omitempty"`
	TrackIssues map[string]domain.TrackIssue `json:"track_issues,omitempty"`
}

// BoardStore persists board configuration and track-issue mappings per project.
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

func (s *BoardStore) load(slug string) (*boardData, error) {
	data, err := os.ReadFile(s.boardPath(slug))
	if os.IsNotExist(err) {
		return &boardData{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read board.json: %w", err)
	}

	var bd boardData
	if err := json.Unmarshal(data, &bd); err != nil {
		return nil, fmt.Errorf("parse board.json: %w", err)
	}
	return &bd, nil
}

func (s *BoardStore) save(slug string, bd *boardData) error {
	dir := filepath.Dir(s.boardPath(slug))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}

	data, err := json.MarshalIndent(bd, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal board.json: %w", err)
	}
	return os.WriteFile(s.boardPath(slug), append(data, '\n'), 0o644)
}

// GetBoardConfig returns the board configuration for a project, or nil if not set up.
func (s *BoardStore) GetBoardConfig(slug string) (*domain.BoardConfig, error) {
	bd, err := s.load(slug)
	if err != nil {
		return nil, err
	}
	return bd.Config, nil
}

// SaveBoardConfig persists the board configuration for a project.
func (s *BoardStore) SaveBoardConfig(slug string, cfg *domain.BoardConfig) error {
	bd, err := s.load(slug)
	if err != nil {
		// If corrupt, start fresh
		bd = &boardData{}
	}
	bd.Config = cfg
	return s.save(slug, bd)
}

// GetTrackIssue returns the track-issue mapping for a track, or nil if not found.
func (s *BoardStore) GetTrackIssue(slug, trackID string) (*domain.TrackIssue, error) {
	bd, err := s.load(slug)
	if err != nil {
		return nil, err
	}
	if bd.TrackIssues == nil {
		return nil, nil
	}
	ti, ok := bd.TrackIssues[trackID]
	if !ok {
		return nil, nil
	}
	return &ti, nil
}

// SaveTrackIssue persists a track-issue mapping.
func (s *BoardStore) SaveTrackIssue(slug string, ti domain.TrackIssue) error {
	bd, err := s.load(slug)
	if err != nil {
		bd = &boardData{}
	}
	if bd.TrackIssues == nil {
		bd.TrackIssues = make(map[string]domain.TrackIssue)
	}
	bd.TrackIssues[ti.TrackID] = ti
	return s.save(slug, bd)
}

// ListTrackIssues returns all track-issue mappings for a project.
func (s *BoardStore) ListTrackIssues(slug string) ([]domain.TrackIssue, error) {
	bd, err := s.load(slug)
	if err != nil {
		return nil, err
	}
	result := make([]domain.TrackIssue, 0, len(bd.TrackIssues))
	for _, ti := range bd.TrackIssues {
		result = append(result, ti)
	}
	return result, nil
}
