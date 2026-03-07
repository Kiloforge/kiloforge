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
	AgentStatusRunning   AgentStatus = "running"
	AgentStatusWaiting   AgentStatus = "waiting"
	AgentStatusHalted    AgentStatus = "halted"
	AgentStatusStopped   AgentStatus = "stopped"
	AgentStatusCompleted AgentStatus = "completed"
	AgentStatusFailed    AgentStatus = "failed"
)

// AgentInfo tracks a spawned Claude agent.
type AgentInfo struct {
	ID          string    `json:"id"`
	Role        string    `json:"role"`
	Ref         string    `json:"ref"`
	Status      string    `json:"status"`
	SessionID   string    `json:"session_id"`
	PID         int       `json:"pid"`
	WorktreeDir string    `json:"worktree_dir"`
	LogFile     string    `json:"log_file"`
	StartedAt   time.Time `json:"started_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
