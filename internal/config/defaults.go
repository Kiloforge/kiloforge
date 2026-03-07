package config

import (
	"os"
	"path/filepath"
)

// DefaultsAdapter provides fallback default values for all config fields.
type DefaultsAdapter struct{}

func (d *DefaultsAdapter) Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return &Config{
		GiteaPort:       3000,
		DataDir:         filepath.Join(home, ".crelay"),
		ContainerName:   "conductor-gitea",
		GiteaImage:      "gitea/gitea:latest",
		GiteaAdminUser:  "conductor",
		GiteaAdminPass:  "conductor123",
		GiteaAdminEmail: "conductor@local.dev",
	}, nil
}
