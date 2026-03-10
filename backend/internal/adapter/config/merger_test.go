package config

import (
	"testing"
)

func TestMerge_PriorityOrdering(t *testing.T) {
	t.Parallel()

	low := &testProvider{cfg: &Config{
		OrchestratorPort: 3000,
		DataDir:          "/low",
		ContainerName:    "low-gitea",
	}}

	high := &testProvider{cfg: &Config{
		OrchestratorPort: 5000,
		DataDir:          "/high",
	}}

	cfg, err := Merge(low, high)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	// High priority wins for fields it sets.
	if cfg.OrchestratorPort != 5000 {
		t.Errorf("OrchestratorPort: want 5000, got %d", cfg.OrchestratorPort)
	}
	if cfg.DataDir != "/high" {
		t.Errorf("DataDir: want %q, got %q", "/high", cfg.DataDir)
	}

	// Low priority fields preserved when high doesn't set them.
	if cfg.ContainerName != "low-gitea" {
		t.Errorf("ContainerName: want %q, got %q", "low-gitea", cfg.ContainerName)
	}
}

func TestMerge_PartialOverlays(t *testing.T) {
	t.Parallel()

	defaults := &testProvider{cfg: &Config{
		OrchestratorPort: 4001,
		DataDir:          "/default",
		ContainerName:    "kf-gitea",
	}}

	jsonCfg := &testProvider{cfg: &Config{
		OrchestratorPort: 4000,
		DataDir:          "/custom",
	}}

	flags := &testProvider{cfg: &Config{
		OrchestratorPort: 9000,
	}}

	cfg, err := Merge(defaults, jsonCfg, flags)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if cfg.OrchestratorPort != 9000 {
		t.Errorf("OrchestratorPort: want 9000 (flags), got %d", cfg.OrchestratorPort)
	}
	if cfg.DataDir != "/custom" {
		t.Errorf("DataDir: want %q (json), got %q", "/custom", cfg.DataDir)
	}
	if cfg.ContainerName != "kf-gitea" {
		t.Errorf("ContainerName: want %q (defaults), got %q", "kf-gitea", cfg.ContainerName)
	}
}

func TestMerge_NoProviders(t *testing.T) {
	t.Parallel()

	cfg, err := Merge()
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if cfg.OrchestratorPort != 0 {
		t.Errorf("OrchestratorPort: want 0, got %d", cfg.OrchestratorPort)
	}
}
