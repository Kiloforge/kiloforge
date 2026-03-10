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
	"kiloforge/internal/core/port"
)

// ProjectServiceConfig holds configuration needed by ProjectService.
type ProjectServiceConfig struct {
	DataDir          string
	OrchestratorPort int
	GiteaAdminUser   string
	APIToken         string
}

// ProjectService handles project registration and removal.
type ProjectService struct {
	store  port.ProjectStore
	gitea  port.ProjectGiteaClient
	config ProjectServiceConfig
}

// NewProjectService creates a new ProjectService.
func NewProjectService(store port.ProjectStore, gitea port.ProjectGiteaClient, cfg ProjectServiceConfig) *ProjectService {
	return &ProjectService{
		store:  store,
		gitea:  gitea,
		config: cfg,
	}
}

// AddProject registers a new project from a remote URL.
func (s *ProjectService) AddProject(ctx context.Context, remoteURL, name string, opts ...domain.AddProjectOpts) (*domain.AddProjectResult, error) {
	var opt domain.AddProjectOpts
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

	if _, err := s.store.Get(slug); err == nil {
		return nil, fmt.Errorf("project %s: %w", slug, domain.ErrProjectExists)
	}

	// Clean up orphaned clone directory from a previous failed attempt.
	cloneDir := filepath.Join(s.config.DataDir, "repos", slug)
	if _, err := os.Stat(cloneDir); err == nil {
		if _, err := s.store.Get(slug); err != nil {
			os.RemoveAll(cloneDir)
		}
	}

	// Clone remote into managed directory.
	if _, err := os.Stat(cloneDir); os.IsNotExist(err) {
		cmd := exec.CommandContext(ctx, "git", "clone", remoteURL, cloneDir)
		cmd.Env = cleanGitEnv()
		if opt.SSHKeyPath != "" {
			cmd.Env = append(cmd.Env,
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

	rmRemote := exec.CommandContext(ctx, "git", "-C", cloneDir, "remote", "remove", "gitea")
	rmRemote.Env = cleanGitEnv()
	_ = rmRemote.Run()
	addRemote := exec.CommandContext(ctx, "git", "-C", cloneDir, "remote", "add", "gitea", giteaRemoteURL)
	addRemote.Env = cleanGitEnv()
	if err := addRemote.Run(); err != nil {
		rollback()
		return nil, fmt.Errorf("add gitea remote: %w", err)
	}

	// Skip push for empty repos (no commits).
	empty := !hasCommits(ctx, cloneDir)
	if !empty {
		pushCmd := exec.CommandContext(ctx, "git", "-C", cloneDir, "push", "-u", "gitea", "main")
		pushCmd.Env = cleanGitEnv()
		if out, err := pushCmd.CombinedOutput(); err != nil {
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

	result := &domain.AddProjectResult{Project: p}
	if empty {
		result.EmptyRepo = true
	}
	return result, nil
}

// CreateProject creates a new project from scratch (no remote URL).
// It initializes a local git repo, creates a Gitea repo, adds a gitea remote,
// creates a webhook, and registers the project in the store.
func (s *ProjectService) CreateProject(ctx context.Context, name string) (*domain.AddProjectResult, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required for creating a project from scratch")
	}

	if _, err := s.store.Get(name); err == nil {
		return nil, fmt.Errorf("project %s: %w", name, domain.ErrProjectExists)
	}

	repoDir := filepath.Join(s.config.DataDir, "repos", name)

	// Clean up orphaned directory from a previous failed attempt.
	if _, err := os.Stat(repoDir); err == nil {
		if _, err := s.store.Get(name); err != nil {
			os.RemoveAll(repoDir)
		}
	}

	// Initialize a fresh git repository.
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		return nil, fmt.Errorf("create repo dir: %w", err)
	}
	initCmd := exec.CommandContext(ctx, "git", "init", repoDir)
	initCmd.Env = cleanGitEnv()
	if out, err := initCmd.CombinedOutput(); err != nil {
		os.RemoveAll(repoDir)
		return nil, fmt.Errorf("git init: %s: %w", string(out), err)
	}

	// Track whether we created the Gitea repo for rollback.
	giteaRepoCreated := false

	if err := s.gitea.CreateRepo(ctx, name); err != nil {
		if !strings.Contains(err.Error(), "409") {
			os.RemoveAll(repoDir)
			return nil, fmt.Errorf("create gitea repo: %w", err)
		}
	} else {
		giteaRepoCreated = true
	}

	rollback := func() {
		if giteaRepoCreated {
			_ = s.gitea.DeleteRepo(ctx, name)
		}
		os.RemoveAll(repoDir)
	}

	// Add gitea remote (embed API token for HTTP auth).
	displayRemoteURL := fmt.Sprintf("%s/%s/%s.git", s.gitea.BaseURL(), s.config.GiteaAdminUser, name)
	parsedURL, err := url.Parse(displayRemoteURL)
	if err != nil {
		rollback()
		return nil, fmt.Errorf("parse gitea URL: %w", err)
	}
	parsedURL.User = url.UserPassword(s.config.GiteaAdminUser, s.config.APIToken)
	giteaRemoteURL := parsedURL.String()

	addRemoteCmd := exec.CommandContext(ctx, "git", "-C", repoDir, "remote", "add", "gitea", giteaRemoteURL)
	addRemoteCmd.Env = cleanGitEnv()
	if err := addRemoteCmd.Run(); err != nil {
		rollback()
		return nil, fmt.Errorf("add gitea remote: %w", err)
	}

	_ = s.gitea.CreateWebhook(ctx, name, s.config.OrchestratorPort)

	logsDir := filepath.Join(s.config.DataDir, "projects", name, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		rollback()
		return nil, fmt.Errorf("create project dir: %w", err)
	}

	p := domain.Project{
		Slug:         name,
		RepoName:     name,
		ProjectDir:   repoDir,
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

	return &domain.AddProjectResult{Project: p}, nil
}

// Store returns the underlying project store.
func (s *ProjectService) Store() port.ProjectStore {
	return s.store
}

// ListProjects returns all registered projects.
func (s *ProjectService) ListProjects() []domain.Project {
	return s.store.List()
}

// GetProject returns a project by slug, or an error if not found.
func (s *ProjectService) GetProject(slug string) (*domain.Project, error) {
	p, err := s.store.Get(slug)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// RemoveProject deregisters a project. If cleanup is true, also deletes
// the Gitea repo and local filesystem data.
func (s *ProjectService) RemoveProject(ctx context.Context, slug string, cleanup bool) error {
	p, err := s.store.Get(slug)
	if err != nil {
		return err
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
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "HEAD")
	cmd.Env = cleanGitEnv()
	return cmd.Run() == nil
}

// cleanGitEnv returns the current environment with GIT_DIR and GIT_WORK_TREE
// removed so git commands operate on their target repo, not the worktree.
func cleanGitEnv() []string {
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GIT_DIR=") || strings.HasPrefix(e, "GIT_WORK_TREE=") {
			continue
		}
		env = append(env, e)
	}
	return env
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
