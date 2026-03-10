package sqlite

import (
	"testing"

	"kiloforge/internal/adapter/config"
)

func TestConfigStore_SaveAndLoad(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewConfigStore(db)

	cfg := &config.Config{
		DataDir: "/opt/kf",
	}

	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.DataDir != "/opt/kf" {
		t.Errorf("DataDir: want /opt/kf, got %q", loaded.DataDir)
	}
}

func TestConfigStore_LoadEmpty(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewConfigStore(db)

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataDir != "" {
		t.Errorf("DataDir: want empty, got %q", cfg.DataDir)
	}
}
