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

	// Should receive the input echo.
	_, data, err = conn.Read(ctx)
	if err != nil {
		t.Fatalf("read echo: %v", err)
	}
	json.Unmarshal(data, &msg)
	if msg.Type != MsgInputEcho || msg.Text != "hello agent" {
		t.Errorf("expected input_echo with text 'hello agent', got type=%s text=%q", msg.Type, msg.Text)
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
	interruptCh := make(chan struct{}, 1)
	done := make(chan struct{})
	bridge := NewSDKBridge("agent-int", func(text string) error { return nil }, done)
	bridge.InterruptHandler = func() {
		select {
		case interruptCh <- struct{}{}:
		default:
		}
	}
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

	// Wait for interrupt handler to be called.
	select {
	case <-interruptCh:
		// success
	case <-time.After(2 * time.Second):
		t.Error("expected InterruptHandler to be called")
	}
}

func TestHandlerAgentWS_InterruptFromSecondSession(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	h := NewHandler(sm, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create an SDK bridge with an interrupt handler.
	interruptCh := make(chan struct{}, 1)
	done := make(chan struct{})
	bridge := NewSDKBridge("agent-int2", func(text string) error { return nil }, done)
	bridge.InterruptHandler = func() {
		select {
		case interruptCh <- struct{}{}:
		default:
		}
	}
	sm.RegisterBridge("agent-int2", bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First connection.
	conn1, _, err := websocket.Dial(ctx, srv.URL+"/ws/agent/agent-int2", nil)
	if err != nil {
		t.Fatalf("dial conn1: %v", err)
	}
	defer conn1.CloseNow()
	_, _, _ = conn1.Read(ctx) // read status

	// Second connection.
	conn2, _, err := websocket.Dial(ctx, srv.URL+"/ws/agent/agent-int2", nil)
	if err != nil {
		t.Fatalf("dial conn2: %v", err)
	}
	defer conn2.CloseNow()
	_, _, _ = conn2.Read(ctx) // read status

	// Second session sends interrupt — should work (all sessions have read loops).
	intMsg, _ := json.Marshal(Message{Type: MsgInterrupt})
	if err := conn2.Write(ctx, websocket.MessageText, intMsg); err != nil {
		t.Fatalf("write interrupt: %v", err)
	}

	select {
	case <-interruptCh:
		// success — any session can interrupt
	case <-time.After(2 * time.Second):
		t.Error("expected InterruptHandler to be called from second session")
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

func TestHandlerAgentWS_BothSessionsCanSendInput(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	h := NewHandler(sm, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// SDK bridge that records input.
	inputCh := make(chan string, 10)
	done := make(chan struct{})
	bridge := NewSDKBridge("agent-multi", func(text string) error {
		inputCh <- text
		return nil
	}, done)
	sm.RegisterBridge("agent-multi", bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First connection.
	conn1, _, err := websocket.Dial(ctx, srv.URL+"/ws/agent/agent-multi", nil)
	if err != nil {
		t.Fatalf("dial conn1: %v", err)
	}
	defer conn1.CloseNow()
	_, _, _ = conn1.Read(ctx) // status

	// Second connection.
	conn2, _, err := websocket.Dial(ctx, srv.URL+"/ws/agent/agent-multi", nil)
	if err != nil {
		t.Fatalf("dial conn2: %v", err)
	}
	defer conn2.CloseNow()
	_, _, _ = conn2.Read(ctx) // status

	// Both send input.
	msg1, _ := json.Marshal(Message{Type: MsgInput, Text: "from-conn1"})
	msg2, _ := json.Marshal(Message{Type: MsgInput, Text: "from-conn2"})

	if err := conn1.Write(ctx, websocket.MessageText, msg1); err != nil {
		t.Fatalf("write conn1: %v", err)
	}
	if err := conn2.Write(ctx, websocket.MessageText, msg2); err != nil {
		t.Fatalf("write conn2: %v", err)
	}

	// Both inputs should arrive.
	got := make(map[string]bool)
	for i := 0; i < 2; i++ {
		select {
		case text := <-inputCh:
			got[text] = true
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for input %d; got so far: %v", i, got)
		}
	}
	if !got["from-conn1"] {
		t.Error("missing input from conn1")
	}
	if !got["from-conn2"] {
		t.Error("missing input from conn2")
	}
}

func TestHandlerAgentWS_ReconnectCanSendInput(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	h := NewHandler(sm, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// SDK bridge that records input.
	inputCh := make(chan string, 10)
	done := make(chan struct{})
	bridge := NewSDKBridge("agent-recon", func(text string) error {
		inputCh <- text
		return nil
	}, done)
	sm.RegisterBridge("agent-recon", bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First connection — send input, then close.
	conn1, _, err := websocket.Dial(ctx, srv.URL+"/ws/agent/agent-recon", nil)
	if err != nil {
		t.Fatalf("dial conn1: %v", err)
	}
	_, _, _ = conn1.Read(ctx) // status
	msg1, _ := json.Marshal(Message{Type: MsgInput, Text: "first"})
	if err := conn1.Write(ctx, websocket.MessageText, msg1); err != nil {
		t.Fatalf("write conn1: %v", err)
	}
	select {
	case <-inputCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first input")
	}
	conn1.Close(websocket.StatusNormalClosure, "bye")

	// Brief pause to let RemoveSession run.
	time.Sleep(50 * time.Millisecond)

	// Reconnect.
	conn2, _, err := websocket.Dial(ctx, srv.URL+"/ws/agent/agent-recon", nil)
	if err != nil {
		t.Fatalf("dial conn2: %v", err)
	}
	defer conn2.CloseNow()

	// Read replayed echo + status.
	for {
		_, data, err := conn2.Read(ctx)
		if err != nil {
			t.Fatalf("read replay: %v", err)
		}
		var m Message
		json.Unmarshal(data, &m)
		if m.Type == MsgStatus {
			break
		}
	}

	// Send input on the reconnected session.
	msg2, _ := json.Marshal(Message{Type: MsgInput, Text: "second"})
	if err := conn2.Write(ctx, websocket.MessageText, msg2); err != nil {
		t.Fatalf("write conn2: %v", err)
	}
	select {
	case text := <-inputCh:
		if text != "second" {
			t.Errorf("got %q, want %q", text, "second")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for reconnected input")
	}
}

func TestHandlerAgentWS_BridgeDone_SendsActualStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		agentStatus    string
		expectedStatus string
		expectExitCode bool
	}{
		{"stopped", string(domain.AgentStatusStopped), "stopped", false},
		{"completed", string(domain.AgentStatusCompleted), "completed", true},
		{"failed", string(domain.AgentStatusFailed), "failed", false},
		{"force-killed", string(domain.AgentStatusForceKilled), "force-killed", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sm := NewSessionManager()
			finder := &fakeAgentFinder{agents: map[string]*domain.AgentInfo{
				"agent-exit": {ID: "agent-exit", Status: tc.agentStatus},
			}}
			h := NewHandler(sm, finder, nil)
			mux := http.NewServeMux()
			h.RegisterRoutes(mux)

			srv := httptest.NewServer(mux)
			defer srv.Close()

			done := make(chan struct{})
			bridge := NewSDKBridge("agent-exit", func(text string) error { return nil }, done)
			sm.RegisterBridge("agent-exit", bridge)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn, _, err := websocket.Dial(ctx, srv.URL+"/ws/agent/agent-exit", nil)
			if err != nil {
				t.Fatalf("dial: %v", err)
			}
			defer conn.CloseNow()

			// Read initial status.
			_, _, err = conn.Read(ctx)
			if err != nil {
				t.Fatalf("read initial status: %v", err)
			}

			// Close the done channel to simulate agent exit.
			close(done)

			// Read exit status.
			_, data, err := conn.Read(ctx)
			if err != nil {
				t.Fatalf("read exit status: %v", err)
			}
			var msg Message
			json.Unmarshal(data, &msg)
			if msg.Type != MsgStatus {
				t.Errorf("expected type=status, got %s", msg.Type)
			}
			if msg.Status != tc.expectedStatus {
				t.Errorf("expected status=%s, got %s", tc.expectedStatus, msg.Status)
			}
			if tc.expectExitCode && msg.ExitCode == nil {
				t.Error("expected exit_code to be set for completed status")
			}
			if !tc.expectExitCode && msg.ExitCode != nil {
				t.Errorf("expected no exit_code for %s status, got %d", tc.name, *msg.ExitCode)
			}
		})
	}
}

func TestHandlerAgentWS_BridgeDone_FallsBackToCompleted(t *testing.T) {
	// When agents finder is nil, should fall back to "completed" with exit code 0.
	t.Parallel()
	sm := NewSessionManager()
	h := NewHandler(sm, nil, nil) // nil agents
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	done := make(chan struct{})
	bridge := NewSDKBridge("agent-fb", func(text string) error { return nil }, done)
	sm.RegisterBridge("agent-fb", bridge)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, srv.URL+"/ws/agent/agent-fb", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	_, _, _ = conn.Read(ctx) // initial status
	close(done)

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var msg Message
	json.Unmarshal(data, &msg)
	if msg.Status != "completed" {
		t.Errorf("expected fallback status=completed, got %s", msg.Status)
	}
	if msg.ExitCode == nil || *msg.ExitCode != 0 {
		t.Errorf("expected exit_code=0, got %v", msg.ExitCode)
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
