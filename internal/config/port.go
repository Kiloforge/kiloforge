package config

// ConfigProvider supplies config values from a single source.
// Returns a Config with only the fields this source knows about.
// Zero-value fields mean "not set by this source."
type ConfigProvider interface {
	Load() (*Config, error)
}
