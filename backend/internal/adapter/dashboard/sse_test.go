package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

// Compile-time check that SSEHub satisfies port.EventBus.
var _ port.EventBus = (*SSEHub)(nil)

func TestSSEHub_ImplementsEventBus(t *testing.T) {
	t.Parallel()
	var bus port.EventBus = NewSSEHub()
	if bus.ClientCount() != 0 {
		t.Errorf("new hub should have 0 clients, got %d", bus.ClientCount())
	}
}

func TestSSEHub_PublishSubscribe(t *testing.T) {
	t.Parallel()
	hub := NewSSEHub()

	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	hub.Publish(domain.Event{Type: "test_event", Data: "payload"})

	select {
	case ev := <-ch:
		if ev.Type != "test_event" {
			t.Errorf("type = %q, want test_event", ev.Type)
		}
		if ev.Data != "payload" {
			t.Errorf("data = %v, want payload", ev.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestSSEHub_UnsubscribeStopsDelivery(t *testing.T) {
	t.Parallel()
	hub := NewSSEHub()

	ch := hub.Subscribe()
	if hub.ClientCount() != 1 {
		t.Fatalf("client count = %d, want 1", hub.ClientCount())
	}

	hub.Unsubscribe(ch)
	if hub.ClientCount() != 0 {
		t.Errorf("client count after unsub = %d, want 0", hub.ClientCount())
	}

	// Channel should be closed.
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after unsubscribe")
	}
}

func TestSSEHub_MultipleSubscribers(t *testing.T) {
	t.Parallel()
	hub := NewSSEHub()

	ch1 := hub.Subscribe()
	ch2 := hub.Subscribe()
	ch3 := hub.Subscribe()
	defer hub.Unsubscribe(ch1)
	defer hub.Unsubscribe(ch2)
	defer hub.Unsubscribe(ch3)

	if hub.ClientCount() != 3 {
		t.Fatalf("client count = %d, want 3", hub.ClientCount())
	}

	hub.Publish(domain.Event{Type: "broadcast", Data: nil})

	for i, ch := range []<-chan domain.Event{ch1, ch2, ch3} {
		select {
		case ev := <-ch:
			if ev.Type != "broadcast" {
				t.Errorf("subscriber %d: type = %q, want broadcast", i, ev.Type)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timed out", i)
		}
	}
}

func TestSSEHub_SlowClientNonBlocking(t *testing.T) {
	t.Parallel()
	hub := NewSSEHub()

	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	// Fill the buffer (capacity 16) and then send one more.
	for i := 0; i < 20; i++ {
		hub.Publish(domain.Event{Type: "flood", Data: i})
	}

	// Should not block or panic — slow client just drops events.
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count > 16 {
		t.Errorf("received %d events, expected at most 16 (buffer size)", count)
	}
	if count == 0 {
		t.Error("received 0 events, expected at least some")
	}
}

func TestSSEHub_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	hub := NewSSEHub()

	var wg sync.WaitGroup
	// Concurrent publishers.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				hub.Publish(domain.Event{Type: fmt.Sprintf("event_%d", n), Data: j})
			}
		}(i)
	}

	// Concurrent subscribe/unsubscribe.
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch := hub.Subscribe()
			time.Sleep(10 * time.Millisecond)
			hub.Unsubscribe(ch)
		}()
	}

	wg.Wait()
	// No race or panic means pass.
}

func TestSSEHub_TypedEventConstructors(t *testing.T) {
	t.Parallel()
	hub := NewSSEHub()
	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	tests := []struct {
		name     string
		event    domain.Event
		wantType string
	}{
		{"agent_update", domain.NewAgentUpdateEvent(map[string]string{"id": "a1"}), domain.EventAgentUpdate},
		{"agent_removed", domain.NewAgentRemovedEvent("a1"), domain.EventAgentRemoved},
		{"quota_update", domain.NewQuotaUpdateEvent(nil), domain.EventQuotaUpdate},
		{"track_update", domain.NewTrackUpdateEvent(nil), domain.EventTrackUpdate},
		{"board_update", domain.NewBoardUpdateEvent(nil), domain.EventBoardUpdate},
		{"project_update", domain.NewProjectUpdateEvent(nil), domain.EventProjectUpdate},
		{"project_removed", domain.NewProjectRemovedEvent("my-proj"), domain.EventProjectRemoved},
		{"lock_update", domain.NewLockUpdateEvent(nil), domain.EventLockUpdate},
		{"lock_released", domain.NewLockReleasedEvent("merge"), domain.EventLockReleased},
	}

	for _, tt := range tests {
		hub.Publish(tt.event)
		select {
		case ev := <-ch:
			if ev.Type != tt.wantType {
				t.Errorf("%s: type = %q, want %q", tt.name, ev.Type, tt.wantType)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s: timed out", tt.name)
		}
	}
}

func TestSSEHandler_Integration(t *testing.T) {
	t.Parallel()
	srv := New(0, &testAgentLister{}, nil, &testProjectLister{}, nil)

	// Use a cancellable context to cleanly stop the handler.
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.handleSSE(rec, req)
	}()

	// Give the handler time to subscribe.
	time.Sleep(50 * time.Millisecond)

	if srv.hub.ClientCount() != 1 {
		t.Fatalf("SSE client count = %d, want 1", srv.hub.ClientCount())
	}

	// Publish an event via the bus.
	srv.hub.Publish(domain.NewAgentUpdateEvent(map[string]string{"id": "test-agent"}))

	// Give the handler time to write, then cancel to stop it.
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done // Wait for handler to exit before reading the recorder.

	body := rec.Body.String()
	if body == "" {
		t.Fatal("expected SSE output, got empty")
	}

	// Verify SSE wire format.
	if !contains(body, "event: agent_update") {
		t.Errorf("missing 'event: agent_update' in SSE output:\n%s", body)
	}
	if !contains(body, "data: ") {
		t.Errorf("missing 'data: ' in SSE output:\n%s", body)
	}

	// Verify the data is valid JSON.
	dataStart := indexOf(body, "data: ") + len("data: ")
	dataEnd := indexOf(body[dataStart:], "\n") + dataStart
	dataJSON := body[dataStart:dataEnd]
	var parsed domain.Event
	if err := json.Unmarshal([]byte(dataJSON), &parsed); err != nil {
		t.Errorf("SSE data is not valid JSON: %v\ndata: %s", err, dataJSON)
	}

	// Verify Content-Type header.
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
}

// fakeResponseWriter implements http.Flusher for the SSE handler test.
// (httptest.ResponseRecorder already implements Flusher in Go 1.20+)
func init() {
	// Verify httptest.ResponseRecorder implements http.Flusher.
	var _ http.Flusher = (*httptest.ResponseRecorder)(nil)
}

func contains(s, substr string) bool {
	return indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
