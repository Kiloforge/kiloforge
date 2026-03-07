package jsonfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"crelay/internal/core/domain"
)

const registryFileName = "projects.json"

// ProjectStore persists projects to a JSON file.
type ProjectStore struct {
	Version  int                       `json:"version"`
	Projects map[string]domain.Project `json:"projects"`
	dataDir  string
}

// LoadProjectStore reads the project registry from the data directory.
func LoadProjectStore(dataDir string) (*ProjectStore, error) {
	path := filepath.Join(dataDir, registryFileName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &ProjectStore{Version: 1, Projects: map[string]domain.Project{}, dataDir: dataDir}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}

	var store ProjectStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	if store.Projects == nil {
		store.Projects = map[string]domain.Project{}
	}
	store.dataDir = dataDir
	return &store, nil
}

// Save writes the registry to the data directory.
func (s *ProjectStore) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}
	path := filepath.Join(s.dataDir, registryFileName)
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// Get returns a project by slug.
func (s *ProjectStore) Get(slug string) (domain.Project, bool) {
	p, ok := s.Projects[slug]
	return p, ok
}

// Add registers a new project. Returns an error if the slug already exists.
func (s *ProjectStore) Add(p domain.Project) error {
	if _, exists := s.Projects[p.Slug]; exists {
		return fmt.Errorf("project %q already registered", p.Slug)
	}
	s.Projects[p.Slug] = p
	return nil
}

// FindByRepoName looks up a project by its Gitea repo name.
func (s *ProjectStore) FindByRepoName(repoName string) (domain.Project, bool) {
	for _, p := range s.Projects {
		if p.RepoName == repoName {
			return p, true
		}
	}
	return domain.Project{}, false
}

// FindByDir looks up a project by its project directory path.
func (s *ProjectStore) FindByDir(dir string) (domain.Project, bool) {
	for _, p := range s.Projects {
		if p.ProjectDir == dir {
			return p, true
		}
	}
	return domain.Project{}, false
}

// List returns all projects sorted by slug.
func (s *ProjectStore) List() []domain.Project {
	projects := make([]domain.Project, 0, len(s.Projects))
	for _, p := range s.Projects {
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
