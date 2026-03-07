package config

// FlagOption sets a field on the flags adapter config.
type FlagOption func(*Config)

func WithGiteaPort(v int) FlagOption       { return func(c *Config) { c.GiteaPort = v } }
func WithRelayPort(v int) FlagOption       { return func(c *Config) { c.RelayPort = v } }
func WithDataDir(v string) FlagOption      { return func(c *Config) { c.DataDir = v } }
func WithAPIToken(v string) FlagOption     { return func(c *Config) { c.APIToken = v } }
func WithComposeFile(v string) FlagOption  { return func(c *Config) { c.ComposeFile = v } }
func WithContainerName(v string) FlagOption { return func(c *Config) { c.ContainerName = v } }
func WithGiteaImage(v string) FlagOption    { return func(c *Config) { c.GiteaImage = v } }
func WithGiteaAdminUser(v string) FlagOption  { return func(c *Config) { c.GiteaAdminUser = v } }
func WithGiteaAdminPass(v string) FlagOption  { return func(c *Config) { c.GiteaAdminPass = v } }
func WithGiteaAdminEmail(v string) FlagOption { return func(c *Config) { c.GiteaAdminEmail = v } }

// FlagsAdapter provides config values from explicitly set CLI flags.
type FlagsAdapter struct {
	cfg Config
}

// NewFlagsAdapter creates a FlagsAdapter with only the explicitly provided options.
func NewFlagsAdapter(opts ...FlagOption) *FlagsAdapter {
	a := &FlagsAdapter{}
	for _, opt := range opts {
		opt(&a.cfg)
	}
	return a
}

func (a *FlagsAdapter) Load() (*Config, error) {
	cfg := a.cfg // copy
	return &cfg, nil
}
