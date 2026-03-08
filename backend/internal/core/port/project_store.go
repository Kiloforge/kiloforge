package port

import "kiloforge/internal/core/domain"

// ProjectStore persists and retrieves registered projects.
type ProjectStore interface {
	Get(slug string) (domain.Project, error)
	List() []domain.Project
	Add(p domain.Project) error
	FindByRepoName(name string) (domain.Project, error)
	FindByDir(dir string) (domain.Project, error)
	Save() error
}
