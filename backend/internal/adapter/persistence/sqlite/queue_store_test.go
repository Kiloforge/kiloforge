package sqlite

import (
	"testing"
	"time"

	"kiloforge/internal/core/domain"
)

func TestQueueStore_EnqueueAndGet(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQueueStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	item := domain.QueueItem{
		TrackID:     "track-abc",
		ProjectSlug: "myapp",
		EnqueuedAt:  now,
	}
	if err := store.Enqueue(item); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	got, err := store.Get("track-abc")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil item")
	}
	if got.TrackID != "track-abc" {
		t.Errorf("TrackID = %q, want %q", got.TrackID, "track-abc")
	}
	if got.ProjectSlug != "myapp" {
		t.Errorf("ProjectSlug = %q, want %q", got.ProjectSlug, "myapp")
	}
	if got.Status != domain.QueueStatusQueued {
		t.Errorf("Status = %q, want %q", got.Status, domain.QueueStatusQueued)
	}
}

func TestQueueStore_EnqueueDuplicate(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQueueStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	item := domain.QueueItem{TrackID: "dup-1", ProjectSlug: "proj", EnqueuedAt: now}

	if err := store.Enqueue(item); err != nil {
		t.Fatalf("Enqueue(1): %v", err)
	}
	// Second enqueue should be ignored (INSERT OR IGNORE).
	if err := store.Enqueue(item); err != nil {
		t.Fatalf("Enqueue(2): %v", err)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item after duplicate enqueue, got %d", len(items))
	}
}

func TestQueueStore_Get_NotFound(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQueueStore(db)

	got, err := store.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for nonexistent item, got %v", got)
	}
}

func TestQueueStore_Dequeue(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQueueStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	store.Enqueue(domain.QueueItem{TrackID: "del-1", ProjectSlug: "proj", EnqueuedAt: now})

	if err := store.Dequeue("del-1"); err != nil {
		t.Fatalf("Dequeue: %v", err)
	}

	got, _ := store.Get("del-1")
	if got != nil {
		t.Error("expected nil after dequeue")
	}
}

func TestQueueStore_Dequeue_NotFound(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQueueStore(db)

	err := store.Dequeue("ghost")
	if err == nil {
		t.Error("expected error for dequeuing nonexistent item")
	}
}

func TestQueueStore_AssignCompleteFlow(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQueueStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	store.Enqueue(domain.QueueItem{TrackID: "flow-1", ProjectSlug: "proj", EnqueuedAt: now})

	// Assign.
	if err := store.Assign("flow-1", "agent-42"); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	got, _ := store.Get("flow-1")
	if got.Status != domain.QueueStatusAssigned {
		t.Errorf("Status after Assign = %q, want %q", got.Status, domain.QueueStatusAssigned)
	}
	if got.AgentID != "agent-42" {
		t.Errorf("AgentID = %q, want %q", got.AgentID, "agent-42")
	}
	if got.AssignedAt == nil {
		t.Error("AssignedAt should be set after Assign")
	}

	// Complete.
	if err := store.Complete("flow-1"); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	got, _ = store.Get("flow-1")
	if got.Status != domain.QueueStatusCompleted {
		t.Errorf("Status after Complete = %q, want %q", got.Status, domain.QueueStatusCompleted)
	}
	if got.CompletedAt == nil {
		t.Error("CompletedAt should be set after Complete")
	}
}

func TestQueueStore_Assign_WrongState(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQueueStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	store.Enqueue(domain.QueueItem{TrackID: "ws-1", ProjectSlug: "proj", EnqueuedAt: now})
	store.Assign("ws-1", "agent-1")

	// Assigning an already-assigned item should fail.
	if err := store.Assign("ws-1", "agent-2"); err == nil {
		t.Error("expected error assigning already-assigned item")
	}
}

func TestQueueStore_Complete_WrongState(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQueueStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	store.Enqueue(domain.QueueItem{TrackID: "cw-1", ProjectSlug: "proj", EnqueuedAt: now})

	// Completing a queued (not assigned) item should fail.
	if err := store.Complete("cw-1"); err == nil {
		t.Error("expected error completing queued item")
	}
}

func TestQueueStore_Fail(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQueueStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	store.Enqueue(domain.QueueItem{TrackID: "fail-1", ProjectSlug: "proj", EnqueuedAt: now})

	if err := store.Fail("fail-1"); err != nil {
		t.Fatalf("Fail: %v", err)
	}
	got, _ := store.Get("fail-1")
	if got.Status != domain.QueueStatusFailed {
		t.Errorf("Status = %q, want %q", got.Status, domain.QueueStatusFailed)
	}
}

func TestQueueStore_Fail_AssignedState(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQueueStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	store.Enqueue(domain.QueueItem{TrackID: "fa-1", ProjectSlug: "proj", EnqueuedAt: now})
	store.Assign("fa-1", "agent-1")

	if err := store.Fail("fa-1"); err != nil {
		t.Fatalf("Fail (assigned): %v", err)
	}
	got, _ := store.Get("fa-1")
	if got.Status != domain.QueueStatusFailed {
		t.Errorf("Status = %q, want %q", got.Status, domain.QueueStatusFailed)
	}
}

