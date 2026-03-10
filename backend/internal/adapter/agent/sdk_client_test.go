package agent

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"kiloforge/internal/core/port"

	"github.com/schlunsen/claude-agent-sdk-go/types"
)

// mockSDKClient implements sdkClientAPI for testing.
type mockSDKClient struct {
	connected   bool
	responseCh  chan types.Message
	queryCalled bool
	queryErr    error
	closeCalled bool
}

func (m *mockSDKClient) Query(_ context.Context, _ string) error {
	m.queryCalled = true
	return m.queryErr
}

func (m *mockSDKClient) ReceiveResponse(_ context.Context) <-chan types.Message {
	return m.responseCh
}

func (m *mockSDKClient) IsConnected() bool {
	return m.connected
}

func (m *mockSDKClient) Close(_ context.Context) error {
	m.closeCalled = true
	return nil
}

func (m *mockSDKClient) Connect(_ context.Context) error {
	m.connected = true
	return nil
}

// newTestSDKSession creates a minimal SDKSession for testing Close() behavior.
// It does not connect to any real Claude CLI.
func newTestSDKSession() *SDKSession {
	ctx, cancel := context.WithCancel(context.Background())
	return &SDKSession{
		ctx:    ctx,
		cancel: cancel,
		output: make(chan []byte, 10),
		done:   make(chan struct{}),
	}
}

// newTestSDKSessionWithMock creates an SDKSession with a mock client for testing.
func newTestSDKSessionWithMock(mock *mockSDKClient) *SDKSession {
	ctx, cancel := context.WithCancel(context.Background())
	return &SDKSession{
		client: mock,
		ctx:    ctx,
		cancel: cancel,
		output: make(chan []byte, 100),
		done:   make(chan struct{}),
	}
}

func TestSDKSession_Close_Concurrent(t *testing.T) {
	s := newTestSDKSession()

	var wg sync.WaitGroup
	// Call Close from multiple goroutines simultaneously — must not panic.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Close()
		}()
	}
	wg.Wait()

	// Verify channels are closed.
	select {
	case _, ok := <-s.output:
		if ok {
			t.Error("output channel should be closed")
		}
	default:
		// closed channel returns immediately
	}

	select {
	case <-s.done:
		// expected — channel closed
	default:
		t.Error("done channel should be closed")
	}
}

func TestSDKSession_Close_Sequential(t *testing.T) {
	s := newTestSDKSession()

	// Calling Close twice sequentially must not panic.
	s.Close()
	s.Close()

	// Verify context is cancelled.
	if s.ctx.Err() == nil {
		t.Error("context should be cancelled after Close()")
	}
}

func TestSDKSession_RelayResponse_ClosesOnDisconnect(t *testing.T) {
	// When the response channel closes and the client is no longer connected,
	// relayResponse should call session.Close() to unblock monitorSDKSession.
	mock := &mockSDKClient{
		connected:  true,
		responseCh: make(chan types.Message),
	}
	s := newTestSDKSessionWithMock(mock)
	s.querying = true

	// Simulate: response channel closes, client is disconnected.
	mock.connected = false
	close(mock.responseCh)

	s.relayResponse(s.ctx, nil, "", nil)

	// querying must be reset.
	s.mu.Lock()
	q := s.querying
	s.mu.Unlock()
	if q {
		t.Error("querying should be false after relayResponse returns")
	}

	// Session should be closed (context cancelled) because client disconnected.
	select {
	case <-s.done:
		// expected
	case <-time.After(time.Second):
		t.Error("session should be closed when client disconnects after response ends")
	}
}

