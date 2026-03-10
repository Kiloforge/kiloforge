package dashboard

// quotaResponse builds the quota summary map used by the SSE watcher.
func (s *Server) quotaResponse() map[string]any {
	if s.quota == nil {
		return map[string]any{
			"estimated_cost_usd":    0,
			"input_tokens":          0,
			"output_tokens":         0,
			"cache_read_tokens":     0,
			"cache_creation_tokens": 0,
			"agent_count":           0,
			"rate_limited":          false,
		}
	}
	total := s.quota.GetTotalUsage()
	resp := map[string]any{
		"estimated_cost_usd":    total.TotalCostUSD,
		"input_tokens":          total.InputTokens,
		"output_tokens":         total.OutputTokens,
		"cache_read_tokens":     total.CacheReadTokens,
		"cache_creation_tokens": total.CacheCreationTokens,
		"agent_count":           total.AgentCount,
		"rate_limited":          s.quota.IsRateLimited(),
	}
	if s.quota.IsRateLimited() {
		resp["retry_after_seconds"] = int(s.quota.RetryAfter().Seconds())
	}

	// Per-agent breakdown.
	if err := s.agents.Load(); err == nil {
		agents := s.agents.Agents()
		var perAgent []map[string]any
		for _, a := range agents {
			if usage := s.quota.GetAgentUsage(a.ID); usage != nil {
				perAgent = append(perAgent, map[string]any{
					"agent_id":              a.ID,
					"estimated_cost_usd":    usage.TotalCostUSD,
					"input_tokens":          usage.InputTokens,
					"output_tokens":         usage.OutputTokens,
					"cache_read_tokens":     usage.CacheReadTokens,
					"cache_creation_tokens": usage.CacheCreationTokens,
				})
			}
		}
		resp["agents"] = perAgent
	}

	return resp
}
