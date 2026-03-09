package ws

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"
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

func TestSDKBridge_WriteInput(t *testing.T) {
	t.Parallel()

	var received string
	handler := func(text string) error {
		received = text
		return nil
	}

	done := make(chan struct{})
	bridge := NewSDKBridge("test-sdk", handler, done)

	if err := bridge.WriteInput("hello SDK"); err != nil {
		t.Fatalf("WriteInput: %v", err)
	}
	if received != "hello SDK" {
		t.Errorf("got %q, want %q", received, "hello SDK")
	}
}

func TestStartOutputRelay(t *testing.T) {
	t.Parallel()

	sm := NewSessionManager()
	_, w := io.Pipe()
	defer w.Close()
	done := make(chan struct{})
	bridge := NewBridge("relay-agent", w, done)
	sm.RegisterBridge("relay-agent", bridge)

	output := make(chan []byte, 5)
	output <- []byte("line 1")
	output <- []byte("line 2")
	close(output)

	sm.StartOutputRelay(context.Background(), "relay-agent", output)

	// Ring buffer should contain both lines.
	lines := bridge.Buffer.Lines()
	if len(lines) != 2 {
		t.Fatalf("buffer lines = %d, want 2", len(lines))
	}
}

func TestStartStructuredRelay(t *testing.T) {
	t.Parallel()

	sm := NewSessionManager()
	done := make(chan struct{})
	bridge := NewSDKBridge("sdk-agent", nil, done)
	sm.RegisterBridge("sdk-agent", bridge)

	messages := make(chan []byte, 5)
	messages <- TurnStartMsg("turn-1")
	messages <- TextMsg("hello", "turn-1")
	messages <- TurnEndMsg("turn-1", 0.05, nil)
	close(messages)

	sm.StartStructuredRelay(context.Background(), "sdk-agent", messages)

	lines := bridge.Buffer.Lines()
	if len(lines) != 3 {
		t.Fatalf("buffer lines = %d, want 3", len(lines))
	}
}

func TestStartStructuredRelay_ContextCancel(t *testing.T) {
	t.Parallel()

	sm := NewSessionManager()
	done := make(chan struct{})
	bridge := NewSDKBridge("cancel-agent", nil, done)
	sm.RegisterBridge("cancel-agent", bridge)

	messages := make(chan []byte, 10)
	ctx, cancel := context.WithCancel(context.Background())

	// Send one message before cancel.
	messages <- TextMsg("before-cancel", "t1")

	relayDone := make(chan struct{})
	go func() {
		sm.StartStructuredRelay(ctx, "cancel-agent", messages)
		close(relayDone)
	}()

	// Give relay time to process the first message.
	// Then cancel context — relay should exit even though channel is still open.
	cancel()

	select {
	case <-relayDone:
		// Relay exited as expected.
	case <-time.After(2 * time.Second):
		t.Fatal("relay did not exit after context cancellation")
	}

	// Verify the channel is still open (not closed by relay).
	select {
	case messages <- TextMsg("after-cancel", "t1"):
		// Channel accepted write — it's still open. Good.
	default:
		t.Fatal("messages channel should still be open after relay cancel")
	}
}

