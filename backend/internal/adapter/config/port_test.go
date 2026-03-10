package config

import (
	"encoding/json"
	"testing"
)

func TestConfigProvider_Interface(t *testing.T) {
	t.Parallel()

	// Verify that ConfigProvider is a valid interface.
	var _ ConfigProvider = (*testProvider)(nil)
}

type testProvider struct {
	cfg *Config
	err error
}

func (p *testProvider) Load() (*Config, error) {
	return p.cfg, p.err
}

func TestConfig_ExpandedFields_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		OrchestratorPort: 4001,
		DataDir:          "/tmp/test",
		ComposeFile:      "/tmp/compose.yml",
		ContainerName:    "my-gitea",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if loaded.ContainerName != "my-gitea" {
		t.Errorf("ContainerName: want %q, got %q", "my-gitea", loaded.ContainerName)
	}
	if loaded.OrchestratorPort != 4001 {
		t.Errorf("OrchestratorPort: want 4001, got %d", loaded.OrchestratorPort)
	}
}
