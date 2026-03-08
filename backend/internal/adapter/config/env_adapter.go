package config

import (
	"os"
	"strconv"
)

// EnvAdapter reads config values from KF_* environment variables.
type EnvAdapter struct{}

func (a *EnvAdapter) Load() (*Config, error) {
	cfg := &Config{
		DataDir:         os.Getenv("KF_DATA_DIR"),
		APIToken:        os.Getenv("KF_API_TOKEN"),
		ComposeFile:     os.Getenv("KF_COMPOSE_FILE"),
		ContainerName:   os.Getenv("KF_CONTAINER_NAME"),
		GiteaImage:      os.Getenv("KF_GITEA_IMAGE"),
		GiteaAdminUser:  os.Getenv("KF_GITEA_ADMIN_USER"),
		GiteaAdminPass:  os.Getenv("KF_GITEA_ADMIN_PASS"),
		GiteaAdminEmail: os.Getenv("KF_GITEA_ADMIN_EMAIL"),
	}

	if v := os.Getenv("KF_GITEA_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.GiteaPort = port
		}
	}

	if v := os.Getenv("KF_DASHBOARD_ENABLED"); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			cfg.DashboardEnabled = &b
		}
	}

	return cfg, nil
}
