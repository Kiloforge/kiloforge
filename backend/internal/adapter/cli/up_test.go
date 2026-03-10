package cli

import (
	"net"
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

func TestDashboardURL_DefaultHost(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{OrchestratorHost: "127.0.0.1", OrchestratorPort: 4001}
	got := dashboardURL(cfg)
	if got != "http://127.0.0.1:4001/" {
		t.Errorf("want http://127.0.0.1:4001/, got %s", got)
	}
}

func TestDashboardURL_WildcardBecomesLocalhost(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{OrchestratorHost: "0.0.0.0", OrchestratorPort: 5000}
	got := dashboardURL(cfg)
	if got != "http://localhost:5000/" {
		t.Errorf("want http://localhost:5000/, got %s", got)
	}
}

func TestPortConflictDetection(t *testing.T) {
	t.Parallel()
	// Occupy a port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	// Try to listen on the same port — should fail.
	port := ln.Addr().(*net.TCPAddr).Port
	ln2, err := net.Listen("tcp", ln.Addr().String())
	if err == nil {
		ln2.Close()
		t.Fatalf("expected port %d to be in use", port)
	}
}

func TestDashboardURL_CustomHost(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{OrchestratorHost: "192.168.1.10", OrchestratorPort: 8080}
	got := dashboardURL(cfg)
	if got != "http://192.168.1.10:8080/" {
		t.Errorf("want http://192.168.1.10:8080/, got %s", got)
	}
}
