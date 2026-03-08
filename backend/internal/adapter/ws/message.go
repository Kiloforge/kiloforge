package ws

import "encoding/json"

// Message types for the WebSocket protocol.
const (
	MsgInput  = "input"  // client → server: user input text
	MsgOutput = "output" // server → client: agent output text
	MsgStatus = "status" // server → client: agent status change
	MsgError  = "error"  // server → client: error message
)

// Message is the WebSocket protocol envelope.
type Message struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Status   string `json:"status,omitempty"`
	Message  string `json:"message,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
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
