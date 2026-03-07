package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// UsageData holds token usage from a CC stream-json result event.
type UsageData struct {
	InputTokens       int `json:"input_tokens"`
	OutputTokens      int `json:"output_tokens"`
	CacheReadTokens   int `json:"cache_read_input_tokens"`
	CacheCreationTokens int `json:"cache_creation_input_tokens"`
}

// StreamEvent represents a parsed CC stream-json event.
type StreamEvent struct {
	Type      string     `json:"type"`
	Subtype   string     `json:"subtype,omitempty"`
	SessionID string     `json:"session_id,omitempty"`
	CostUSD   float64    `json:"total_cost_usd,omitempty"`
	Usage     *UsageData `json:"usage,omitempty"`
}

// ParseStreamLine parses a single line of CC stream-json output.
// Returns an error for malformed or empty input.
func ParseStreamLine(line string) (StreamEvent, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return StreamEvent{}, fmt.Errorf("empty line")
	}

	var ev StreamEvent
	if err := json.Unmarshal([]byte(line), &ev); err != nil {
		return StreamEvent{}, fmt.Errorf("parse stream-json: %w", err)
	}

	// Normalize: if usage has all zeros from a non-result event, nil it out.
	if ev.Usage != nil && ev.Type != "result" {
		ev.Usage = nil
	}

	return ev, nil
}
