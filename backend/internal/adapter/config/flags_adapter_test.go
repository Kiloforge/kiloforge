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
	)

	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.OrchestratorPort != 9000 {
		t.Errorf("OrchestratorPort: want 9000, got %d", cfg.OrchestratorPort)
	}
	if cfg.DataDir != "/flags" {
		t.Errorf("DataDir: want %q, got %q", "/flags", cfg.DataDir)
	}
}
