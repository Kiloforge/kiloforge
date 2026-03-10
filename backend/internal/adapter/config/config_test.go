package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_SaveAndLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	cfg := &Config{
		OrchestratorPort: 4001,
		DataDir:          dir,
		ComposeFile:      filepath.Join(dir, "docker-compose.yml"),
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadFrom(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.OrchestratorPort != cfg.OrchestratorPort {
		t.Errorf("OrchestratorPort: want %d, got %d", cfg.OrchestratorPort, loaded.OrchestratorPort)
	}
	if loaded.DataDir != cfg.DataDir {
		t.Errorf("DataDir: want %q, got %q", cfg.DataDir, loaded.DataDir)
	}
	if loaded.ComposeFile != cfg.ComposeFile {
		t.Errorf("ComposeFile: want %q, got %q", cfg.ComposeFile, loaded.ComposeFile)
	}
}

func TestConfig_IsDashboardEnabled(t *testing.T) {
	t.Parallel()

	// nil means enabled (default).
	cfg := &Config{}
	if !cfg.IsDashboardEnabled() {
		t.Error("nil DashboardEnabled should return true")
	}

	// Explicit true.
	tr := true
	cfg.DashboardEnabled = &tr
	if !cfg.IsDashboardEnabled() {
		t.Error("true DashboardEnabled should return true")
	}

	// Explicit false.
	f := false
	cfg.DashboardEnabled = &f
	if cfg.IsDashboardEnabled() {
		t.Error("false DashboardEnabled should return false")
	}
}

func TestConfig_IsAnalyticsEnabled(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	if !cfg.IsAnalyticsEnabled() {
		t.Error("nil AnalyticsEnabled should return true")
	}

	tr := true
	cfg.AnalyticsEnabled = &tr
	if !cfg.IsAnalyticsEnabled() {
		t.Error("true AnalyticsEnabled should return true")
	}

	f := false
	cfg.AnalyticsEnabled = &f
	if cfg.IsAnalyticsEnabled() {
		t.Error("false AnalyticsEnabled should return false")
	}
}

func TestConfig_NoProjectFields(t *testing.T) {
	t.Parallel()

	// Write a config with old project fields — they should be ignored on load.
	dir := t.TempDir()
	data := []byte(`{"orchestrator_port":4001,"data_dir":"` + dir + `","repo_name":"old","project_dir":"/old"}`)
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	loaded, err := LoadFrom(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// Config should load without error; unknown fields are silently ignored by encoding/json.
	if loaded.OrchestratorPort != 4001 {
		t.Errorf("OrchestratorPort: want 4001, got %d", loaded.OrchestratorPort)
	}
}
