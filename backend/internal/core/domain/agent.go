package domain

import "time"

// AgentRole defines the role of a spawned agent.
type AgentRole string

const (
	AgentRoleDeveloper AgentRole = "developer"
	AgentRoleReviewer  AgentRole = "reviewer"
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
)

// AgentInfo tracks a spawned Claude agent.
type AgentInfo struct {
	ID             string     `json:"id"`
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
	ShutdownReason string     `json:"shutdown_reason,omitempty"`
	ResumeError    string     `json:"resume_error,omitempty"`
	Model          string     `json:"model,omitempty"`
}
