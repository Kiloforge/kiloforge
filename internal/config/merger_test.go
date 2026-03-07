package config

import (
	"testing"
)

func TestMerge_PriorityOrdering(t *testing.T) {
	t.Parallel()

	low := &testProvider{cfg: &Config{
		GiteaPort:      3000,
		DataDir:        "/low",
		ContainerName:  "low-gitea",
		GiteaAdminUser: "lowadmin",
	}}

	high := &testProvider{cfg: &Config{
		GiteaPort: 5000,
		DataDir:   "/high",
	}}

	cfg, err := Merge(low, high)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	// High priority wins for fields it sets.
	if cfg.GiteaPort != 5000 {
		t.Errorf("GiteaPort: want 5000, got %d", cfg.GiteaPort)
	}
	if cfg.DataDir != "/high" {
		t.Errorf("DataDir: want %q, got %q", "/high", cfg.DataDir)
	}

	// Low priority fields preserved when high doesn't set them.
	if cfg.ContainerName != "low-gitea" {
		t.Errorf("ContainerName: want %q, got %q", "low-gitea", cfg.ContainerName)
	}
	if cfg.GiteaAdminUser != "lowadmin" {
		t.Errorf("GiteaAdminUser: want %q, got %q", "lowadmin", cfg.GiteaAdminUser)
	}
}

func TestMerge_PartialOverlays(t *testing.T) {
	t.Parallel()

	defaults := &testProvider{cfg: &Config{
		GiteaPort:       3000,
		DataDir:         "/default",
		ContainerName:   "conductor-gitea",
		GiteaImage:      "gitea/gitea:latest",
		GiteaAdminUser:  "conductor",
		GiteaAdminPass:  "conductor123",
		GiteaAdminEmail: "conductor@local.dev",
	}}

	jsonCfg := &testProvider{cfg: &Config{
		GiteaPort: 4000,
		DataDir:   "/custom",
	}}

	env := &testProvider{cfg: &Config{
		GiteaAdminPass: "env-secret",
	}}

	flags := &testProvider{cfg: &Config{
		GiteaPort: 9000,
	}}

	cfg, err := Merge(defaults, jsonCfg, env, flags)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if cfg.GiteaPort != 9000 {
		t.Errorf("GiteaPort: want 9000 (flags), got %d", cfg.GiteaPort)
	}
	if cfg.DataDir != "/custom" {
		t.Errorf("DataDir: want %q (json), got %q", "/custom", cfg.DataDir)
	}
	if cfg.GiteaAdminPass != "env-secret" {
		t.Errorf("GiteaAdminPass: want %q (env), got %q", "env-secret", cfg.GiteaAdminPass)
	}
	if cfg.GiteaAdminUser != "conductor" {
		t.Errorf("GiteaAdminUser: want %q (defaults), got %q", "conductor", cfg.GiteaAdminUser)
	}
	if cfg.ContainerName != "conductor-gitea" {
		t.Errorf("ContainerName: want %q (defaults), got %q", "conductor-gitea", cfg.ContainerName)
	}
}

func TestMerge_NoProviders(t *testing.T) {
	t.Parallel()

	cfg, err := Merge()
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if cfg.GiteaPort != 0 {
		t.Errorf("GiteaPort: want 0, got %d", cfg.GiteaPort)
	}
}
