package config

import (
	"testing"
)

func TestFlagsAdapter_ImplementsConfigProvider(t *testing.T) {
	t.Parallel()
	var _ ConfigProvider = (*FlagsAdapter)(nil)
}

func TestFlagsAdapter_OnlySetFields(t *testing.T) {
	t.Parallel()

	adapter := NewFlagsAdapter(
		WithGiteaPort(5000),
		WithDataDir("/custom/dir"),
	)

	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.GiteaPort != 5000 {
		t.Errorf("GiteaPort: want 5000, got %d", cfg.GiteaPort)
	}
	if cfg.DataDir != "/custom/dir" {
		t.Errorf("DataDir: want %q, got %q", "/custom/dir", cfg.DataDir)
	}

	// Unset fields should be zero.
	if cfg.APIToken != "" {
		t.Errorf("APIToken: want empty, got %q", cfg.APIToken)
	}
	if cfg.ContainerName != "" {
		t.Errorf("ContainerName: want empty, got %q", cfg.ContainerName)
	}
}

func TestFlagsAdapter_NoOptions_ZeroConfig(t *testing.T) {
	t.Parallel()

	adapter := NewFlagsAdapter()
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.GiteaPort != 0 {
		t.Errorf("GiteaPort: want 0, got %d", cfg.GiteaPort)
	}
}

func TestFlagsAdapter_AllOptions(t *testing.T) {
	t.Parallel()

	adapter := NewFlagsAdapter(
		WithGiteaPort(9000),
		WithDataDir("/flags"),
		WithAPIToken("flag-tok"),
		WithComposeFile("/flags/compose.yml"),
		WithContainerName("flag-gitea"),
		WithGiteaImage("gitea/gitea:1.22"),
		WithGiteaAdminUser("flagadmin"),
		WithGiteaAdminPass("flagpass"),
		WithGiteaAdminEmail("flag@test.com"),
	)

	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.GiteaPort != 9000 {
		t.Errorf("GiteaPort: want 9000, got %d", cfg.GiteaPort)
	}
	if cfg.ContainerName != "flag-gitea" {
		t.Errorf("ContainerName: want %q, got %q", "flag-gitea", cfg.ContainerName)
	}
	if cfg.GiteaAdminEmail != "flag@test.com" {
		t.Errorf("GiteaAdminEmail: want %q, got %q", "flag@test.com", cfg.GiteaAdminEmail)
	}
}
