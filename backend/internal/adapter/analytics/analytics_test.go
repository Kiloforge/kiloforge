package analytics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"kiloforge/internal/core/port"
)

var (
	_ port.AnalyticsTracker = (*PostHog)(nil)
	_ port.AnalyticsTracker = (*Noop)(nil)
)

func TestNoop_Track(t *testing.T) {
	t.Parallel()
	n := &Noop{}
	n.Track(context.Background(), "test", map[string]any{"k": "v"})
	if err := n.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown: %v", err)
	}
}

func TestPostHog_Track_SendsToServer(t *testing.T) {
	var received atomic.Int32
	var lastPayload capturePayload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		json.NewDecoder(r.Body).Decode(&lastPayload)
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := NewPostHog("phc_test", "anon-123")
	p.captureURL = srv.URL

	p.Track(context.Background(), "test_event", map[string]any{"color": "blue"})
	p.Shutdown(context.Background())

	if received.Load() != 1 {
		t.Fatalf("expected 1 event, got %d", received.Load())
	}
	if lastPayload.APIKey != "phc_test" {
		t.Errorf("APIKey: want %q, got %q", "phc_test", lastPayload.APIKey)
	}
	if lastPayload.Event != "test_event" {
		t.Errorf("Event: want %q, got %q", "test_event", lastPayload.Event)
	}
	if lastPayload.DistinctID != "anon-123" {
		t.Errorf("DistinctID: want %q, got %q", "anon-123", lastPayload.DistinctID)
	}
}

func TestPostHog_MultipleEvents(t *testing.T) {
	var received atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body.Close()
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := NewPostHog("phc_key", "anon")
	p.captureURL = srv.URL

	for i := 0; i < 5; i++ {
		p.Track(context.Background(), "evt", nil)
	}
	p.Shutdown(context.Background())

	if received.Load() != 5 {
		t.Errorf("expected 5 events, got %d", received.Load())
	}
}

func TestPostHog_BufferFull_NoPanic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		r.Body.Close()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := NewPostHog("phc_key", "anon")
	p.captureURL = srv.URL

	for i := 0; i < bufferSize+10; i++ {
		p.Track(context.Background(), "overflow", nil)
	}
	p.Shutdown(context.Background())
}

func TestPostHog_ServerDown_NoError(t *testing.T) {
	p := NewPostHog("phc_key", "anon")
	p.captureURL = "http://localhost:1"
	p.Track(context.Background(), "fail", nil)
	p.Shutdown(context.Background())
}

func TestAnonymousID_Stable(t *testing.T) {
	t.Parallel()
	id1 := AnonymousID("/data")
	id2 := AnonymousID("/data")
	if id1 != id2 {
		t.Errorf("not stable: %q != %q", id1, id2)
	}
	if len(id1) != 32 {
		t.Errorf("length: want 32, got %d", len(id1))
	}
}

func TestAnonymousID_Different(t *testing.T) {
	t.Parallel()
	if AnonymousID("/a") == AnonymousID("/b") {
		t.Error("should differ for different inputs")
	}
}
