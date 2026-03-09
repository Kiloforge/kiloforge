package port

import "kiloforge/internal/core/domain"

// AgentStore persists and retrieves tracked agent state.
type AgentStore interface {
	Load() error
	Save() error
	AddAgent(info domain.AgentInfo)
	FindAgent(idPrefix string) (*domain.AgentInfo, error)
	FindByRef(ref string) *domain.AgentInfo
	UpdateStatus(idPrefix, status string)
	HaltAgent(idPrefix string) error
	RemoveAgent(id string) error
	Agents() []domain.AgentInfo
	AgentsByStatus(statuses ...string) []domain.AgentInfo
}
