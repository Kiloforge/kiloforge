package config

import (
	"os"
	"path/filepath"
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
		OrchestratorPort: 4001,
		DataDir:          dir,
	}

	if err := adapter.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.OrchestratorPort != 4001 {
		t.Errorf("OrchestratorPort: want 4001, got %d", loaded.OrchestratorPort)
	}
	if loaded.DataDir != dir {
		t.Errorf("DataDir: want %q, got %q", dir, loaded.DataDir)
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

	if cfg.OrchestratorPort != 0 {
		t.Errorf("OrchestratorPort: want 0, got %d", cfg.OrchestratorPort)
	}
	if cfg.DataDir != "" {
		t.Errorf("DataDir: want empty, got %q", cfg.DataDir)
	}
}

func TestJSONAdapter_PartialFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	data := []byte(`{"orchestrator_port":5000}`)
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	adapter := NewJSONAdapter(dir)
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.OrchestratorPort != 5000 {
		t.Errorf("OrchestratorPort: want 5000, got %d", cfg.OrchestratorPort)
	}
	if cfg.DataDir != "" {
		t.Errorf("DataDir: want empty, got %q", cfg.DataDir)
	}
}
