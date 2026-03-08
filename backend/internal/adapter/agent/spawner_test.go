package agent

import (
	"testing"
	"time"

	"kiloforge/internal/adapter/config"
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

func TestCheckQuota_BudgetIgnored(t *testing.T) {
	t.Parallel()

	// MaxSessionCostUSD is deprecated — budget should no longer block spawns.
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

	if err := s.checkQuota(); err != nil {
		t.Errorf("budget should be ignored (deprecated), got: %v", err)
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

func TestSetCompletionCallback(t *testing.T) {
	t.Parallel()

	s := NewSpawner(&config.Config{}, nil, nil)

	var called bool
	var gotID, gotRef, gotStatus string
	s.SetCompletionCallback(func(agentID, ref, status string) {
		called = true
		gotID = agentID
		gotRef = ref
		gotStatus = status
	})

	// Invoke callback directly to test it's wired.
	s.onCompletion("agent-123", "track-abc", "completed")

	if !called {
		t.Fatal("completion callback was not called")
	}
	if gotID != "agent-123" {
		t.Errorf("agentID = %q, want %q", gotID, "agent-123")
	}
	if gotRef != "track-abc" {
		t.Errorf("ref = %q, want %q", gotRef, "track-abc")
	}
	if gotStatus != "completed" {
		t.Errorf("status = %q, want %q", gotStatus, "completed")
	}
}

func TestOnCompletion_NilCallback(t *testing.T) {
	t.Parallel()

	s := NewSpawner(&config.Config{}, nil, nil)
	// Should not panic when no callback is set.
	s.onCompletion("agent-123", "track-abc", "completed")
}

func TestCheckQuota_HighCostAllowed(t *testing.T) {
	t.Parallel()

	// Budget enforcement is deprecated — high cost should not block spawns.
	tracker := NewQuotaTracker("")
	tracker.RecordEvent("agent-1", StreamEvent{
		Type:    "result",
		CostUSD: 100.0,
		Usage:   &UsageData{InputTokens: 1000000},
	})

	s := &Spawner{
		cfg:     &config.Config{},
		tracker: tracker,
	}

	if err := s.checkQuota(); err != nil {
		t.Errorf("should always allow spawn (budget deprecated), got: %v", err)
	}
}
