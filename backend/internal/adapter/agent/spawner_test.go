package agent

import (
	"testing"
	"time"

	"crelay/internal/adapter/config"
)

func TestCheckQuota_NilTracker(t *testing.T) {
	t.Parallel()

	s := &Spawner{cfg: &config.Config{}}
	if err := s.checkQuota(); err != nil {
		t.Errorf("nil tracker should not error, got: %v", err)
	}
}

func TestCheckQuota_NotRateLimited(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")
	s := &Spawner{cfg: &config.Config{}, tracker: tracker}
	if err := s.checkQuota(); err != nil {
		t.Errorf("should not error when not rate limited, got: %v", err)
	}
}

func TestCheckQuota_RateLimited(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")
	tracker.mu.Lock()
	tracker.rateLimitUntil = time.Now().Add(5 * time.Minute)
	tracker.mu.Unlock()

	s := &Spawner{cfg: &config.Config{}, tracker: tracker}
	err := s.checkQuota()
	if err == nil {
		t.Fatal("expected error when rate limited")
	}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestCheckQuota_BudgetExceeded(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")
	tracker.RecordEvent("agent-1", StreamEvent{
		Type:    "result",
		CostUSD: 10.0,
		Usage:   &UsageData{InputTokens: 100000},
	})

	s := &Spawner{
		cfg:     &config.Config{MaxSessionCostUSD: 5.0},
		tracker: tracker,
	}

	err := s.checkQuota()
	if err == nil {
		t.Fatal("expected error when budget exceeded")
	}
}

func TestCheckQuota_UnderBudget(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")
	tracker.RecordEvent("agent-1", StreamEvent{
		Type:    "result",
		CostUSD: 1.0,
		Usage:   &UsageData{InputTokens: 10000},
	})

	s := &Spawner{
		cfg:     &config.Config{MaxSessionCostUSD: 5.0},
		tracker: tracker,
	}

	if err := s.checkQuota(); err != nil {
		t.Errorf("should allow spawn under budget, got: %v", err)
	}
}

func TestSetTracer(t *testing.T) {
	t.Parallel()

	s := NewSpawner(&config.Config{}, nil, nil)
	// Default tracer should be NoopTracer.
	if s.tracer == nil {
		t.Fatal("expected non-nil default tracer")
	}

	// SetTracer with nil should not replace the default.
	s.SetTracer(nil)
	if s.tracer == nil {
		t.Fatal("SetTracer(nil) should not set nil")
	}
}

func TestCheckQuota_NoBudgetConfigured(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")
	tracker.RecordEvent("agent-1", StreamEvent{
		Type:    "result",
		CostUSD: 100.0,
		Usage:   &UsageData{InputTokens: 1000000},
	})

	s := &Spawner{
		cfg:     &config.Config{MaxSessionCostUSD: 0}, // no budget
		tracker: tracker,
	}

	if err := s.checkQuota(); err != nil {
		t.Errorf("no budget limit should always allow spawn, got: %v", err)
	}
}
