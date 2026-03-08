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
		GiteaPort:      3000,
		DataDir:        "/opt/kf",
		GiteaAdminUser: "admin",
		GiteaAdminPass: "secret",
	}

	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.GiteaPort != 3000 {
		t.Errorf("GiteaPort: want 3000, got %d", loaded.GiteaPort)
	}
	if loaded.DataDir != "/opt/kf" {
		t.Errorf("DataDir: want /opt/kf, got %q", loaded.DataDir)
	}
	// Password should be stripped.
	if loaded.GiteaAdminPass != "" {
		t.Errorf("GiteaAdminPass: want empty (stripped), got %q", loaded.GiteaAdminPass)
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
	if cfg.GiteaPort != 0 {
		t.Errorf("GiteaPort: want 0, got %d", cfg.GiteaPort)
	}
}
