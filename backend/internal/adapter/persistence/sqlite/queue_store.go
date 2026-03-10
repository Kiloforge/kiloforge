package sqlite

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"kiloforge/internal/core/domain"
)

// QueueStore implements port.QueueStore using SQLite.
type QueueStore struct {
	db *sql.DB
}

// NewQueueStore creates a new SQLite-backed queue store.
func NewQueueStore(db *sql.DB) *QueueStore {
	return &QueueStore{db: db}
}

func (s *QueueStore) Enqueue(item domain.QueueItem) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO queue_items (track_id, project_slug, status, enqueued_at)
		 VALUES (?, ?, ?, ?)`,
		item.TrackID, item.ProjectSlug, domain.QueueStatusQueued,
		item.EnqueuedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *QueueStore) Dequeue(trackID string) error {
	res, err := s.db.Exec(`DELETE FROM queue_items WHERE track_id = ?`, trackID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("queue item not found: %s", trackID)
	}
	return nil
}

func (s *QueueStore) Assign(trackID, agentID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`UPDATE queue_items SET status = ?, agent_id = ?, assigned_at = ?
		 WHERE track_id = ? AND status = ?`,
		domain.QueueStatusAssigned, agentID, now, trackID, domain.QueueStatusQueued,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("queue item not found or not in queued state: %s", trackID)
	}
	return nil
}

func (s *QueueStore) Complete(trackID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`UPDATE queue_items SET status = ?, completed_at = ?
		 WHERE track_id = ? AND status = ?`,
		domain.QueueStatusCompleted, now, trackID, domain.QueueStatusAssigned,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("queue item not found or not in assigned state: %s", trackID)
	}
	return nil
}

func (s *QueueStore) Fail(trackID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`UPDATE queue_items SET status = ?, completed_at = ?
		 WHERE track_id = ? AND status IN (?, ?)`,
		domain.QueueStatusFailed, now, trackID,
		domain.QueueStatusQueued, domain.QueueStatusAssigned,
	)
	return err
}

func (s *QueueStore) List(statuses ...string) ([]domain.QueueItem, error) {
	var query string
	var args []any
	if len(statuses) == 0 {
		query = `SELECT track_id, project_slug, status, agent_id, enqueued_at, assigned_at, completed_at
		         FROM queue_items ORDER BY enqueued_at ASC`
	} else {
		placeholders := make([]string, len(statuses))
		for i, s := range statuses {
			placeholders[i] = "?"
			args = append(args, s)
		}
		query = fmt.Sprintf(
			`SELECT track_id, project_slug, status, agent_id, enqueued_at, assigned_at, completed_at
			 FROM queue_items WHERE status IN (%s) ORDER BY enqueued_at ASC`,
			strings.Join(placeholders, ","),
		)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.QueueItem
	for rows.Next() {
		var item domain.QueueItem
		var agentID, enqueuedAt, assignedAt, completedAt sql.NullString
		if err := rows.Scan(&item.TrackID, &item.ProjectSlug, &item.Status,
			&agentID, &enqueuedAt, &assignedAt, &completedAt); err != nil {
			return nil, err
		}
		item.AgentID = agentID.String
		if enqueuedAt.Valid {
			if t, err := time.Parse(time.RFC3339, enqueuedAt.String); err == nil {
				item.EnqueuedAt = t
			}
		}
		if assignedAt.Valid {
			if t, err := time.Parse(time.RFC3339, assignedAt.String); err == nil {
				item.AssignedAt = &t
			}
		}
		if completedAt.Valid {
			if t, err := time.Parse(time.RFC3339, completedAt.String); err == nil {
				item.CompletedAt = &t
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *QueueStore) Get(trackID string) (*domain.QueueItem, error) {
	var item domain.QueueItem
	var agentID, enqueuedAt, assignedAt, completedAt sql.NullString
	err := s.db.QueryRow(
		`SELECT track_id, project_slug, status, agent_id, enqueued_at, assigned_at, completed_at
		 FROM queue_items WHERE track_id = ?`, trackID,
	).Scan(&item.TrackID, &item.ProjectSlug, &item.Status,
		&agentID, &enqueuedAt, &assignedAt, &completedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item.AgentID = agentID.String
	if enqueuedAt.Valid {
		if t, err := time.Parse(time.RFC3339, enqueuedAt.String); err == nil {
			item.EnqueuedAt = t
		}
	}
	if assignedAt.Valid {
		if t, err := time.Parse(time.RFC3339, assignedAt.String); err == nil {
			item.AssignedAt = &t
		}
	}
	if completedAt.Valid {
		if t, err := time.Parse(time.RFC3339, completedAt.String); err == nil {
			item.CompletedAt = &t
		}
	}
	return &item, nil
}

func (s *QueueStore) Clear() error {
	_, err := s.db.Exec(`DELETE FROM queue_items`)
	return err
}

// ListPaginated returns a paginated list of queue items with optional filters.
func (s *QueueStore) ListPaginated(opts domain.PageOpts, projectSlug string, statuses ...string) (domain.Page[domain.QueueItem], error) {
	opts.Normalize()

	var whereParts []string
	var args []any
	if projectSlug != "" {
		whereParts = append(whereParts, "project_slug = ?")
		args = append(args, projectSlug)
	}
	if len(statuses) > 0 {
		ph := make([]string, len(statuses))
		for i, st := range statuses {
			ph[i] = "?"
			args = append(args, st)
		}
		whereParts = append(whereParts, "status IN ("+strings.Join(ph, ",")+")")
	}
	if opts.Cursor != "" {
		cur := domain.DecodeCursor(opts.Cursor)
		if cur.SortVal != "" {
			whereParts = append(whereParts, "(enqueued_at > ? OR (enqueued_at = ? AND track_id > ?))")
			args = append(args, cur.SortVal, cur.SortVal, cur.ID)
		}
	}

	where := ""
	if len(whereParts) > 0 {
		where = " WHERE " + strings.Join(whereParts, " AND ")
	}

	// Count total (without cursor).
	var countParts []string
	var countArgs []any
	if projectSlug != "" {
		countParts = append(countParts, "project_slug = ?")
		countArgs = append(countArgs, projectSlug)
	}
	if len(statuses) > 0 {
		ph := make([]string, len(statuses))
		for i, st := range statuses {
			ph[i] = "?"
			countArgs = append(countArgs, st)
		}
		countParts = append(countParts, "status IN ("+strings.Join(ph, ",")+")")
	}
	countWhere := ""
	if len(countParts) > 0 {
		countWhere = " WHERE " + strings.Join(countParts, " AND ")
	}
	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM queue_items"+countWhere, countArgs...).Scan(&total); err != nil {
		return domain.Page[domain.QueueItem]{}, fmt.Errorf("count queue: %w", err)
	}

	query := `SELECT track_id, project_slug, status, agent_id, enqueued_at, assigned_at, completed_at
	          FROM queue_items` + where + ` ORDER BY enqueued_at ASC, track_id ASC LIMIT ?`
	args = append(args, opts.Limit+1)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return domain.Page[domain.QueueItem]{}, fmt.Errorf("list queue: %w", err)
	}
	defer rows.Close()

	var items []domain.QueueItem
	for rows.Next() {
		var item domain.QueueItem
		var agentID, enqueuedAt, assignedAt, completedAt sql.NullString
		if err := rows.Scan(&item.TrackID, &item.ProjectSlug, &item.Status,
			&agentID, &enqueuedAt, &assignedAt, &completedAt); err != nil {
			return domain.Page[domain.QueueItem]{}, err
		}
		item.AgentID = agentID.String
		if enqueuedAt.Valid {
			if t, err := time.Parse(time.RFC3339, enqueuedAt.String); err == nil {
				item.EnqueuedAt = t
			}
		}
		if assignedAt.Valid {
			if t, err := time.Parse(time.RFC3339, assignedAt.String); err == nil {
				item.AssignedAt = &t
			}
		}
		if completedAt.Valid {
			if t, err := time.Parse(time.RFC3339, completedAt.String); err == nil {
				item.CompletedAt = &t
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.Page[domain.QueueItem]{}, err
	}

	var nextCursor string
	if len(items) > opts.Limit {
		last := items[opts.Limit-1]
		nextCursor = domain.EncodeCursor(last.EnqueuedAt.UTC().Format(time.RFC3339), last.TrackID)
		items = items[:opts.Limit]
	}

	return domain.Page[domain.QueueItem]{
		Items:      items,
		NextCursor: nextCursor,
		TotalCount: total,
	}, nil
}
