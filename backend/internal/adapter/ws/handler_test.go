package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kiloforge/internal/core/domain"

	"nhooyr.io/websocket"
)

// fakeAgentFinder implements AgentFinder for tests.
type fakeAgentFinder struct {
	agents map[string]*domain.AgentInfo
}

func (f *fakeAgentFinder) FindAgent(idPrefix string) (*domain.AgentInfo, error) {
	if a, ok := f.agents[idPrefix]; ok {
		return a, nil
	}
	return nil, fmt.Errorf("not found")
}

func TestHandlerAgentWS_NotFound(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	h := NewHandler(sm, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, _, err := websocket.Dial(ctx, srv.URL+"/ws/agent/nonexistent", nil)
	if err == nil {
		t.Fatal("expected error connecting to nonexistent agent")
	}
}

func TestHandlerAgentWS_Connect(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	h := NewHandler(sm, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create a bridge for a fake agent.
	r, w := io.Pipe()
	defer r.Close()
	done := make(chan struct{})
	bridge := NewBridge("agent-1", w, done)
	sm.RegisterBridge("agent-1", bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, srv.URL+"/ws/agent/agent-1", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Should receive a status message.
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var msg Message
	json.Unmarshal(data, &msg)
	if msg.Type != MsgStatus || msg.Status != "running" {
		t.Errorf("expected status=running, got type=%s status=%s", msg.Type, msg.Status)
	}

	// Send input.
	input, _ := json.Marshal(Message{Type: MsgInput, Text: "hello agent"})
	if err := conn.Write(ctx, websocket.MessageText, input); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read from agent stdin pipe.
	buf := make([]byte, 64)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("read stdin: %v", err)
	}
	if string(buf[:n]) != "hello agent\n" {
		t.Errorf("stdin got %q, want %q", string(buf[:n]), "hello agent\n")
	}

	// Close the agent.
	w.Close()
	close(done)

	// Should receive completed status.
	_, data, err = conn.Read(ctx)
	if err != nil {
		t.Fatalf("read final: %v", err)
	}
	json.Unmarshal(data, &msg)
	if msg.Type != MsgStatus || msg.Status != "completed" {
		t.Errorf("expected status=completed, got type=%s status=%s", msg.Type, msg.Status)
	}
}

func TestHandlerAgentWS_OutputBroadcast(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	h := NewHandler(sm, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, w := io.Pipe()
	defer w.Close()
	done := make(chan struct{})
	bridge := NewBridge("agent-2", w, done)

	// Pre-buffer some output.
	bridge.Buffer.Write(OutputMsg("line 1"))
	bridge.Buffer.Write(OutputMsg("line 2"))
	sm.RegisterBridge("agent-2", bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, srv.URL+"/ws/agent/agent-2", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Should receive 2 buffered lines, then status.
	for i := 0; i < 3; i++ {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		var msg Message
		json.Unmarshal(data, &msg)
		if i < 2 && msg.Type != MsgOutput {
			t.Errorf("msg %d: type = %s, want output", i, msg.Type)
		}
		if i == 2 && msg.Type != MsgStatus {
			t.Errorf("msg %d: type = %s, want status", i, msg.Type)
		}
	}
}

func TestHandlerAgentWS_InterruptFromPrimary(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	h := NewHandler(sm, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create an SDK bridge with an interrupt handler.
	var interrupted bool
	done := make(chan struct{})
	bridge := NewSDKBridge("agent-int", func(text string) error { return nil }, done)
	bridge.InterruptHandler = func() { interrupted = true }
	sm.RegisterBridge("agent-int", bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, srv.URL+"/ws/agent/agent-int", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	// Read the initial status message.
	_, _, err = conn.Read(ctx)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}

	// Send an interrupt message.
	intMsg, _ := json.Marshal(Message{Type: MsgInterrupt})
	if err := conn.Write(ctx, websocket.MessageText, intMsg); err != nil {
		t.Fatalf("write interrupt: %v", err)
	}

	// Give readLoop time to process.
	time.Sleep(100 * time.Millisecond)

	if !interrupted {
		t.Error("expected InterruptHandler to be called")
	}
}

func TestHandlerAgentWS_InterruptFromObserver(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	h := NewHandler(sm, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create an SDK bridge with an interrupt handler.
	var interrupted bool
	done := make(chan struct{})
	bridge := NewSDKBridge("agent-int2", func(text string) error { return nil }, done)
	bridge.InterruptHandler = func() { interrupted = true }
	sm.RegisterBridge("agent-int2", bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First connection is primary.
	conn1, _, err := websocket.Dial(ctx, srv.URL+"/ws/agent/agent-int2", nil)
	if err != nil {
		t.Fatalf("dial primary: %v", err)
	}
	defer conn1.CloseNow()
	_, _, _ = conn1.Read(ctx) // read status

	// Second connection is observer (read-only).
	conn2, _, err := websocket.Dial(ctx, srv.URL+"/ws/agent/agent-int2", nil)
	if err != nil {
		t.Fatalf("dial observer: %v", err)
	}
	defer conn2.CloseNow()
	_, _, _ = conn2.Read(ctx) // read status

	// Observer sends interrupt — should be ignored (no readLoop for observers).
	intMsg, _ := json.Marshal(Message{Type: MsgInterrupt})
	if err := conn2.Write(ctx, websocket.MessageText, intMsg); err != nil {
		t.Fatalf("write interrupt: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if interrupted {
		t.Error("observer should NOT be able to trigger interrupt")
	}
}

func TestHandlerAgentWS_NoBridge_TerminalAgent(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	finder := &fakeAgentFinder{agents: map[string]*domain.AgentInfo{
		"agent-done": {ID: "agent-done", Status: string(domain.AgentStatusCompleted)},
	}}
	h := NewHandler(sm, finder, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// No bridge registered — agent is completed. Should get a WS connection
	// with a status message then clean close.
	conn, _, err := websocket.Dial(ctx, srv.URL+"/ws/agent/agent-done", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var msg Message
	json.Unmarshal(data, &msg)
	if msg.Type != MsgStatus || msg.Status != "completed" {
		t.Errorf("expected status=completed, got type=%s status=%s", msg.Type, msg.Status)
	}

	// Connection should be closed by server after status.
	_, _, err = conn.Read(ctx)
	if err == nil {
		t.Error("expected connection to be closed after terminal status")
	}
}

func TestHandlerAgentWS_NoBridge_NonTerminalAgent(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	finder := &fakeAgentFinder{agents: map[string]*domain.AgentInfo{
		"agent-limbo": {ID: "agent-limbo", Status: string(domain.AgentStatusSuspending)},
	}}
	h := NewHandler(sm, finder, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Agent exists but is not terminal and has no bridge — should return 503.
	_, _, err := websocket.Dial(ctx, srv.URL+"/ws/agent/agent-limbo", nil)
	if err == nil {
		t.Fatal("expected error connecting to non-terminal agent without bridge")
	}
}

func TestHandlerAgentWS_InitialStatusFromStore(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	finder := &fakeAgentFinder{agents: map[string]*domain.AgentInfo{
		"agent-wait": {ID: "agent-wait", Status: string(domain.AgentStatusWaiting)},
	}}
	h := NewHandler(sm, finder, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Register a bridge so the normal path is taken.
	_, w := io.Pipe()
	defer w.Close()
	done := make(chan struct{})
	bridge := NewBridge("agent-wait", w, done)
	sm.RegisterBridge("agent-wait", bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, srv.URL+"/ws/agent/agent-wait", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	// Should receive the actual agent status from the store, not hardcoded "running".
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var msg Message
	json.Unmarshal(data, &msg)
	if msg.Type != MsgStatus || msg.Status != "waiting" {
		t.Errorf("expected status=waiting, got type=%s status=%s", msg.Type, msg.Status)
	}
}
