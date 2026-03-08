package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsAdapter_Load(t *testing.T) {
	t.Parallel()

	adapter := &DefaultsAdapter{}
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.GiteaPort != 3000 {
		t.Errorf("GiteaPort: want 3000, got %d", cfg.GiteaPort)
	}

	home, _ := os.UserHomeDir()
	wantDataDir := filepath.Join(home, ".kiloforge")
	if cfg.DataDir != wantDataDir {
		t.Errorf("DataDir: want %q, got %q", wantDataDir, cfg.DataDir)
	}

	if cfg.ContainerName != "conductor-gitea" {
		t.Errorf("ContainerName: want %q, got %q", "conductor-gitea", cfg.ContainerName)
	}
	if cfg.GiteaImage != "gitea/gitea:latest" {
		t.Errorf("GiteaImage: want %q, got %q", "gitea/gitea:latest", cfg.GiteaImage)
	}
	if cfg.GiteaAdminUser != "conductor" {
		t.Errorf("GiteaAdminUser: want %q, got %q", "conductor", cfg.GiteaAdminUser)
	}
	if cfg.GiteaAdminPass != "" {
		t.Errorf("GiteaAdminPass: want empty (resolved elsewhere), got %q", cfg.GiteaAdminPass)
	}
	if cfg.GiteaAdminEmail != "conductor@local.dev" {
		t.Errorf("GiteaAdminEmail: want %q, got %q", "conductor@local.dev", cfg.GiteaAdminEmail)
	}
	// DashboardEnabled defaults to nil (meaning enabled via IsDashboardEnabled()).
	if cfg.DashboardEnabled != nil {
		t.Errorf("DashboardEnabled: want nil, got %v", *cfg.DashboardEnabled)
	}
}

func TestDefaultsAdapter_ImplementsConfigProvider(t *testing.T) {
	t.Parallel()
	var _ ConfigProvider = (*DefaultsAdapter)(nil)
}
