package config

import (
	"testing"
)

func TestEnvAdapter_ImplementsConfigProvider(t *testing.T) {
	t.Parallel()
	var _ ConfigProvider = (*EnvAdapter)(nil)
}

func TestEnvAdapter_Load(t *testing.T) {
	// Not parallel — modifies env vars.
	t.Setenv("CRELAY_GITEA_PORT", "4000")
	t.Setenv("CRELAY_DATA_DIR", "/opt/crelay")
	t.Setenv("CRELAY_API_TOKEN", "env-token")
	t.Setenv("CRELAY_COMPOSE_FILE", "/opt/compose.yml")
	t.Setenv("CRELAY_CONTAINER_NAME", "env-gitea")
	t.Setenv("CRELAY_GITEA_IMAGE", "gitea/gitea:1.21")
	t.Setenv("CRELAY_GITEA_ADMIN_USER", "envadmin")
	t.Setenv("CRELAY_GITEA_ADMIN_PASS", "envpass")
	t.Setenv("CRELAY_GITEA_ADMIN_EMAIL", "env@test.com")

	adapter := &EnvAdapter{}
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.GiteaPort != 4000 {
		t.Errorf("GiteaPort: want 4000, got %d", cfg.GiteaPort)
	}
	if cfg.DataDir != "/opt/crelay" {
		t.Errorf("DataDir: want %q, got %q", "/opt/crelay", cfg.DataDir)
	}
	if cfg.APIToken != "env-token" {
		t.Errorf("APIToken: want %q, got %q", "env-token", cfg.APIToken)
	}
	if cfg.ComposeFile != "/opt/compose.yml" {
		t.Errorf("ComposeFile: want %q, got %q", "/opt/compose.yml", cfg.ComposeFile)
	}
	if cfg.ContainerName != "env-gitea" {
		t.Errorf("ContainerName: want %q, got %q", "env-gitea", cfg.ContainerName)
	}
	if cfg.GiteaImage != "gitea/gitea:1.21" {
		t.Errorf("GiteaImage: want %q, got %q", "gitea/gitea:1.21", cfg.GiteaImage)
	}
	if cfg.GiteaAdminUser != "envadmin" {
		t.Errorf("GiteaAdminUser: want %q, got %q", "envadmin", cfg.GiteaAdminUser)
	}
	if cfg.GiteaAdminPass != "envpass" {
		t.Errorf("GiteaAdminPass: want %q, got %q", "envpass", cfg.GiteaAdminPass)
	}
	if cfg.GiteaAdminEmail != "env@test.com" {
		t.Errorf("GiteaAdminEmail: want %q, got %q", "env@test.com", cfg.GiteaAdminEmail)
	}
}

func TestEnvAdapter_UnsetVars_ReturnZero(t *testing.T) {
	// Not parallel — relies on clean env.
	// t.Setenv not called — vars are unset.

	adapter := &EnvAdapter{}
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.GiteaPort != 0 {
		t.Errorf("GiteaPort: want 0, got %d", cfg.GiteaPort)
	}
	if cfg.DataDir != "" {
		t.Errorf("DataDir: want empty, got %q", cfg.DataDir)
	}
}

func TestEnvAdapter_DashboardPort(t *testing.T) {
	t.Setenv("CRELAY_DASHBOARD_PORT", "4002")

	adapter := &EnvAdapter{}
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DashboardPort != 4002 {
		t.Errorf("DashboardPort: want 4002, got %d", cfg.DashboardPort)
	}
}

func TestEnvAdapter_DashboardEnabled(t *testing.T) {
	t.Setenv("CRELAY_DASHBOARD_ENABLED", "false")

	adapter := &EnvAdapter{}
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DashboardEnabled == nil {
		t.Fatal("DashboardEnabled: want non-nil")
	}
	if *cfg.DashboardEnabled != false {
		t.Errorf("DashboardEnabled: want false, got %v", *cfg.DashboardEnabled)
	}
}

func TestEnvAdapter_InvalidPort_Ignored(t *testing.T) {
	t.Setenv("CRELAY_GITEA_PORT", "notanumber")

	adapter := &EnvAdapter{}
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.GiteaPort != 0 {
		t.Errorf("GiteaPort: want 0 for invalid, got %d", cfg.GiteaPort)
	}
}
