package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/service"
)

const (
	quotaFile    = "quota-usage.json"
	maxSnapshots = 60 // ~1 per minute, keep last hour of data
)

// RateSnapshot records a point-in-time usage sample for rate computation.
type RateSnapshot struct {
	Timestamp    time.Time `json:"timestamp"`
	CostUSD      float64   `json:"cost_usd"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
}

// AgentUsage holds cumulative usage for a single agent.
type AgentUsage struct {
	AgentID             string  `json:"agent_id"`
	TotalCostUSD        float64 `json:"total_cost_usd"`
	InputTokens         int     `json:"input_tokens"`
	OutputTokens        int     `json:"output_tokens"`
	CacheReadTokens     int     `json:"cache_read_tokens"`
	CacheCreationTokens int     `json:"cache_creation_tokens"`
	ResultCount         int     `json:"result_count"`
}

// TotalUsage holds aggregate usage across all agents.
type TotalUsage struct {
	TotalCostUSD        float64 `json:"total_cost_usd"`
	InputTokens         int     `json:"input_tokens"`
	OutputTokens        int     `json:"output_tokens"`
	CacheReadTokens     int     `json:"cache_read_tokens"`
	CacheCreationTokens int     `json:"cache_creation_tokens"`
	AgentCount          int     `json:"agent_count"`
}

// quotaSnapshot is the JSON-serializable state for persistence.
type quotaSnapshot struct {
	Agents    map[string]*AgentUsage `json:"agents"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// QuotaTracker aggregates token usage and cost across multiple CC agents.
// It is thread-safe for concurrent use.
type QuotaTracker struct {
	mu             sync.RWMutex
	agents         map[string]*AgentUsage
	snapshots      []RateSnapshot
	rateLimitUntil time.Time
	dataDir        string
	reliabilitySvc *service.ReliabilityService
}

// NewQuotaTracker creates a new tracker. If dataDir is empty, persistence is disabled.
func NewQuotaTracker(dataDir string) *QuotaTracker {
	return &QuotaTracker{
		agents:  make(map[string]*AgentUsage),
		dataDir: dataDir,
	}
}

// SetReliabilityService sets the reliability service for recording quota events.
func (t *QuotaTracker) SetReliabilityService(svc *service.ReliabilityService) {
	t.reliabilitySvc = svc
}

// RecordEvent processes a stream-json event and updates usage counters.
// Only result events with usage data are recorded.
func (t *QuotaTracker) RecordEvent(agentID string, event StreamEvent) {
	if event.Type != "result" {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Check for rate-limiting signals.
	if event.Subtype == "error_max_budget_usd" {
		t.rateLimitUntil = time.Now().Add(5 * time.Minute)
		if t.reliabilitySvc != nil {
			go func() {
				_ = t.reliabilitySvc.RecordEvent(domain.RelEvtQuotaExceeded, domain.SeverityWarn, agentID, "", map[string]any{
					"subtype":     event.Subtype,
					"retry_after": "5m",
				})
			}()
		}
	}

	if event.Usage == nil && event.CostUSD == 0 {
		return
	}

	usage, ok := t.agents[agentID]
	if !ok {
		usage = &AgentUsage{AgentID: agentID}
		t.agents[agentID] = usage
	}

	usage.TotalCostUSD += event.CostUSD
	usage.ResultCount++

	if event.Usage != nil {
		usage.InputTokens += event.Usage.InputTokens
		usage.OutputTokens += event.Usage.OutputTokens
		usage.CacheReadTokens += event.Usage.CacheReadTokens
		usage.CacheCreationTokens += event.Usage.CacheCreationTokens
	}

	// Append rate snapshot for time-windowed metrics.
	snap := RateSnapshot{
		Timestamp: time.Now(),
		CostUSD:   event.CostUSD,
	}
	if event.Usage != nil {
		snap.InputTokens = event.Usage.InputTokens
		snap.OutputTokens = event.Usage.OutputTokens
	}
	t.snapshots = append(t.snapshots, snap)
	if len(t.snapshots) > maxSnapshots {
		t.snapshots = t.snapshots[len(t.snapshots)-maxSnapshots:]
	}
}

// GetAgentUsage returns usage for a specific agent, or nil if not found.
func (t *QuotaTracker) GetAgentUsage(agentID string) *AgentUsage {
	t.mu.RLock()
	defer t.mu.RUnlock()

	usage, ok := t.agents[agentID]
	if !ok {
		return nil
	}
	// Return a copy.
	cp := *usage
	return &cp
}

// GetTotalUsage returns aggregate usage across all agents.
func (t *QuotaTracker) GetTotalUsage() TotalUsage {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var total TotalUsage
	for range t.agents {
		total.AgentCount++
	}
	for _, u := range t.agents {
		total.TotalCostUSD += u.TotalCostUSD
		total.InputTokens += u.InputTokens
		total.OutputTokens += u.OutputTokens
		total.CacheReadTokens += u.CacheReadTokens
		total.CacheCreationTokens += u.CacheCreationTokens
	}
	return total
}

// IsRateLimited returns true if any agent has signaled rate limiting recently.
func (t *QuotaTracker) IsRateLimited() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return time.Now().Before(t.rateLimitUntil)
}

// TokensPerMin returns the rate of total tokens (input+output) per minute
// over the given time window, computed from recent snapshots.
func (t *QuotaTracker) TokensPerMin(window time.Duration) float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	cutoff := time.Now().Add(-window)
	var totalTokens int
	var oldest time.Time
	var count int
	for _, s := range t.snapshots {
		if s.Timestamp.Before(cutoff) {
			continue
		}
		totalTokens += s.InputTokens + s.OutputTokens
		if count == 0 || s.Timestamp.Before(oldest) {
			oldest = s.Timestamp
		}
		count++
	}
	if count == 0 {
		return 0
	}
	elapsed := time.Since(oldest).Minutes()
	if elapsed < 0.01 {
		return 0
	}
	return float64(totalTokens) / elapsed
}

