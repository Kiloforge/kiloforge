package ws

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nhooyr.io/websocket"
)

func TestHandlerAgentWS_NotFound(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()
	h := NewHandler(sm, nil)
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
	h := NewHandler(sm, nil)
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
	h := NewHandler(sm, nil)
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
