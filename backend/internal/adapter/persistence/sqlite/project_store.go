package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

var _ port.ProjectStore = (*ProjectStore)(nil)

// ProjectStore persists projects to SQLite.
type ProjectStore struct {
	db *sql.DB
}

// NewProjectStore creates a ProjectStore backed by the given database.
func NewProjectStore(db *sql.DB) *ProjectStore {
	return &ProjectStore{db: db}
}

func (s *ProjectStore) Get(slug string) (domain.Project, error) {
	var p domain.Project
	var regAt string
	var active int
	err := s.db.QueryRow(
		`SELECT slug, repo_name, project_dir, origin_remote, ssh_key_path, registered_at, active
		 FROM projects WHERE slug = ?`, slug,
	).Scan(&p.Slug, &p.RepoName, &p.ProjectDir, &p.OriginRemote, &p.SSHKeyPath, &regAt, &active)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Project{}, fmt.Errorf("project %s: %w", slug, domain.ErrProjectNotFound)
		}
		return domain.Project{}, fmt.Errorf("get project %s: %w", slug, err)
	}
	p.RegisteredAt, _ = time.Parse(time.RFC3339, regAt)
	p.Active = active != 0
	return p, nil
}

func (s *ProjectStore) List() []domain.Project {
	rows, err := s.db.Query(
		`SELECT slug, repo_name, project_dir, origin_remote, ssh_key_path, registered_at, active
		 FROM projects ORDER BY slug`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		var p domain.Project
		var regAt string
		var active int
		if err := rows.Scan(&p.Slug, &p.RepoName, &p.ProjectDir, &p.OriginRemote, &p.SSHKeyPath, &regAt, &active); err != nil {
			continue
		}
		p.RegisteredAt, _ = time.Parse(time.RFC3339, regAt)
		p.Active = active != 0
		projects = append(projects, p)
	}
	return projects
}

func (s *ProjectStore) Add(p domain.Project) error {
	_, err := s.db.Exec(
		`INSERT INTO projects (slug, repo_name, project_dir, origin_remote, ssh_key_path, registered_at, active)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.Slug, p.RepoName, p.ProjectDir, p.OriginRemote, p.SSHKeyPath,
		p.RegisteredAt.Format(time.RFC3339), boolToInt(p.Active),
	)
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}
	return nil
}

func (s *ProjectStore) Remove(slug string) error {
	res, err := s.db.Exec("DELETE FROM projects WHERE slug = ?", slug)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("project %q not found", slug)
	}
	return nil
}

func (s *ProjectStore) FindByRepoName(name string) (domain.Project, bool) {
	var p domain.Project
	var regAt string
	var active int
	err := s.db.QueryRow(
		`SELECT slug, repo_name, project_dir, origin_remote, ssh_key_path, registered_at, active
		 FROM projects WHERE repo_name = ?`, name,
	).Scan(&p.Slug, &p.RepoName, &p.ProjectDir, &p.OriginRemote, &p.SSHKeyPath, &regAt, &active)
	if err != nil {
		return domain.Project{}, false
	}
	p.RegisteredAt, _ = time.Parse(time.RFC3339, regAt)
	p.Active = active != 0
	return p, true
}

func (s *ProjectStore) FindByDir(dir string) (domain.Project, bool) {
	var p domain.Project
	var regAt string
	var active int
	err := s.db.QueryRow(
		`SELECT slug, repo_name, project_dir, origin_remote, ssh_key_path, registered_at, active
		 FROM projects WHERE project_dir = ?`, dir,
	).Scan(&p.Slug, &p.RepoName, &p.ProjectDir, &p.OriginRemote, &p.SSHKeyPath, &regAt, &active)
	if err != nil {
		return domain.Project{}, false
	}
	p.RegisteredAt, _ = time.Parse(time.RFC3339, regAt)
	p.Active = active != 0
	return p, true
}

// ListPaginated returns a paginated list of projects ordered by slug ASC.
func (s *ProjectStore) ListPaginated(opts domain.PageOpts) (domain.Page[domain.Project], error) {
	opts.Normalize()

	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM projects").Scan(&total); err != nil {
		return domain.Page[domain.Project]{}, fmt.Errorf("count projects: %w", err)
	}

	var args []any
	where := ""
	if opts.Cursor != "" {
		cur := domain.DecodeCursor(opts.Cursor)
		if cur.SortVal != "" {
			where = " WHERE slug > ?"
			args = append(args, cur.SortVal)
		}
	}

	query := `SELECT slug, repo_name, project_dir, origin_remote, ssh_key_path, registered_at, active
	          FROM projects` + where + ` ORDER BY slug ASC LIMIT ?`
	args = append(args, opts.Limit+1)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return domain.Page[domain.Project]{}, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		var p domain.Project
		var regAt string
		var active int
		if err := rows.Scan(&p.Slug, &p.RepoName, &p.ProjectDir, &p.OriginRemote, &p.SSHKeyPath, &regAt, &active); err != nil {
			continue
		}
		p.RegisteredAt, _ = time.Parse(time.RFC3339, regAt)
		p.Active = active != 0
		projects = append(projects, p)
	}

	var nextCursor string
	if len(projects) > opts.Limit {
		last := projects[opts.Limit-1]
		nextCursor = domain.EncodeCursor(last.Slug, last.Slug)
		projects = projects[:opts.Limit]
	}

	return domain.Page[domain.Project]{
		Items:      projects,
		NextCursor: nextCursor,
		TotalCount: total,
	}, nil
}

// Save is a no-op for SQLite — writes are immediate.
func (s *ProjectStore) Save() error { return nil }

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
