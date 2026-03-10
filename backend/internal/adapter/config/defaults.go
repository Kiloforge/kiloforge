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
		OrchestratorPort: 4001,
		DataDir:          filepath.Join(home, ".kiloforge"),
		Model:            "opus",
	}, nil
}
