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
		GiteaPort:       4000,
		OrchestratorPort: 4001,
		DataDir:         filepath.Join(home, ".kiloforge"),
		ContainerName:   "kf-gitea",
		GiteaImage:      "gitea/gitea:latest",
		GiteaAdminUser:  "kiloforger",
		// GiteaAdminPass intentionally omitted — resolved via flag, saved config, or generated.
		GiteaAdminEmail: "kiloforger@local.dev",
		Model:           "opus",
	}, nil
}
