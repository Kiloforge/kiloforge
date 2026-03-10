package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestQuotaTracker_RecordAndGetAgent(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")
	tracker.RecordEvent("agent-1", StreamEvent{
		Type:    "result",
		CostUSD: 0.05,
		Usage:   &UsageData{InputTokens: 1000, OutputTokens: 500},
	})

	usage := tracker.GetAgentUsage("agent-1")
	if usage == nil {
		t.Fatal("agent usage is nil")
	}
	if usage.TotalCostUSD != 0.05 {
		t.Errorf("TotalCostUSD: want 0.05, got %f", usage.TotalCostUSD)
	}
	if usage.InputTokens != 1000 {
		t.Errorf("InputTokens: want 1000, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 500 {
		t.Errorf("OutputTokens: want 500, got %d", usage.OutputTokens)
	}
}

func TestQuotaTracker_GetAgentUsage_NotFound(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")
	usage := tracker.GetAgentUsage("nonexistent")
	if usage != nil {
		t.Error("expected nil for unknown agent")
	}
}

func TestQuotaTracker_AggregatesMultipleEvents(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")

	tracker.RecordEvent("agent-1", StreamEvent{
		Type:    "result",
		CostUSD: 0.05,
		Usage:   &UsageData{InputTokens: 1000, OutputTokens: 500},
	})
	tracker.RecordEvent("agent-1", StreamEvent{
		Type:    "result",
		CostUSD: 0.03,
		Usage:   &UsageData{InputTokens: 800, OutputTokens: 300},
	})

	usage := tracker.GetAgentUsage("agent-1")
	if usage.TotalCostUSD != 0.08 {
		t.Errorf("TotalCostUSD: want 0.08, got %f", usage.TotalCostUSD)
	}
	if usage.InputTokens != 1800 {
		t.Errorf("InputTokens: want 1800, got %d", usage.InputTokens)
	}
}

func TestQuotaTracker_GetTotalUsage(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")

	tracker.RecordEvent("agent-1", StreamEvent{
		Type:    "result",
		CostUSD: 0.05,
		Usage:   &UsageData{InputTokens: 1000, OutputTokens: 500, CacheReadTokens: 200, CacheCreationTokens: 50},
	})
	tracker.RecordEvent("agent-2", StreamEvent{
		Type:    "result",
		CostUSD: 0.03,
		Usage:   &UsageData{InputTokens: 800, OutputTokens: 300, CacheReadTokens: 100, CacheCreationTokens: 25},
	})

	total := tracker.GetTotalUsage()
	if total.TotalCostUSD != 0.08 {
		t.Errorf("TotalCostUSD: want 0.08, got %f", total.TotalCostUSD)
	}
	if total.InputTokens != 1800 {
		t.Errorf("InputTokens: want 1800, got %d", total.InputTokens)
	}
	if total.OutputTokens != 800 {
		t.Errorf("OutputTokens: want 800, got %d", total.OutputTokens)
	}
	if total.CacheReadTokens != 300 {
		t.Errorf("CacheReadTokens: want 300, got %d", total.CacheReadTokens)
	}
	if total.CacheCreationTokens != 75 {
		t.Errorf("CacheCreationTokens: want 75, got %d", total.CacheCreationTokens)
	}
	if total.AgentCount != 2 {
		t.Errorf("AgentCount: want 2, got %d", total.AgentCount)
	}
}

func TestQuotaTracker_IgnoresNonResultEvents(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")

	tracker.RecordEvent("agent-1", StreamEvent{Type: "message"})
	tracker.RecordEvent("agent-1", StreamEvent{Type: "tool_use"})
	tracker.RecordEvent("agent-1", StreamEvent{Type: "init"})

	usage := tracker.GetAgentUsage("agent-1")
	if usage != nil {
		t.Error("expected nil — non-result events should not create agent entries")
	}
}

func TestQuotaTracker_RateLimited(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")

	if tracker.IsRateLimited() {
		t.Error("should not be rate limited initially")
	}

	tracker.RecordEvent("agent-1", StreamEvent{
		Type:    "result",
		Subtype: "error_during_execution",
		CostUSD: 0,
	})
	// error_during_execution alone doesn't mean rate limited
	if tracker.IsRateLimited() {
		t.Error("error_during_execution alone should not flag rate limited")
	}

	// Record a budget exceeded event
	tracker.RecordEvent("agent-2", StreamEvent{
		Type:    "result",
		Subtype: "error_max_budget_usd",
		CostUSD: 5.0,
	})
	if !tracker.IsRateLimited() {
		t.Error("should be rate limited after budget exceeded")
	}
}

func TestQuotaTracker_RetryAfter(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")

	if tracker.RetryAfter() != 0 {
		t.Error("RetryAfter should be 0 when not rate limited")
	}

	tracker.RecordEvent("agent-1", StreamEvent{
		Type:    "result",
		Subtype: "error_max_budget_usd",
	})

	ra := tracker.RetryAfter()
	if ra <= 0 {
		t.Error("RetryAfter should be positive after budget exceeded")
	}
}

func TestQuotaTracker_RateLimitExpires(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")

	// Directly set an expired rate limit for testing
	tracker.mu.Lock()
	tracker.rateLimitUntil = time.Now().Add(-1 * time.Second)
	tracker.mu.Unlock()

	if tracker.IsRateLimited() {
		t.Error("expired rate limit should not flag as rate limited")
	}
}

func TestQuotaTracker_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			agentID := "agent-" + string(rune('A'+id%10))
			tracker.RecordEvent(agentID, StreamEvent{
				Type:    "result",
				CostUSD: 0.01,
				Usage:   &UsageData{InputTokens: 100, OutputTokens: 50},
			})
			_ = tracker.GetAgentUsage(agentID)
			_ = tracker.GetTotalUsage()
			_ = tracker.IsRateLimited()
			_ = tracker.RetryAfter()
		}(i)
	}
	wg.Wait()

	total := tracker.GetTotalUsage()
	if total.InputTokens != 10000 {
		t.Errorf("InputTokens: want 10000, got %d", total.InputTokens)
	}
}

