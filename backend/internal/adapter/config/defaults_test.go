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

	if cfg.GiteaPort != 4000 {
		t.Errorf("GiteaPort: want 4000, got %d", cfg.GiteaPort)
	}

	home, _ := os.UserHomeDir()
	wantDataDir := filepath.Join(home, ".kiloforge")
	if cfg.DataDir != wantDataDir {
		t.Errorf("DataDir: want %q, got %q", wantDataDir, cfg.DataDir)
	}

	if cfg.ContainerName != "kf-gitea" {
		t.Errorf("ContainerName: want %q, got %q", "kf-gitea", cfg.ContainerName)
	}
	if cfg.GiteaImage != "gitea/gitea:latest" {
		t.Errorf("GiteaImage: want %q, got %q", "gitea/gitea:latest", cfg.GiteaImage)
	}
	if cfg.GiteaAdminUser != "kiloforger" {
		t.Errorf("GiteaAdminUser: want %q, got %q", "kiloforger", cfg.GiteaAdminUser)
	}
	if cfg.GiteaAdminPass != "" {
		t.Errorf("GiteaAdminPass: want empty (resolved elsewhere), got %q", cfg.GiteaAdminPass)
	}
	if cfg.GiteaAdminEmail != "kiloforger@local.dev" {
		t.Errorf("GiteaAdminEmail: want %q, got %q", "kiloforger@local.dev", cfg.GiteaAdminEmail)
	}
	// DashboardEnabled defaults to nil (meaning enabled via IsDashboardEnabled()).
	if cfg.DashboardEnabled != nil {
		t.Errorf("DashboardEnabled: want nil, got %v", *cfg.DashboardEnabled)
	}
	if cfg.Model != "opus" {
		t.Errorf("Model: want %q, got %q", "opus", cfg.Model)
	}
}

func TestDefaultsAdapter_ImplementsConfigProvider(t *testing.T) {
	t.Parallel()
	var _ ConfigProvider = (*DefaultsAdapter)(nil)
}
