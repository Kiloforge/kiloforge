package service

import (
	"context"
	"fmt"
	"path/filepath"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

// ImplementConsentStore checks and records user consent for agent permissions.
type ImplementConsentStore interface {
	HasAgentPermissionsConsent() bool
	RecordAgentPermissionsConsent() error
}

// ImplementWorktreePool manages worktree acquisition and return.
type ImplementWorktreePool interface {
	Acquire() (*ImplementWorktree, error)
	Prepare(wt *ImplementWorktree, trackID string) error
	ReturnByTrackID(trackID string) error
	Save(dataDir string) error
}

// ImplementWorktree represents a worktree slot from the pool.
type ImplementWorktree struct {
	Path    string
	AgentID string
}

// ImplementSkillValidator checks that required agent skills are installed.
type ImplementSkillValidator interface {
	ValidateSkills(role, workDir string) error
}

// ImplementAgentSpawner spawns developer agents.
type ImplementAgentSpawner interface {
	SetTracer(t port.Tracer)
	SetCompletionCallback(fn func(agentID, ref, status string))
	SpawnDeveloper(ctx context.Context, opts SpawnDeveloperOpts) (*domain.AgentInfo, error)
	ValidateSkills(role, workDir string) error
}

// SpawnDeveloperOpts configures a developer agent spawn.
type SpawnDeveloperOpts struct {
	TrackID     string
	Flags       string
	WorktreeDir string
	LogDir      string
	Model       string
}

// ImplementAuthChecker verifies Claude CLI authentication.
type ImplementAuthChecker interface {
	CheckAuth(ctx context.Context) error
}

// ImplementTracingProvider initializes and provides tracing.
type ImplementTracingProvider interface {
	Init(ctx context.Context) (port.Tracer, func(context.Context) error, error)
	ExtractTraceID(ctx context.Context) string
}

// ImplementService orchestrates the "kf implement" workflow:
// track validation, consent, worktree acquisition, agent spawning,
// and board state transitions.
type ImplementService struct {
	consent  ImplementConsentStore
	board    *NativeBoardService
	dataDir  string
	model    string
}

// NewImplementService creates a new ImplementService.
func NewImplementService(
	consent ImplementConsentStore,
	board *NativeBoardService,
	dataDir string,
	model string,
) *ImplementService {
	return &ImplementService{
		consent:  consent,
		board:    board,
		dataDir:  dataDir,
		model:    model,
	}
}

// ValidateTrack checks that a track exists in the project and is in pending status.
func (s *ImplementService) ValidateTrack(projectDir, trackID string) (*port.TrackEntry, error) {
	tracks, err := DiscoverTracks(projectDir)
	if err != nil {
		return nil, fmt.Errorf("discover tracks: %w", err)
	}

	var found *port.TrackEntry
	for i := range tracks {
		if tracks[i].ID == trackID {
			found = &tracks[i]
			break
		}
	}
	if found == nil {
		return nil, &TrackNotFoundError{TrackID: trackID}
	}
	if found.Status == StatusComplete {
		return nil, &TrackAlreadyCompleteError{TrackID: trackID}
	}
	if found.Status == StatusInProgress {
		return nil, &TrackInProgressError{TrackID: trackID}
	}

	return found, nil
}

// HasConsent returns true if the user has already consented to agent permissions.
func (s *ImplementService) HasConsent() bool {
	return s.consent.HasAgentPermissionsConsent()
}

// RecordConsent stores the user's consent for agent permissions.
func (s *ImplementService) RecordConsent() error {
	return s.consent.RecordAgentPermissionsConsent()
}

// ImplementResult contains the output of a successful implement execution.
type ImplementResult struct {
	AgentInfo    *domain.AgentInfo
	WorktreePath string
	TraceID      string
	LogFile      string
}

// ImplementOpts configures the implement execution.
type ImplementOpts struct {
	TrackID  string
	TrackTitle string
	ProjectSlug string
	ProjectDir  string
}

// LogDir returns the log directory for a project.
func (s *ImplementService) LogDir(projectSlug string) string {
	return filepath.Join(s.dataDir, "projects", projectSlug, "logs")
}

// MoveCardToInProgress moves a track card to the In Progress column.
func (s *ImplementService) MoveCardToInProgress(projectSlug, trackID string) (string, string, error) {
	result, err := s.board.MoveCard(projectSlug, trackID, domain.ColumnInProgress)
	if err != nil {
		return "", "", err
	}
	return result.FromColumn, result.ToColumn, nil
}

// StoreTraceID stores the trace ID on a board card.
func (s *ImplementService) StoreTraceID(projectSlug, trackID, traceID string) {
	_ = s.board.StoreTraceID(projectSlug, trackID, traceID)
}

// MoveCardToDone moves a track card to the Done column (for completion callbacks).
func (s *ImplementService) MoveCardToDone(projectSlug, trackID string) error {
	_, err := s.board.MoveCard(projectSlug, trackID, domain.ColumnDone)
	return err
}

// TrackNotFoundError indicates the track was not found in the project.
type TrackNotFoundError struct {
	TrackID string
}

func (e *TrackNotFoundError) Error() string {
	return fmt.Sprintf("track %q not found", e.TrackID)
}

// TrackAlreadyCompleteError indicates the track is already complete.
type TrackAlreadyCompleteError struct {
	TrackID string
}

func (e *TrackAlreadyCompleteError) Error() string {
	return fmt.Sprintf("track %q is already complete", e.TrackID)
}

// TrackInProgressError indicates the track is already in progress.
type TrackInProgressError struct {
	TrackID string
}

func (e *TrackInProgressError) Error() string {
	return fmt.Sprintf("track %q is already in progress", e.TrackID)
}

// ListPendingTracks returns tracks with pending status in the given project.
func (s *ImplementService) ListPendingTracks(projectDir string) ([]port.TrackEntry, error) {
	tracks, err := DiscoverTracks(projectDir)
	if err != nil {
		return nil, fmt.Errorf("discover tracks: %w", err)
	}
	return FilterByStatus(tracks, StatusPending), nil
}
