package gitea

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"crelay/internal/config"
)

// Manager handles Docker lifecycle for Gitea.
type Manager struct {
	cfg *config.Config
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{cfg: cfg}
}

// Start launches the Gitea Docker container if not already running.
func (m *Manager) Start(ctx context.Context) error {
	// Check if container already exists and is running.
	out, err := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Status}}", config.ContainerName).Output()
	if err == nil {
		status := strings.TrimSpace(string(out))
		if status == "running" {
			return nil // Already running.
		}
		if status == "exited" || status == "created" {
			// Restart existing container.
			return exec.CommandContext(ctx, "docker", "start", config.ContainerName).Run()
		}
	}

	// Run a new container.
	args := []string{
		"run", "-d",
		"--name", config.ContainerName,
		"-p", fmt.Sprintf("%d:3000", m.cfg.GiteaPort),
		"-v", fmt.Sprintf("%s/gitea-data:/data", m.cfg.DataDir),
		"-e", "GITEA__security__INSTALL_LOCK=true",
		"-e", "GITEA__server__ROOT_URL=" + m.cfg.GiteaURL() + "/",
		"-e", "GITEA__server__HTTP_PORT=3000",
		"-e", "GITEA__database__DB_TYPE=sqlite3",
		"-e", "GITEA__service__DISABLE_REGISTRATION=true",
		"-e", "GITEA__webhook__ALLOWED_HOST_LIST=*",
		config.GiteaImage,
	}

	if err := exec.CommandContext(ctx, "docker", args...).Run(); err != nil {
		return fmt.Errorf("docker run: %w", err)
	}

	// Wait for Gitea to be ready.
	return m.waitReady(ctx)
}

func (m *Manager) waitReady(ctx context.Context) error {
	deadline := time.After(60 * time.Second)
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("gitea did not become ready within 60 seconds")
		case <-tick.C:
			out, err := exec.CommandContext(ctx, "curl", "-sf", fmt.Sprintf("http://localhost:%d/api/v1/version", m.cfg.GiteaPort)).Output()
			if err == nil && len(out) > 0 {
				return nil
			}
		}
	}
}

// Configure creates the admin user, API token, and repository.
// Returns a Client configured with the token.
func (m *Manager) Configure(ctx context.Context) (*Client, error) {
	// Create admin user (ignore error if already exists).
	_ = exec.CommandContext(ctx, "docker", "exec", config.ContainerName,
		"gitea", "admin", "user", "create",
		"--username", config.GiteaAdminUser,
		"--password", config.GiteaAdminPass,
		"--email", config.GiteaAdminEmail,
		"--admin",
	).Run()

	client := NewClient(m.cfg.GiteaURL(), config.GiteaAdminUser, config.GiteaAdminPass)

	// Create API token.
	token, err := client.CreateToken(ctx, "crelay")
	if err != nil {
		// Token may already exist; try to use basic auth for remaining calls.
		// We'll proceed with basic auth.
		fmt.Println("    (using basic auth — token may already exist)")
	} else {
		client.SetToken(token)
		m.cfg.APIToken = token
	}

	// Create repository if it doesn't exist.
	if err := client.CreateRepo(ctx, m.cfg.RepoName); err != nil {
		// Repo may already exist.
		fmt.Printf("    (repo may already exist: %v)\n", err)
	}

	return client, nil
}

// SetupGitRemote adds a 'gitea' remote to the project and pushes.
func (m *Manager) SetupGitRemote(ctx context.Context, cfg *config.Config) error {
	remoteURL := fmt.Sprintf("http://%s:%s@localhost:%d/%s/%s.git",
		config.GiteaAdminUser, config.GiteaAdminPass, cfg.GiteaPort, config.GiteaAdminUser, cfg.RepoName)

	// Remove existing remote if any.
	_ = exec.CommandContext(ctx, "git", "-C", cfg.ProjectDir, "remote", "remove", "gitea").Run()

	// Add remote.
	if err := exec.CommandContext(ctx, "git", "-C", cfg.ProjectDir, "remote", "add", "gitea", remoteURL).Run(); err != nil {
		return fmt.Errorf("add remote: %w", err)
	}

	// Push current branch.
	out, err := exec.CommandContext(ctx, "git", "-C", cfg.ProjectDir, "push", "-u", "gitea", "main").CombinedOutput()
	if err != nil {
		return fmt.Errorf("push to gitea: %w\n%s", err, out)
	}

	return nil
}
