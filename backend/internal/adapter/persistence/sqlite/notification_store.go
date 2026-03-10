package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"kiloforge/internal/core/domain"
)

// NotificationStore implements port.NotificationStore using SQLite.
type NotificationStore struct {
	db *sql.DB
}

// NewNotificationStore creates a new SQLite-backed notification store.
func NewNotificationStore(db *sql.DB) *NotificationStore {
	return &NotificationStore{db: db}
}

// Insert persists a new notification.
func (s *NotificationStore) Insert(n domain.Notification) error {
	var ackedAt sql.NullString
	if n.AcknowledgedAt != nil {
		ackedAt = sql.NullString{String: n.AcknowledgedAt.UTC().Format(time.RFC3339Nano), Valid: true}
	}
	_, err := s.db.Exec(
		`INSERT INTO notifications (id, agent_id, title, body, created_at, acknowledged_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		n.ID, n.AgentID, n.Title, n.Body,
		n.CreatedAt.UTC().Format(time.RFC3339Nano),
		ackedAt,
	)
	return err
}

// ListActive returns all unacknowledged notifications, optionally filtered by agent ID.
// Results are sorted by created_at DESC.
func (s *NotificationStore) ListActive(agentID string) ([]domain.Notification, error) {
	query := `SELECT id, agent_id, title, body, created_at, acknowledged_at
	          FROM notifications WHERE acknowledged_at IS NULL`
	var args []any
	if agentID != "" {
		query += " AND agent_id = ?"
		args = append(args, agentID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list active notifications: %w", err)
	}
	defer rows.Close()

	var items []domain.Notification
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, n)
	}
	return items, rows.Err()
}

// Acknowledge sets acknowledged_at on a notification. Returns error if not found.
func (s *NotificationStore) Acknowledge(id string) error {
	res, err := s.db.Exec(
		`UPDATE notifications SET acknowledged_at = ? WHERE id = ? AND acknowledged_at IS NULL`,
		time.Now().UTC().Format(time.RFC3339Nano), id,
	)
	if err != nil {
		return fmt.Errorf("acknowledge notification: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("notification not found or already acknowledged: %s", id)
	}
	return nil
}

// DeleteForAgent removes all notifications for an agent (used on terminal status cleanup).
func (s *NotificationStore) DeleteForAgent(agentID string) error {
	_, err := s.db.Exec(`DELETE FROM notifications WHERE agent_id = ?`, agentID)
	return err
}

// FindActiveByAgent returns the active (unacknowledged) notification for an agent, or nil.
func (s *NotificationStore) FindActiveByAgent(agentID string) (*domain.Notification, error) {
	row := s.db.QueryRow(
		`SELECT id, agent_id, title, body, created_at, acknowledged_at
		 FROM notifications WHERE agent_id = ? AND acknowledged_at IS NULL
		 ORDER BY created_at DESC LIMIT 1`, agentID,
	)
	var n domain.Notification
	var createdAtStr string
	var ackedAt sql.NullString
	err := row.Scan(&n.ID, &n.AgentID, &n.Title, &n.Body, &createdAtStr, &ackedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find active notification: %w", err)
	}
	if t, err := time.Parse(time.RFC3339Nano, createdAtStr); err == nil {
		n.CreatedAt = t
	}
	return &n, nil
}

func scanNotification(rows *sql.Rows) (domain.Notification, error) {
	var n domain.Notification
	var createdAtStr string
	var ackedAt sql.NullString
	if err := rows.Scan(&n.ID, &n.AgentID, &n.Title, &n.Body, &createdAtStr, &ackedAt); err != nil {
		return n, err
	}
	if t, err := time.Parse(time.RFC3339Nano, createdAtStr); err == nil {
		n.CreatedAt = t
	}
	if ackedAt.Valid {
		if t, err := time.Parse(time.RFC3339Nano, ackedAt.String); err == nil {
			n.AcknowledgedAt = &t
		}
	}
	return n, nil
}