func TestQuotaTracker_SaveLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tracker := NewQuotaTracker(dir)

	tracker.RecordEvent("agent-1", StreamEvent{
		Type:    "result",
		CostUSD: 0.05,
		Usage:   &UsageData{InputTokens: 1000, OutputTokens: 500, CacheReadTokens: 200},
	})
	tracker.RecordEvent("agent-2", StreamEvent{
		Type:    "result",
		CostUSD: 0.03,
		Usage:   &UsageData{InputTokens: 800, OutputTokens: 300},
	})

	if err := tracker.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists
	path := filepath.Join(dir, "quota-usage.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	// Should be valid JSON
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}

	// Load into a new tracker
	tracker2 := NewQuotaTracker(dir)
	if err := tracker2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	total := tracker2.GetTotalUsage()
	if total.TotalCostUSD != 0.08 {
		t.Errorf("TotalCostUSD after load: want 0.08, got %f", total.TotalCostUSD)
	}
	if total.InputTokens != 1800 {
		t.Errorf("InputTokens after load: want 1800, got %d", total.InputTokens)
	}
	if total.AgentCount != 2 {
		t.Errorf("AgentCount after load: want 2, got %d", total.AgentCount)
	}
}

func TestQuotaTracker_LoadMissing(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker(t.TempDir())
	// Loading from a directory without the file should not error (fresh start).
	if err := tracker.Load(); err != nil {
		t.Errorf("Load from empty dir should not error, got: %v", err)
	}
}

func TestQuotaTracker_SaveNoDataDir(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")
	// Save with no dataDir should be a no-op, not an error.
	if err := tracker.Save(); err != nil {
		t.Errorf("Save with empty dataDir should not error, got: %v", err)
	}
}

