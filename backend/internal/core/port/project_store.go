package port

import "kiloforge/internal/core/domain"

// ProjectStore persists and retrieves registered projects.
type ProjectStore interface {
	Get(slug string) (domain.Project, bool)
	List() []domain.Project
	Add(p domain.Project) error
	Remove(slug string) error
	FindByRepoName(name string) (domain.Project, bool)
	FindByDir(dir string) (domain.Project, bool)
	Save() error
}
