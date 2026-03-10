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
