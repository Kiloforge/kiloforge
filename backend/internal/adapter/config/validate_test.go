package config

import (
	"testing"
)

func TestConfig_Validate_Valid(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		OrchestratorPort: 4001,
		OrchestratorHost: "127.0.0.1",
		DataDir:          "/tmp/kf",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got: %v", err)
	}
}

func TestConfig_Validate_EmptyHost(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		OrchestratorPort: 4001,
		OrchestratorHost: "",
		DataDir:          "/tmp/kf",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty OrchestratorHost")
	}
}

func TestConfig_Validate_WildcardHostAllowed(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		OrchestratorPort: 4001,
		OrchestratorHost: "0.0.0.0",
		DataDir:          "/tmp/kf",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected 0.0.0.0 to be valid, got: %v", err)
	}
}

func TestConfig_Validate_EmptyDataDir(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		OrchestratorPort: 4001,
		OrchestratorHost: "127.0.0.1",
		DataDir:          "",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty DataDir")
	}
}

func TestConfig_Validate_PortTooLow(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		OrchestratorPort: 0,
		OrchestratorHost: "127.0.0.1",
		DataDir:          "/tmp/kf",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for port 0")
	}
}

func TestConfig_Validate_PortTooHigh(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		OrchestratorPort: 70000,
		OrchestratorHost: "127.0.0.1",
		DataDir:          "/tmp/kf",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for port 70000")
	}
}

func TestConfig_Validate_PortBoundaries(t *testing.T) {
	t.Parallel()
	for _, port := range []int{1, 65535} {
		cfg := &Config{
			OrchestratorPort: port,
			OrchestratorHost: "127.0.0.1",
			DataDir:          "/tmp/kf",
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("port %d should be valid, got: %v", port, err)
		}
	}
}
