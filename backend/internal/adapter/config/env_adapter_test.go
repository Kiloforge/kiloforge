package config

import (
	"testing"
	"time"
)

func TestEnvAdapter_ImplementsConfigProvider(t *testing.T) {
	t.Parallel()
	var _ ConfigProvider = (*EnvAdapter)(nil)
}

func TestEnvAdapter_Load(t *testing.T) {
	// Not parallel — modifies env vars.
	t.Setenv("KF_GITEA_PORT", "4000")
	t.Setenv("KF_DATA_DIR", "/opt/kf")
	t.Setenv("KF_API_TOKEN", "env-token")
	t.Setenv("KF_COMPOSE_FILE", "/opt/compose.yml")
	t.Setenv("KF_CONTAINER_NAME", "env-gitea")
	t.Setenv("KF_GITEA_IMAGE", "gitea/gitea:1.21")
	t.Setenv("KF_GITEA_ADMIN_USER", "envadmin")
	t.Setenv("KF_GITEA_ADMIN_PASS", "envpass")
	t.Setenv("KF_GITEA_ADMIN_EMAIL", "env@test.com")

	adapter := &EnvAdapter{}
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.GiteaPort != 4000 {
		t.Errorf("GiteaPort: want 4000, got %d", cfg.GiteaPort)
	}
	if cfg.DataDir != "/opt/kf" {
		t.Errorf("DataDir: want %q, got %q", "/opt/kf", cfg.DataDir)
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

func TestEnvAdapter_DashboardEnabled(t *testing.T) {
	t.Setenv("KF_DASHBOARD_ENABLED", "false")

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

func TestEnvAdapter_OrchestratorPort(t *testing.T) {
	t.Setenv("KF_ORCH_PORT", "4001")

	adapter := &EnvAdapter{}
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OrchestratorPort != 4001 {
		t.Errorf("OrchestratorPort: want 4001, got %d", cfg.OrchestratorPort)
	}
}

func TestEnvAdapter_Model(t *testing.T) {
	t.Setenv("KF_MODEL", "sonnet")

	adapter := &EnvAdapter{}
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Model != "sonnet" {
		t.Errorf("Model: want %q, got %q", "sonnet", cfg.Model)
	}
}

func TestEnvAdapter_AgentMaxDuration(t *testing.T) {
	t.Setenv("KF_AGENT_MAX_DURATION", "30m")

	adapter := &EnvAdapter{}
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AgentMaxDuration != "30m" {
		t.Errorf("AgentMaxDuration: want %q, got %q", "30m", cfg.AgentMaxDuration)
	}
}

func TestGetAgentMaxDuration_Default(t *testing.T) {
	t.Parallel()
	cfg := &Config{}
	if got := cfg.GetAgentMaxDuration(); got != 2*time.Hour {
		t.Errorf("GetAgentMaxDuration default: want 2h, got %v", got)
	}
}

func TestGetAgentMaxDuration_Custom(t *testing.T) {
	t.Parallel()
	cfg := &Config{AgentMaxDuration: "45m"}
	if got := cfg.GetAgentMaxDuration(); got != 45*time.Minute {
		t.Errorf("GetAgentMaxDuration custom: want 45m, got %v", got)
	}
}

func TestGetAgentMaxDuration_Zero_DisablesTimeout(t *testing.T) {
	t.Parallel()
	cfg := &Config{AgentMaxDuration: "0s"}
	if got := cfg.GetAgentMaxDuration(); got != 0 {
		t.Errorf("GetAgentMaxDuration zero: want 0 (disabled), got %v", got)
	}
}

func TestGetAgentMaxDuration_InvalidFallsBackToDefault(t *testing.T) {
	t.Parallel()
	cfg := &Config{AgentMaxDuration: "notaduration"}
	if got := cfg.GetAgentMaxDuration(); got != 2*time.Hour {
		t.Errorf("GetAgentMaxDuration invalid: want 2h (default), got %v", got)
	}
}

func TestEnvAdapter_InvalidPort_Ignored(t *testing.T) {
	t.Setenv("KF_GITEA_PORT", "notanumber")

	adapter := &EnvAdapter{}
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.GiteaPort != 0 {
		t.Errorf("GiteaPort: want 0 for invalid, got %d", cfg.GiteaPort)
	}
}
