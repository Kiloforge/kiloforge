package agent

import (
	"testing"
)

func TestParseStreamLine_ResultWithUsage(t *testing.T) {
	t.Parallel()

	line := `{"type":"result","subtype":"success","total_cost_usd":0.0342,"session_id":"abc123","usage":{"input_tokens":12500,"output_tokens":3200,"cache_read_input_tokens":8000,"cache_creation_input_tokens":1500}}`

	ev, err := ParseStreamLine(line)
	if err != nil {
		t.Fatalf("ParseStreamLine: %v", err)
	}

	if ev.Type != "result" {
		t.Errorf("Type: want %q, got %q", "result", ev.Type)
	}
	if ev.Subtype != "success" {
		t.Errorf("Subtype: want %q, got %q", "success", ev.Subtype)
	}
	if ev.CostUSD != 0.0342 {
		t.Errorf("CostUSD: want %f, got %f", 0.0342, ev.CostUSD)
	}
	if ev.Usage == nil {
		t.Fatal("Usage is nil")
	}
	if ev.Usage.InputTokens != 12500 {
		t.Errorf("InputTokens: want 12500, got %d", ev.Usage.InputTokens)
	}
	if ev.Usage.OutputTokens != 3200 {
		t.Errorf("OutputTokens: want 3200, got %d", ev.Usage.OutputTokens)
	}
	if ev.Usage.CacheReadTokens != 8000 {
		t.Errorf("CacheReadTokens: want 8000, got %d", ev.Usage.CacheReadTokens)
	}
	if ev.Usage.CacheCreationTokens != 1500 {
		t.Errorf("CacheCreationTokens: want 1500, got %d", ev.Usage.CacheCreationTokens)
	}
}

func TestParseStreamLine_ResultBudgetExceeded(t *testing.T) {
	t.Parallel()

	line := `{"type":"result","subtype":"error_max_budget_usd","total_cost_usd":5.00,"usage":{"input_tokens":50000,"output_tokens":10000}}`

	ev, err := ParseStreamLine(line)
	if err != nil {
		t.Fatalf("ParseStreamLine: %v", err)
	}

	if ev.Type != "result" {
		t.Errorf("Type: want %q, got %q", "result", ev.Type)
	}
	if ev.Subtype != "error_max_budget_usd" {
		t.Errorf("Subtype: want %q, got %q", "error_max_budget_usd", ev.Subtype)
	}
	if ev.CostUSD != 5.00 {
		t.Errorf("CostUSD: want %f, got %f", 5.00, ev.CostUSD)
	}
}

func TestParseStreamLine_MessageEvent(t *testing.T) {
	t.Parallel()

	line := `{"type":"message","role":"assistant","content":[{"type":"text","text":"Hello"}]}`

	ev, err := ParseStreamLine(line)
	if err != nil {
		t.Fatalf("ParseStreamLine: %v", err)
	}

	if ev.Type != "message" {
		t.Errorf("Type: want %q, got %q", "message", ev.Type)
	}
	if ev.Usage != nil {
		t.Error("Usage should be nil for message events")
	}
}

func TestParseStreamLine_ToolUseEvent(t *testing.T) {
	t.Parallel()

	line := `{"type":"tool_use","name":"Read","input":{"file_path":"/tmp/test.go"}}`

	ev, err := ParseStreamLine(line)
	if err != nil {
		t.Fatalf("ParseStreamLine: %v", err)
	}

	if ev.Type != "tool_use" {
		t.Errorf("Type: want %q, got %q", "tool_use", ev.Type)
	}
}

func TestParseStreamLine_InitEvent(t *testing.T) {
	t.Parallel()

	line := `{"type":"init","session_id":"sess-123","timestamp":"2026-03-07T12:00:00Z"}`

	ev, err := ParseStreamLine(line)
	if err != nil {
		t.Fatalf("ParseStreamLine: %v", err)
	}

	if ev.Type != "init" {
		t.Errorf("Type: want %q, got %q", "init", ev.Type)
	}
	if ev.SessionID != "sess-123" {
		t.Errorf("SessionID: want %q, got %q", "sess-123", ev.SessionID)
	}
}

func TestParseStreamLine_MalformedJSON(t *testing.T) {
	t.Parallel()

	lines := []string{
		"not json at all",
		"{malformed",
		"",
		"   ",
	}

	for _, line := range lines {
		_, err := ParseStreamLine(line)
		if err == nil {
			t.Errorf("expected error for %q", line)
		}
	}
}

func TestParseStreamLine_ResultWithoutUsage(t *testing.T) {
	t.Parallel()

	line := `{"type":"result","subtype":"error_during_execution"}`

	ev, err := ParseStreamLine(line)
	if err != nil {
		t.Fatalf("ParseStreamLine: %v", err)
	}

	if ev.Type != "result" {
		t.Errorf("Type: want %q, got %q", "result", ev.Type)
	}
	if ev.Usage != nil {
		t.Error("Usage should be nil when not present")
	}
}

func TestParseStreamLine_ResultWithZeroCost(t *testing.T) {
	t.Parallel()

	line := `{"type":"result","subtype":"success","total_cost_usd":0,"usage":{"input_tokens":0,"output_tokens":0}}`

	ev, err := ParseStreamLine(line)
	if err != nil {
		t.Fatalf("ParseStreamLine: %v", err)
	}

	if ev.CostUSD != 0 {
		t.Errorf("CostUSD: want 0, got %f", ev.CostUSD)
	}
}
