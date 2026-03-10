package agent

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

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