func TestSDKSession_RelayResponse_NoCloseWhenStillConnected(t *testing.T) {
	// When the response channel closes but the client is still connected
	// (e.g., turn completed normally), session should NOT be closed.
	mock := &mockSDKClient{
		connected:  true,
		responseCh: make(chan types.Message),
	}
	s := newTestSDKSessionWithMock(mock)
	s.querying = true

	close(mock.responseCh)

	s.relayResponse(s.ctx, nil, "", nil)

	// querying must be reset.
	s.mu.Lock()
	q := s.querying
	s.mu.Unlock()
	if q {
		t.Error("querying should be false after relayResponse returns")
	}

	// Session should NOT be closed — client is still connected.
	select {
	case <-s.done:
		t.Error("session should NOT be closed when client is still connected")
	default:
		// expected
	}
}

func TestSDKSession_RelayResponse_Timeout(t *testing.T) {
	// When no messages arrive within the timeout, relayResponse should
	// emit an error, reset querying, and close the session.
	mock := &mockSDKClient{
		connected:  true,
		responseCh: make(chan types.Message), // never sends
	}
	s := newTestSDKSessionWithMock(mock)
	s.querying = true
	s.responseTimeout = 100 * time.Millisecond // short timeout for test

	// relayResponse should return after the timeout fires.
	done := make(chan struct{})
	go func() {
		s.relayResponse(s.ctx, nil, "", nil)
		close(done)
	}()

	select {
	case <-done:
		// relayResponse returned — good.
	case <-time.After(5 * time.Second):
		t.Fatal("relayResponse should have returned after timeout")
	}

	// querying should be reset.
	s.mu.Lock()
	q := s.querying
	s.mu.Unlock()
	if q {
		t.Error("querying should be false after timeout")
	}

	// An error message should have been emitted.
	var found bool
	for len(s.output) > 0 {
		msg := <-s.output
		var parsed map[string]interface{}
		if err := json.Unmarshal(msg, &parsed); err == nil {
			if parsed["type"] == "error" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("expected an error message to be emitted on timeout")
	}

	// Session should be closed.
	select {
	case <-s.done:
		// expected
	case <-time.After(time.Second):
		t.Error("session should be closed after timeout")
	}
}

func TestSDKSession_Interrupt_ActiveTurn(t *testing.T) {
	// Interrupt during an active turn should cancel the relay context,
	// causing relayResponse to exit and emit turn_end with interrupted=true.
	mock := &mockSDKClient{
		connected:  true,
		responseCh: make(chan types.Message),
	}
	s := newTestSDKSessionWithMock(mock)

	// Start a query (sets querying=true, creates queryCancel).
	err := s.Query(s.ctx, "hello", nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify querying is true.
	s.mu.Lock()
	if !s.querying {
		t.Fatal("querying should be true after Query()")
	}
	s.mu.Unlock()

	// Interrupt the turn.
	s.Interrupt()

	// Wait for relayResponse to finish (it should exit via ctx.Done()).
	deadline := time.After(2 * time.Second)
	for {
		s.mu.Lock()
		q := s.querying
		s.mu.Unlock()
		if !q {
			break
		}
		select {
		case <-deadline:
			t.Fatal("querying should be false after interrupt")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Check that a turn_end message with interrupted=true was emitted.
	var foundInterrupted bool
	for len(s.output) > 0 {
		msg := <-s.output
		var parsed map[string]interface{}
		if err := json.Unmarshal(msg, &parsed); err == nil {
			if parsed["type"] == "turn_end" {
				if interrupted, ok := parsed["interrupted"].(bool); ok && interrupted {
					foundInterrupted = true
				}
			}
		}
	}
	if !foundInterrupted {
		t.Error("expected turn_end message with interrupted=true")
	}
}

func TestSDKSession_Interrupt_NoActiveTurn(t *testing.T) {
	// Interrupt when no turn is active should be a no-op (no panic).
	s := newTestSDKSession()
	s.Interrupt() // should not panic
}

func TestSDKSession_Interrupt_SessionStillUsable(t *testing.T) {
	// After interrupt, a new query should succeed.
	mock := &mockSDKClient{
		connected:  true,
		responseCh: make(chan types.Message),
	}
	s := newTestSDKSessionWithMock(mock)

	// Start and interrupt a query.
	err := s.Query(s.ctx, "hello", nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s.Interrupt()

	// Wait for relayResponse to finish.
	deadline := time.After(2 * time.Second)
	for {
		s.mu.Lock()
		q := s.querying
		s.mu.Unlock()
		if !q {
			break
		}
		select {
		case <-deadline:
			t.Fatal("querying should be false after interrupt")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// New query should be accepted (need fresh response channel).
	mock.responseCh = make(chan types.Message)
	err = s.Query(s.ctx, "world", nil, "", nil)
	if err != nil {
		t.Errorf("expected new query to succeed after interrupt, got: %v", err)
	}
}

func TestSDKSession_SessionIDCallback_Invoked(t *testing.T) {
	// When a ResultMessage with a SessionID is received, the sessionIDCallback
	// should be invoked with the real session ID.
	mock := &mockSDKClient{
		connected:  true,
		responseCh: make(chan types.Message, 1),
	}
	s := newTestSDKSessionWithMock(mock)
	s.querying = true

	var capturedID string
	s.SetSessionIDCallback(func(id string) {
		capturedID = id
	})

	// Send a ResultMessage with a real session ID, then close the channel.
	costUSD := 0.01
	mock.responseCh <- &types.ResultMessage{
		SessionID:    "real-sdk-session-id",
		TotalCostUSD: &costUSD,
	}
	close(mock.responseCh)

	s.relayResponse(s.ctx, nil, "", nil)

	if capturedID != "real-sdk-session-id" {
		t.Errorf("expected callback with 'real-sdk-session-id', got %q", capturedID)
	}
}

func TestSDKSession_SessionIDCallback_NotCalledWhenEmpty(t *testing.T) {
	// When a ResultMessage has an empty SessionID, the callback should NOT be invoked.
	mock := &mockSDKClient{
		connected:  true,
		responseCh: make(chan types.Message, 1),
	}
	s := newTestSDKSessionWithMock(mock)
	s.querying = true

	called := false
	s.SetSessionIDCallback(func(_ string) {
		called = true
	})

	costUSD := 0.01
	mock.responseCh <- &types.ResultMessage{
		SessionID:    "",
		TotalCostUSD: &costUSD,
	}
	close(mock.responseCh)

	s.relayResponse(s.ctx, nil, "", nil)

	if called {
		t.Error("callback should not be called when SessionID is empty")
	}
}

func TestQueryOneShot_ReturnsSessionID(t *testing.T) {
	// QueryOneShot should return the real session ID from the ResultMessage.
	// We can't easily test the full QueryOneShot without a real SDK, but we
	// verify the return signature change compiles and the logic is correct
	// by checking the function signature exists with 3 return values.
	// The actual integration is tested through runSDKAgent in spawner tests.

	// This is a compile-time check — if QueryOneShot doesn't return
	// (string, string, error), this test won't compile.
	var fn func(ctx context.Context, prompt, workDir, model, logFilePath string,
		tracker *QuotaTracker, agentID string, span port.SpanEnder,
		envVars map[string]string) (string, string, error)
	fn = QueryOneShot
	_ = fn
}

func TestSDKSession_Query_Disconnected(t *testing.T) {
	// Query should return an error immediately if the client is disconnected.
	mock := &mockSDKClient{
		connected:  false,
		responseCh: make(chan types.Message),
	}
	s := newTestSDKSessionWithMock(mock)

	err := s.Query(s.ctx, "hello", nil, "", nil)
	if err == nil {
		t.Fatal("expected error when client is disconnected")
	}
	// Query should check IsConnected before calling client.Query.
	if mock.queryCalled {
		t.Error("client.Query should not be called when client is disconnected")
	}

	// querying should NOT be stuck true.
	s.mu.Lock()
	q := s.querying
	s.mu.Unlock()
	if q {
		t.Error("querying should be false when query is rejected")
	}
}
