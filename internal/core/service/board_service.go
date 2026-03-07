package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"crelay/internal/core/domain"
	"crelay/internal/core/port"
)

// Standard label definitions for track boards.
var standardLabels = []struct {
	Name  string
	Color string
}{
	{"type:feature", "#0075ca"},
	{"type:bug", "#e11d48"},
	{"type:refactor", "#e4e669"},
	{"type:chore", "#808080"},
	{"status:suggested", "#d4d4d4"},
	{"status:approved", "#22c55e"},
	{"status:in-progress", "#f97316"},
	{"status:in-review", "#a855f7"},
}

// Standard kanban column names in order.
var standardColumns = []string{
	"Suggested",
	"Approved",
	"In Progress",
	"In Review",
	"Completed",
}

// BoardStore abstracts persistence for board config and track-issue mappings.
type BoardStore interface {
	GetBoardConfig(slug string) (*domain.BoardConfig, error)
	SaveBoardConfig(slug string, cfg *domain.BoardConfig) error
	GetTrackIssue(slug, trackID string) (*domain.TrackIssue, error)
	SaveTrackIssue(slug string, ti domain.TrackIssue) error
	ListTrackIssues(slug string) ([]domain.TrackIssue, error)
}

// BoardService manages Gitea project board sync for conductor tracks.
type BoardService struct {
	gitea port.BoardGiteaClient
	store BoardStore
}

// NewBoardService creates a new BoardService.
func NewBoardService(gitea port.BoardGiteaClient, store BoardStore) *BoardService {
	return &BoardService{gitea: gitea, store: store}
}

// SetupBoard creates a kanban board with standard labels and columns for a project.
// Idempotent — returns existing config if board already exists.
func (s *BoardService) SetupBoard(ctx context.Context, project domain.Project) (*domain.BoardConfig, error) {
	// Check if already set up.
	existing, err := s.store.GetBoardConfig(project.Slug)
	if err != nil {
		return nil, fmt.Errorf("check existing board: %w", err)
	}
	if existing != nil && existing.ProjectBoardID > 0 {
		return existing, nil
	}

	cfg := &domain.BoardConfig{
		Columns: make(map[string]int),
		Labels:  make(map[string]int),
	}

	// Create labels.
	for _, l := range standardLabels {
		id, err := s.gitea.EnsureLabel(ctx, project.RepoName, l.Name, l.Color)
		if err != nil {
			return nil, fmt.Errorf("ensure label %q: %w", l.Name, err)
		}
		cfg.Labels[l.Name] = id
	}

	// Check if project board already exists.
	projects, err := s.gitea.ListProjects(ctx, project.RepoName)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	for _, p := range projects {
		if p.Title == "Tracks" {
			cfg.ProjectBoardID = p.ID
			break
		}
	}

	// Create board if not found.
	if cfg.ProjectBoardID == 0 {
		boardID, err := s.gitea.CreateProject(ctx, project.RepoName, "Tracks", "Conductor track board")
		if err != nil {
			return nil, fmt.Errorf("create project board: %w", err)
		}
		cfg.ProjectBoardID = boardID
	}

	// Create columns (check existing first).
	existingCols, err := s.gitea.ListColumns(ctx, cfg.ProjectBoardID)
	if err != nil {
		return nil, fmt.Errorf("list columns: %w", err)
	}
	existingColMap := make(map[string]int)
	for _, col := range existingCols {
		existingColMap[col.Title] = col.ID
	}

	for _, colName := range standardColumns {
		if id, ok := existingColMap[colName]; ok {
			cfg.Columns[columnKey(colName)] = id
			continue
		}
		id, err := s.gitea.CreateColumn(ctx, cfg.ProjectBoardID, colName)
		if err != nil {
			return nil, fmt.Errorf("create column %q: %w", colName, err)
		}
		cfg.Columns[columnKey(colName)] = id
	}

	if err := s.store.SaveBoardConfig(project.Slug, cfg); err != nil {
		return nil, fmt.Errorf("save board config: %w", err)
	}

	return cfg, nil
}

// SyncResult holds the results of a track sync operation.
type SyncResult struct {
	Created   int
	Updated   int
	Unchanged int
}

// PublishTrack creates a Gitea issue from a track and places it on the board.
// Idempotent — skips if already published.
func (s *BoardService) PublishTrack(ctx context.Context, project domain.Project, track TrackEntry, trackType, specContent string) error {
	existing, err := s.store.GetTrackIssue(project.Slug, track.ID)
	if err != nil {
		return fmt.Errorf("check existing: %w", err)
	}
	if existing != nil {
		return nil
	}

	cfg, err := s.store.GetBoardConfig(project.Slug)
	if err != nil || cfg == nil {
		return fmt.Errorf("board not set up for project %q", project.Slug)
	}

	// Build label list.
	var labels []string
	if trackType != "" {
		labels = append(labels, "type:"+trackType)
	}
	statusLabel := StatusToColumn(track.Status)
	if statusLabel != "" {
		labels = append(labels, "status:"+track.Status)
	}

	// Create issue.
	body := specContent
	if body == "" {
		body = track.Title
	}
	issueNum, err := s.gitea.CreateIssue(ctx, project.RepoName, track.Title, body, labels)
	if err != nil {
		return fmt.Errorf("create issue: %w", err)
	}

	// Place card in the appropriate column.
	colName := StatusToColumn(track.Status)
	colID, ok := cfg.Columns[columnKey(colName)]
	if !ok {
		colID = cfg.Columns["suggested"]
	}

	cardID, err := s.gitea.CreateCard(ctx, colID, issueNum)
	if err != nil {
		return fmt.Errorf("create card: %w", err)
	}

	return s.store.SaveTrackIssue(project.Slug, domain.TrackIssue{
		TrackID:     track.ID,
		IssueNumber: issueNum,
		CardID:      cardID,
		Column:      columnKey(colName),
		LastSynced:  time.Now().Truncate(time.Second),
	})
}

