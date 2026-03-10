package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolve_DefaultChain(t *testing.T) {
	// Not parallel — uses env vars.

	// Isolate from user's real config by pointing to an empty temp dir.
	t.Setenv("KF_DATA_DIR", t.TempDir())

	// With no JSON file and no env vars, Resolve should return defaults.
	cfg, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if cfg.OrchestratorPort != 39517 {
		t.Errorf("OrchestratorPort: want 39517, got %d", cfg.OrchestratorPort)
	}
}

func TestResolve_WithFlags(t *testing.T) {
	// Not parallel — uses env vars.
	t.Setenv("KF_DATA_DIR", t.TempDir())

	cfg, err := Resolve(
		NewFlagsAdapter(WithOrchestratorPort(8080)),
	)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if cfg.OrchestratorPort != 8080 {
		t.Errorf("OrchestratorPort: want 8080, got %d", cfg.OrchestratorPort)
	}
}

func TestResolve_FullChain(t *testing.T) {
	// Not parallel — uses env vars.
	dir := t.TempDir()

	// Write a JSON config.
	data := []byte(`{"orchestrator_port":4001,"data_dir":"` + dir + `"}`)
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Set data dir env so the JSON adapter can find the file.
	t.Setenv("KF_DATA_DIR", dir)

	cfg, err := Resolve(
		NewFlagsAdapter(WithOrchestratorPort(9999)),
	)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Flags override everything.
	if cfg.OrchestratorPort != 9999 {
		t.Errorf("OrchestratorPort: want 9999 (flags), got %d", cfg.OrchestratorPort)
	}
}
