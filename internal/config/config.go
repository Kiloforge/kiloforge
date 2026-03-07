package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	ContainerName = "conductor-gitea"
	GiteaImage    = "gitea/gitea:latest"
	GiteaAdminUser = "conductor"
	GiteaAdminPass = "conductor123"
	GiteaAdminEmail = "conductor@local.dev"
	ConfigFileName = "config.json"
)

type Config struct {
	GiteaPort  int    `json:"gitea_port"`
	RelayPort  int    `json:"relay_port"`
	RepoName   string `json:"repo_name"`
	ProjectDir string `json:"project_dir"`
	DataDir    string `json:"data_dir"`
	APIToken   string `json:"api_token,omitempty"`
}

func (c *Config) GiteaURL() string {
	return fmt.Sprintf("http://localhost:%d", c.GiteaPort)
}

func (c *Config) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.DataDir, ConfigFileName), data, 0o644)
}

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dataDir := filepath.Join(home, ".crelay")
	return LoadFrom(dataDir)
}

func LoadFrom(dataDir string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(dataDir, ConfigFileName))
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
