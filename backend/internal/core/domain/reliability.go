package domain

import "time"

// ReliabilityEventType categorizes reliability events.
type ReliabilityEventType string

const (
	RelEvtLockContention    ReliabilityEventType = "lock_contention"
	RelEvtLockTimeout       ReliabilityEventType = "lock_timeout"
	RelEvtAgentTimeout      ReliabilityEventType = "agent_timeout"
	RelEvtAgentSpawnFailure ReliabilityEventType = "agent_spawn_failure"
	RelEvtAgentResumeFail   ReliabilityEventType = "agent_resume_failure"
	RelEvtMergeConflict     ReliabilityEventType = "merge_conflict"
	RelEvtQuotaExceeded     ReliabilityEventType = "quota_exceeded"
)

// ValidReliabilityEventTypes lists all recognised event types.
var ValidReliabilityEventTypes = map[ReliabilityEventType]bool{
	RelEvtLockContention:    true,
	RelEvtLockTimeout:       true,
	RelEvtAgentTimeout:      true,
	RelEvtAgentSpawnFailure: true,
	RelEvtAgentResumeFail:   true,
	RelEvtMergeConflict:     true,
	RelEvtQuotaExceeded:     true,
}

// Severity levels for reliability events.
type Severity string

const (
	SeverityWarn     Severity = "warn"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// ValidSeverities lists all recognised severity levels.
var ValidSeverities = map[Severity]bool{
	SeverityWarn:     true,
	SeverityError:    true,
	SeverityCritical: true,
}

// ReliabilityEvent records a single reliability incident.
type ReliabilityEvent struct {
	ID        string               `json:"id"`
	EventType ReliabilityEventType `json:"event_type"`
	Severity  Severity             `json:"severity"`
	AgentID   string               `json:"agent_id,omitempty"`
	Scope     string               `json:"scope,omitempty"`
	Detail    map[string]any       `json:"detail,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
}

// ReliabilityFilter defines query filters for listing reliability events.
type ReliabilityFilter struct {
	EventTypes []ReliabilityEventType
	Severities []Severity
	AgentID    string
	Since      *time.Time
	Until      *time.Time
}

// ReliabilityBucket holds aggregated counts for a single time bucket.
type ReliabilityBucket struct {
	Timestamp time.Time      `json:"timestamp"`
	Counts    map[string]int `json:"counts"`
}

// ReliabilitySummary contains bucketed and total aggregations.
type ReliabilitySummary struct {
	Buckets []ReliabilityBucket `json:"buckets"`
	Totals  map[string]int      `json:"totals"`
}
