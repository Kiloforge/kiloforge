package ws

import "encoding/json"

// Message types for the WebSocket protocol.
const (
	MsgInput  = "input"  // client → server: user input text
	MsgOutput = "output" // server → client: agent output text
	MsgStatus = "status" // server → client: agent status change
	MsgError  = "error"  // server → client: error message

	MsgInterrupt = "interrupt" // client → server: interrupt current turn

	// Enriched message types for SDK-based agents.
	MsgTurnStart  = "turn_start"  // server → client: new turn begins
	MsgText       = "text"        // server → client: text content block
	MsgToolUse    = "tool_use"    // server → client: tool invocation
	MsgToolResult = "tool_result" // server → client: tool execution result
	MsgThinking   = "thinking"    // server → client: thinking content
	MsgTurnEnd    = "turn_end"    // server → client: turn completed with cost/usage
	MsgSystem     = "system"      // server → client: system notification
)

// Message is the WebSocket protocol envelope.
type Message struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Status   string `json:"status,omitempty"`
	Message  string `json:"message,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
}

// UsageInfo holds token usage data for turn_end messages.
type UsageInfo struct {
	InputTokens         int `json:"input_tokens,omitempty"`
	OutputTokens        int `json:"output_tokens,omitempty"`
	CacheReadTokens     int `json:"cache_read_tokens,omitempty"`
	CacheCreationTokens int `json:"cache_creation_tokens,omitempty"`
}

// TurnStartMessage is sent when a new turn begins.
type TurnStartMessage struct {
	Type   string `json:"type"`
	TurnID string `json:"turn_id"`
}

// TextMessage is sent for text content blocks.
type TextMessage struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	TurnID string `json:"turn_id"`
}

// ToolUseMessage is sent for tool invocations.
type ToolUseMessage struct {
	Type     string      `json:"type"`
	ToolName string      `json:"tool_name"`
	ToolID   string      `json:"tool_id"`
	Input    interface{} `json:"input,omitempty"`
	TurnID   string      `json:"turn_id"`
}

// ToolResultMessage is sent for tool execution results.
type ToolResultMessage struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error"`
	TurnID    string `json:"turn_id"`
}

// ThinkingMessage is sent for thinking content.
type ThinkingMessage struct {
	Type     string `json:"type"`
	Thinking string `json:"thinking"`
	TurnID   string `json:"turn_id"`
}

// TurnEndMessage is sent when a turn completes.
type TurnEndMessage struct {
	Type        string     `json:"type"`
	TurnID      string     `json:"turn_id"`
	CostUSD     float64    `json:"cost_usd,omitempty"`
	Usage       *UsageInfo `json:"usage,omitempty"`
	Interrupted bool       `json:"interrupted,omitempty"`
}

// SystemNotification is sent for system events.
type SystemNotification struct {
	Type    string      `json:"type"`
	Subtype string      `json:"subtype"`
	Data    interface{} `json:"data,omitempty"`
}

// OutputMsg creates an output message.
func OutputMsg(text string) []byte {
	b, _ := json.Marshal(Message{Type: MsgOutput, Text: text})
	return b
}

// StatusMsg creates a status message.
func StatusMsg(status string, exitCode *int) []byte {
	b, _ := json.Marshal(Message{Type: MsgStatus, Status: status, ExitCode: exitCode})
	return b
}

// ErrorMsg creates an error message.
func ErrorMsg(msg string) []byte {
	b, _ := json.Marshal(Message{Type: MsgError, Message: msg})
	return b
}

// TurnStartMsg creates a turn_start message.
func TurnStartMsg(turnID string) []byte {
	b, _ := json.Marshal(TurnStartMessage{Type: MsgTurnStart, TurnID: turnID})
	return b
}

// TextMsg creates a text content message.
func TextMsg(text, turnID string) []byte {
	b, _ := json.Marshal(TextMessage{Type: MsgText, Text: text, TurnID: turnID})
	return b
}

// ToolUseMsg creates a tool_use message.
func ToolUseMsg(toolName, toolID, turnID string, input interface{}) []byte {
	b, _ := json.Marshal(ToolUseMessage{
		Type:     MsgToolUse,
		ToolName: toolName,
		ToolID:   toolID,
		Input:    input,
		TurnID:   turnID,
	})
	return b
}

// ToolResultMsg creates a tool_result message.
func ToolResultMsg(toolUseID, content, turnID string, isError bool) []byte {
	b, _ := json.Marshal(ToolResultMessage{
		Type:      MsgToolResult,
		ToolUseID: toolUseID,
		Content:   content,
		IsError:   isError,
		TurnID:    turnID,
	})
	return b
}

// ThinkingMsg creates a thinking message.
func ThinkingMsg(thinking, turnID string) []byte {
	b, _ := json.Marshal(ThinkingMessage{Type: MsgThinking, Thinking: thinking, TurnID: turnID})
	return b
}

// TurnEndMsg creates a turn_end message with cost/usage data.
func TurnEndMsg(turnID string, costUSD float64, usage *UsageInfo) []byte {
	b, _ := json.Marshal(TurnEndMessage{
		Type:    MsgTurnEnd,
		TurnID:  turnID,
		CostUSD: costUSD,
		Usage:   usage,
	})
	return b
}

// TurnEndInterruptedMsg creates a turn_end message indicating the turn was interrupted.
func TurnEndInterruptedMsg(turnID string) []byte {
	b, _ := json.Marshal(TurnEndMessage{
		Type:        MsgTurnEnd,
		TurnID:      turnID,
		Interrupted: true,
	})
	return b
}

// SystemMsg creates a system notification message.
func SystemMsg(subtype string, data interface{}) []byte {
	b, _ := json.Marshal(SystemNotification{Type: MsgSystem, Subtype: subtype, Data: data})
	return b
}