func TestQueueStore_List_AllItems(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQueueStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	store.Enqueue(domain.QueueItem{TrackID: "la-1", ProjectSlug: "proj", EnqueuedAt: now})
	store.Enqueue(domain.QueueItem{TrackID: "la-2", ProjectSlug: "proj", EnqueuedAt: now.Add(time.Second)})

	items, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestQueueStore_List_ByStatus(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQueueStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	store.Enqueue(domain.QueueItem{TrackID: "ls-1", ProjectSlug: "proj", EnqueuedAt: now})
	store.Enqueue(domain.QueueItem{TrackID: "ls-2", ProjectSlug: "proj", EnqueuedAt: now.Add(time.Second)})
	store.Assign("ls-2", "agent-1")

	queued, err := store.List(domain.QueueStatusQueued)
	if err != nil {
		t.Fatalf("List(queued): %v", err)
	}
	if len(queued) != 1 {
		t.Errorf("queued: expected 1, got %d", len(queued))
	}

	assigned, err := store.List(domain.QueueStatusAssigned)
	if err != nil {
		t.Fatalf("List(assigned): %v", err)
	}
	if len(assigned) != 1 {
		t.Errorf("assigned: expected 1, got %d", len(assigned))
	}
}

func TestQueueStore_List_MultipleStatuses(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQueueStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	store.Enqueue(domain.QueueItem{TrackID: "ms-1", ProjectSlug: "proj", EnqueuedAt: now})
	store.Enqueue(domain.QueueItem{TrackID: "ms-2", ProjectSlug: "proj", EnqueuedAt: now.Add(time.Second)})
	store.Assign("ms-2", "agent-1")
	store.Complete("ms-2")

	items, err := store.List(domain.QueueStatusQueued, domain.QueueStatusCompleted)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items (queued+completed), got %d", len(items))
	}
}

func TestQueueStore_Clear(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQueueStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	store.Enqueue(domain.QueueItem{TrackID: "cl-1", ProjectSlug: "proj", EnqueuedAt: now})
	store.Enqueue(domain.QueueItem{TrackID: "cl-2", ProjectSlug: "proj", EnqueuedAt: now.Add(time.Second)})

	if err := store.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	items, _ := store.List()
	if len(items) != 0 {
		t.Errorf("expected 0 items after clear, got %d", len(items))
	}
}

func TestQueueStore_ListPaginated_Basic(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQueueStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 5; i++ {
		store.Enqueue(domain.QueueItem{
			TrackID:     "pg-" + string(rune('a'+i)),
			ProjectSlug: "proj",
			EnqueuedAt:  now.Add(time.Duration(i) * time.Second),
		})
	}

	// First page: limit 2.
	page, err := store.ListPaginated(domain.PageOpts{Limit: 2}, "proj")
	if err != nil {
		t.Fatalf("ListPaginated: %v", err)
	}
	if len(page.Items) != 2 {
		t.Errorf("page1: expected 2 items, got %d", len(page.Items))
	}
	if page.TotalCount != 5 {
		t.Errorf("TotalCount = %d, want 5", page.TotalCount)
	}
	if page.NextCursor == "" {
		t.Error("expected non-empty NextCursor for first page")
	}

	// Second page using cursor.
	page2, err := store.ListPaginated(domain.PageOpts{Limit: 2, Cursor: page.NextCursor}, "proj")
	if err != nil {
		t.Fatalf("ListPaginated(page2): %v", err)
	}
	if len(page2.Items) != 2 {
		t.Errorf("page2: expected 2 items, got %d", len(page2.Items))
	}

	// Third page should have 1 item.
	page3, err := store.ListPaginated(domain.PageOpts{Limit: 2, Cursor: page2.NextCursor}, "proj")
	if err != nil {
		t.Fatalf("ListPaginated(page3): %v", err)
	}
	if len(page3.Items) != 1 {
		t.Errorf("page3: expected 1 item, got %d", len(page3.Items))
	}
	if page3.NextCursor != "" {
		t.Errorf("page3: expected empty NextCursor, got %q", page3.NextCursor)
	}
}

func TestQueueStore_ListPaginated_StatusFilter(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQueueStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	store.Enqueue(domain.QueueItem{TrackID: "pf-1", ProjectSlug: "proj", EnqueuedAt: now})
	store.Enqueue(domain.QueueItem{TrackID: "pf-2", ProjectSlug: "proj", EnqueuedAt: now.Add(time.Second)})
	store.Assign("pf-2", "agent-1")

	page, err := store.ListPaginated(domain.PageOpts{Limit: 10}, "proj", domain.QueueStatusQueued)
	if err != nil {
		t.Fatalf("ListPaginated: %v", err)
	}
	if len(page.Items) != 1 {
		t.Errorf("expected 1 queued item, got %d", len(page.Items))
	}
	if page.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", page.TotalCount)
	}
}

func TestQueueStore_ListPaginated_ProjectFilter(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewQueueStore(db)

	now := time.Now().UTC().Truncate(time.Second)
	store.Enqueue(domain.QueueItem{TrackID: "pp-1", ProjectSlug: "alpha", EnqueuedAt: now})
	store.Enqueue(domain.QueueItem{TrackID: "pp-2", ProjectSlug: "beta", EnqueuedAt: now.Add(time.Second)})

	page, err := store.ListPaginated(domain.PageOpts{Limit: 10}, "alpha")
	if err != nil {
		t.Fatalf("ListPaginated: %v", err)
	}
	if len(page.Items) != 1 {
		t.Errorf("expected 1 item for alpha, got %d", len(page.Items))
	}
	if page.Items[0].TrackID != "pp-1" {
		t.Errorf("expected pp-1, got %s", page.Items[0].TrackID)
	}
}
