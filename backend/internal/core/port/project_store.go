package port

import "kiloforge/internal/core/domain"

// ProjectStore persists and retrieves registered projects.
type ProjectStore interface {
	Get(slug string) (domain.Project, error)
	List() []domain.Project
	// ListPaginated returns a paginated list of projects.
	ListPaginated(opts domain.PageOpts) (domain.Page[domain.Project], error)
	Add(p domain.Project) error
	Remove(slug string) error
	FindByRepoName(name string) (domain.Project, bool)
	FindByDir(dir string) (domain.Project, bool)
	Save() error
}
