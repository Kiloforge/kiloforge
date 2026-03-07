package dashboard

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"crelay/internal/core/service"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleAgents(w http.ResponseWriter, _ *http.Request) {
	if err := s.agents.Load(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load agent state")
		return
	}
	agents := s.agents.Agents()
	result := make([]map[string]any, 0, len(agents))
	for _, a := range agents {
		m := agentToJSON(a)
		// Add computed uptime.
		if !a.StartedAt.IsZero() {
			m["uptime_seconds"] = int(time.Since(a.StartedAt).Seconds())
		}
		// Attach per-agent quota if available.
		if s.quota != nil {
			if usage := s.quota.GetAgentUsage(a.ID); usage != nil {
				m["cost_usd"] = usage.TotalCostUSD
				m["input_tokens"] = usage.InputTokens
				m["output_tokens"] = usage.OutputTokens
			}
		}
		result = append(result, m)
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing agent id")
		return
	}
	if err := s.agents.Load(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load agent state")
		return
	}
	a, err := s.agents.FindAgent(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	m := agentToJSON(*a)
	if !a.StartedAt.IsZero() {
		m["uptime_seconds"] = int(time.Since(a.StartedAt).Seconds())
	}
	if s.quota != nil {
		if usage := s.quota.GetAgentUsage(a.ID); usage != nil {
			m["cost_usd"] = usage.TotalCostUSD
			m["input_tokens"] = usage.InputTokens
			m["output_tokens"] = usage.OutputTokens
		}
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) handleAgentLog(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing agent id")
		return
	}
	if err := s.agents.Load(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load agent state")
		return
	}
	a, err := s.agents.FindAgent(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	if a.LogFile == "" {
		writeError(w, http.StatusNotFound, "no log file for agent")
		return
	}

	f, err := os.Open(a.LogFile)
	if err != nil {
		writeError(w, http.StatusNotFound, "log file not accessible")
		return
	}
	defer f.Close()

	lines := 100
	if n := r.URL.Query().Get("lines"); n != "" {
		if parsed, err := strconv.Atoi(n); err == nil && parsed > 0 {
			lines = parsed
		}
	}

	// Read all lines and return the last N.
	var allLines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	start := 0
	if len(allLines) > lines {
		start = len(allLines) - lines
	}
	tail := allLines[start:]

	follow := r.URL.Query().Get("follow") == "true"
	if follow {
		// SSE-style streaming of log lines.
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeError(w, http.StatusInternalServerError, "streaming not supported")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		// Send existing tail first.
		for _, line := range tail {
			fmt.Fprintf(w, "data: %s\n\n", line)
		}
		flusher.Flush()

		// Then poll for new lines.
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		offset := len(allLines)
		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				f2, err := os.Open(a.LogFile)
				if err != nil {
					return
				}
				scanner2 := bufio.NewScanner(f2)
				lineNum := 0
				for scanner2.Scan() {
					lineNum++
					if lineNum > offset {
						fmt.Fprintf(w, "data: %s\n\n", scanner2.Text())
					}
				}
				f2.Close()
				if lineNum > offset {
					offset = lineNum
					flusher.Flush()
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"agent_id": a.ID,
		"lines":    tail,
		"total":    len(allLines),
	})
}

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

func (s *Server) handleQuota(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.quotaResponse())
}

func (s *Server) handleTracks(w http.ResponseWriter, _ *http.Request) {
	tracks, err := service.DiscoverTracks(s.projectDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read tracks")
		return
	}
	result := make([]map[string]string, 0, len(tracks))
	for _, t := range tracks {
		result = append(result, map[string]string{
			"id":     t.ID,
			"title":  t.Title,
			"status": t.Status,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	if err := s.agents.Load(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load state")
		return
	}
	agents := s.agents.Agents()
	counts := make(map[string]int)
	for _, a := range agents {
		counts[a.Status]++
	}

	status := map[string]any{
		"gitea_url":    s.giteaURL,
		"agent_counts": counts,
		"total_agents": len(agents),
		"sse_clients":  s.hub.ClientCount(),
	}
	if s.quota != nil {
		status["rate_limited"] = s.quota.IsRateLimited()
		total := s.quota.GetTotalUsage()
		status["total_cost_usd"] = total.TotalCostUSD
	}
	writeJSON(w, http.StatusOK, status)
}
