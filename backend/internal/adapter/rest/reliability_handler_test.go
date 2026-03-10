package rest

import (
	"context"
	"testing"
	"time"

	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/persistence/sqlite"
	"kiloforge/internal/adapter/rest/gen"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/service"
)

func newReliabilityTestHandler(t *testing.T) *APIHandler {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.Open(dir)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store := sqlite.NewReliabilityStore(db)
	svc := service.NewReliabilityService(store, nil)

	return NewAPIHandler(APIHandlerOpts{
		Agents:         &stubAgentLister{},
		Quota:          &stubQuotaReader{},
		LockMgr:        lock.New(""),
		Projects:       &stubProjectLister{},
		SSEClients:     func() int { return 0 },
		ReliabilitySvc: svc,
	})
}

func TestListReliabilityEvents_EmptyStore(t *testing.T) {
	t.Parallel()
	h := newReliabilityTestHandler(t)
	resp, err := h.ListReliabilityEvents(context.Background(), gen.ListReliabilityEventsRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.ListReliabilityEvents200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if r.TotalCount != 0 {
		t.Errorf("expected 0 total, got %d", r.TotalCount)
	}
	if len(r.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(r.Items))
	}
}

func TestReliabilityEvents_RecordAndList(t *testing.T) {
	t.Parallel()
	h := newReliabilityTestHandler(t)

	// Record events via the service.
	err := h.reliabilitySvc.RecordEvent(domain.RelEvtLockContention, domain.SeverityWarn, "", "merge", map[string]any{
		"requester":      "worker-1",
		"current_holder": "worker-2",
	})
	if err != nil {
		t.Fatalf("record event: %v", err)
	}
	err = h.reliabilitySvc.RecordEvent(domain.RelEvtAgentTimeout, domain.SeverityError, "agent-abc", "track-xyz", map[string]any{
		"reason": "exceeded max duration",
	})
	if err != nil {
		t.Fatalf("record event: %v", err)
	}

	// List all events.
	resp, err := h.ListReliabilityEvents(context.Background(), gen.ListReliabilityEventsRequestObject{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	r := resp.(gen.ListReliabilityEvents200JSONResponse)
	if r.TotalCount != 2 {
		t.Errorf("expected 2 total, got %d", r.TotalCount)
	}
	if len(r.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(r.Items))
	}
	// Verify both event types are present (order may vary within same second).
	types := map[gen.ReliabilityEventEventType]bool{}
	for _, item := range r.Items {
		types[item.EventType] = true
	}
	if !types[gen.ReliabilityEventEventType(domain.RelEvtLockContention)] {
		t.Error("expected lock_contention event")
	}
	if !types[gen.ReliabilityEventEventType(domain.RelEvtAgentTimeout)] {
		t.Error("expected agent_timeout event")
	}
	// Verify agent_timeout has agent_id.
	for _, item := range r.Items {
		if item.EventType == gen.ReliabilityEventEventType(domain.RelEvtAgentTimeout) {
			if item.AgentId == nil || *item.AgentId != "agent-abc" {
				t.Errorf("expected agent_id=agent-abc")
			}
		}
	}

	// Filter by type.
	evtType := "lock_contention"
	resp2, err := h.ListReliabilityEvents(context.Background(), gen.ListReliabilityEventsRequestObject{
		Params: gen.ListReliabilityEventsParams{EventType: &evtType},
	})
	if err != nil {
		t.Fatalf("list filtered: %v", err)
	}
	r2 := resp2.(gen.ListReliabilityEvents200JSONResponse)
	if r2.TotalCount != 1 {
		t.Errorf("expected 1 filtered result, got %d", r2.TotalCount)
	}
}

func TestReliabilitySummary(t *testing.T) {
	t.Parallel()
	h := newReliabilityTestHandler(t)

	// Record a few events.
	_ = h.reliabilitySvc.RecordEvent(domain.RelEvtLockContention, domain.SeverityWarn, "", "merge", nil)
	_ = h.reliabilitySvc.RecordEvent(domain.RelEvtAgentTimeout, domain.SeverityError, "agent-1", "", nil)
	_ = h.reliabilitySvc.RecordEvent(domain.RelEvtLockContention, domain.SeverityWarn, "", "merge", nil)

	since := time.Now().Add(-1 * time.Hour)
	until := time.Now().Add(1 * time.Hour)
	bucket := gen.Hour
	resp, err := h.GetReliabilitySummary(context.Background(), gen.GetReliabilitySummaryRequestObject{
		Params: gen.GetReliabilitySummaryParams{
			Since:  &since,
			Until:  &until,
			Bucket: &bucket,
		},
	})
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	r := resp.(gen.GetReliabilitySummary200JSONResponse)
	if r.Totals["lock_contention"] != 2 {
		t.Errorf("expected 2 lock_contention, got %d", r.Totals["lock_contention"])
	}
	if r.Totals["agent_timeout"] != 1 {
		t.Errorf("expected 1 agent_timeout, got %d", r.Totals["agent_timeout"])
	}
	if len(r.Buckets) == 0 {
		t.Error("expected at least 1 bucket")
	}
}

func TestLockContention_EmitsReliabilityEvent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	db, err := sqlite.Open(dir)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store := sqlite.NewReliabilityStore(db)
	svc := service.NewReliabilityService(store, nil)
	lockMgr := lock.New("")

	h := NewAPIHandler(APIHandlerOpts{
		Agents:         &stubAgentLister{},
		Quota:          &stubQuotaReader{},
		LockMgr:        lockMgr,
		Projects:       &stubProjectLister{},
		SSEClients:     func() int { return 0 },
		ReliabilitySvc: svc,
	})

	// Acquire lock as worker-1.
	holder1 := "worker-1"
	ttl := 60
	_, err = h.AcquireLock(context.Background(), gen.AcquireLockRequestObject{
		Scope: "merge",
		Body: &gen.LockAcquireRequest{
			Holder:     holder1,
			TtlSeconds: &ttl,
		},
	})
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}

	// Try to acquire as worker-2 — should get 409 and emit reliability event.
	resp, err := h.AcquireLock(context.Background(), gen.AcquireLockRequestObject{
		Scope: "merge",
		Body: &gen.LockAcquireRequest{
			Holder:     "worker-2",
			TtlSeconds: &ttl,
		},
	})
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	if _, ok := resp.(gen.AcquireLock409JSONResponse); !ok {
		t.Fatalf("expected 409, got %T", resp)
	}

	// Verify reliability event was recorded.
	page, err := svc.ListEvents(domain.ReliabilityFilter{
		EventTypes: []domain.ReliabilityEventType{domain.RelEvtLockContention},
	}, domain.PageOpts{Limit: 10})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 lock_contention event, got %d", len(page.Items))
	}
	evt := page.Items[0]
	if evt.Scope != "merge" {
		t.Errorf("expected scope=merge, got %s", evt.Scope)
	}
	if evt.Detail["requester"] != "worker-2" {
		t.Errorf("expected requester=worker-2, got %v", evt.Detail["requester"])
	}
	if evt.Detail["current_holder"] != "worker-1" {
		t.Errorf("expected current_holder=worker-1, got %v", evt.Detail["current_holder"])
	}
}
