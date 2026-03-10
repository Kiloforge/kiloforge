package service

import (
	"context"
	"fmt"
	"os"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

// ColumnOrder defines the priority ordering of board columns.
// Delegates to domain.ColumnOrder for the canonical column names.
var ColumnOrder = domain.ColumnOrder

// LifecycleService handles agent lifecycle control driven by board state changes.
type LifecycleService struct {
	agents    port.AgentStore
	spawner   port.AgentSpawner
	pool      port.PoolReturner
	logger    port.Logger
	analytics port.AnalyticsTracker
}

// NewLifecycleService creates a new LifecycleService.
func NewLifecycleService(agents port.AgentStore, spawner port.AgentSpawner, pool port.PoolReturner, logger port.Logger) *LifecycleService {
	return &LifecycleService{
		agents:  agents,
		spawner: spawner,
		pool:    pool,
		logger:  logger,
	}
}

// SetAnalyticsTracker sets the analytics tracker for lifecycle events.
func (s *LifecycleService) SetAnalyticsTracker(t port.AnalyticsTracker) {
	s.analytics = t
}

// HandleBackwardMove processes a backward column transition for a track.
// Stashes agent WIP before halting so it can be restored on re-promotion.
func (s *LifecycleService) HandleBackwardMove(ctx context.Context, trackID, fromCol, toCol string, prTracking *domain.PRTracking) {
	// Stash agent work before halting.
	if s.pool != nil {
		if err := s.pool.StashByTrackID(trackID); err != nil {
			s.logger.Printf("stash for %s: %v", trackID, err)
		}
	}

	agent := s.agents.FindByRef(trackID)

	if agent != nil {
		s.haltIfActive(agent, "board-demotion")
	}

	_ = s.agents.Save()
}

// HandleRepromotion processes a forward re-promotion for a halted track.
func (s *LifecycleService) HandleRepromotion(ctx context.Context, trackID, toCol string, prTracking *domain.PRTracking) (resumed bool, reason string) {
	if toCol == "in_progress" {
		return s.resumeDeveloper(ctx, trackID)
	}
	return false, "no action for column " + toCol
}

// HandleRejection terminates an agent and returns its worktree when a track is rejected.
func (s *LifecycleService) HandleRejection(ctx context.Context, trackID string, prTracking *domain.PRTracking) {
	agent := s.agents.FindByRef(trackID)

	if agent != nil {
		status := agent.Status
		if status == string(domain.AgentStatusRunning) || status == string(domain.AgentStatusWaiting) || status == "waiting-review" || status == string(domain.AgentStatusHalted) {
			_ = s.agents.HaltAgent(agent.ID)
			_ = s.agents.UpdateStatus(agent.ID, string(domain.AgentStatusStopped))
			if a := s.agents.FindByRef(trackID); a != nil {
				a.ShutdownReason = "track-rejected"
			}
		}
	}

	if s.pool != nil {
		if err := s.pool.ReturnByTrackID(trackID); err != nil {
			s.logger.Printf("pool return for %s: %v", trackID, err)
		}
	}

	_ = s.agents.Save()

	if s.analytics != nil {
		s.analytics.Track(ctx, "track_rejected", map[string]any{
			"track_id": trackID,
		})
	}
}

// IsBackwardMove returns true if toCol is earlier in the workflow than fromCol.
func IsBackwardMove(fromCol, toCol string) bool {
	return domain.IsBackwardMove(fromCol, toCol)
}

// IsForwardMove returns true if toCol is later in the workflow than fromCol.
func IsForwardMove(fromCol, toCol string) bool {
	return domain.IsForwardMove(fromCol, toCol)
}

func (s *LifecycleService) haltIfActive(agent *domain.AgentInfo, reason string) {
	status := agent.Status
	switch status {
	case string(domain.AgentStatusRunning), string(domain.AgentStatusWaiting), "waiting-review":
		err := s.agents.HaltAgent(agent.ID)
		if err != nil {
			s.logger.Printf("halt agent %s: %v (marking halted anyway)", agent.ID, err)
		}
		_ = s.agents.UpdateStatus(agent.ID, string(domain.AgentStatusHalted))
		if a := s.agents.FindByRef(agent.Ref); a != nil {
			a.ShutdownReason = reason
		}
	case string(domain.AgentStatusCompleted), string(domain.AgentStatusStopped):
		s.logger.Printf("agent %s already %s, skipping halt", agent.ID, status)
	case string(domain.AgentStatusHalted):
		s.logger.Printf("agent %s already halted, skipping", agent.ID)
	}
}

func (s *LifecycleService) resumeDeveloper(ctx context.Context, trackID string) (bool, string) {
	agent := s.agents.FindByRef(trackID)
	if agent == nil {
		return false, "no agent found for track"
	}
	if agent.Status != string(domain.AgentStatusHalted) {
		return false, fmt.Sprintf("agent not halted (status: %s)", agent.Status)
	}
	if agent.SessionID == "" {
		_ = s.agents.UpdateStatus(agent.ID, string(domain.AgentStatusResumeFailed))
		_ = s.agents.Save()
		return false, "no session to resume"
	}
	if agent.WorktreeDir != "" {
		if _, err := os.Stat(agent.WorktreeDir); os.IsNotExist(err) {
			_ = s.agents.UpdateStatus(agent.ID, string(domain.AgentStatusResumeFailed))
			_ = s.agents.Save()
			return false, "worktree not found"
		}
	}

	if err := s.spawner.ResumeDeveloper(ctx, agent.SessionID, agent.WorktreeDir); err != nil {
		_ = s.agents.UpdateStatus(agent.ID, string(domain.AgentStatusResumeFailed))
		if a := s.agents.FindByRef(trackID); a != nil {
			a.ResumeError = err.Error()
		}
		_ = s.agents.Save()
		return false, fmt.Sprintf("resume failed: %v", err)
	}

	_ = s.agents.UpdateStatus(agent.ID, string(domain.AgentStatusRunning))
	if a := s.agents.FindByRef(trackID); a != nil {
		a.ShutdownReason = ""
	}
	_ = s.agents.Save()
	return true, ""
}
