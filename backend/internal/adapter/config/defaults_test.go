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

	if cfg.OrchestratorPort != 4001 {
		t.Errorf("OrchestratorPort: want 4001, got %d", cfg.OrchestratorPort)
	}

	home, _ := os.UserHomeDir()
	wantDataDir := filepath.Join(home, ".kiloforge")
	if cfg.DataDir != wantDataDir {
		t.Errorf("DataDir: want %q, got %q", wantDataDir, cfg.DataDir)
	}

	if cfg.ContainerName != "kf-gitea" {
		t.Errorf("ContainerName: want %q, got %q", "kf-gitea", cfg.ContainerName)
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
