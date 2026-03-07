package dashboard

import (
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// quotaResponse builds the quota summary map used by the SSE watcher.
func (s *Server) quotaResponse() map[string]any {
	if s.quota == nil {
		return map[string]any{
			"total_cost_usd": 0,
			"rate_limited":   false,
		}
	}
	total := s.quota.GetTotalUsage()
	resp := map[string]any{
		"total_cost_usd": total.TotalCostUSD,
		"input_tokens":   total.InputTokens,
		"output_tokens":  total.OutputTokens,
		"agent_count":    total.AgentCount,
		"rate_limited":   s.quota.IsRateLimited(),
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
					"agent_id":      a.ID,
					"cost_usd":      usage.TotalCostUSD,
					"input_tokens":  usage.InputTokens,
					"output_tokens": usage.OutputTokens,
				})
			}
		}
		resp["agents"] = perAgent
	}

	return resp
}
