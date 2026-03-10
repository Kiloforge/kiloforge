package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	claude "github.com/schlunsen/claude-agent-sdk-go"
	"github.com/schlunsen/claude-agent-sdk-go/types"

	"kiloforge/internal/adapter/ws"
	"kiloforge/internal/core/port"

	"github.com/google/uuid"
)

// defaultResponseTimeout is the maximum time to wait for the first response
// message after sending a query. If no message arrives within this window,
// the session is closed to avoid permanently stuck turns.
const defaultResponseTimeout = 2 * time.Minute

// sdkClientAPI abstracts the Claude SDK Client methods used by SDKSession.
// This enables testing without a real CLI process.
type sdkClientAPI interface {
	Query(ctx context.Context, prompt string) error
	ReceiveResponse(ctx context.Context) <-chan types.Message
	IsConnected() bool
	Close(ctx context.Context) error
	Connect(ctx context.Context) error
}

// SDKSession wraps a claude.Client for interactive agent sessions.
type SDKSession struct {
	client  sdkClientAPI
	ctx     context.Context
	cancel  context.CancelFunc
	output  chan []byte // structured messages for WS relay
	done    chan struct{}
	logFile *os.File

	responseTimeout time.Duration // timeout for waiting on response messages; 0 = default

	mu                sync.Mutex
	querying          bool               // prevents concurrent turns
	queryCancel       context.CancelFunc // cancels the current turn's relay context
	onTurnEnd         func()             // called after each turn completes (e.g., to drain queued input)
	sessionIDCallback func(string)       // called when the real Claude session ID is received
	closeOnce         sync.Once
}

// NewSDKSession creates an SDK client configured for an interactive agent.
// The ctx parameter controls the session's process lifetime — pass context.Background()
// to decouple from any HTTP request context. The session has its own cancel func
// (called by Close) for explicit shutdown.
func NewSDKSession(ctx context.Context, workDir, model, logFilePath string, envVars map[string]string) (*SDKSession, error) {
	opts := types.NewClaudeAgentOptions().
		WithCWD(workDir).
		WithDangerouslySkipPermissions(true).
		WithAllowDangerouslySkipPermissions(true).
		WithVerbose(true)

	if model != "" {
		opts = opts.WithModel(model)
	}

	if logFilePath != "" {
		opts = opts.WithCustomStderrLogFile(logFilePath)
	}

	for k, v := range envVars {
		opts = opts.WithEnvVar(k, v)
	}

	client, err := claude.NewClient(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("create SDK client: %w", err)
	}

	sessionCtx, cancel := context.WithCancel(ctx)

	return &SDKSession{
		client:  client,
		ctx:     sessionCtx,
		cancel:  cancel,
		output:  make(chan []byte, 100),
		done:    make(chan struct{}),
		logFile: nil, // set externally if needed
	}, nil
}

// NewSDKSessionWithResume creates an SDK session that resumes a previous session.
// The ctx parameter controls the session's process lifetime — pass context.Background()
// to decouple from any HTTP request context.
func NewSDKSessionWithResume(ctx context.Context, workDir, model, logFilePath, sessionID string, envVars map[string]string) (*SDKSession, error) {
	opts := types.NewClaudeAgentOptions().
		WithCWD(workDir).
		WithDangerouslySkipPermissions(true).
		WithAllowDangerouslySkipPermissions(true).
		WithVerbose(true).
		WithResume(sessionID)

	if model != "" {
		opts = opts.WithModel(model)
	}

	if logFilePath != "" {
		opts = opts.WithCustomStderrLogFile(logFilePath)
	}

	for k, v := range envVars {
		opts = opts.WithEnvVar(k, v)
	}

	client, err := claude.NewClient(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("create SDK client: %w", err)
	}

	sessionCtx, cancel := context.WithCancel(ctx)

	return &SDKSession{
		client: client,
		ctx:    sessionCtx,
		cancel: cancel,
		output: make(chan []byte, 100),
		done:   make(chan struct{}),
	}, nil
}

// Connect establishes the SDK connection to the Claude CLI.
func (s *SDKSession) Connect(ctx context.Context) error {
	return s.client.Connect(ctx)
}

