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
		GiteaPort:   3000,
		DataDir:     dir,
		APIToken:    "test-token",
		ComposeFile: filepath.Join(dir, "docker-compose.yml"),
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadFrom(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.GiteaPort != cfg.GiteaPort {
		t.Errorf("GiteaPort: want %d, got %d", cfg.GiteaPort, loaded.GiteaPort)
	}
	if loaded.DataDir != cfg.DataDir {
		t.Errorf("DataDir: want %q, got %q", cfg.DataDir, loaded.DataDir)
	}
	if loaded.APIToken != cfg.APIToken {
		t.Errorf("APIToken: want %q, got %q", cfg.APIToken, loaded.APIToken)
	}
	if loaded.ComposeFile != cfg.ComposeFile {
		t.Errorf("ComposeFile: want %q, got %q", cfg.ComposeFile, loaded.ComposeFile)
	}
}

func TestConfig_GiteaURL(t *testing.T) {
	t.Parallel()

	cfg := &Config{GiteaPort: 4000}
	want := "http://localhost:4000"
	if got := cfg.GiteaURL(); got != want {
		t.Errorf("GiteaURL: want %q, got %q", want, got)
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

func TestConfig_NoProjectFields(t *testing.T) {
	t.Parallel()

	// Write a config with old project fields — they should be ignored on load.
	dir := t.TempDir()
	data := []byte(`{"gitea_port":3000,"data_dir":"` + dir + `","repo_name":"old","project_dir":"/old"}`)
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	loaded, err := LoadFrom(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// Config should load without error; unknown fields are silently ignored by encoding/json.
	if loaded.GiteaPort != 3000 {
		t.Errorf("GiteaPort: want 3000, got %d", loaded.GiteaPort)
	}
}
