package config

import (
	"fmt"
)

const (
	ConfigFileName = "config.json"
)

type Config struct {
	GiteaPort       int    `json:"gitea_port"`
	RelayPort       int    `json:"relay_port"`
	DataDir         string `json:"data_dir"`
	APIToken        string `json:"api_token,omitempty"`
	ComposeFile     string `json:"compose_file,omitempty"`
	ContainerName   string `json:"container_name,omitempty"`
	GiteaImage      string `json:"gitea_image,omitempty"`
	GiteaAdminUser  string `json:"gitea_admin_user,omitempty"`
	GiteaAdminPass  string `json:"gitea_admin_pass,omitempty"`
	GiteaAdminEmail    string  `json:"gitea_admin_email,omitempty"`
	MaxSessionCostUSD  float64 `json:"max_session_cost_usd,omitempty"`
	DashboardPort      int     `json:"dashboard_port,omitempty"`
	DashboardEnabled   *bool   `json:"dashboard_enabled,omitempty"`
}

// IsDashboardEnabled returns whether the dashboard is enabled.
// Defaults to true when DashboardEnabled is nil.
func (c *Config) IsDashboardEnabled() bool {
	if c.DashboardEnabled == nil {
		return true
	}
	return *c.DashboardEnabled
}

func (c *Config) GiteaURL() string {
	return fmt.Sprintf("http://localhost:%d", c.GiteaPort)
}

func (c *Config) Save() error {
	return NewJSONAdapter(c.DataDir).Save(c)
}

// Resolve chains config providers: defaults → JSON file → env → extra providers.
// Extra providers (typically a FlagsAdapter) have the highest priority.
func Resolve(extra ...ConfigProvider) (*Config, error) {
	// First pass: resolve data dir from all sources so we know where the JSON file is.
	defaults := &DefaultsAdapter{}
	env := &EnvAdapter{}

	preProviders := []ConfigProvider{defaults, env}
	preProviders = append(preProviders, extra...)
	preCfg, err := Merge(preProviders...)
	if err != nil {
		return nil, err
	}

	// Build the full chain with the JSON adapter using the resolved data dir.
	providers := []ConfigProvider{
		defaults,
		NewJSONAdapter(preCfg.DataDir),
		env,
	}
	providers = append(providers, extra...)

	return Merge(providers...)
}

// Load resolves config using the default chain (defaults → JSON → env).
// Retained for backward compatibility.
func Load() (*Config, error) {
	return Resolve()
}

// LoadFrom resolves config using a specific data directory for the JSON file.
func LoadFrom(dataDir string) (*Config, error) {
	return Resolve(NewFlagsAdapter(WithDataDir(dataDir)))
}
