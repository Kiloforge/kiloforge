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
		OrchestratorPort: 39517,
		OrchestratorHost: "127.0.0.1",
		DataDir:          filepath.Join(home, ".kiloforge"),
		Model:            "opus",
		SkillsRepo:       "kiloforge/kiloforge-skills",
	}, nil
}
