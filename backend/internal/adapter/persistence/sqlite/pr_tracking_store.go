package sqlite

import (
	"database/sql"
	"fmt"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

var _ port.PRTrackingStore = (*PRTrackingStore)(nil)

// PRTrackingStore persists PR tracking records to SQLite.
type PRTrackingStore struct {
	db *sql.DB
}

// NewPRTrackingStore creates a PRTrackingStore backed by the given database.
func NewPRTrackingStore(db *sql.DB) *PRTrackingStore {
	return &PRTrackingStore{db: db}
}

func (s *PRTrackingStore) LoadPRTracking(slug string) (*domain.PRTracking, error) {
	var t domain.PRTracking
	err := s.db.QueryRow(
		`SELECT pr_number, project_slug, track_id,
		        developer_agent_id, developer_session, developer_work_dir,
		        reviewer_agent_id, reviewer_session,
		        review_cycle_count, max_review_cycles, status
		 FROM pr_tracking WHERE project_slug = ? ORDER BY pr_number DESC LIMIT 1`, slug,
	).Scan(
		&t.PRNumber, &t.ProjectSlug, &t.TrackID,
		&t.DeveloperAgentID, &t.DeveloperSession, &t.DeveloperWorkDir,
		&t.ReviewerAgentID, &t.ReviewerSession,
		&t.ReviewCycleCount, &t.MaxReviewCycles, &t.Status,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pr tracking for %s: %w", slug, domain.ErrPRTrackingNotFound)
		}
		return nil, fmt.Errorf("load pr tracking: %w", err)
	}
	return &t, nil
}

func (s *PRTrackingStore) SavePRTracking(slug string, t *domain.PRTracking) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO pr_tracking
		 (pr_number, project_slug, track_id,
		  developer_agent_id, developer_session, developer_work_dir,
		  reviewer_agent_id, reviewer_session,
		  review_cycle_count, max_review_cycles, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.PRNumber, slug, t.TrackID,
		t.DeveloperAgentID, t.DeveloperSession, t.DeveloperWorkDir,
		t.ReviewerAgentID, t.ReviewerSession,
		t.ReviewCycleCount, t.MaxReviewCycles, t.Status,
	)
	return err
}
