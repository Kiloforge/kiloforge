package gitea

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"kiloforge/internal/adapter/compose"
	"kiloforge/internal/adapter/config"
)

// Manager handles Gitea lifecycle via docker compose.
type Manager struct {
	cfg    *config.Config
	runner *compose.Runner
}

// NewManager creates a Manager with the given config and compose runner.
func NewManager(cfg *config.Config, runner *compose.Runner) *Manager {
	return &Manager{cfg: cfg, runner: runner}
}

// Start launches Gitea via docker compose up.
func (m *Manager) Start(ctx context.Context) error {
	if err := m.runner.Up(ctx, m.cfg.DataDir); err != nil {
		return fmt.Errorf("compose up: %w", err)
	}
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
			req, err := http.NewRequestWithContext(ctx, "GET", m.cfg.GiteaURL()+"/api/v1/version", nil)
			if err != nil {
				continue
			}
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
	}
}

// Configure creates the admin user and API token.
// Returns a Client configured with the token.
func (m *Manager) Configure(ctx context.Context) (*Client, error) {
	// Create admin user via compose exec (ignore error if already exists).
	// Gitea refuses to run CLI commands as root — run as the "git" user.
	_, _ = m.runner.Exec(ctx, m.cfg.DataDir, "gitea", "git",
		"gitea", "admin", "user", "create",
		"--username", m.cfg.GiteaAdminUser,
		"--password", m.cfg.GiteaAdminPass,
		"--email", m.cfg.GiteaAdminEmail,
		"--admin",
	)

	client := NewClient(m.cfg.GiteaURL(), m.cfg.GiteaAdminUser, m.cfg.GiteaAdminPass)

	// Create API token.
	token, err := client.CreateToken(ctx, "kiloforge")
	if err != nil {
		fmt.Println("    (using basic auth — token may already exist)")
	} else {
		client.SetToken(token)
		m.cfg.APIToken = token
	}

	return client, nil
}
