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

	if cfg.OrchestratorPort != 39517 {
		t.Errorf("OrchestratorPort: want 39517, got %d", cfg.OrchestratorPort)
	}

	home, _ := os.UserHomeDir()
	wantDataDir := filepath.Join(home, ".kiloforge")
	if cfg.DataDir != wantDataDir {
		t.Errorf("DataDir: want %q, got %q", wantDataDir, cfg.DataDir)
	}

	// DashboardEnabled defaults to nil (meaning enabled via IsDashboardEnabled()).
	if cfg.DashboardEnabled != nil {
		t.Errorf("DashboardEnabled: want nil, got %v", *cfg.DashboardEnabled)
	}
	if cfg.Model != "opus" {
		t.Errorf("Model: want %q, got %q", "opus", cfg.Model)
	}
	if cfg.OrchestratorHost != "127.0.0.1" {
		t.Errorf("OrchestratorHost: want %q, got %q", "127.0.0.1", cfg.OrchestratorHost)
	}
}

func TestDefaultsAdapter_ImplementsConfigProvider(t *testing.T) {
	t.Parallel()
	var _ ConfigProvider = (*DefaultsAdapter)(nil)
}
