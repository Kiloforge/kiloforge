package ws

import (
	"encoding/json"
	"testing"
)

func TestInputEchoMsg(t *testing.T) {
	t.Parallel()

	raw := InputEchoMsg("hello world")

	var msg Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Type != MsgInputEcho {
		t.Errorf("type = %q, want %q", msg.Type, MsgInputEcho)
	}
	if msg.Text != "hello world" {
		t.Errorf("text = %q, want %q", msg.Text, "hello world")
	}
}
