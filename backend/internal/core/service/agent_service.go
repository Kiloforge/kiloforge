package service

import (
	"fmt"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

// AgentService provides agent operations for CLI commands and API handlers.
type AgentService struct {
	agents    port.AgentStore
	projects  port.ProjectStore
	prTracker port.PRTrackingStore
}

// NewAgentService creates a new AgentService.
func NewAgentService(agents port.AgentStore, projects port.ProjectStore, prTracker port.PRTrackingStore) *AgentService {
	return &AgentService{
		agents:    agents,
		projects:  projects,
		prTracker: prTracker,
	}
}

// ListAgents returns all tracked agents.
func (s *AgentService) ListAgents() []domain.AgentInfo {
	return s.agents.Agents()
}

// GetAgent finds an agent by ID prefix.
func (s *AgentService) GetAgent(idPrefix string) (*domain.AgentInfo, error) {
	return s.agents.FindAgent(idPrefix)
}

// StopAgent sends SIGINT to a running agent and updates its status.
func (s *AgentService) StopAgent(idPrefix string) (*domain.AgentInfo, error) {
	agent, err := s.agents.FindAgent(idPrefix)
	if err != nil {
		return nil, err
	}

	if agent.Status != "running" && agent.Status != "waiting" {
		return agent, fmt.Errorf("agent %s is not running (status: %s)", idPrefix, agent.Status)
	}

	if err := s.agents.HaltAgent(idPrefix); err != nil {
		return agent, fmt.Errorf("halt agent: %w", err)
	}

	if err := s.agents.UpdateStatus(idPrefix, "stopped"); err != nil {
		return agent, fmt.Errorf("update status: %w", err)
	}
	if err := s.agents.Save(); err != nil {
		return agent, fmt.Errorf("save state: %w", err)
	}

	return agent, nil
}

// AttachAgent finds an agent and optionally halts it for interactive takeover.
func (s *AgentService) AttachAgent(idPrefix string) (*domain.AgentInfo, error) {
	agent, err := s.agents.FindAgent(idPrefix)
	if err != nil {
		return nil, err
	}

	if agent.Status == "running" && agent.PID > 0 {
		if err := s.agents.HaltAgent(idPrefix); err != nil {
			return agent, fmt.Errorf("halt agent: %w", err)
		}
		if err := s.agents.UpdateStatus(idPrefix, "halted"); err != nil {
			return agent, fmt.Errorf("update status: %w", err)
		}
		_ = s.agents.Save()
	}

	return agent, nil
}

// GetEscalated returns all PRs that have been escalated across all projects.
func (s *AgentService) GetEscalated() []domain.EscalatedItem {
	var items []domain.EscalatedItem
	for _, proj := range s.projects.List() {
		tracking, err := s.prTracker.LoadPRTracking(proj.Slug)
		if err != nil {
			continue
		}
		if tracking.Status == "escalated" {
			items = append(items, domain.EscalatedItem{
				Slug:    proj.Slug,
				PR:      tracking.PRNumber,
				TrackID: tracking.TrackID,
				Cycles:  tracking.ReviewCycleCount,
			})
		}
	}
	return items
}
