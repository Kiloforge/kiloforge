package kf_test

import (
	"os"
	"path/filepath"
	"testing"

	"kiloforge/pkg/kf"
)

func TestDefaultConfig(t *testing.T) {
	cfg := kf.DefaultConfig()
	if cfg.PrimaryBranch != "main" {
		t.Errorf("PrimaryBranch = %q, want %q", cfg.PrimaryBranch, "main")
	}
	if !cfg.EnforceDepOrdering {
		t.Error("EnforceDepOrdering should default to true")
	}
}

func TestGetConfig_FullConfig(t *testing.T) {
	dir := t.TempDir()
	kfDir := filepath.Join(dir, ".agent", "kf")
	os.MkdirAll(kfDir, 0o755)

	content := `primary_branch: develop
enforce_dep_ordering: false
`
	os.WriteFile(filepath.Join(kfDir, "config.yaml"), []byte(content), 0o644)

	client := kf.NewClient(kfDir)
	cfg, err := client.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PrimaryBranch != "develop" {
		t.Errorf("PrimaryBranch = %q, want %q", cfg.PrimaryBranch, "develop")
	}
	if cfg.EnforceDepOrdering {
		t.Error("EnforceDepOrdering should be false")
	}
}

func TestGetConfig_PartialConfig(t *testing.T) {
	dir := t.TempDir()
	kfDir := filepath.Join(dir, ".agent", "kf")
	os.MkdirAll(kfDir, 0o755)

	// Only primary_branch set — enforce_dep_ordering should default to true
	content := `primary_branch: staging
`
	os.WriteFile(filepath.Join(kfDir, "config.yaml"), []byte(content), 0o644)

	client := kf.NewClient(kfDir)
	cfg, err := client.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PrimaryBranch != "staging" {
		t.Errorf("PrimaryBranch = %q, want %q", cfg.PrimaryBranch, "staging")
	}
	if !cfg.EnforceDepOrdering {
		t.Error("EnforceDepOrdering should default to true when not set")
	}
}

func TestGetConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	kfDir := filepath.Join(dir, ".agent", "kf")
	os.MkdirAll(kfDir, 0o755)

	// No config.yaml — all defaults
	client := kf.NewClient(kfDir)
	cfg, err := client.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PrimaryBranch != "main" {
		t.Errorf("PrimaryBranch = %q, want %q", cfg.PrimaryBranch, "main")
	}
	if !cfg.EnforceDepOrdering {
		t.Error("EnforceDepOrdering should default to true")
	}
}

func TestSaveConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	kfDir := filepath.Join(dir, ".agent", "kf")
	os.MkdirAll(kfDir, 0o755)

	client := kf.NewClient(kfDir)

	// Write a config.
	cfg := &kf.ProjectConfig{
		PrimaryBranch:      "develop",
		EnforceDepOrdering: false,
	}
	if err := client.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// Read it back.
	got, err := client.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig after save: %v", err)
	}
	if got.PrimaryBranch != "develop" {
		t.Errorf("PrimaryBranch = %q, want %q", got.PrimaryBranch, "develop")
	}
	if got.EnforceDepOrdering {
		t.Error("EnforceDepOrdering should be false")
	}
}

func TestSaveConfig_DefaultsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	kfDir := filepath.Join(dir, ".agent", "kf")
	os.MkdirAll(kfDir, 0o755)

	client := kf.NewClient(kfDir)

	// Save defaults.
	cfg := kf.DefaultConfig()
	if err := client.SaveConfig(&cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// Read it back — should match defaults.
	got, err := client.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if got.PrimaryBranch != "main" {
		t.Errorf("PrimaryBranch = %q, want %q", got.PrimaryBranch, "main")
	}
	if !got.EnforceDepOrdering {
		t.Error("EnforceDepOrdering should be true")
	}
}

func TestSaveConfig_HasHeader(t *testing.T) {
	dir := t.TempDir()
	kfDir := filepath.Join(dir, ".agent", "kf")
	os.MkdirAll(kfDir, 0o755)

	client := kf.NewClient(kfDir)
	cfg := kf.DefaultConfig()
	if err := client.SaveConfig(&cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(kfDir, "config.yaml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("config.yaml is empty")
	}
	// Should start with a comment header.
	if data[0] != '#' {
		t.Error("expected config.yaml to start with comment header")
	}
}

func TestGetConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	kfDir := filepath.Join(dir, ".agent", "kf")
	os.MkdirAll(kfDir, 0o755)

	content := `{{{invalid yaml`
	os.WriteFile(filepath.Join(kfDir, "config.yaml"), []byte(content), 0o644)

	client := kf.NewClient(kfDir)
	_, err := client.GetConfig()
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
