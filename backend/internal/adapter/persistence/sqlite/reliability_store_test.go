package sqlite

import (
	"fmt"
	"testing"
	"time"

	"kiloforge/internal/core/domain"
)

func TestReliabilityStore_InsertAndList(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewReliabilityStore(db)

	now := time.Now().UTC().Truncate(time.Millisecond)
	events := []domain.ReliabilityEvent{
		{ID: "ev-1", EventType: domain.RelEventLockContention, Severity: domain.SeverityWarn, Scope: "merge", CreatedAt: now.Add(-2 * time.Minute)},
		{ID: "ev-2", EventType: domain.RelEventAgentTimeout, Severity: domain.SeverityError, AgentID: "agent-1", CreatedAt: now.Add(-1 * time.Minute)},
		{ID: "ev-3", EventType: domain.RelEventLockContention, Severity: domain.SeverityWarn, Scope: "merge", Detail: map[string]any{"holder": "dev-1"}, CreatedAt: now},
	}
	for _, ev := range events {
		if err := store.Insert(ev); err != nil {
			t.Fatalf("Insert %s: %v", ev.ID, err)
		}
	}

	// List all — should be newest first.
	page, err := store.List(domain.ReliabilityFilter{}, domain.PageOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.TotalCount != 3 {
		t.Errorf("TotalCount: want 3, got %d", page.TotalCount)
	}
	if len(page.Items) != 3 {
		t.Fatalf("Items: want 3, got %d", len(page.Items))
	}
	if page.Items[0].ID != "ev-3" {
		t.Errorf("first item: want ev-3, got %s", page.Items[0].ID)
	}
	// Check detail was preserved.
	if page.Items[0].Detail["holder"] != "dev-1" {
		t.Errorf("detail.holder: want dev-1, got %v", page.Items[0].Detail["holder"])
	}
}

func TestReliabilityStore_FilterByType(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewReliabilityStore(db)

	now := time.Now().UTC()
	store.Insert(domain.ReliabilityEvent{ID: "ev-1", EventType: domain.RelEventLockContention, Severity: domain.SeverityWarn, CreatedAt: now.Add(-2 * time.Minute)})
	store.Insert(domain.ReliabilityEvent{ID: "ev-2", EventType: domain.RelEventAgentTimeout, Severity: domain.SeverityError, CreatedAt: now.Add(-1 * time.Minute)})
	store.Insert(domain.ReliabilityEvent{ID: "ev-3", EventType: domain.RelEventLockContention, Severity: domain.SeverityWarn, CreatedAt: now})

	page, err := store.List(domain.ReliabilityFilter{EventTypes: []string{domain.RelEventLockContention}}, domain.PageOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.TotalCount != 2 {
		t.Errorf("TotalCount: want 2, got %d", page.TotalCount)
	}
	for _, item := range page.Items {
		if item.EventType != domain.RelEventLockContention {
			t.Errorf("unexpected type: %s", item.EventType)
		}
	}
}

func TestReliabilityStore_FilterBySeverity(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewReliabilityStore(db)

	now := time.Now().UTC()
	store.Insert(domain.ReliabilityEvent{ID: "ev-1", EventType: domain.RelEventLockContention, Severity: domain.SeverityWarn, CreatedAt: now})
	store.Insert(domain.ReliabilityEvent{ID: "ev-2", EventType: domain.RelEventAgentTimeout, Severity: domain.SeverityError, CreatedAt: now})

	page, err := store.List(domain.ReliabilityFilter{Severities: []string{domain.SeverityError}}, domain.PageOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.TotalCount != 1 {
		t.Errorf("TotalCount: want 1, got %d", page.TotalCount)
	}
}

func TestReliabilityStore_FilterByTimeRange(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewReliabilityStore(db)

	now := time.Now().UTC()
	store.Insert(domain.ReliabilityEvent{ID: "ev-old", EventType: domain.RelEventLockContention, Severity: domain.SeverityWarn, CreatedAt: now.Add(-48 * time.Hour)})
	store.Insert(domain.ReliabilityEvent{ID: "ev-new", EventType: domain.RelEventAgentTimeout, Severity: domain.SeverityError, CreatedAt: now})

	since := now.Add(-1 * time.Hour)
	page, err := store.List(domain.ReliabilityFilter{Since: &since}, domain.PageOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.TotalCount != 1 {
		t.Errorf("TotalCount: want 1, got %d", page.TotalCount)
	}
	if page.Items[0].ID != "ev-new" {
		t.Errorf("ID: want ev-new, got %s", page.Items[0].ID)
	}
}

func TestReliabilityStore_Pagination(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewReliabilityStore(db)

	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		store.Insert(domain.ReliabilityEvent{
			ID:        fmt.Sprintf("ev-%d", i),
			EventType: domain.RelEventLockContention,
			Severity:  domain.SeverityWarn,
			CreatedAt: now.Add(time.Duration(i) * time.Minute),
		})
	}

	// First page of 2.
	page1, err := store.List(domain.ReliabilityFilter{}, domain.PageOpts{Limit: 2})
	if err != nil {
		t.Fatalf("List page1: %v", err)
	}
	if len(page1.Items) != 2 {
		t.Fatalf("page1 items: want 2, got %d", len(page1.Items))
	}
	if page1.NextCursor == "" {
		t.Fatal("page1 should have next cursor")
	}
	if page1.TotalCount != 5 {
		t.Errorf("TotalCount: want 5, got %d", page1.TotalCount)
	}

	// Second page.
	page2, err := store.List(domain.ReliabilityFilter{}, domain.PageOpts{Limit: 2, Cursor: page1.NextCursor})
	if err != nil {
		t.Fatalf("List page2: %v", err)
	}
	if len(page2.Items) != 2 {
		t.Fatalf("page2 items: want 2, got %d", len(page2.Items))
	}

	// Third page (last item).
	page3, err := store.List(domain.ReliabilityFilter{}, domain.PageOpts{Limit: 2, Cursor: page2.NextCursor})
	if err != nil {
		t.Fatalf("List page3: %v", err)
	}
	if len(page3.Items) != 1 {
		t.Fatalf("page3 items: want 1, got %d", len(page3.Items))
	}
	if page3.NextCursor != "" {
		t.Error("page3 should not have next cursor")
	}
}

func TestReliabilityStore_Summary(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewReliabilityStore(db)

	now := time.Now().UTC()
	store.Insert(domain.ReliabilityEvent{ID: "ev-1", EventType: domain.RelEventLockContention, Severity: domain.SeverityWarn, CreatedAt: now.Add(-3 * time.Hour)})
	store.Insert(domain.ReliabilityEvent{ID: "ev-2", EventType: domain.RelEventAgentTimeout, Severity: domain.SeverityError, CreatedAt: now.Add(-2 * time.Hour)})
	store.Insert(domain.ReliabilityEvent{ID: "ev-3", EventType: domain.RelEventLockContention, Severity: domain.SeverityWarn, CreatedAt: now.Add(-1 * time.Hour)})

	since := now.Add(-4 * time.Hour)
	summary, err := store.Summary(domain.ReliabilityFilter{Since: &since, Until: &now}, 4)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}

	if summary.Totals.Total != 3 {
		t.Errorf("total: want 3, got %d", summary.Totals.Total)
	}
	if summary.Totals.ByType[domain.RelEventLockContention] != 2 {
		t.Errorf("by_type lock_contention: want 2, got %d", summary.Totals.ByType[domain.RelEventLockContention])
	}
	if summary.Totals.BySeverity[domain.SeverityWarn] != 2 {
		t.Errorf("by_severity warn: want 2, got %d", summary.Totals.BySeverity[domain.SeverityWarn])
	}
	if len(summary.Buckets) != 4 {
		t.Errorf("buckets: want 4, got %d", len(summary.Buckets))
	}
}
