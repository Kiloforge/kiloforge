package analytics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestPostHogTracker_SendsEvent(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var received []captureEvent

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ev captureEvent
		if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
			t.Errorf("decode: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, ev)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tracker := newPostHogTracker("phc_test", "anon-123", srv.URL, srv.Client())

	tracker.Track(context.Background(), "test_event", map[string]any{
		"key": "value",
	})

	// Give the sender goroutine time to process.
	if err := tracker.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}
	if received[0].Event != "test_event" {
		t.Errorf("event: want test_event, got %q", received[0].Event)
	}
	if received[0].APIKey != "phc_test" {
		t.Errorf("api_key: want phc_test, got %q", received[0].APIKey)
	}
	if received[0].DistinctID != "anon-123" {
		t.Errorf("distinct_id: want anon-123, got %q", received[0].DistinctID)
	}
	if received[0].Properties["key"] != "value" {
		t.Errorf("properties.key: want value, got %v", received[0].Properties["key"])
	}
}

func TestPostHogTracker_DropsWhenFull(t *testing.T) {
	t.Parallel()

	// Create a tracker that sends to a non-routable address (connection will fail fast).
	client := &http.Client{Timeout: 50 * time.Millisecond}
	tracker := newPostHogTracker("phc_test", "anon", "http://192.0.2.1:1/capture", client)

	// Fill the buffer.
	for i := 0; i < bufferSize+10; i++ {
		tracker.Track(context.Background(), "spam", nil)
	}

	// Should not block or panic.
	if err := tracker.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestNoopTracker(t *testing.T) {
	t.Parallel()

	var tracker NoopTracker
	tracker.Track(context.Background(), "test", map[string]any{"a": 1})
	if err := tracker.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestAnonymousID(t *testing.T) {
	t.Parallel()

	id1 := AnonymousID("/tmp/test1")
	id2 := AnonymousID("/tmp/test2")

	if id1 == "" {
		t.Error("AnonymousID returned empty string")
	}
	if len(id1) != 32 {
		t.Errorf("expected 32-char hex, got %d chars", len(id1))
	}
	if id1 == id2 {
		t.Error("different dataDirs should produce different IDs")
	}

	// Deterministic.
	if AnonymousID("/tmp/test1") != id1 {
		t.Error("AnonymousID is not deterministic")
	}
}

func TestPostHogTracker_GracefulShutdown(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	count := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		mu.Lock()
		count++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tracker := newPostHogTracker("phc_test", "anon", srv.URL, srv.Client())

	for i := 0; i < 5; i++ {
		tracker.Track(context.Background(), "shutdown_test", nil)
	}

	if err := tracker.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if count < 1 {
		t.Error("expected at least 1 event to be sent during shutdown drain")
	}
}
