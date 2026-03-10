package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// UsageData holds token usage from a CC stream-json result event.
type UsageData struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	CacheReadTokens     int `json:"cache_read_input_tokens"`
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

// ExtractText extracts human-readable text from a CC stream-json line.
// Returns empty string if the line contains no displayable text.
func ExtractText(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return ""
	}

	var eventType string
	if err := json.Unmarshal(raw["type"], &eventType); err != nil {
		return ""
	}

	switch eventType {
	case "content_block_delta":
		var delta struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(raw["delta"], &delta); err == nil && delta.Type == "text_delta" {
			return delta.Text
		}
	case "assistant", "message":
		if text := extractContentText(raw["content"]); text != "" {
			return text
		}
		// Try nested message.content.
		var msg map[string]json.RawMessage
		if json.Unmarshal(raw["message"], &msg) == nil {
			return extractContentText(msg["content"])
		}
	}
	return ""
}

func extractContentText(data json.RawMessage) string {
	if data == nil {
		return ""
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &blocks); err != nil {
		return ""
	}
	var sb strings.Builder
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			sb.WriteString(b.Text)
		}
	}
	return sb.String()
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
