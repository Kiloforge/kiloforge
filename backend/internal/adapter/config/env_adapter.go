package config

import (
	"os"
	"strconv"
)

// EnvAdapter reads config values from KF_* environment variables.
type EnvAdapter struct{}

func (a *EnvAdapter) Load() (*Config, error) {
	cfg := &Config{
		DataDir:          os.Getenv("KF_DATA_DIR"),
		Model:            os.Getenv("KF_MODEL"),
		AgentMaxDuration: os.Getenv("KF_AGENT_MAX_DURATION"),
	}

	if v := os.Getenv("KF_ORCH_HOST"); v != "" {
		cfg.OrchestratorHost = v
	}

	if v := os.Getenv("KF_ORCH_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.OrchestratorPort = port
		}
	}

	if v := os.Getenv("KF_DASHBOARD_ENABLED"); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			cfg.DashboardEnabled = &b
		}
	}

	if v := os.Getenv("KF_ANALYTICS_ENABLED"); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			cfg.AnalyticsEnabled = &b
		}
	}

	if v := os.Getenv("KF_POSTHOG_API_KEY"); v != "" {
		cfg.PostHogAPIKey = v
	}

	if v := os.Getenv("KF_AGENT_IDLE_SUSPEND_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.IdleSuspendSeconds = &n
		}
	}

	return cfg, nil
}
