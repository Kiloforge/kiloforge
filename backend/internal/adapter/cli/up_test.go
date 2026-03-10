package cli

import (
	"os"
	"path/filepath"
	"testing"

	"kiloforge/internal/adapter/config"
)

func TestIsFirstRun_NoConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if !isFirstRun(dir) {
		t.Error("expected isFirstRun=true for empty dir")
	}
}

func TestIsFirstRun_WithConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.ConfigFileName), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if isFirstRun(dir) {
		t.Error("expected isFirstRun=false when config.json exists")
	}
}
