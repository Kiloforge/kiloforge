package ws

import (
	"io"
	"testing"
)

func TestSessionManager_AddRemove(t *testing.T) {
	t.Parallel()
	sm := NewSessionManager()

	// No bridge registered — GetBridge returns false.
	_, ok := sm.GetBridge("agent-1")
	if ok {
		t.Error("expected no bridge")
	}

	// Register a bridge.
	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()
	done := make(chan struct{})
	bridge := NewBridge("agent-1", w, done)
	sm.RegisterBridge("agent-1", bridge)

	b, ok := sm.GetBridge("agent-1")
	if !ok {
		t.Fatal("expected bridge")
	}
	if b.AgentID != "agent-1" {
		t.Errorf("bridge agent = %q, want %q", b.AgentID, "agent-1")
	}

	// Unregister.
	sm.UnregisterBridge("agent-1")
	_, ok = sm.GetBridge("agent-1")
	if ok {
		t.Error("expected no bridge after unregister")
	}
}

func TestBridge_WriteInput(t *testing.T) {
	t.Parallel()
	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()

	done := make(chan struct{})
	bridge := NewBridge("test", w, done)

	go func() {
		_ = bridge.WriteInput("hello")
	}()

	buf := make([]byte, 64)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(buf[:n]) != "hello\n" {
		t.Errorf("got %q, want %q", string(buf[:n]), "hello\n")
	}
}

func TestMessage_Constructors(t *testing.T) {
	t.Parallel()

	out := OutputMsg("hello world")
	if len(out) == 0 {
		t.Error("empty output message")
	}

	status := StatusMsg("running", nil)
	if len(status) == 0 {
		t.Error("empty status message")
	}

	errMsg := ErrorMsg("something broke")
	if len(errMsg) == 0 {
		t.Error("empty error message")
	}
}