// SyncTracks diffs local tracks against published ones, creating or updating as needed.
func (s *BoardService) SyncTracks(ctx context.Context, project domain.Project, tracks []TrackEntry, trackTypes map[string]string, specReader func(trackID string) string) (*SyncResult, error) {
	cfg, err := s.store.GetBoardConfig(project.Slug)
	if err != nil || cfg == nil {
		return nil, fmt.Errorf("board not set up for project %q", project.Slug)
	}

	existing, err := s.store.ListTrackIssues(project.Slug)
	if err != nil {
		return nil, fmt.Errorf("list track issues: %w", err)
	}
	existingMap := make(map[string]domain.TrackIssue, len(existing))
	for _, ti := range existing {
		existingMap[ti.TrackID] = ti
	}

	result := &SyncResult{}

	for _, track := range tracks {
		ti, published := existingMap[track.ID]

		if !published {
			// New track — publish it.
			spec := ""
			if specReader != nil {
				spec = specReader(track.ID)
			}
			trackType := ""
			if trackTypes != nil {
				trackType = trackTypes[track.ID]
			}
			if err := s.PublishTrack(ctx, project, track, trackType, spec); err != nil {
				return nil, fmt.Errorf("publish %s: %w", track.ID, err)
			}
			result.Created++
			continue
		}

		// Check if column changed.
		targetCol := columnKey(StatusToColumn(track.Status))
		if ti.Column == targetCol {
			result.Unchanged++
			continue
		}

		// Move card to new column.
		colID, ok := cfg.Columns[targetCol]
		if !ok {
			result.Unchanged++
			continue
		}
		if err := s.gitea.MoveCard(ctx, ti.CardID, colID); err != nil {
			return nil, fmt.Errorf("move card for %s: %w", track.ID, err)
		}

		ti.Column = targetCol
		ti.LastSynced = time.Now().Truncate(time.Second)
		if err := s.store.SaveTrackIssue(project.Slug, ti); err != nil {
			return nil, fmt.Errorf("update mapping for %s: %w", track.ID, err)
		}
		result.Updated++
	}

	return result, nil
}

// ReadTrackSpec reads the spec.md for a track from the project directory.
func ReadTrackSpec(projectDir, trackID string) string {
	paths := []string{
		filepath.Join(projectDir, ".agent", "conductor", "tracks", trackID, "spec.md"),
		filepath.Join(projectDir, ".agent", "conductor", "tracks", "_archive", trackID, "spec.md"),
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			return string(data)
		}
	}
	return ""
}

// StatusToColumn maps a track status to a kanban column name.
func StatusToColumn(status string) string {
	switch status {
	case StatusPending:
		return "Suggested"
	case StatusApproved:
		return "Approved"
	case StatusInProgress:
		return "In Progress"
	case StatusInReview:
		return "In Review"
	case StatusComplete:
		return "Completed"
	default:
		return "Suggested"
	}
}

// ColumnToStatus maps a kanban column name to a track status.
func ColumnToStatus(column string) string {
	switch column {
	case "Suggested":
		return StatusPending
	case "Approved":
		return StatusApproved
	case "In Progress":
		return StatusInProgress
	case "In Review":
		return StatusInReview
	case "Completed":
		return StatusComplete
	default:
		return StatusPending
	}
}

// MoveCard moves a card to a different column on the Gitea board.
func (s *BoardService) MoveCard(ctx context.Context, slug string, cardID, columnID int) error {
	return s.gitea.MoveCard(ctx, cardID, columnID)
}

// CloseTrackIssue closes a Gitea issue for a track.
func (s *BoardService) CloseTrackIssue(ctx context.Context, project domain.Project, issueNum int) error {
	return s.gitea.UpdateIssue(ctx, project.RepoName, issueNum, "", "", "closed")
}

// ColumnKeyFromName normalizes a column name to a map key. Exported for use by board sync.
func ColumnKeyFromName(name string) string {
	return columnKey(name)
}

// columnKey normalizes a column name to a map key (lowercase, hyphens).
func columnKey(name string) string {
	switch name {
	case "Suggested":
		return "suggested"
	case "Approved":
		return "approved"
	case "In Progress":
		return "in_progress"
	case "In Review":
		return "in_review"
	case "Completed":
		return "completed"
	default:
		return "suggested"
	}
}
