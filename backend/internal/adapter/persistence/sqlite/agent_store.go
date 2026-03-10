package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

var _ port.AgentStore = (*AgentStore)(nil)

// AgentStore persists agent state to SQLite.
type AgentStore struct {
	db *sql.DB
}

// NewAgentStore creates an AgentStore backed by the given database.
func NewAgentStore(db *sql.DB) *AgentStore {
	return &AgentStore{db: db}
}

// Load is a no-op for SQLite — state is always current.
func (s *AgentStore) Load() error { return nil }

// Save is a no-op for SQLite — writes are immediate.
func (s *AgentStore) Save() error { return nil }

func (s *AgentStore) Agents() []domain.AgentInfo {
	rows, err := s.db.Query(
		`SELECT id, name, role, ref, status, session_id, pid, worktree_dir, log_file,
		        started_at, updated_at, suspended_at, finished_at, shutdown_reason, resume_error, model
		 FROM agents ORDER BY started_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	agents, err := scanAgents(rows)
	if err != nil {
		log.Printf("warn: agents query: %v", err)
	}
	return agents
}

func (s *AgentStore) AddAgent(info domain.AgentInfo) error {
	var suspAt, finAt *string
	if info.SuspendedAt != nil {
		v := info.SuspendedAt.Format(time.RFC3339)
		suspAt = &v
	}
	if info.FinishedAt != nil {
		v := info.FinishedAt.Format(time.RFC3339)
		finAt = &v
	}
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO agents
		 (id, name, role, ref, status, session_id, pid, worktree_dir, log_file,
		  started_at, updated_at, suspended_at, finished_at, shutdown_reason, resume_error, model)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		info.ID, info.Name, info.Role, info.Ref, info.Status, info.SessionID, info.PID,
		info.WorktreeDir, info.LogFile,
		info.StartedAt.Format(time.RFC3339), info.UpdatedAt.Format(time.RFC3339),
		suspAt, finAt, info.ShutdownReason, info.ResumeError, info.Model,
	)
	if err != nil {
		return fmt.Errorf("agent store: add agent %s: %w", info.ID, err)
	}
	return nil
}

func (s *AgentStore) FindAgent(idPrefix string) (*domain.AgentInfo, error) {
	rows, err := s.db.Query(
		`SELECT id, name, role, ref, status, session_id, pid, worktree_dir, log_file,
		        started_at, updated_at, suspended_at, finished_at, shutdown_reason, resume_error, model
		 FROM agents WHERE id = ? OR id LIKE ?`, idPrefix, idPrefix+"%")
	if err != nil {
		return nil, fmt.Errorf("query agents: %w", err)
	}
	defer rows.Close()

	agents, err := scanAgents(rows)
	if err != nil {
		return nil, err
	}
	if len(agents) == 0 {
		return nil, fmt.Errorf("agent %s: %w", idPrefix, domain.ErrAgentNotFound)
	}
	return &agents[0], nil
}

func (s *AgentStore) FindByRef(ref string) *domain.AgentInfo {
	rows, err := s.db.Query(
		`SELECT id, name, role, ref, status, session_id, pid, worktree_dir, log_file,
		        started_at, updated_at, suspended_at, finished_at, shutdown_reason, resume_error, model
		 FROM agents WHERE ref = ? ORDER BY started_at DESC LIMIT 1`, ref)
	if err != nil {
		return nil
	}
	defer rows.Close()

	agents, err := scanAgents(rows)
	if err != nil {
		log.Printf("warn: find by ref: %v", err)
		return nil
	}
	if len(agents) == 0 {
		return nil
	}
	return &agents[0]
}

func (s *AgentStore) UpdateStatus(idPrefix, status string) error {
	now := time.Now().Format(time.RFC3339)
	a := domain.AgentInfo{Status: status}
	var err error
	if a.IsTerminal() {
		_, err = s.db.Exec(
			`UPDATE agents SET status = ?, updated_at = ?, finished_at = ? WHERE id = ? OR id LIKE ?`,
			status, now, now, idPrefix, idPrefix+"%")
	} else {
		_, err = s.db.Exec(
			`UPDATE agents SET status = ?, updated_at = ? WHERE id = ? OR id LIKE ?`,
			status, now, idPrefix, idPrefix+"%")
	}
	if err != nil {
		return fmt.Errorf("agent store: update status %s for %s: %w", status, idPrefix, err)
	}
	return nil
}

func (s *AgentStore) HaltAgent(idPrefix string) error {
	agent, err := s.FindAgent(idPrefix)
	if err != nil {
		return err
	}
	if agent.PID <= 0 {
		return fmt.Errorf("no PID recorded for agent %s", idPrefix)
	}
	proc, err := os.FindProcess(agent.PID)
	if err != nil {
		return fmt.Errorf("find process: %w", err)
	}
	return proc.Signal(syscall.SIGINT)
}

func (s *AgentStore) RemoveAgent(id string) error {
	res, err := s.db.Exec(`DELETE FROM agents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("agent %s: %w", id, domain.ErrAgentNotFound)
	}
	return nil
}