// Query sends a prompt and relays structured messages to the output channel.
// Returns immediately; messages are sent asynchronously.
func (s *SDKSession) Query(ctx context.Context, prompt string, tracker *QuotaTracker, agentID string, span port.SpanEnder) error {
	s.mu.Lock()
	if s.querying {
		s.mu.Unlock()
		return fmt.Errorf("turn already in progress")
	}
	s.querying = true
	// Create per-query context so Interrupt() can cancel just this turn.
	queryCtx, queryCancel := context.WithCancel(s.ctx)
	s.queryCancel = queryCancel
	s.mu.Unlock()

	// Check client is still connected before attempting the query.
	if s.client == nil || !s.client.IsConnected() {
		s.mu.Lock()
		s.querying = false
		s.queryCancel = nil
		queryCancel()
		s.mu.Unlock()
		return fmt.Errorf("client disconnected")
	}

	if err := s.client.Query(ctx, prompt); err != nil {
		s.mu.Lock()
		s.querying = false
		s.queryCancel = nil
		queryCancel()
		s.mu.Unlock()
		return fmt.Errorf("send query: %w", err)
	}

	go s.relayResponse(queryCtx, tracker, agentID, span)
	return nil
}

// SetOnTurnEnd sets a callback invoked after each turn completes (relayResponse exits).
// Used to wire input queue draining from the Bridge.
func (s *SDKSession) SetOnTurnEnd(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onTurnEnd = fn
}

// SetSessionIDCallback sets a callback invoked when the real Claude SDK session ID
// is received in a ResultMessage. Used to persist the real session ID for resume.
func (s *SDKSession) SetSessionIDCallback(fn func(string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionIDCallback = fn
}

// Interrupt cancels the current turn's relay context if a turn is in progress.
// If no turn is active, this is a no-op.
func (s *SDKSession) Interrupt() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.querying && s.queryCancel != nil {
		s.queryCancel()
	}
}

