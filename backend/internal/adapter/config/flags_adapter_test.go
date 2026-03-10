package config

import (
	"testing"
)

func TestFlagsAdapter_ImplementsConfigProvider(t *testing.T) {
	t.Parallel()
	var _ ConfigProvider = (*FlagsAdapter)(nil)
}

func TestFlagsAdapter_OnlySetFields(t *testing.T) {
	t.Parallel()

	adapter := NewFlagsAdapter(
		WithOrchestratorPort(5000),
		WithDataDir("/custom/dir"),
	)

	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.OrchestratorPort != 5000 {
		t.Errorf("OrchestratorPort: want 5000, got %d", cfg.OrchestratorPort)
	}
	if cfg.DataDir != "/custom/dir" {
		t.Errorf("DataDir: want %q, got %q", "/custom/dir", cfg.DataDir)
	}

	// Unset fields should be zero.
	if cfg.ContainerName != "" {
		t.Errorf("ContainerName: want empty, got %q", cfg.ContainerName)
	}
}

func TestFlagsAdapter_NoOptions_ZeroConfig(t *testing.T) {
	t.Parallel()

	adapter := NewFlagsAdapter()
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.OrchestratorPort != 0 {
		t.Errorf("OrchestratorPort: want 0, got %d", cfg.OrchestratorPort)
	}
}

func TestFlagsAdapter_AllOptions(t *testing.T) {
	t.Parallel()

	adapter := NewFlagsAdapter(
		WithOrchestratorPort(9000),
		WithDataDir("/flags"),
		WithComposeFile("/flags/compose.yml"),
		WithContainerName("flag-gitea"),
	)

	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.OrchestratorPort != 9000 {
		t.Errorf("OrchestratorPort: want 9000, got %d", cfg.OrchestratorPort)
	}
	if cfg.ContainerName != "flag-gitea" {
		t.Errorf("ContainerName: want %q, got %q", "flag-gitea", cfg.ContainerName)
	}
}
