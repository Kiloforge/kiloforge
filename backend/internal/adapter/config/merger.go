package config

// Merge chains providers in order (first = lowest priority, last = highest).
// For each provider, non-zero fields override the accumulator.
func Merge(providers ...ConfigProvider) (*Config, error) {
	result := &Config{}

	for _, p := range providers {
		cfg, err := p.Load()
		if err != nil {
			return nil, err
		}
		overlay(result, cfg)
	}

	return result, nil
}

func overlay(dst, src *Config) {
	if src.GiteaPort != 0 {
		dst.GiteaPort = src.GiteaPort
	}
	if src.RelayPort != 0 {
		dst.RelayPort = src.RelayPort
	}
	if src.DataDir != "" {
		dst.DataDir = src.DataDir
	}
	if src.APIToken != "" {
		dst.APIToken = src.APIToken
	}
	if src.ComposeFile != "" {
		dst.ComposeFile = src.ComposeFile
	}
	if src.ContainerName != "" {
		dst.ContainerName = src.ContainerName
	}
	if src.GiteaImage != "" {
		dst.GiteaImage = src.GiteaImage
	}
	if src.GiteaAdminUser != "" {
		dst.GiteaAdminUser = src.GiteaAdminUser
	}
	if src.GiteaAdminPass != "" {
		dst.GiteaAdminPass = src.GiteaAdminPass
	}
	if src.GiteaAdminEmail != "" {
		dst.GiteaAdminEmail = src.GiteaAdminEmail
	}
	if src.MaxSessionCostUSD != 0 {
		dst.MaxSessionCostUSD = src.MaxSessionCostUSD
	}
	if src.DashboardEnabled != nil {
		dst.DashboardEnabled = src.DashboardEnabled
	}
}
