package service

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"kiloforge/internal/core/domain"
)

// ProjectGiteaClient abstracts the Gitea operations needed by ProjectService.
type ProjectGiteaClient interface {
	CreateRepo(ctx context.Context, name string) error
	CreateWebhook(ctx context.Context, repoName string, orchPort int) error
	DeleteRepo(ctx context.Context, repoName string) error
	DeleteAllWebhooks(ctx context.Context, repoName string) error
	BaseURL() string
}

// ProjectStoreWriter provides read/write access to the project registry.
type ProjectStoreWriter interface {
	Get(slug string) (domain.Project, bool)
	List() []domain.Project
	Add(p domain.Project) error
	Remove(slug string) error
	Save() error
}

// ProjectServiceConfig holds configuration needed by ProjectService.
type ProjectServiceConfig struct {
	DataDir          string
	OrchestratorPort int
	GiteaAdminUser   string
	APIToken         string
}

// ProjectService handles project registration and removal.
type ProjectService struct {
	store  ProjectStoreWriter
	gitea  ProjectGiteaClient
	config ProjectServiceConfig
}

// NewProjectService creates a new ProjectService.
func NewProjectService(store ProjectStoreWriter, gitea ProjectGiteaClient, cfg ProjectServiceConfig) *ProjectService {
	return &ProjectService{
		store:  store,
		gitea:  gitea,
		config: cfg,
	}
}

// AddProjectResult contains details about a newly added project.
type AddProjectResult struct {
	Project   domain.Project
	EmptyRepo bool // true if the repo had no commits (push was skipped)
}

// AddProjectOpts contains optional parameters for AddProject.
type AddProjectOpts struct {
	SSHKeyPath string // Path to SSH private key for cloning.
}

// AddProject registers a new project from a remote URL.
func (s *ProjectService) AddProject(ctx context.Context, remoteURL, name string, opts ...AddProjectOpts) (*AddProjectResult, error) {
	var opt AddProjectOpts
	if len(opts) > 0 {
		opt = opts[0]
	}
	if !isRemoteURL(remoteURL) {
		return nil, fmt.Errorf("invalid remote URL: %s", remoteURL)
	}

	repoName, err := repoNameFromURL(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("parse remote URL: %w", err)
	}

	slug := repoName
	if name != "" {
		slug = name
	}

	if _, exists := s.store.Get(slug); exists {
		return nil, &ProjectExistsError{Slug: slug}
	}

	// Clean up orphaned clone directory from a previous failed attempt.
	cloneDir := filepath.Join(s.config.DataDir, "repos", slug)
	if _, err := os.Stat(cloneDir); err == nil {
		if _, registered := s.store.Get(slug); !registered {
			os.RemoveAll(cloneDir)
		}
	}

	// Clone remote into managed directory.
	if _, err := os.Stat(cloneDir); os.IsNotExist(err) {
		cmd := exec.CommandContext(ctx, "git", "clone", remoteURL, cloneDir)
		if opt.SSHKeyPath != "" {
			cmd.Env = append(os.Environ(),
				fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o IdentitiesOnly=yes", opt.SSHKeyPath),
			)
		}
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("clone: %s: %w", string(out), err)
		}
	}

	// Track whether we created the Gitea repo (vs. pre-existing) for rollback.
	giteaRepoCreated := false

	// Create Gitea repo.
	if err := s.gitea.CreateRepo(ctx, repoName); err != nil {
		if !strings.Contains(err.Error(), "409") {
			os.RemoveAll(cloneDir)
			return nil, fmt.Errorf("create gitea repo: %w", err)
		}
	} else {
		giteaRepoCreated = true
	}

	// rollback cleans up the Gitea repo (if we created it) and the clone dir.
	rollback := func() {
		if giteaRepoCreated {
			_ = s.gitea.DeleteRepo(ctx, repoName)
		}
		os.RemoveAll(cloneDir)
	}

	// Add gitea remote and push (embed API token for HTTP auth).
	displayRemoteURL := fmt.Sprintf("%s/%s/%s.git", s.gitea.BaseURL(), s.config.GiteaAdminUser, repoName)
	parsedURL, err := url.Parse(displayRemoteURL)
	if err != nil {
		rollback()
		return nil, fmt.Errorf("parse gitea URL: %w", err)
	}
	parsedURL.User = url.UserPassword(s.config.GiteaAdminUser, s.config.APIToken)
	giteaRemoteURL := parsedURL.String()

	_ = exec.CommandContext(ctx, "git", "-C", cloneDir, "remote", "remove", "gitea").Run()
	if err := exec.CommandContext(ctx, "git", "-C", cloneDir, "remote", "add", "gitea", giteaRemoteURL).Run(); err != nil {
		rollback()
		return nil, fmt.Errorf("add gitea remote: %w", err)
	}

	// Skip push for empty repos (no commits).
	empty := !hasCommits(ctx, cloneDir)
	if !empty {
		if out, err := exec.CommandContext(ctx, "git", "-C", cloneDir, "push", "-u", "gitea", "main").CombinedOutput(); err != nil {
			rollback()
			return nil, fmt.Errorf("push to gitea: %s: %w", string(out), err)
		}
	}

	// Create webhook (non-blocking — failure doesn't trigger rollback).
	_ = s.gitea.CreateWebhook(ctx, repoName, s.config.OrchestratorPort)

	// Create project data directory.
	logsDir := filepath.Join(s.config.DataDir, "projects", slug, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		rollback()
		return nil, fmt.Errorf("create project dir: %w", err)
	}

	// Register in store.
	p := domain.Project{
		Slug:         slug,
		RepoName:     repoName,
		ProjectDir:   cloneDir,
		OriginRemote: remoteURL,
		RegisteredAt: time.Now().Truncate(time.Second),
		Active:       true,
	}
	if err := s.store.Add(p); err != nil {
		rollback()
		return nil, fmt.Errorf("register project: %w", err)
	}
	if err := s.store.Save(); err != nil {
		rollback()
		return nil, fmt.Errorf("save registry: %w", err)
	}

	result := &AddProjectResult{Project: p}
	if empty {
		result.EmptyRepo = true
	}
	return result, nil
}

