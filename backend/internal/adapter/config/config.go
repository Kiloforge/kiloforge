package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	ConfigFileName = "config.json"
)

type Config struct {
	OrchestratorPort int    `json:"orchestrator_port"`
	DataDir          string `json:"data_dir"`
	// Deprecated: MaxSessionCostUSD is no longer enforced. Subscription rate
	// limits are the primary constraint. Retained for backward compatibility.
	MaxSessionCostUSD float64 `json:"max_session_cost_usd,omitempty"`
	DashboardEnabled  *bool   `json:"dashboard_enabled,omitempty"`
	SkillsRepo        string  `json:"skills_repo,omitempty"`
	SkillsVersion     string  `json:"skills_version,omitempty"`
	AutoUpdateSkills  *bool   `json:"auto_update_skills,omitempty"`
	SkillsDir         string  `json:"skills_dir,omitempty"`
	Model             string  `json:"model,omitempty"`
	MaxSwarmSize      int     `json:"max_swarm_size,omitempty"`
	MaxWorkers        int     `json:"max_workers,omitempty"` // Deprecated: use MaxSwarmSize. Kept for backwards compat on load.
	QueueEnabled      *bool   `json:"queue_enabled,omitempty"`
	AgentMaxDuration  string  `json:"agent_max_duration,omitempty"`
	AnalyticsEnabled  *bool   `json:"analytics_enabled,omitempty"`
	PostHogAPIKey     string  `json:"posthog_api_key,omitempty"`
	BudgetUSD            float64 `json:"budget_usd,omitempty"`
	IdleSuspendSeconds   *int    `json:"idle_suspend_seconds,omitempty"`
}

// GetMaxSwarmSize returns the configured max swarm size, defaulting to 3.
// Falls back to the deprecated MaxWorkers field for backwards compatibility.
func (c *Config) GetMaxSwarmSize() int {
	if c.MaxSwarmSize > 0 {
		return c.MaxSwarmSize
	}
	if c.MaxWorkers > 0 {
		return c.MaxWorkers
	}
	return 3
}

// GetMaxWorkers is deprecated — use GetMaxSwarmSize.
func (c *Config) GetMaxWorkers() int {
	return c.GetMaxSwarmSize()
}

// GetAgentMaxDuration returns the configured agent max duration, defaulting to 2 hours.
// A zero duration means timeout enforcement is disabled.
func (c *Config) GetAgentMaxDuration() time.Duration {
	if c.AgentMaxDuration == "" {
		return 2 * time.Hour
	}
	d, err := time.ParseDuration(c.AgentMaxDuration)
	if err != nil {
		return 2 * time.Hour
	}
	return d
}

// IsQueueEnabled returns whether the work queue is enabled. Defaults to false.
func (c *Config) IsQueueEnabled() bool {
	if c.QueueEnabled == nil {
		return false
	}
	return *c.QueueEnabled
}

// IsDashboardEnabled returns whether the dashboard is enabled.
// Defaults to true when DashboardEnabled is nil.
func (c *Config) IsDashboardEnabled() bool {
	if c.DashboardEnabled == nil {
		return true
	}
	return *c.DashboardEnabled
}

// IsAnalyticsEnabled returns whether analytics is enabled.
// Defaults to true when AnalyticsEnabled is nil.
func (c *Config) IsAnalyticsEnabled() bool {
	if c.AnalyticsEnabled == nil {
		return true
	}
	return *c.AnalyticsEnabled
}

// GetIdleSuspendSeconds returns the idle disconnect grace period in seconds.
// Default is 30. A value of 0 disables auto-suspension.
func (c *Config) GetIdleSuspendSeconds() int {
	if c.IdleSuspendSeconds != nil {
		return *c.IdleSuspendSeconds
	}
	return 30
}

// GetSkillsDir returns the configured skills directory, or the default (~/.claude/skills).
func (c *Config) GetSkillsDir() string {
	if c.SkillsDir != "" {
		return c.SkillsDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "skills")
}

// Validate checks that critical configuration values are within acceptable bounds.
func (c *Config) Validate() error {
	if c.DataDir == "" {
		return fmt.Errorf("config: data_dir must not be empty")
	}
	if c.OrchestratorPort < 1 || c.OrchestratorPort > 65535 {
		return fmt.Errorf("config: orchestrator_port %d out of range (1-65535)", c.OrchestratorPort)
	}
	return nil
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

	cfg, err := Merge(providers...)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
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
