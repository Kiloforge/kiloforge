package lock

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestAcquire_FreeLock(t *testing.T) {
	t.Parallel()
	m := New("")
	ctx := context.Background()

	l, err := m.Acquire(ctx, "merge", "dev-1", 30*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.Scope != "merge" || l.Holder != "dev-1" {
		t.Errorf("got scope=%s holder=%s, want merge/dev-1", l.Scope, l.Holder)
	}
}

func TestAcquire_BlocksUntilReleased(t *testing.T) {
	t.Parallel()
	m := New("")
	ctx := context.Background()

	// Acquire first lock.
	_, err := m.Acquire(ctx, "merge", "dev-1", 30*time.Second)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	// Second acquire should block, then succeed after release.
	done := make(chan *Lock, 1)
	go func() {
		l, err := m.Acquire(ctx, "merge", "dev-2", 30*time.Second)
		if err != nil {
			t.Errorf("second acquire: %v", err)
		}
		done <- l
	}()

	// Give goroutine time to register as waiter.
	time.Sleep(50 * time.Millisecond)
	if err := m.Release("merge", "dev-1"); err != nil {
		t.Fatalf("release: %v", err)
	}

	select {
	case l := <-done:
		if l.Holder != "dev-2" {
			t.Errorf("expected holder dev-2, got %s", l.Holder)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second acquire did not complete after release")
	}
}

func TestAcquire_Timeout(t *testing.T) {
	t.Parallel()
	m := New("")
	ctx := context.Background()

	_, _ = m.Acquire(ctx, "merge", "dev-1", 30*time.Second)

	ctx2, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	_, err := m.Acquire(ctx2, "merge", "dev-2", 30*time.Second)
	if err != ErrTimeout {
		t.Fatalf("expected ErrTimeout, got %v", err)
	}
}

func TestRelease_NonHolder(t *testing.T) {
	t.Parallel()
	m := New("")
	ctx := context.Background()

	_, _ = m.Acquire(ctx, "merge", "dev-1", 30*time.Second)

	err := m.Release("merge", "dev-2")
	if err != ErrNotHolder {
		t.Fatalf("expected ErrNotHolder, got %v", err)
	}
}

func TestRelease_NotFound(t *testing.T) {
	t.Parallel()
	m := New("")

	err := m.Release("nonexistent", "dev-1")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestHeartbeat_ExtendsTTL(t *testing.T) {
	t.Parallel()
	m := New("")
	ctx := context.Background()

	l, _ := m.Acquire(ctx, "merge", "dev-1", 2*time.Second)
	origExpiry := l.ExpiresAt

	time.Sleep(50 * time.Millisecond)
	l2, err := m.Heartbeat("merge", "dev-1", 30*time.Second)
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if !l2.ExpiresAt.After(origExpiry) {
		t.Errorf("heartbeat did not extend TTL: orig=%v new=%v", origExpiry, l2.ExpiresAt)
	}
}

func TestHeartbeat_NonHolder(t *testing.T) {
	t.Parallel()
	m := New("")
	ctx := context.Background()

	_, _ = m.Acquire(ctx, "merge", "dev-1", 30*time.Second)

	_, err := m.Heartbeat("merge", "dev-2", 30*time.Second)
	if err != ErrNotHolder {
		t.Fatalf("expected ErrNotHolder, got %v", err)
	}
}

func TestTTLExpiry_AutoRelease(t *testing.T) {
	t.Parallel()
	m := New("")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Acquire with very short TTL.
	_, _ = m.Acquire(ctx, "merge", "dev-1", 200*time.Millisecond)

	m.StartReaper(ctx)

	// Wait for expiry.
	time.Sleep(500 * time.Millisecond)

	// Should be able to acquire now.
	l, err := m.Acquire(ctx, "merge", "dev-2", 30*time.Second)
	if err != nil {
		t.Fatalf("acquire after expiry: %v", err)
	}
	if l.Holder != "dev-2" {
		t.Errorf("expected holder dev-2, got %s", l.Holder)
	}
}

func TestTTLExpiry_UnblocksWaiter(t *testing.T) {
	t.Parallel()
	m := New("")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _ = m.Acquire(ctx, "merge", "dev-1", 200*time.Millisecond)

	m.StartReaper(ctx)

	// This should block and then succeed when TTL expires.
	acquireCtx, acquireCancel := context.WithTimeout(ctx, 2*time.Second)
	defer acquireCancel()

	l, err := m.Acquire(acquireCtx, "merge", "dev-2", 30*time.Second)
	if err != nil {
		t.Fatalf("acquire after TTL expiry: %v", err)
	}
	if l.Holder != "dev-2" {
		t.Errorf("expected dev-2, got %s", l.Holder)
	}
}

func TestConcurrentAcquire(t *testing.T) {
	t.Parallel()
	m := New("")
	ctx := context.Background()

	const n = 10
	var wg sync.WaitGroup
	acquired := make(chan string, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			holder := "dev-" + string(rune('0'+id))
			l, err := m.Acquire(ctx, "merge", holder, 50*time.Millisecond)
			if err != nil {
				t.Errorf("acquire %s: %v", holder, err)
				return
			}
			acquired <- l.Holder
			// Hold briefly then release.
			time.Sleep(10 * time.Millisecond)
			m.Release("merge", holder)
		}(i)
	}

	wg.Wait()
	close(acquired)

	count := 0
	for range acquired {
		count++
	}
	if count != n {
		t.Errorf("expected %d successful acquires, got %d", n, count)
	}
}

func TestList(t *testing.T) {
	t.Parallel()
	m := New("")
	ctx := context.Background()

	// Empty.
	if locks := m.List(); len(locks) != 0 {
		t.Errorf("expected 0 locks, got %d", len(locks))
	}

	_, _ = m.Acquire(ctx, "merge", "dev-1", 30*time.Second)
	_, _ = m.Acquire(ctx, "deploy", "dev-2", 30*time.Second)

	locks := m.List()
	if len(locks) != 2 {
		t.Errorf("expected 2 locks, got %d", len(locks))
	}
}

// Persistence tests

func TestPersistence_RoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	m := New(dir)
	ctx := context.Background()

	_, _ = m.Acquire(ctx, "merge", "dev-1", 30*time.Second)
	_, _ = m.Acquire(ctx, "deploy", "dev-2", 30*time.Second)

	// Load into new manager.
	m2 := New(dir)
	locks := m2.List()
	if len(locks) != 2 {
		t.Fatalf("expected 2 locks after load, got %d", len(locks))
	}
}

func TestPersistence_ExpiredDiscarded(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	m := New(dir)
	ctx := context.Background()

	_, _ = m.Acquire(ctx, "merge", "dev-1", 100*time.Millisecond)

	time.Sleep(200 * time.Millisecond)

	m2 := New(dir)
	if locks := m2.List(); len(locks) != 0 {
		t.Errorf("expected expired lock to be discarded, got %d locks", len(locks))
	}
}

func TestPersistence_MissingFile(t *testing.T) {
	t.Parallel()
	m := New(t.TempDir())
	if locks := m.List(); len(locks) != 0 {
		t.Errorf("expected 0 locks from missing file, got %d", len(locks))
	}
}

func TestPersistence_CorruptFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, lockFile), []byte("not json"), 0o644)

	m := New(dir)
	// Should not crash, just start empty.
	if locks := m.List(); len(locks) != 0 {
		t.Errorf("expected 0 locks from corrupt file, got %d", len(locks))
	}
}

func TestReentrantAcquire(t *testing.T) {
	t.Parallel()
	m := New("")
	ctx := context.Background()

	_, _ = m.Acquire(ctx, "merge", "dev-1", 30*time.Second)

	// Same holder re-acquires — should succeed (re-entrant).
	l, err := m.Acquire(ctx, "merge", "dev-1", 60*time.Second)
	if err != nil {
		t.Fatalf("re-entrant acquire: %v", err)
	}
	if l.Holder != "dev-1" {
		t.Errorf("expected dev-1, got %s", l.Holder)
	}
}