func (s *AgentStore) AgentsByStatus(statuses ...string) []domain.AgentInfo {
	if len(statuses) == 0 {
		return nil
	}
	placeholders := strings.Repeat("?,", len(statuses))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(statuses))
	for i, st := range statuses {
		args[i] = st
	}
	rows, err := s.db.Query(
		`SELECT id, name, role, ref, status, session_id, pid, worktree_dir, log_file,
		        started_at, updated_at, suspended_at, finished_at, shutdown_reason, resume_error, model
		 FROM agents WHERE status IN (`+placeholders+`) ORDER BY started_at DESC`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	agents, err := scanAgents(rows)
	if err != nil {
		log.Printf("warn: agents by status: %v", err)
	}
	return agents
}

// ListAgents returns a paginated list of agents, optionally filtered by statuses.
func (s *AgentStore) ListAgents(opts domain.PageOpts, statuses ...string) (domain.Page[domain.AgentInfo], error) {
	opts.Normalize()

	var whereParts []string
	var args []any
	if len(statuses) > 0 {
		placeholders := make([]string, len(statuses))
		for i, st := range statuses {
			placeholders[i] = "?"
			args = append(args, st)
		}
		whereParts = append(whereParts, "status IN ("+strings.Join(placeholders, ",")+")")
	}

	if opts.Cursor != "" {
		cur := domain.DecodeCursor(opts.Cursor)
		if cur.SortVal != "" {
			whereParts = append(whereParts, "(started_at < ? OR (started_at = ? AND id < ?))")
			args = append(args, cur.SortVal, cur.SortVal, cur.ID)
		}
	}

	where := ""
	if len(whereParts) > 0 {
		where = " WHERE " + strings.Join(whereParts, " AND ")
	}

	// Count total (without cursor filter).
	var countParts []string
	var countArgs []any
	if len(statuses) > 0 {
		placeholders := make([]string, len(statuses))
		for i, st := range statuses {
			placeholders[i] = "?"
			countArgs = append(countArgs, st)
		}
		countParts = append(countParts, "status IN ("+strings.Join(placeholders, ",")+")")
	}
	countWhere := ""
	if len(countParts) > 0 {
		countWhere = " WHERE " + strings.Join(countParts, " AND ")
	}
	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM agents"+countWhere, countArgs...).Scan(&total); err != nil {
		return domain.Page[domain.AgentInfo]{}, fmt.Errorf("count agents: %w", err)
	}

	query := `SELECT id, name, role, ref, status, session_id, pid, worktree_dir, log_file,
	          started_at, updated_at, suspended_at, finished_at, shutdown_reason, resume_error, model
	          FROM agents` + where + ` ORDER BY started_at DESC, id DESC LIMIT ?`
	args = append(args, opts.Limit+1)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return domain.Page[domain.AgentInfo]{}, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	agents, err := scanAgents(rows)
	if err != nil {
		return domain.Page[domain.AgentInfo]{}, err
	}
	var nextCursor string
	if len(agents) > opts.Limit {
		last := agents[opts.Limit-1]
		nextCursor = domain.EncodeCursor(last.StartedAt.Format(time.RFC3339), last.ID)
		agents = agents[:opts.Limit]
	}

	return domain.Page[domain.AgentInfo]{
		Items:      agents,
		NextCursor: nextCursor,
		TotalCount: total,
	}, nil
}

func scanAgents(rows *sql.Rows) ([]domain.AgentInfo, error) {
	var agents []domain.AgentInfo
	for rows.Next() {
		var a domain.AgentInfo
		var startedAt, updatedAt string
		var suspendedAt, finishedAt *string
		if err := rows.Scan(
			&a.ID, &a.Name, &a.Role, &a.Ref, &a.Status, &a.SessionID, &a.PID,
			&a.WorktreeDir, &a.LogFile,
			&startedAt, &updatedAt, &suspendedAt, &finishedAt, &a.ShutdownReason, &a.ResumeError, &a.Model,
		); err != nil {
			log.Printf("warn: scan agent row: %v", err)
			continue
		}
		a.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
		a.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		if suspendedAt != nil {
			t, _ := time.Parse(time.RFC3339, *suspendedAt)
			a.SuspendedAt = &t
		}
		if finishedAt != nil {
			t, _ := time.Parse(time.RFC3339, *finishedAt)
			a.FinishedAt = &t
		}
		agents = append(agents, a)
	}
	if err := rows.Err(); err != nil {
		return agents, fmt.Errorf("iterate agent rows: %w", err)
	}
	return agents, nil
}
