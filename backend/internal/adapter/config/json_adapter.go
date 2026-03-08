package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// JSONAdapter reads and writes config from a JSON file.
type JSONAdapter struct {
	dataDir string
}

// NewJSONAdapter creates a JSONAdapter that reads/writes {dataDir}/config.json.
func NewJSONAdapter(dataDir string) *JSONAdapter {
	return &JSONAdapter{dataDir: dataDir}
}

// Load reads config from the JSON file. Returns a zero Config (not an error)
// if the file does not exist.
func (a *JSONAdapter) Load() (*Config, error) {
	data, err := os.ReadFile(filepath.Join(a.dataDir, ConfigFileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes the config to the JSON file.
// Sensitive fields (e.g. GiteaAdminPass) are stripped before writing.
func (a *JSONAdapter) Save(cfg *Config) error {
	safe := *cfg
	safe.GiteaAdminPass = ""
	data, err := json.MarshalIndent(&safe, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(a.dataDir, ConfigFileName), data, 0o644)
}
