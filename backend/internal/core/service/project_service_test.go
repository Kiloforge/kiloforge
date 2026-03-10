package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"kiloforge/internal/core/domain"
)

// mockGiteaClient is a mock Gitea client for testing ProjectService.
type mockGiteaClient struct {
	createRepoErr   error
	deleteRepoErr   error
	deleteHooksErr  error
	createHookErr   error
	deletedRepo     string
	deletedHooksFor string
}

func (m *mockGiteaClient) CreateRepo(_ context.Context, name string) error {
	return m.createRepoErr
}

func (m *mockGiteaClient) CreateWebhook(_ context.Context, _ string, _ int) error {
	return m.createHookErr
}

func (m *mockGiteaClient) DeleteRepo(_ context.Context, repoName string) error {
	m.deletedRepo = repoName
	return m.deleteRepoErr
}

func (m *mockGiteaClient) DeleteAllWebhooks(_ context.Context, repoName string) error {
	m.deletedHooksFor = repoName
	return m.deleteHooksErr
}

func (m *mockGiteaClient) BaseURL() string {
	return "http://localhost:3000"
}

// mockProjectStore is an in-memory project store for testing.
type mockProjectStore struct {
	projects map[string]domain.Project
	saveErr  error
}

func newMockProjectStore() *mockProjectStore {
	return &mockProjectStore{projects: map[string]domain.Project{}}
}

