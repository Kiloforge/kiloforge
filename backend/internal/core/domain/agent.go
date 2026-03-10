package domain

import (
	"strings"
	"time"
)

// AgentRole defines the role of a spawned agent.
type AgentRole string

const (
	AgentRoleDeveloper          AgentRole = "developer"
	AgentRoleReviewer           AgentRole = "reviewer"
	AgentRoleArchitect          AgentRole = "architect"
	AgentRoleAdvisorProduct     AgentRole = "advisor-product"
	AgentRoleAdvisorReliability AgentRole = "advisor-reliability"
)

// AgentStatus defines the lifecycle state of an agent.
type AgentStatus string

const (
	AgentStatusRunning      AgentStatus = "running"
	AgentStatusWaiting      AgentStatus = "waiting"
	AgentStatusHalted       AgentStatus = "halted"
	AgentStatusStopped      AgentStatus = "stopped"
	AgentStatusCompleted    AgentStatus = "completed"
	AgentStatusFailed       AgentStatus = "failed"
	AgentStatusSuspended    AgentStatus = "suspended"
	AgentStatusSuspending   AgentStatus = "suspending"
	AgentStatusForceKilled  AgentStatus = "force-killed"
	AgentStatusResumeFailed AgentStatus = "resume-failed"
	AgentStatusReplaced     AgentStatus = "replaced"
)

// IsAdvisorRole returns true if the role string is an advisor role.
func IsAdvisorRole(role string) bool {
	return strings.HasPrefix(role, "advisor-")
}

// AgentInfo tracks a spawned Claude agent.
type AgentInfo struct {
	ID             string     `json:"id"`
	Name           string     `json:"name,omitempty"`
	Role           string     `json:"role"`
	Ref            string     `json:"ref"`
	Status         string     `json:"status"`
	SessionID      string     `json:"session_id"`
	PID            int        `json:"pid"`
	WorktreeDir    string     `json:"worktree_dir"`
	LogFile        string     `json:"log_file"`
	StartedAt      time.Time  `json:"started_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	SuspendedAt    *time.Time `json:"suspended_at,omitempty"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
	ShutdownReason string     `json:"shutdown_reason,omitempty"`
	ResumeError    string     `json:"resume_error,omitempty"`
	Model          string     `json:"model,omitempty"`
}

// IsActive returns true if the agent is in a non-terminal status.
func (a AgentInfo) IsActive() bool {
	return a.Status == string(AgentStatusRunning) || a.Status == string(AgentStatusWaiting)
}

// IsTerminal returns true if the agent is in a terminal status.
func (a AgentInfo) IsTerminal() bool {
	switch AgentStatus(a.Status) {
	case AgentStatusStopped, AgentStatusCompleted, AgentStatusFailed,
		AgentStatusForceKilled, AgentStatusResumeFailed, AgentStatusReplaced:
		return true
	}
	return false
}
