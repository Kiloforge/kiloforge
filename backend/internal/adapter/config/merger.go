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
	if src.OrchestratorPort != 0 {
		dst.OrchestratorPort = src.OrchestratorPort
	}
	if src.DataDir != "" {
		dst.DataDir = src.DataDir
	}
	if src.ComposeFile != "" {
		dst.ComposeFile = src.ComposeFile
	}
	if src.ContainerName != "" {
		dst.ContainerName = src.ContainerName
	}
	if src.MaxSessionCostUSD != 0 {
		dst.MaxSessionCostUSD = src.MaxSessionCostUSD
	}
	if src.DashboardEnabled != nil {
		dst.DashboardEnabled = src.DashboardEnabled
	}
	if src.SkillsRepo != "" {
		dst.SkillsRepo = src.SkillsRepo
	}
	if src.SkillsVersion != "" {
		dst.SkillsVersion = src.SkillsVersion
	}
	if src.AutoUpdateSkills != nil {
		dst.AutoUpdateSkills = src.AutoUpdateSkills
	}
	if src.SkillsDir != "" {
		dst.SkillsDir = src.SkillsDir
	}
	if src.Model != "" {
		dst.Model = src.Model
	}
	if src.MaxSwarmSize != 0 {
		dst.MaxSwarmSize = src.MaxSwarmSize
	}
	if src.MaxWorkers != 0 {
		dst.MaxWorkers = src.MaxWorkers
	}
	if src.QueueEnabled != nil {
		dst.QueueEnabled = src.QueueEnabled
	}
	if src.AgentMaxDuration != "" {
		dst.AgentMaxDuration = src.AgentMaxDuration
	}
	if src.AnalyticsEnabled != nil {
		dst.AnalyticsEnabled = src.AnalyticsEnabled
	}
	if src.PostHogAPIKey != "" {
		dst.PostHogAPIKey = src.PostHogAPIKey
	}
}
