package rest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/persistence/sqlite"
	"kiloforge/internal/adapter/rest/gen"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/service"
)

func newTestReliabilityHandler(t *testing.T) (*APIHandler, *service.ReliabilityService) {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.Open(dir)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store := sqlite.NewReliabilityStore(db)
	svc := service.NewReliabilityService(store, nil)

	h := NewAPIHandler(APIHandlerOpts{
		Agents:         &stubAgentLister{},
		Quota:          &stubQuotaReader{},
		Projects:       &stubProjectLister{},
		SSEClients:     func() int { return 0 },
		ReliabilitySvc: svc,
	})
	return h, svc
}

func TestGetReliabilityEvents(t *testing.T) {
	h, svc := newTestReliabilityHandler(t)
	ctx := context.Background()

	// Seed events.
	for i := 0; i < 5; i++ {
		err := svc.RecordEvent(
			domain.RelEventLockContention,
			domain.SeverityWarn,
			fmt.Sprintf("agent-%d", i),
			"merge",
			map[string]any{"iteration": i},
		)
		if err != nil {
			t.Fatalf("seed event %d: %v", i, err)
		}
	}
	err := svc.RecordEvent(
		domain.RelEventAgentTimeout,
		domain.SeverityError,
		"agent-timeout",
		"dev-1",
		nil,
	)
	if err != nil {
		t.Fatalf("seed timeout event: %v", err)
	}

	t.Run("list all", func(t *testing.T) {
		resp, err := h.GetReliabilityEvents(ctx, gen.GetReliabilityEventsRequestObject{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r, ok := resp.(gen.GetReliabilityEvents200JSONResponse)
		if !ok {
			t.Fatalf("expected 200, got %T", resp)
		}
		if r.TotalCount != 6 {
			t.Errorf("expected 6 events, got %d", r.TotalCount)
		}
	})

	t.Run("filter by event type", func(t *testing.T) {
		et := "agent_timeout"
		resp, err := h.GetReliabilityEvents(ctx, gen.GetReliabilityEventsRequestObject{
			Params: gen.GetReliabilityEventsParams{EventType: &et},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := resp.(gen.GetReliabilityEvents200JSONResponse)
		if r.TotalCount != 1 {
			t.Errorf("expected 1 event, got %d", r.TotalCount)
		}
		if len(r.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(r.Items))
		}
		if r.Items[0].EventType != gen.AgentTimeout {
			t.Errorf("expected agent_timeout, got %s", r.Items[0].EventType)
		}
	})

	t.Run("filter by severity", func(t *testing.T) {
		sev := "error"
		resp, err := h.GetReliabilityEvents(ctx, gen.GetReliabilityEventsRequestObject{
			Params: gen.GetReliabilityEventsParams{Severity: &sev},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := resp.(gen.GetReliabilityEvents200JSONResponse)
		if r.TotalCount != 1 {
			t.Errorf("expected 1 event, got %d", r.TotalCount)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		limit := 3
		resp, err := h.GetReliabilityEvents(ctx, gen.GetReliabilityEventsRequestObject{
			Params: gen.GetReliabilityEventsParams{Limit: &limit},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := resp.(gen.GetReliabilityEvents200JSONResponse)
		if len(r.Items) != 3 {
			t.Errorf("expected 3 items, got %d", len(r.Items))
		}
		if r.NextCursor == nil {
			t.Fatal("expected next_cursor to be set")
		}
		if r.TotalCount != 6 {
			t.Errorf("expected total_count 6, got %d", r.TotalCount)
		}

		// Fetch next page.
		resp2, err := h.GetReliabilityEvents(ctx, gen.GetReliabilityEventsRequestObject{
			Params: gen.GetReliabilityEventsParams{Limit: &limit, Cursor: r.NextCursor},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r2 := resp2.(gen.GetReliabilityEvents200JSONResponse)
		if len(r2.Items) != 3 {
			t.Errorf("expected 3 items on page 2, got %d", len(r2.Items))
		}
	})
}

func TestGetReliabilitySummary(t *testing.T) {
	h, svc := newTestReliabilityHandler(t)
	ctx := context.Background()

	// Seed events.
	for i := 0; i < 3; i++ {
		_ = svc.RecordEvent(domain.RelEventLockContention, domain.SeverityWarn, "a1", "merge", nil)
	}
	_ = svc.RecordEvent(domain.RelEventAgentTimeout, domain.SeverityError, "a2", "dev", nil)
	_ = svc.RecordEvent(domain.RelEventQuotaExceeded, domain.SeverityCritical, "a3", "", nil)

	t.Run("default window", func(t *testing.T) {
		resp, err := h.GetReliabilitySummary(ctx, gen.GetReliabilitySummaryRequestObject{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r, ok := resp.(gen.GetReliabilitySummary200JSONResponse)
		if !ok {
			t.Fatalf("expected 200, got %T", resp)
		}
		if r.Window != "24h" {
			t.Errorf("expected window 24h, got %s", r.Window)
		}
		if r.Totals.Total != 5 {
			t.Errorf("expected 5 total events, got %d", r.Totals.Total)
		}
		if r.Totals.ByType["lock_contention"] != 3 {
			t.Errorf("expected 3 lock_contention, got %d", r.Totals.ByType["lock_contention"])
		}
		if r.Totals.BySeverity["warn"] != 3 {
			t.Errorf("expected 3 warn, got %d", r.Totals.BySeverity["warn"])
		}
		if len(r.Buckets) != 12 {
			t.Errorf("expected 12 buckets, got %d", len(r.Buckets))
		}
	})

	t.Run("custom buckets", func(t *testing.T) {
		window := gen.N1h
		buckets := 4
		resp, err := h.GetReliabilitySummary(ctx, gen.GetReliabilitySummaryRequestObject{
			Params: gen.GetReliabilitySummaryParams{Window: &window, Buckets: &buckets},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := resp.(gen.GetReliabilitySummary200JSONResponse)
		if len(r.Buckets) != 4 {
			t.Errorf("expected 4 buckets, got %d", len(r.Buckets))
		}
	})
}

func TestLockContentionRecordsReliabilityEvent(t *testing.T) {
	dir := t.TempDir()
	db, err := sqlite.Open(dir)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer db.Close()

	store := sqlite.NewReliabilityStore(db)
	svc := service.NewReliabilityService(store, nil)

	lockMgr := lock.New(dir)

	h := NewAPIHandler(APIHandlerOpts{
		Agents:         &stubAgentLister{},
		Quota:          &stubQuotaReader{},
		LockMgr:        lockMgr,
		Projects:       &stubProjectLister{},
		SSEClients:     func() int { return 0 },
		ReliabilitySvc: svc,
	})

	ctx := context.Background()
	ttl := 60
	timeout := 0

	// First acquire succeeds.
	_, err = h.AcquireLock(ctx, gen.AcquireLockRequestObject{
		Scope: "merge",
		Body: &gen.AcquireLockJSONRequestBody{
			Holder:         "worker-1",
			TtlSeconds:     &ttl,
			TimeoutSeconds: &timeout,
		},
	})
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	// Second acquire should fail with 409 and record a reliability event.
	resp, err := h.AcquireLock(ctx, gen.AcquireLockRequestObject{
		Scope: "merge",
		Body: &gen.AcquireLockJSONRequestBody{
			Holder:         "worker-2",
			TtlSeconds:     &ttl,
			TimeoutSeconds: &timeout,
		},
	})
	if err != nil {
		t.Fatalf("second acquire: %v", err)
	}
	if _, ok := resp.(gen.AcquireLock409JSONResponse); !ok {
		t.Fatalf("expected 409, got %T", resp)
	}

	// Verify reliability event was recorded.
	time.Sleep(10 * time.Millisecond)
	page, err := svc.ListEvents(domain.ReliabilityFilter{
		EventTypes: []string{"lock_contention"},
	}, domain.PageOpts{Limit: 10})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if page.TotalCount != 1 {
		t.Fatalf("expected 1 lock_contention event, got %d", page.TotalCount)
	}
	ev := page.Items[0]
	if ev.AgentID != "worker-2" {
		t.Errorf("expected agent_id worker-2, got %s", ev.AgentID)
	}
	if ev.Scope != "merge" {
		t.Errorf("expected scope merge, got %s", ev.Scope)
	}
}
