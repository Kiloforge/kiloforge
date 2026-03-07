package dashboard

import (
	"context"
	"time"

	"crelay/internal/core/domain"
)

type watcherState struct {
	agents      map[string]string // id -> status
	rateLimited bool
	totalCost   float64
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
			s.hub.Broadcast(SSEEvent{
				Type: "agent_update",
				Data: agentToJSON(a),
			})
		}
	}
	// Detect removed agents.
	for id := range prev.agents {
		if _, ok := cur.agents[id]; !ok {
			s.hub.Broadcast(SSEEvent{
				Type: "agent_removed",
				Data: map[string]string{"id": id},
			})
		}
	}

	// Check quota changes.
	if s.quota != nil {
		total := s.quota.GetTotalUsage()
		cur.totalCost = total.TotalCostUSD
		cur.rateLimited = s.quota.IsRateLimited()

		if cur.totalCost != prev.totalCost || cur.rateLimited != prev.rateLimited {
			s.hub.Broadcast(SSEEvent{
				Type: "quota_update",
				Data: s.quotaResponse(),
			})
		}
	}

	return cur
}

func agentToJSON(a domain.AgentInfo) map[string]any {
	return map[string]any{
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
}