// relayResponse reads SDK messages and forwards them as structured WS messages.
// It applies a timeout for the initial response — if no message arrives within
// the timeout, it emits an error and closes the session.
// After the response channel closes, if the SDK client has disconnected (process
// exited), it closes the session to unblock monitorSDKSession.
func (s *SDKSession) relayResponse(ctx context.Context, tracker *QuotaTracker, agentID string, span port.SpanEnder) {
	defer func() {
		s.mu.Lock()
		s.querying = false
		s.queryCancel = nil
		cb := s.onTurnEnd
		s.mu.Unlock()
		if cb != nil {
			cb()
		}
	}()

	turnID := uuid.New().String()

	// Emit turn_start.
	s.emit(ws.TurnStartMsg(turnID))

	timeout := s.responseTimeout
	if timeout == 0 {
		timeout = defaultResponseTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	responseCh := s.client.ReceiveResponse(ctx)
	gotMessage := false

	for {
		select {
		case msg, ok := <-responseCh:
			if !ok {
				// Response channel closed — turn is done.
				// If the SDK client has disconnected (CLI process exited),
				// close the session to unblock monitorSDKSession.
				if s.client != nil && !s.client.IsConnected() {
					s.logLine("[relay] client disconnected after response ended — closing session")
					s.Close()
				}
				return
			}

			gotMessage = true
			// Reset timer on each message — timeout only applies to gaps.
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(timeout)

			s.handleResponseMessage(msg, turnID, tracker, agentID, span)

		case <-timer.C:
			// No message received within timeout.
			s.logLine(fmt.Sprintf("[relay] response timeout after %s (got_message=%v)", timeout, gotMessage))
			s.emit(ws.ErrorMsg("agent not responding — no output received"))
			s.Close()
			return

		case <-ctx.Done():
			s.emit(ws.TurnEndInterruptedMsg(turnID))
			return
		}
	}
}

// handleResponseMessage processes a single SDK response message.
func (s *SDKSession) handleResponseMessage(msg types.Message, turnID string, tracker *QuotaTracker, agentID string, span port.SpanEnder) {
	switch m := msg.(type) {
	case *types.AssistantMessage:
		for _, block := range m.Content {
			switch b := block.(type) {
			case *types.TextBlock:
				s.emit(ws.TextMsg(b.Text, turnID))
				s.logLine(fmt.Sprintf("[text] %s", b.Text))
			case *types.ToolUseBlock:
				s.emit(ws.ToolUseMsg(b.Name, b.ID, turnID, b.Input))
				s.logLine(fmt.Sprintf("[tool_use] %s (id=%s)", b.Name, b.ID))
			case *types.ToolResultBlock:
				content := normalizeToolResultContent(b.Content)
				isError := b.IsError != nil && *b.IsError
				s.emit(ws.ToolResultMsg(b.ToolUseID, content, turnID, isError))
				if isError {
					s.logLine(fmt.Sprintf("[tool_result] ERROR id=%s", b.ToolUseID))
				} else {
					s.logLine(fmt.Sprintf("[tool_result] id=%s len=%d", b.ToolUseID, len(content)))
				}
			case *types.ThinkingBlock:
				s.emit(ws.ThinkingMsg(b.Thinking, turnID))
				s.logLine("[thinking] ...")
			}
		}

	case *types.SystemMessage:
		s.emit(ws.SystemMsg(m.Subtype, m.Data))
		s.logLine(fmt.Sprintf("[system] subtype=%s", m.Subtype))

	case *types.ResultMessage:
		var costUSD float64
		if m.TotalCostUSD != nil {
			costUSD = *m.TotalCostUSD
		}

		usage := extractUsageInfo(m.Usage)
		s.emit(ws.TurnEndMsg(turnID, costUSD, usage))

		if tracker != nil {
			ev := resultToStreamEvent(m)
			tracker.RecordEvent(agentID, ev)
		}

		if span != nil && usage != nil {
			span.SetAttributes(
				port.IntAttr("tokens.input", usage.InputTokens),
				port.IntAttr("tokens.output", usage.OutputTokens),
				port.IntAttr("tokens.cache_read", usage.CacheReadTokens),
				port.IntAttr("tokens.cache_create", usage.CacheCreationTokens),
				port.Float64Attr("cost.usd", costUSD),
			)
		}

		s.logLine(fmt.Sprintf("[result] cost=$%.4f session=%s", costUSD, m.SessionID))

		// Invoke session ID callback if a real session ID was received.
		if m.SessionID != "" {
			s.mu.Lock()
			cb := s.sessionIDCallback
			s.mu.Unlock()
			if cb != nil {
				cb(m.SessionID)
			}
		}
	}
}

// normalizeToolResultContent converts the SDK's ToolResultBlock.Content
// (which can be a string or []map[string]interface{}) into a plain string.
func normalizeToolResultContent(content interface{}) string {
	if content == nil {
		return ""
	}
	if s, ok := content.(string); ok {
		return s
	}
	// Content can be an array of content blocks (e.g., [{type: "text", text: "..."}]).
	if arr, ok := content.([]interface{}); ok {
		var parts []string
		for _, item := range arr {
			if m, ok := item.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	// Fallback: JSON-encode whatever it is.
	b, err := json.Marshal(content)
	if err != nil {
		return fmt.Sprintf("%v", content)
	}
	return string(b)
}

// Close terminates the SDK session. It is safe to call multiple times
// concurrently — only the first call performs cleanup.
func (s *SDKSession) Close() {
	s.closeOnce.Do(func() {
		s.cancel()
		if s.client != nil {
			_ = s.client.Close(context.Background())
		}
		close(s.output)
		close(s.done)
		if s.logFile != nil {
			s.logFile.Close()
		}
	})
}

// Output returns the channel of structured WS messages.
func (s *SDKSession) Output() <-chan []byte {
	return s.output
}

// Done returns a channel closed when the session ends.
func (s *SDKSession) Done() <-chan struct{} {
	return s.done
}

// SetLogFile sets the log file for the session.
func (s *SDKSession) SetLogFile(f *os.File) {
	s.logFile = f
}

func (s *SDKSession) emit(msg []byte) {
	select {
	case s.output <- msg:
	default:
		// Drop if channel is full to avoid blocking.
	}
}

func (s *SDKSession) logLine(line string) {
	if s.logFile != nil {
		fmt.Fprintln(s.logFile, line)
	}
}

// extractUsageInfo converts the SDK Usage map to our UsageInfo struct.
func extractUsageInfo(usage map[string]interface{}) *ws.UsageInfo {
	if usage == nil {
		return nil
	}
	return &ws.UsageInfo{
		InputTokens:         intFromMap(usage, "input_tokens"),
		OutputTokens:        intFromMap(usage, "output_tokens"),
		CacheReadTokens:     intFromMap(usage, "cache_read_input_tokens"),
		CacheCreationTokens: intFromMap(usage, "cache_creation_input_tokens"),
	}
}

func intFromMap(m map[string]interface{}, key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case int:
		return n
	default:
		return 0
	}
}

// resultToStreamEvent converts an SDK ResultMessage to our internal StreamEvent
// for quota tracker compatibility.
func resultToStreamEvent(m *types.ResultMessage) StreamEvent {
	ev := StreamEvent{
		Type:      "result",
		Subtype:   m.Subtype,
		SessionID: m.SessionID,
	}
	if m.TotalCostUSD != nil {
		ev.CostUSD = *m.TotalCostUSD
	}
	if m.Usage != nil {
		ev.Usage = &UsageData{
			InputTokens:         intFromMap(m.Usage, "input_tokens"),
			OutputTokens:        intFromMap(m.Usage, "output_tokens"),
			CacheReadTokens:     intFromMap(m.Usage, "cache_read_input_tokens"),
			CacheCreationTokens: intFromMap(m.Usage, "cache_creation_input_tokens"),
		}
	}
	return ev
}

// QueryOneShot executes a one-shot SDK query (for non-interactive agents)
// and processes messages until completion. Returns the final status.
func QueryOneShot(ctx context.Context, prompt, workDir, model, logFilePath string,
	tracker *QuotaTracker, agentID string, span port.SpanEnder, envVars map[string]string) (string, error) {

	opts := types.NewClaudeAgentOptions().
		WithCWD(workDir).
		WithDangerouslySkipPermissions(true).
		WithAllowDangerouslySkipPermissions(true).
		WithVerbose(true)

	if model != "" {
		opts = opts.WithModel(model)
	}

	if logFilePath != "" {
		opts = opts.WithCustomStderrLogFile(logFilePath)
	}

	for k, v := range envVars {
		opts = opts.WithEnvVar(k, v)
	}

	messages, err := claude.Query(ctx, prompt, opts)
	if err != nil {
		return "failed", fmt.Errorf("SDK query: %w", err)
	}

	var lf *os.File
	if logFilePath != "" {
		lf, _ = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if lf != nil {
			defer lf.Close()
		}
	}

	for msg := range messages {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			for _, block := range m.Content {
				switch b := block.(type) {
				case *types.TextBlock:
					if lf != nil {
						fmt.Fprintf(lf, "[text] %s\n", b.Text)
					}
				case *types.ToolUseBlock:
					if lf != nil {
						fmt.Fprintf(lf, "[tool_use] %s (id=%s)\n", b.Name, b.ID)
					}
				}
			}

		case *types.ResultMessage:
			if tracker != nil {
				ev := resultToStreamEvent(m)
				tracker.RecordEvent(agentID, ev)
			}

			if span != nil {
				var costUSD float64
				if m.TotalCostUSD != nil {
					costUSD = *m.TotalCostUSD
				}
				usage := extractUsageInfo(m.Usage)
				if usage != nil {
					span.SetAttributes(
						port.IntAttr("tokens.input", usage.InputTokens),
						port.IntAttr("tokens.output", usage.OutputTokens),
						port.IntAttr("tokens.cache_read", usage.CacheReadTokens),
						port.IntAttr("tokens.cache_create", usage.CacheCreationTokens),
						port.Float64Attr("cost.usd", costUSD),
					)
				}
			}

			if lf != nil {
				var costUSD float64
				if m.TotalCostUSD != nil {
					costUSD = *m.TotalCostUSD
				}
				fmt.Fprintf(lf, "[result] cost=$%.4f session=%s\n", costUSD, m.SessionID)
			}

		case *types.SystemMessage:
			if lf != nil {
				fmt.Fprintf(lf, "[system] subtype=%s\n", m.Subtype)
			}
		}
	}

	return "completed", nil
}
