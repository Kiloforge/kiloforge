package kf

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProjectConfig holds project-level settings from .agent/kf/config.yaml.
type ProjectConfig struct {
	PrimaryBranch      string `yaml:"primary_branch"`
	EnforceDepOrdering bool   `yaml:"enforce_dep_ordering"`
}

// DefaultConfig returns a ProjectConfig with all default values.
func DefaultConfig() ProjectConfig {
	return ProjectConfig{
		PrimaryBranch:      "main",
		EnforceDepOrdering: true,
	}
}

// rawConfig is the internal parsing struct that distinguishes missing from false.
type rawConfig struct {
	PrimaryBranch      string `yaml:"primary_branch"`
	EnforceDepOrdering *bool  `yaml:"enforce_dep_ordering"`
}

func (c *Client) configFile() string {
	return filepath.Join(c.KFDir, "config.yaml")
}

// GetConfig reads .agent/kf/config.yaml and returns a ProjectConfig with
// defaults applied for any missing fields.
func (c *Client) GetConfig() (*ProjectConfig, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(c.configFile())
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return nil, fmt.Errorf("read config.yaml: %w", err)
	}

	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse config.yaml: %w", err)
	}

	if raw.PrimaryBranch != "" {
		cfg.PrimaryBranch = raw.PrimaryBranch
	}
	if raw.EnforceDepOrdering != nil {
		cfg.EnforceDepOrdering = *raw.EnforceDepOrdering
	}

	return &cfg, nil
}

// SaveConfig writes the given ProjectConfig to .agent/kf/config.yaml.
func (c *Client) SaveConfig(cfg *ProjectConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	header := "# Kiloforge Project Configuration\n#\n# primary_branch: the branch used for merges and track state\n# enforce_dep_ordering: require dependency satisfaction before claiming tracks\n\n"
	return os.WriteFile(c.configFile(), append([]byte(header), data...), 0o644)
}
