package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const quotaFile = "quota-usage.json"

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
	rateLimitUntil time.Time
	dataDir        string
}

// NewQuotaTracker creates a new tracker. If dataDir is empty, persistence is disabled.
func NewQuotaTracker(dataDir string) *QuotaTracker {
	return &QuotaTracker{
		agents:  make(map[string]*AgentUsage),
		dataDir: dataDir,
	}
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