func (m *mockProjectStore) Get(slug string) (domain.Project, error) {
	p, ok := m.projects[slug]
	if !ok {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	return p, nil
}

func (m *mockProjectStore) List() []domain.Project {
	result := make([]domain.Project, 0, len(m.projects))
	for _, p := range m.projects {
		result = append(result, p)
	}
	return result
}

func (m *mockProjectStore) Add(p domain.Project) error {
	if _, exists := m.projects[p.Slug]; exists {
		return fmt.Errorf("project %q already exists", p.Slug)
	}
	m.projects[p.Slug] = p
	return nil
}

func (m *mockProjectStore) Remove(slug string) error {
	if _, exists := m.projects[slug]; !exists {
		return fmt.Errorf("project %q not found", slug)
	}
	delete(m.projects, slug)
	return nil
}

func (m *mockProjectStore) FindByRepoName(name string) (domain.Project, bool) {
	for _, p := range m.projects {
		if p.RepoName == name {
			return p, true
		}
	}
	return domain.Project{}, false
}

func (m *mockProjectStore) FindByDir(dir string) (domain.Project, bool) {
	for _, p := range m.projects {
		if p.ProjectDir == dir {
			return p, true
		}
	}
	return domain.Project{}, false
}

func (m *mockProjectStore) ListPaginated(_ domain.PageOpts) (domain.Page[domain.Project], error) {
	return domain.Page[domain.Project]{Items: m.List()}, nil
}

func (m *mockProjectStore) Save() error {
	return m.saveErr
}

func TestProjectService_ListProjects(t *testing.T) {
	t.Parallel()

	store := newMockProjectStore()
	store.projects["app1"] = domain.Project{Slug: "app1"}
	store.projects["app2"] = domain.Project{Slug: "app2"}

	svc := NewProjectService(store, nil, ProjectServiceConfig{})
	projects := svc.ListProjects()

	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestProjectService_GetProject(t *testing.T) {
	t.Parallel()

	store := newMockProjectStore()
	store.projects["myapp"] = domain.Project{Slug: "myapp", RepoName: "myapp"}

	svc := NewProjectService(store, nil, ProjectServiceConfig{})

	t.Run("found", func(t *testing.T) {
		p, err := svc.GetProject("myapp")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Slug != "myapp" {
			t.Errorf("expected slug myapp, got %s", p.Slug)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetProject("nonexistent")
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, domain.ErrProjectNotFound) {
			t.Errorf("expected ErrProjectNotFound, got: %v", err)
		}
	})
}

func TestProjectService_RemoveProject(t *testing.T) {
	t.Parallel()

	store := newMockProjectStore()
	store.projects["myapp"] = domain.Project{Slug: "myapp", RepoName: "myapp"}
	gitea := &mockGiteaClient{}

	svc := NewProjectService(store, gitea, ProjectServiceConfig{
		DataDir: t.TempDir(),
	})

	err := svc.RemoveProject(context.Background(), "myapp", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := store.Get("myapp"); err == nil {
		t.Error("project should have been removed from store")
	}
}

func TestProjectService_RemoveProject_NotFound(t *testing.T) {
	t.Parallel()

	store := newMockProjectStore()
	gitea := &mockGiteaClient{}

	svc := NewProjectService(store, gitea, ProjectServiceConfig{
		DataDir: t.TempDir(),
	})

	err := svc.RemoveProject(context.Background(), "nonexistent", false)
	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Errorf("expected ErrProjectNotFound, got: %v", err)
	}
}

func TestProjectService_RemoveProject_WithCleanup(t *testing.T) {
	t.Parallel()

	store := newMockProjectStore()
	store.projects["myapp"] = domain.Project{Slug: "myapp", RepoName: "myapp"}
	gitea := &mockGiteaClient{}

	svc := NewProjectService(store, gitea, ProjectServiceConfig{
		DataDir: t.TempDir(),
	})

	err := svc.RemoveProject(context.Background(), "myapp", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gitea.deletedRepo != "myapp" {
		t.Errorf("expected Gitea repo deletion for 'myapp', got %q", gitea.deletedRepo)
	}
	if gitea.deletedHooksFor != "myapp" {
		t.Errorf("expected webhook deletion for 'myapp', got %q", gitea.deletedHooksFor)
	}
}

func TestProjectService_AddProject_DuplicateSlug(t *testing.T) {
	t.Parallel()

	store := newMockProjectStore()
	store.projects["myapp"] = domain.Project{Slug: "myapp", RepoName: "myapp"}
	gitea := &mockGiteaClient{}

	svc := NewProjectService(store, gitea, ProjectServiceConfig{
		DataDir: t.TempDir(),
	})

	_, err := svc.AddProject(context.Background(), "git@github.com:user/myapp.git", "")
	if err == nil {
		t.Fatal("expected error for duplicate")
	}

	if !errors.Is(err, domain.ErrProjectExists) {
		t.Errorf("expected ErrProjectExists, got: %v", err)
	}
}

func TestProjectService_AddProject_InvalidURL(t *testing.T) {
	t.Parallel()

	store := newMockProjectStore()
	gitea := &mockGiteaClient{}

	svc := NewProjectService(store, gitea, ProjectServiceConfig{
		DataDir: t.TempDir(),
	})

	_, err := svc.AddProject(context.Background(), "/local/path", "")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestProjectService_CreateProject_Success(t *testing.T) {
	t.Parallel()

	store := newMockProjectStore()
	gitea := &mockGiteaClient{}
	dataDir := t.TempDir()

	svc := NewProjectService(store, gitea, ProjectServiceConfig{
		DataDir:          dataDir,
		OrchestratorPort: 4001,
		GiteaAdminUser:   "kf-admin",
		APIToken:         "test-token",
	})

	result, err := svc.CreateProject(context.Background(), "my-new-project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Project.Slug != "my-new-project" {
		t.Errorf("expected slug my-new-project, got %s", result.Project.Slug)
	}
	if result.Project.RepoName != "my-new-project" {
		t.Errorf("expected repo name my-new-project, got %s", result.Project.RepoName)
	}
	if result.Project.OriginRemote != "" {
		t.Errorf("expected empty origin remote, got %s", result.Project.OriginRemote)
	}
	if !result.Project.Active {
		t.Error("expected project to be active")
	}

	p, err := store.Get("my-new-project")
	if err != nil {
		t.Fatalf("project not found in store: %v", err)
	}
	if p.Slug != "my-new-project" {
		t.Errorf("store slug mismatch: got %s", p.Slug)
	}
}

func TestProjectService_CreateProject_EmptyName(t *testing.T) {
	t.Parallel()

	store := newMockProjectStore()
	gitea := &mockGiteaClient{}

	svc := NewProjectService(store, gitea, ProjectServiceConfig{DataDir: t.TempDir()})

	_, err := svc.CreateProject(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestProjectService_CreateProject_DuplicateSlug(t *testing.T) {
	t.Parallel()

	store := newMockProjectStore()
	store.projects["existing"] = domain.Project{Slug: "existing", RepoName: "existing"}
	gitea := &mockGiteaClient{}

	svc := NewProjectService(store, gitea, ProjectServiceConfig{DataDir: t.TempDir()})

	_, err := svc.CreateProject(context.Background(), "existing")
	if err == nil {
		t.Fatal("expected error for duplicate slug")
	}
	if !errors.Is(err, domain.ErrProjectExists) {
		t.Errorf("expected ErrProjectExists, got: %v", err)
	}
}

func TestProjectService_CreateProject_RollbackOnGiteaFailure(t *testing.T) {
	t.Parallel()

	store := newMockProjectStore()
	gitea := &mockGiteaClient{createRepoErr: fmt.Errorf("gitea down")}
	dataDir := t.TempDir()

	svc := NewProjectService(store, gitea, ProjectServiceConfig{DataDir: dataDir})

	_, err := svc.CreateProject(context.Background(), "fail-project")
	if err == nil {
		t.Fatal("expected error")
	}

	if _, err := store.Get("fail-project"); err == nil {
		t.Error("project should not be in store after rollback")
	}
}

func TestIsRemoteURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  bool
	}{
		{"git@github.com:user/repo.git", true},
		{"https://github.com/user/repo.git", true},
		{"http://github.com/user/repo.git", true},
		{"ssh://git@host/user/repo.git", true},
		{"/local/path", false},
		{"relative/path", false},
	}

	for _, tt := range tests {
		if got := isRemoteURL(tt.input); got != tt.want {
			t.Errorf("isRemoteURL(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestRepoNameFromURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"git@github.com:user/myapp.git", "myapp"},
		{"https://github.com/user/myapp.git", "myapp"},
		{"ssh://git@host/user/myapp.git", "myapp"},
		{"https://github.com/user/myapp", "myapp"},
	}

	for _, tt := range tests {
		got, err := repoNameFromURL(tt.input)
		if err != nil {
			t.Errorf("repoNameFromURL(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("repoNameFromURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
