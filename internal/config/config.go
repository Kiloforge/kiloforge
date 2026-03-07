package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	ConfigFileName = "config.json"
)

type Config struct {
	GiteaPort       int    `json:"gitea_port"`
	DataDir         string `json:"data_dir"`
	APIToken        string `json:"api_token,omitempty"`
	ComposeFile     string `json:"compose_file,omitempty"`
	ContainerName   string `json:"container_name,omitempty"`
	GiteaImage      string `json:"gitea_image,omitempty"`
	GiteaAdminUser  string `json:"gitea_admin_user,omitempty"`
	GiteaAdminPass  string `json:"gitea_admin_pass,omitempty"`
	GiteaAdminEmail string `json:"gitea_admin_email,omitempty"`
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
