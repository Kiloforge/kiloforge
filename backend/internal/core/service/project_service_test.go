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

func (m *mockProjectStore) Get(slug string) (domain.Project, bool) {
	p, ok := m.projects[slug]
	return p, ok
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

func (m *mockProjectStore) Save() error {
	return m.saveErr
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

	if _, exists := store.Get("myapp"); exists {
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

	var notFound *ProjectNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected ProjectNotFoundError, got %T: %v", err, err)
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

	var existsErr *ProjectExistsError
	if !errors.As(err, &existsErr) {
		t.Errorf("expected ProjectExistsError, got %T: %v", err, err)
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
