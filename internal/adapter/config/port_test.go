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
		GiteaPort:       4000,
		DataDir:         "/tmp/test",
		APIToken:        "tok-123",
		ComposeFile:     "/tmp/compose.yml",
		ContainerName:   "my-gitea",
		GiteaImage:      "gitea/gitea:1.21",
		GiteaAdminUser:  "admin",
		GiteaAdminPass:  "secret",
		GiteaAdminEmail: "admin@test.com",
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
	if loaded.GiteaImage != "gitea/gitea:1.21" {
		t.Errorf("GiteaImage: want %q, got %q", "gitea/gitea:1.21", loaded.GiteaImage)
	}
	if loaded.GiteaAdminUser != "admin" {
		t.Errorf("GiteaAdminUser: want %q, got %q", "admin", loaded.GiteaAdminUser)
	}
	if loaded.GiteaAdminPass != "secret" {
		t.Errorf("GiteaAdminPass: want %q, got %q", "secret", loaded.GiteaAdminPass)
	}
	if loaded.GiteaAdminEmail != "admin@test.com" {
		t.Errorf("GiteaAdminEmail: want %q, got %q", "admin@test.com", loaded.GiteaAdminEmail)
	}
}
