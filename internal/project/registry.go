package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"crelay/internal/core/domain"
)

const registryFileName = "projects.json"

// Registry tracks all registered projects.
type Registry struct {
	Version  int                       `json:"version"`
	Projects map[string]domain.Project `json:"projects"`
}

// LoadRegistry reads the project registry from the data directory.
// Returns an empty registry if the file does not exist.
func LoadRegistry(dataDir string) (*Registry, error) {
	path := filepath.Join(dataDir, registryFileName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Registry{Version: 1, Projects: map[string]domain.Project{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	if reg.Projects == nil {
		reg.Projects = map[string]domain.Project{}
	}
	return &reg, nil
}

// Save writes the registry to the data directory.
func (r *Registry) Save(dataDir string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}
	path := filepath.Join(dataDir, registryFileName)
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// Get returns a project by slug.
func (r *Registry) Get(slug string) (domain.Project, bool) {
	p, ok := r.Projects[slug]
	return p, ok
}

// Add registers a new project. Returns an error if the slug already exists.
func (r *Registry) Add(p domain.Project) error {
	if _, exists := r.Projects[p.Slug]; exists {
		return fmt.Errorf("project %q already registered", p.Slug)
	}
	r.Projects[p.Slug] = p
	return nil
}

// FindByRepoName looks up a project by its Gitea repo name.
func (r *Registry) FindByRepoName(repoName string) (domain.Project, bool) {
	for _, p := range r.Projects {
		if p.RepoName == repoName {
			return p, true
		}
	}
	return domain.Project{}, false
}

// FindByDir looks up a project by its project directory path.
func (r *Registry) FindByDir(dir string) (domain.Project, bool) {
	for _, p := range r.Projects {
		if p.ProjectDir == dir {
			return p, true
		}
	}
	return domain.Project{}, false
}

// List returns all projects sorted by slug.
func (r *Registry) List() []domain.Project {
	projects := make([]domain.Project, 0, len(r.Projects))
	for _, p := range r.Projects {
		projects = append(projects, p)
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Slug < projects[j].Slug
	})
	return projects
}

// EnsureProjectDir creates the project data directory structure.
func EnsureProjectDir(dataDir, slug string) error {
	logsDir := filepath.Join(dataDir, "projects", slug, "logs")
	return os.MkdirAll(logsDir, 0o755)
}
