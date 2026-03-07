package config

import (
	"os"
	"strconv"
)

// EnvAdapter reads config values from CRELAY_* environment variables.
type EnvAdapter struct{}

func (a *EnvAdapter) Load() (*Config, error) {
	cfg := &Config{
		DataDir:         os.Getenv("CRELAY_DATA_DIR"),
		APIToken:        os.Getenv("CRELAY_API_TOKEN"),
		ComposeFile:     os.Getenv("CRELAY_COMPOSE_FILE"),
		ContainerName:   os.Getenv("CRELAY_CONTAINER_NAME"),
		GiteaImage:      os.Getenv("CRELAY_GITEA_IMAGE"),
		GiteaAdminUser:  os.Getenv("CRELAY_GITEA_ADMIN_USER"),
		GiteaAdminPass:  os.Getenv("CRELAY_GITEA_ADMIN_PASS"),
		GiteaAdminEmail: os.Getenv("CRELAY_GITEA_ADMIN_EMAIL"),
	}

	if v := os.Getenv("CRELAY_GITEA_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.GiteaPort = port
		}
	}

	if v := os.Getenv("CRELAY_DASHBOARD_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.DashboardPort = port
		}
	}

	if v := os.Getenv("CRELAY_DASHBOARD_ENABLED"); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			cfg.DashboardEnabled = &b
		}
	}

	return cfg, nil
}
