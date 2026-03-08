package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSONAdapter_ImplementsConfigProvider(t *testing.T) {
	t.Parallel()
	var _ ConfigProvider = (*JSONAdapter)(nil)
}

func TestJSONAdapter_SaveAndLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	adapter := NewJSONAdapter(dir)

	cfg := &Config{
		GiteaPort:      4000,
		DataDir:        dir,
		APIToken:       "tok-abc",
		ComposeFile:    filepath.Join(dir, "docker-compose.yml"),
		ContainerName:  "custom-gitea",
		GiteaAdminUser: "admin",
	}

	if err := adapter.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.GiteaPort != 4000 {
		t.Errorf("GiteaPort: want 4000, got %d", loaded.GiteaPort)
	}
	if loaded.APIToken != "tok-abc" {
		t.Errorf("APIToken: want %q, got %q", "tok-abc", loaded.APIToken)
	}
	if loaded.ContainerName != "custom-gitea" {
		t.Errorf("ContainerName: want %q, got %q", "custom-gitea", loaded.ContainerName)
	}
	if loaded.GiteaAdminUser != "admin" {
		t.Errorf("GiteaAdminUser: want %q, got %q", "admin", loaded.GiteaAdminUser)
	}
}

func TestJSONAdapter_MissingFile_ReturnsZeroConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	adapter := NewJSONAdapter(dir)

	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load should not error on missing file: %v", err)
	}

	if cfg.GiteaPort != 0 {
		t.Errorf("GiteaPort: want 0, got %d", cfg.GiteaPort)
	}
	if cfg.DataDir != "" {
		t.Errorf("DataDir: want empty, got %q", cfg.DataDir)
	}
}

func TestJSONAdapter_EmptyPasswordOmitted(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	adapter := NewJSONAdapter(dir)

	cfg := &Config{
		GiteaPort:      3000,
		DataDir:        dir,
		APIToken:       "tok-abc",
		GiteaAdminUser: "kiloforger",
		GiteaAdminPass: "", // cleared after init
	}

	if err := adapter.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Read the raw JSON and verify gitea_admin_pass is absent.
	data, err := os.ReadFile(filepath.Join(dir, ConfigFileName))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(data), "gitea_admin_pass") {
		t.Errorf("config.json should not contain gitea_admin_pass when empty, got:\n%s", data)
	}
}

func TestJSONAdapter_OldConfigWithPassword_LoadsGracefully(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Simulate an old config that still has the password field.
	data := []byte(`{"gitea_port":3000,"api_token":"tok-abc","gitea_admin_user":"kiloforger","gitea_admin_pass":"old-secret"}`)
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	adapter := NewJSONAdapter(dir)
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// The password field loads fine (backward compat) but won't be used post-init.
	if cfg.GiteaAdminPass != "old-secret" {
		t.Errorf("GiteaAdminPass: want %q, got %q", "old-secret", cfg.GiteaAdminPass)
	}
	if cfg.APIToken != "tok-abc" {
		t.Errorf("APIToken: want %q, got %q", "tok-abc", cfg.APIToken)
	}
}


func TestJSONAdapter_Save_StripsPassword(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	adapter := NewJSONAdapter(dir)

	cfg := &Config{
		GiteaPort:      3000,
		DataDir:        dir,
		GiteaAdminUser: "kiloforger",
		GiteaAdminPass: "super-secret-pass", // non-empty password
	}

	if err := adapter.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Raw JSON should never contain the password.
	data, err := os.ReadFile(filepath.Join(dir, ConfigFileName))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(data), "gitea_admin_pass") {
		t.Errorf("config.json must not contain gitea_admin_pass, got:\n%s", data)
	}
	if strings.Contains(string(data), "super-secret-pass") {
		t.Errorf("config.json must not contain the password value, got:\n%s", data)
	}

	// The in-memory config should still have the password (not mutated).
	if cfg.GiteaAdminPass != "super-secret-pass" {
		t.Errorf("Save should not mutate in-memory config, got GiteaAdminPass=%q", cfg.GiteaAdminPass)
	}

	// Loaded config should have no password.
	loaded, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.GiteaAdminPass != "" {
		t.Errorf("loaded GiteaAdminPass should be empty, got %q", loaded.GiteaAdminPass)
	}
}

func TestJSONAdapter_PartialFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	data := []byte(`{"gitea_port":5000}`)
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	adapter := NewJSONAdapter(dir)
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.GiteaPort != 5000 {
		t.Errorf("GiteaPort: want 5000, got %d", cfg.GiteaPort)
	}
	if cfg.DataDir != "" {
		t.Errorf("DataDir: want empty, got %q", cfg.DataDir)
	}
}
