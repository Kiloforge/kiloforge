package domain

import "time"

// Reliability event type constants.
const (
	RelEventLockContention    = "lock_contention"
	RelEventLockTimeout       = "lock_timeout"
	RelEventAgentTimeout      = "agent_timeout"
	RelEventAgentSpawnFailure = "agent_spawn_failure"
	RelEventAgentResumeFail   = "agent_resume_failure"
	RelEventMergeConflict     = "merge_conflict"
	RelEventQuotaExceeded     = "quota_exceeded"
	RelEventAgentReplaced     = "agent_replaced"
)

// Reliability severity constants.
const (
	SeverityWarn     = "warn"
	SeverityError    = "error"
	SeverityCritical = "critical"
)

// ReliabilityEvent represents a recorded reliability incident.
type ReliabilityEvent struct {
	ID        string         `json:"id"`
	EventType string         `json:"event_type"`
	Severity  string         `json:"severity"`
	AgentID   string         `json:"agent_id,omitempty"`
	Scope     string         `json:"scope,omitempty"`
	Detail    map[string]any `json:"detail,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// ReliabilityFilter holds filter parameters for querying reliability events.
type ReliabilityFilter struct {
	EventTypes []string
	Severities []string
	Since      *time.Time
	Until      *time.Time
}

// ReliabilitySummary holds aggregated counts for a time window.
type ReliabilitySummary struct {
	Window         string              `json:"window"`
	BucketDuration string              `json:"bucket_duration"`
	Buckets        []ReliabilityBucket `json:"buckets"`
	Totals         ReliabilityTotals   `json:"totals"`
}

// ReliabilityBucket holds event counts for a single time bucket.
type ReliabilityBucket struct {
	Start  time.Time      `json:"start"`
	End    time.Time      `json:"end"`
	Counts map[string]int `json:"counts"`
}

// ReliabilityTotals holds total event counts across the entire window.
type ReliabilityTotals struct {
	Total      int            `json:"total"`
	ByType     map[string]int `json:"by_type"`
	BySeverity map[string]int `json:"by_severity"`
}

// ValidReliabilityEventTypes returns all valid event type values.
func ValidReliabilityEventTypes() []string {
	return []string{
		RelEventLockContention,
		RelEventLockTimeout,
		RelEventAgentTimeout,
		RelEventAgentSpawnFailure,
		RelEventAgentResumeFail,
		RelEventMergeConflict,
		RelEventQuotaExceeded,
		RelEventAgentReplaced,
	}
}

// ValidSeverities returns all valid severity values.
func ValidSeverities() []string {
	return []string{SeverityWarn, SeverityError, SeverityCritical}
}
