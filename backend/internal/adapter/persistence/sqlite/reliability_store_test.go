package sqlite

import (
	"testing"
	"time"

	"kiloforge/internal/core/domain"
)

func newTestReliabilityStore(t *testing.T) *ReliabilityStore {
	t.Helper()
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewReliabilityStore(db)
}

func makeEvent(id string, evtType domain.ReliabilityEventType, sev domain.Severity, agentID string, at time.Time) domain.ReliabilityEvent {
	return domain.ReliabilityEvent{
		ID:        id,
		EventType: evtType,
		Severity:  sev,
		AgentID:   agentID,
		Scope:     "test-scope",
		Detail:    map[string]any{"key": "value"},
		CreatedAt: at,
	}
}

func TestReliabilityStore_InsertAndList(t *testing.T) {
	t.Parallel()
	s := newTestReliabilityStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	e1 := makeEvent("evt-1", domain.RelEvtLockContention, domain.SeverityWarn, "agent-1", now)
	e2 := makeEvent("evt-2", domain.RelEvtAgentTimeout, domain.SeverityError, "agent-2", now.Add(time.Second))

	if err := s.Insert(e1); err != nil {
		t.Fatalf("Insert e1: %v", err)
	}
	if err := s.Insert(e2); err != nil {
		t.Fatalf("Insert e2: %v", err)
	}

	// List all — should return newest first.
	page, err := s.List(domain.ReliabilityFilter{}, domain.PageOpts{Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.TotalCount != 2 {
		t.Errorf("TotalCount: want 2, got %d", page.TotalCount)
	}
	if len(page.Items) != 2 {
		t.Fatalf("Items: want 2, got %d", len(page.Items))
	}
	if page.Items[0].ID != "evt-2" {
		t.Errorf("first item: want evt-2, got %s", page.Items[0].ID)
	}
	if page.Items[0].Detail["key"] != "value" {
		t.Errorf("detail not preserved")
	}
}

func TestReliabilityStore_FilterByType(t *testing.T) {
	t.Parallel()
	s := newTestReliabilityStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	s.Insert(makeEvent("evt-1", domain.RelEvtLockContention, domain.SeverityWarn, "", now))
	s.Insert(makeEvent("evt-2", domain.RelEvtAgentTimeout, domain.SeverityError, "", now.Add(time.Second)))
	s.Insert(makeEvent("evt-3", domain.RelEvtQuotaExceeded, domain.SeverityWarn, "", now.Add(2*time.Second)))

	page, err := s.List(domain.ReliabilityFilter{
		EventTypes: []domain.ReliabilityEventType{domain.RelEvtLockContention},
	}, domain.PageOpts{Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.TotalCount != 1 {
		t.Errorf("TotalCount: want 1, got %d", page.TotalCount)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "evt-1" {
		t.Errorf("expected evt-1 only")
	}
}

func TestReliabilityStore_FilterByTimeRange(t *testing.T) {
	t.Parallel()
	s := newTestReliabilityStore(t)
	base := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

	s.Insert(makeEvent("old", domain.RelEvtAgentTimeout, domain.SeverityError, "", base))
	s.Insert(makeEvent("mid", domain.RelEvtAgentTimeout, domain.SeverityError, "", base.Add(time.Hour)))
	s.Insert(makeEvent("new", domain.RelEvtAgentTimeout, domain.SeverityError, "", base.Add(2*time.Hour)))

	since := base.Add(30 * time.Minute)
	until := base.Add(90 * time.Minute)
	page, err := s.List(domain.ReliabilityFilter{
		Since: &since,
		Until: &until,
	}, domain.PageOpts{Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.TotalCount != 1 {
		t.Errorf("TotalCount: want 1, got %d", page.TotalCount)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "mid" {
		t.Errorf("expected mid only, got %v", page.Items)
	}
}

func TestReliabilityStore_Pagination(t *testing.T) {
	t.Parallel()
	s := newTestReliabilityStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	for i := 0; i < 5; i++ {
		s.Insert(makeEvent(
			"evt-"+string(rune('a'+i)),
			domain.RelEvtLockContention,
			domain.SeverityWarn,
			"",
			now.Add(time.Duration(i)*time.Second),
		))
	}

	// First page of 2.
	page1, err := s.List(domain.ReliabilityFilter{}, domain.PageOpts{Limit: 2})
	if err != nil {
		t.Fatalf("List page 1: %v", err)
	}
	if len(page1.Items) != 2 {
		t.Fatalf("page 1 items: want 2, got %d", len(page1.Items))
	}
	if page1.NextCursor == "" {
		t.Fatal("expected next cursor")
	}
	if page1.TotalCount != 5 {
		t.Errorf("TotalCount: want 5, got %d", page1.TotalCount)
	}

	// Second page.
	page2, err := s.List(domain.ReliabilityFilter{}, domain.PageOpts{Limit: 2, Cursor: page1.NextCursor})
	if err != nil {
		t.Fatalf("List page 2: %v", err)
	}
	if len(page2.Items) != 2 {
		t.Fatalf("page 2 items: want 2, got %d", len(page2.Items))
	}

	// Third page (last item).
	page3, err := s.List(domain.ReliabilityFilter{}, domain.PageOpts{Limit: 2, Cursor: page2.NextCursor})
	if err != nil {
		t.Fatalf("List page 3: %v", err)
	}
	if len(page3.Items) != 1 {
		t.Fatalf("page 3 items: want 1, got %d", len(page3.Items))
	}
	if page3.NextCursor != "" {
		t.Error("expected empty next cursor on last page")
	}
}

func TestReliabilityStore_Summary(t *testing.T) {
	t.Parallel()
	s := newTestReliabilityStore(t)
	base := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)

	// Two events in hour 10, one in hour 11.
	s.Insert(makeEvent("evt-1", domain.RelEvtLockContention, domain.SeverityWarn, "", base.Add(5*time.Minute)))
	s.Insert(makeEvent("evt-2", domain.RelEvtAgentTimeout, domain.SeverityError, "", base.Add(30*time.Minute)))
	s.Insert(makeEvent("evt-3", domain.RelEvtLockContention, domain.SeverityWarn, "", base.Add(70*time.Minute)))

	summary, err := s.Summary(base, base.Add(2*time.Hour), "hour")
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}

	if len(summary.Buckets) != 2 {
		t.Fatalf("buckets: want 2, got %d", len(summary.Buckets))
	}

	// First bucket (hour 10): 1 lock_contention, 1 agent_timeout.
	b0 := summary.Buckets[0]
	if b0.Counts["lock_contention"] != 1 {
		t.Errorf("bucket 0 lock_contention: want 1, got %d", b0.Counts["lock_contention"])
	}
	if b0.Counts["agent_timeout"] != 1 {
		t.Errorf("bucket 0 agent_timeout: want 1, got %d", b0.Counts["agent_timeout"])
	}

	// Second bucket (hour 11): 1 lock_contention.
	b1 := summary.Buckets[1]
	if b1.Counts["lock_contention"] != 1 {
		t.Errorf("bucket 1 lock_contention: want 1, got %d", b1.Counts["lock_contention"])
	}

	// Totals.
	if summary.Totals["lock_contention"] != 2 {
		t.Errorf("total lock_contention: want 2, got %d", summary.Totals["lock_contention"])
	}
	if summary.Totals["agent_timeout"] != 1 {
		t.Errorf("total agent_timeout: want 1, got %d", summary.Totals["agent_timeout"])
	}
}

func TestReliabilityStore_FilterBySeverity(t *testing.T) {
	t.Parallel()
	s := newTestReliabilityStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	s.Insert(makeEvent("evt-1", domain.RelEvtLockContention, domain.SeverityWarn, "", now))
	s.Insert(makeEvent("evt-2", domain.RelEvtAgentTimeout, domain.SeverityError, "", now.Add(time.Second)))
	s.Insert(makeEvent("evt-3", domain.RelEvtQuotaExceeded, domain.SeverityCritical, "", now.Add(2*time.Second)))

	page, err := s.List(domain.ReliabilityFilter{
		Severities: []domain.Severity{domain.SeverityError, domain.SeverityCritical},
	}, domain.PageOpts{Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.TotalCount != 2 {
		t.Errorf("TotalCount: want 2, got %d", page.TotalCount)
	}
}

func TestReliabilityStore_FilterByAgentID(t *testing.T) {
	t.Parallel()
	s := newTestReliabilityStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	s.Insert(makeEvent("evt-1", domain.RelEvtLockContention, domain.SeverityWarn, "agent-A", now))
	s.Insert(makeEvent("evt-2", domain.RelEvtAgentTimeout, domain.SeverityError, "agent-B", now.Add(time.Second)))

	page, err := s.List(domain.ReliabilityFilter{AgentID: "agent-A"}, domain.PageOpts{Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.TotalCount != 1 {
		t.Errorf("TotalCount: want 1, got %d", page.TotalCount)
	}
	if page.Items[0].AgentID != "agent-A" {
		t.Errorf("expected agent-A, got %s", page.Items[0].AgentID)
	}
}