func TestQuotaTracker_RateSnapshotsAppended(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")
	tracker.RecordEvent("agent-1", StreamEvent{
		Type:    "result",
		CostUSD: 0.05,
		Usage:   &UsageData{InputTokens: 1000, OutputTokens: 500},
	})
	tracker.RecordEvent("agent-1", StreamEvent{
		Type:    "result",
		CostUSD: 0.03,
		Usage:   &UsageData{InputTokens: 800, OutputTokens: 300},
	})

	tracker.mu.RLock()
	defer tracker.mu.RUnlock()
	if len(tracker.snapshots) != 2 {
		t.Errorf("snapshots: want 2, got %d", len(tracker.snapshots))
	}
	if tracker.snapshots[0].CostUSD != 0.05 {
		t.Errorf("snapshot[0].CostUSD: want 0.05, got %f", tracker.snapshots[0].CostUSD)
	}
	if tracker.snapshots[1].InputTokens != 800 {
		t.Errorf("snapshot[1].InputTokens: want 800, got %d", tracker.snapshots[1].InputTokens)
	}
}

func TestQuotaTracker_RingBufferCap(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")
	// Fill beyond maxSnapshots (60).
	for i := 0; i < 70; i++ {
		tracker.RecordEvent("agent-1", StreamEvent{
			Type:    "result",
			CostUSD: float64(i),
			Usage:   &UsageData{InputTokens: i * 10},
		})
	}

	tracker.mu.RLock()
	defer tracker.mu.RUnlock()
	if len(tracker.snapshots) != maxSnapshots {
		t.Errorf("snapshots: want %d, got %d", maxSnapshots, len(tracker.snapshots))
	}
	// Oldest should be entry 10 (0-9 evicted).
	if tracker.snapshots[0].CostUSD != 10 {
		t.Errorf("oldest snapshot CostUSD: want 10, got %f", tracker.snapshots[0].CostUSD)
	}
}

func TestQuotaTracker_TokensPerMin(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")
	now := time.Now()

	// Inject snapshots manually to control timing.
	tracker.mu.Lock()
	tracker.snapshots = []RateSnapshot{
		{Timestamp: now.Add(-4 * time.Minute), InputTokens: 500, OutputTokens: 200},
		{Timestamp: now.Add(-3 * time.Minute), InputTokens: 600, OutputTokens: 300},
		{Timestamp: now.Add(-1 * time.Minute), InputTokens: 400, OutputTokens: 100},
	}
	tracker.mu.Unlock()

	// 5-minute window should capture all 3 snapshots.
	rate := tracker.TokensPerMin(5 * time.Minute)
	// Total tokens in window: (500+200) + (600+300) + (400+100) = 2100
	// Elapsed from oldest to now: 4 minutes
	want := 2100.0 / 4.0
	if rate < want-1 || rate > want+1 {
		t.Errorf("TokensPerMin(5m): want ~%.1f, got %.1f", want, rate)
	}

	// 2-minute window should capture only the last snapshot.
	rate2 := tracker.TokensPerMin(2 * time.Minute)
	// Total: 500, elapsed ~1 min
	want2 := 500.0 / 1.0
	if rate2 < want2-50 || rate2 > want2+50 {
		t.Errorf("TokensPerMin(2m): want ~%.1f, got %.1f", want2, rate2)
	}
}

func TestQuotaTracker_CostPerHour(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")
	now := time.Now()

	tracker.mu.Lock()
	tracker.snapshots = []RateSnapshot{
		{Timestamp: now.Add(-20 * time.Minute), CostUSD: 0.10},
		{Timestamp: now.Add(-10 * time.Minute), CostUSD: 0.15},
		{Timestamp: now.Add(-5 * time.Minute), CostUSD: 0.05},
	}
	tracker.mu.Unlock()

	// 30-minute window captures all.
	rate := tracker.CostPerHour(30 * time.Minute)
	// Total cost: 0.30, elapsed 20 min => per hour: 0.30 / 20 * 60 = 0.90
	want := 0.30 / 20.0 * 60.0
	if rate < want-0.01 || rate > want+0.01 {
		t.Errorf("CostPerHour(30m): want ~%.3f, got %.3f", want, rate)
	}
}

func TestQuotaTracker_RatesEmptySnapshots(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")

	if rate := tracker.TokensPerMin(5 * time.Minute); rate != 0 {
		t.Errorf("TokensPerMin with no snapshots: want 0, got %f", rate)
	}
	if rate := tracker.CostPerHour(30 * time.Minute); rate != 0 {
		t.Errorf("CostPerHour with no snapshots: want 0, got %f", rate)
	}
}