func TestStartStructuredRelay_ReplacePrevious(t *testing.T) {
	t.Parallel()

	sm := NewSessionManager()
	done := make(chan struct{})
	bridge := NewSDKBridge("replace-agent", nil, done)
	sm.RegisterBridge("replace-agent", bridge)

	// Start first relay.
	messages1 := make(chan []byte, 10)
	ctx1, cancel1 := context.WithCancel(context.Background())
	relay1Done := make(chan struct{})
	go func() {
		sm.StartStructuredRelay(ctx1, "replace-agent", messages1)
		close(relay1Done)
	}()

	// Cancel first relay (simulating resume).
	cancel1()
	select {
	case <-relay1Done:
	case <-time.After(2 * time.Second):
		t.Fatal("first relay did not exit after cancel")
	}

	// Start second relay on new channel — should work without issues.
	messages2 := make(chan []byte, 10)
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	messages2 <- TextMsg("from-relay-2", "t2")
	close(messages2)

	sm.StartStructuredRelay(ctx2, "replace-agent", messages2)

	lines := bridge.Buffer.Lines()
	found := false
	for _, line := range lines {
		if string(line) != "" {
			found = true
		}
	}
	if !found {
		t.Error("second relay should have buffered messages")
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

func TestEnrichedMessage_Constructors(t *testing.T) {
	t.Parallel()

	// TurnStart
	ts := TurnStartMsg("turn-1")
	var tsMsg TurnStartMessage
	if err := json.Unmarshal(ts, &tsMsg); err != nil {
		t.Fatalf("unmarshal TurnStart: %v", err)
	}
	if tsMsg.Type != MsgTurnStart {
		t.Errorf("type = %q, want %q", tsMsg.Type, MsgTurnStart)
	}
	if tsMsg.TurnID != "turn-1" {
		t.Errorf("turn_id = %q, want %q", tsMsg.TurnID, "turn-1")
	}

	// Text
	txt := TextMsg("hello world", "turn-1")
	var txtMsg TextMessage
	if err := json.Unmarshal(txt, &txtMsg); err != nil {
		t.Fatalf("unmarshal Text: %v", err)
	}
	if txtMsg.Type != MsgText {
		t.Errorf("type = %q, want %q", txtMsg.Type, MsgText)
	}
	if txtMsg.Text != "hello world" {
		t.Errorf("text = %q, want %q", txtMsg.Text, "hello world")
	}

	// ToolUse
	tu := ToolUseMsg("Read", "toolu_123", "turn-1", map[string]string{"file": "test.go"})
	var tuMsg ToolUseMessage
	if err := json.Unmarshal(tu, &tuMsg); err != nil {
		t.Fatalf("unmarshal ToolUse: %v", err)
	}
	if tuMsg.Type != MsgToolUse {
		t.Errorf("type = %q, want %q", tuMsg.Type, MsgToolUse)
	}
	if tuMsg.ToolName != "Read" {
		t.Errorf("tool_name = %q, want %q", tuMsg.ToolName, "Read")
	}

	// Thinking
	th := ThinkingMsg("I am thinking...", "turn-1")
	var thMsg ThinkingMessage
	if err := json.Unmarshal(th, &thMsg); err != nil {
		t.Fatalf("unmarshal Thinking: %v", err)
	}
	if thMsg.Type != MsgThinking {
		t.Errorf("type = %q, want %q", thMsg.Type, MsgThinking)
	}

	// TurnEnd
	usage := &UsageInfo{InputTokens: 100, OutputTokens: 50}
	te := TurnEndMsg("turn-1", 0.034, usage)
	var teMsg TurnEndMessage
	if err := json.Unmarshal(te, &teMsg); err != nil {
		t.Fatalf("unmarshal TurnEnd: %v", err)
	}
	if teMsg.Type != MsgTurnEnd {
		t.Errorf("type = %q, want %q", teMsg.Type, MsgTurnEnd)
	}
	if teMsg.CostUSD != 0.034 {
		t.Errorf("cost_usd = %f, want %f", teMsg.CostUSD, 0.034)
	}
	if teMsg.Usage == nil {
		t.Fatal("usage is nil")
	}
	if teMsg.Usage.InputTokens != 100 {
		t.Errorf("input_tokens = %d, want 100", teMsg.Usage.InputTokens)
	}

	// System
	sys := SystemMsg("init", map[string]string{"version": "1.0"})
	var sysMsg SystemNotification
	if err := json.Unmarshal(sys, &sysMsg); err != nil {
		t.Fatalf("unmarshal System: %v", err)
	}
	if sysMsg.Type != MsgSystem {
		t.Errorf("type = %q, want %q", sysMsg.Type, MsgSystem)
	}
	if sysMsg.Subtype != "init" {
		t.Errorf("subtype = %q, want %q", sysMsg.Subtype, "init")
	}
}

func TestOutputMsg_BackwardCompat(t *testing.T) {
	t.Parallel()

	msg := OutputMsg("legacy text")
	var m Message
	if err := json.Unmarshal(msg, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.Type != MsgOutput {
		t.Errorf("type = %q, want %q", m.Type, MsgOutput)
	}
	if m.Text != "legacy text" {
		t.Errorf("text = %q, want %q", m.Text, "legacy text")
	}
}
