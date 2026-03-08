package sqlite

import (
	"database/sql"
	"time"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

var _ port.BoardStore = (*BoardStore)(nil)

// BoardStore persists board state per project to SQLite.
type BoardStore struct {
	db *sql.DB
}

// NewBoardStore creates a BoardStore backed by the given database.
func NewBoardStore(db *sql.DB) *BoardStore {
	return &BoardStore{db: db}
}

func (s *BoardStore) GetBoard(slug string) (*domain.BoardState, error) {
	rows, err := s.db.Query(
		`SELECT track_id, title, type, column_name, position,
		        agent_id, agent_status, assigned_worker, pr_number, trace_id,
		        moved_at, created_at
		 FROM board_cards WHERE project_slug = ?`, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cards := make(map[string]domain.BoardCard)
	for rows.Next() {
		var c domain.BoardCard
		var movedAt, createdAt string
		if err := rows.Scan(
			&c.TrackID, &c.Title, &c.Type, &c.Column, &c.Position,
			&c.AgentID, &c.AgentStatus, &c.AssignedWorker, &c.PRNumber, &c.TraceID,
			&movedAt, &createdAt,
		); err != nil {
			continue
		}
		c.MovedAt, _ = time.Parse(time.RFC3339, movedAt)
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		cards[c.TrackID] = c
	}

	if len(cards) == 0 {
		return nil, nil
	}

	return &domain.BoardState{
		Columns: domain.BoardColumns,
		Cards:   cards,
	}, nil
}

func (s *BoardStore) SaveBoard(slug string, board *domain.BoardState) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing cards for this project.
	if _, err := tx.Exec("DELETE FROM board_cards WHERE project_slug = ?", slug); err != nil {
		return err
	}

	// Insert all cards.
	for _, c := range board.Cards {
		if _, err := tx.Exec(
			`INSERT INTO board_cards
			 (track_id, project_slug, title, type, column_name, position,
			  agent_id, agent_status, assigned_worker, pr_number, trace_id,
			  moved_at, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			c.TrackID, slug, c.Title, c.Type, c.Column, c.Position,
			c.AgentID, c.AgentStatus, c.AssignedWorker, c.PRNumber, c.TraceID,
			c.MovedAt.Format(time.RFC3339), c.CreatedAt.Format(time.RFC3339),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