// CostPerHour returns the cost rate in USD/hour over the given time window.
func (t *QuotaTracker) CostPerHour(window time.Duration) float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	cutoff := time.Now().Add(-window)
	var totalCost float64
	var oldest time.Time
	var count int
	for _, s := range t.snapshots {
		if s.Timestamp.Before(cutoff) {
			continue
		}
		totalCost += s.CostUSD
		if count == 0 || s.Timestamp.Before(oldest) {
			oldest = s.Timestamp
		}
		count++
	}
	if count == 0 {
		return 0
	}
	elapsed := time.Since(oldest).Minutes()
	if elapsed < 0.01 {
		return 0
	}
	return totalCost / elapsed * 60.0
}

// RetryAfter returns the duration until the rate limit expires.
// Returns 0 if not rate limited.
func (t *QuotaTracker) RetryAfter() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	remaining := time.Until(t.rateLimitUntil)
	if remaining <= 0 {
		return 0
	}
	return remaining
}

// Save writes the current usage state to disk. No-op if dataDir is empty.
func (t *QuotaTracker) Save() error {
	if t.dataDir == "" {
		return nil
	}

	t.mu.RLock()
	snap := quotaSnapshot{
		Agents:    t.agents,
		UpdatedAt: time.Now(),
	}
	t.mu.RUnlock()

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal quota usage: %w", err)
	}

	return os.WriteFile(filepath.Join(t.dataDir, quotaFile), append(data, '\n'), 0o644)
}

// Load restores usage state from disk. No-op if file doesn't exist.
func (t *QuotaTracker) Load() error {
	if t.dataDir == "" {
		return nil
	}

	data, err := os.ReadFile(filepath.Join(t.dataDir, quotaFile))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read quota usage: %w", err)
	}

	var snap quotaSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("parse quota usage: %w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if snap.Agents != nil {
		t.agents = snap.Agents
	}
	return nil
}
