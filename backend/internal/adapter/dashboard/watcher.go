package dashboard

import (
	"context"
	"time"

	"kiloforge/internal/core/domain"
)

type watcherState struct {
	agents       map[string]string // id -> status
	rateLimited  bool
	totalCost    float64
	inputTokens  int
	outputTokens int
	tracks       map[string]string // id -> status
	traceCount   int
}

// StartWatcher launches the background state watcher goroutine.
func (s *Server) StartWatcher(ctx context.Context) {
	go s.watchState(ctx)
}

func (s *Server) watchState(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var prev watcherState
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			prev = s.checkAndBroadcast(prev)
		}
	}
}

func (s *Server) checkAndBroadcast(prev watcherState) watcherState {
	// Reload agent state from disk.
	if err := s.agents.Load(); err != nil {
		return prev
	}

	agents := s.agents.Agents()
	cur := watcherState{
		agents: make(map[string]string, len(agents)),
	}
	for _, a := range agents {
		cur.agents[a.ID] = a.Status
	}

	// Detect agent changes.
	for _, a := range agents {
		oldStatus, existed := prev.agents[a.ID]
		if !existed || oldStatus != a.Status {
			s.hub.Publish(domain.NewAgentUpdateEvent(agentToJSON(a)))
		}
	}
	// Detect removed agents.
	for id := range prev.agents {
		if _, ok := cur.agents[id]; !ok {
			s.hub.Publish(domain.NewAgentRemovedEvent(id))
		}
	}

	// Check quota changes.
	if s.quota != nil {
		total := s.quota.GetTotalUsage()
		cur.totalCost = total.TotalCostUSD
		cur.inputTokens = total.InputTokens
		cur.outputTokens = total.OutputTokens
		cur.rateLimited = s.quota.IsRateLimited()

		if cur.totalCost != prev.totalCost || cur.rateLimited != prev.rateLimited ||
			cur.inputTokens != prev.inputTokens || cur.outputTokens != prev.outputTokens {
			s.hub.Publish(domain.NewQuotaUpdateEvent(s.quotaResponse()))
		}
	}

	// Detect track changes across all projects.
	cur.tracks = make(map[string]string)
	if s.projects != nil {
		for _, p := range s.projects.List() {
			tracks, err := s.trackReader.DiscoverTracks(p.ProjectDir)
			if err != nil {
				continue
			}
			for _, t := range tracks {
				cur.tracks[t.ID] = t.Status
			}
		}
	}
	if prev.tracks != nil {
		for id, status := range cur.tracks {
			oldStatus, existed := prev.tracks[id]
			if !existed || oldStatus != status {
				s.hub.Publish(domain.NewTrackUpdateEvent(map[string]string{
					"id":     id,
					"status": status,
				}))
			}
		}
		for id := range prev.tracks {
			if _, ok := cur.tracks[id]; !ok {
				s.hub.Publish(domain.NewTrackRemovedEvent(id))
			}
		}
	}

	// Detect new traces.
	if s.traceStore != nil {
		traces := s.traceStore.ListTraces()
		cur.traceCount = len(traces)
		if cur.traceCount > prev.traceCount && prev.traceCount > 0 {
			// Emit update for each new trace (approximate — just emit all current).
			// Since traces are append-only, new ones are the delta.
			for i := 0; i < cur.traceCount-prev.traceCount && i < len(traces); i++ {
				t := traces[i]
				s.hub.Publish(domain.NewTraceUpdateEvent(map[string]any{
					"trace_id":   t.TraceID,
					"root_name":  t.RootName,
					"span_count": t.SpanCount,
				}))
			}
		}
	}

	return cur
}

func agentToJSON(a domain.AgentInfo) map[string]any {
	m := map[string]any{
		"id":           a.ID,
		"role":         a.Role,
		"ref":          a.Ref,
		"status":       a.Status,
		"session_id":   a.SessionID,
		"pid":          a.PID,
		"worktree_dir": a.WorktreeDir,
		"log_file":     a.LogFile,
		"started_at":   a.StartedAt,
		"updated_at":   a.UpdatedAt,
	}
	if a.SuspendedAt != nil {
		m["suspended_at"] = a.SuspendedAt
	}
	if a.ShutdownReason != "" {
		m["shutdown_reason"] = a.ShutdownReason
	}
	return m
}
