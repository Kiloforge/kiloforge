package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolve_DefaultChain(t *testing.T) {
	t.Parallel()

	// With no JSON file and no env vars, Resolve should return defaults.
	cfg, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if cfg.GiteaPort != 3000 {
		t.Errorf("GiteaPort: want 3000, got %d", cfg.GiteaPort)
	}
	if cfg.GiteaAdminUser != "conductor" {
		t.Errorf("GiteaAdminUser: want %q, got %q", "conductor", cfg.GiteaAdminUser)
	}
}

func TestResolve_WithFlags(t *testing.T) {
	t.Parallel()

	cfg, err := Resolve(
		NewFlagsAdapter(WithGiteaPort(8080)),
	)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if cfg.GiteaPort != 8080 {
		t.Errorf("GiteaPort: want 8080, got %d", cfg.GiteaPort)
	}
	// Defaults still fill in unset fields.
	if cfg.GiteaAdminUser != "conductor" {
		t.Errorf("GiteaAdminUser: want %q, got %q", "conductor", cfg.GiteaAdminUser)
	}
}

func TestResolve_FullChain(t *testing.T) {
	// Not parallel — uses env vars.
	dir := t.TempDir()

	// Write a JSON config.
	data := []byte(`{"gitea_port":4000,"data_dir":"` + dir + `","api_token":"json-tok"}`)
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Set an env var that overrides JSON.
	t.Setenv("CRELAY_API_TOKEN", "env-tok")
	// Set data dir env so the JSON adapter can find the file.
	t.Setenv("CRELAY_DATA_DIR", dir)

	cfg, err := Resolve(
		NewFlagsAdapter(WithGiteaPort(9999)),
	)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Flags override everything.
	if cfg.GiteaPort != 9999 {
		t.Errorf("GiteaPort: want 9999 (flags), got %d", cfg.GiteaPort)
	}
	// Env overrides JSON.
	if cfg.APIToken != "env-tok" {
		t.Errorf("APIToken: want %q (env), got %q", "env-tok", cfg.APIToken)
	}
	// Defaults fill in the rest.
	if cfg.GiteaAdminUser != "conductor" {
		t.Errorf("GiteaAdminUser: want %q (defaults), got %q", "conductor", cfg.GiteaAdminUser)
	}
}
