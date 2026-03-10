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
	t.Setenv("KF_DATA_DIR", "/opt/kf")

	adapter := &EnvAdapter{}
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.DataDir != "/opt/kf" {
		t.Errorf("DataDir: want %q, got %q", "/opt/kf", cfg.DataDir)
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

func TestEnvAdapter_AnalyticsEnabled(t *testing.T) {
	t.Setenv("KF_ANALYTICS_ENABLED", "false")
	adapter := &EnvAdapter{}
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AnalyticsEnabled == nil {
		t.Fatal("AnalyticsEnabled: want non-nil")
	}
	if *cfg.AnalyticsEnabled != false {
		t.Errorf("AnalyticsEnabled: want false, got %v", *cfg.AnalyticsEnabled)
	}
}

func TestEnvAdapter_OrchestratorHost(t *testing.T) {
	t.Setenv("KF_ORCH_HOST", "0.0.0.0")

	adapter := &EnvAdapter{}
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OrchestratorHost != "0.0.0.0" {
		t.Errorf("OrchestratorHost: want %q, got %q", "0.0.0.0", cfg.OrchestratorHost)
	}
}

func TestEnvAdapter_PostHogAPIKey(t *testing.T) {
	t.Setenv("KF_POSTHOG_API_KEY", "phc_test123")
	adapter := &EnvAdapter{}
	cfg, err := adapter.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PostHogAPIKey != "phc_test123" {
		t.Errorf("PostHogAPIKey: want %q, got %q", "phc_test123", cfg.PostHogAPIKey)
	}
}