// Store returns the underlying project store writer.
func (s *ProjectService) Store() ProjectStoreWriter {
	return s.store
}

// ListProjects returns all registered projects.
func (s *ProjectService) ListProjects() []domain.Project {
	return s.store.List()
}

// GetProject returns a project by slug, or an error if not found.
func (s *ProjectService) GetProject(slug string) (*domain.Project, error) {
	p, ok := s.store.Get(slug)
	if !ok {
		return nil, &ProjectNotFoundError{Slug: slug}
	}
	return &p, nil
}

// RemoveProject deregisters a project. If cleanup is true, also deletes
// the Gitea repo and local filesystem data.
func (s *ProjectService) RemoveProject(ctx context.Context, slug string, cleanup bool) error {
	p, exists := s.store.Get(slug)
	if !exists {
		return &ProjectNotFoundError{Slug: slug}
	}

	if cleanup {
		// Delete webhooks and repo from Gitea (best-effort).
		_ = s.gitea.DeleteAllWebhooks(ctx, p.RepoName)
		_ = s.gitea.DeleteRepo(ctx, p.RepoName)

		// Remove local directories.
		repoDir := filepath.Join(s.config.DataDir, "repos", slug)
		_ = os.RemoveAll(repoDir)
		projectDir := filepath.Join(s.config.DataDir, "projects", slug)
		_ = os.RemoveAll(projectDir)
	}

	if err := s.store.Remove(slug); err != nil {
		return fmt.Errorf("remove from store: %w", err)
	}
	if err := s.store.Save(); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	return nil
}

// ProjectExistsError indicates a project with the given slug already exists.
type ProjectExistsError struct {
	Slug string
}

func (e *ProjectExistsError) Error() string {
	return fmt.Sprintf("project %q already exists", e.Slug)
}

// ProjectNotFoundError indicates a project with the given slug was not found.
type ProjectNotFoundError struct {
	Slug string
}

func (e *ProjectNotFoundError) Error() string {
	return fmt.Sprintf("project %q not found", e.Slug)
}

// isRemoteURL returns true if the argument looks like a git remote URL.
func isRemoteURL(arg string) bool {
	if strings.HasPrefix(arg, "https://") || strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "ssh://") {
		return true
	}
	if strings.Contains(arg, "@") && strings.Contains(arg, ":") {
		return true
	}
	return false
}

// hasCommits returns true if the git repository at dir has at least one commit.
func hasCommits(ctx context.Context, dir string) bool {
	return exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "HEAD").Run() == nil
}

// repoNameFromURL extracts the repository name from a git remote URL.
func repoNameFromURL(rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("empty URL")
	}

	var path string
	if idx := strings.Index(rawURL, ":"); idx != -1 && strings.Contains(rawURL[:idx], "@") && !strings.HasPrefix(rawURL, "ssh://") {
		path = rawURL[idx+1:]
	} else {
		u := rawURL
		for _, prefix := range []string{"https://", "http://", "ssh://"} {
			if strings.HasPrefix(u, prefix) {
				u = u[len(prefix):]
				break
			}
		}
		if idx := strings.Index(u, "/"); idx != -1 {
			path = u[idx+1:]
		}
	}

	if path == "" {
		return "", fmt.Errorf("cannot extract repo name from URL: %s", rawURL)
	}

	path = strings.TrimRight(path, "/")
	name := filepath.Base(path)
	name = strings.TrimSuffix(name, ".git")

	if name == "" || name == "." {
		return "", fmt.Errorf("cannot extract repo name from URL: %s", rawURL)
	}

	return name, nil
}
