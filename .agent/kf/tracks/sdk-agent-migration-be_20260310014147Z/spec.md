# Specification: Migrate Agent Spawner to Claude Agent SDK (Backend)

**Track ID:** sdk-agent-migration-be_20260310014147Z
**Type:** Refactor
**Created:** 2026-03-10T01:41:47Z
**Status:** Draft

## Summary

Replace the raw `exec.Command("claude")` + manual stream-json parsing in the agent spawner with the unofficial Go Claude Agent SDK (`github.com/schlunsen/claude-agent-sdk-go` v0.5.1), and enrich the WebSocket protocol with structured turn-based messages.

## Context

The current agent spawner manually constructs `exec.Command` invocations, pipes stdin/stdout, and scans stdout line-by-line to parse stream-json events. The parser (`parser.go`) only extracts raw text from `content_block_delta` and `assistant`/`message` events, ignoring tool use, thinking blocks, and system events. The WebSocket protocol is flat — only `output` (raw text), `status`, and `error` message types. This means the dashboard cannot display structured agent activity (tool calls, thinking, turn boundaries).

The Claude Agent SDK provides:
- `Client` with `Connect/Query/ReceiveResponse` lifecycle for interactive sessions
- `Query()` function for one-shot (non-interactive) agents
- Typed messages: `AssistantMessage` (with `TextBlock`, `ToolUseBlock`, `ThinkingBlock`), `SystemMessage`, `ResultMessage` (with cost/usage/session data), `StreamEvent`
- Subprocess transport management, CLI discovery, environment cleaning
- Session resumption via `WithResume(sessionID)`

## Codebase Analysis

- **`backend/internal/adapter/agent/spawner.go`** (~570 lines): Contains `SpawnReviewer`, `SpawnDeveloper`, `SpawnInteractive` — all build CLI args manually, create `exec.Command`, set up pipes, and monitor via goroutines. `CleanClaudeEnv()` duplicates SDK functionality. `monitorAgent` and `monitorInteractive` do line-by-line scanning.
- **`backend/internal/adapter/agent/parser.go`** (~105 lines): `ExtractText()` only handles `content_block_delta` and `assistant`/`message` types. `ParseStreamLine()` extracts basic stream event metadata. These are replaced by SDK's `UnmarshalMessage` and typed content blocks.
- **`backend/internal/adapter/ws/message.go`** (~38 lines): Flat protocol with 4 message types (`input`, `output`, `status`, `error`). Needs enrichment with structured message types.
- **`backend/internal/adapter/ws/session.go`** (~143 lines): `StartOutputRelay` broadcasts raw `[]byte` text. Needs to broadcast structured JSON messages.
- **`backend/internal/adapter/rest/api_handler.go`**: Calls `SpawnInteractive` — interface change needed if return type changes.
- **`backend/internal/adapter/agent/tracker.go`**: Quota tracker uses `StreamEvent` from parser — needs to work with SDK `ResultMessage` for cost/usage.

## Acceptance Criteria

- [ ] Interactive agents use SDK `Client` (Connect/Query/ReceiveResponse pattern)
- [ ] Non-interactive agents (developer, reviewer) use SDK `Query()` function
- [ ] `CleanClaudeEnv()` removed — SDK handles environment cleaning
- [ ] SDK typed messages (AssistantMessage, SystemMessage, ResultMessage) parsed and forwarded
- [ ] WS protocol extended with message types: `turn_start`, `text`, `tool_use`, `thinking`, `turn_end`, `system`
- [ ] `turn_end` messages include cost/usage data from `ResultMessage`
- [ ] Quota tracker updated to extract usage from SDK `ResultMessage`
- [ ] Agent tracing spans updated with SDK message metadata
- [ ] All existing tests pass or are updated for new interfaces
- [ ] `parser.go` ExtractText/ParseStreamLine removed or deprecated (replaced by SDK)
- [ ] Interactive agent stdin input triggers new SDK `Query()` call (turn-based)

## Dependencies

None — SDK is already in go.mod (v0.5.1).

## Out of Scope

- Frontend changes (separate track: `sdk-agent-migration-fe_20260310014148Z`)
- SDK library modifications or forking
- Changes to non-agent REST endpoints
- Agent persistence/recovery changes

## Technical Notes

### Migration Strategy

**Interactive agents (SpawnInteractive):**
1. Create SDK `Client` with `types.NewClaudeAgentOptions()` configured from existing opts
2. Call `client.Connect(ctx)` to start the subprocess
3. If initial prompt provided, call `client.Query(ctx, prompt)`
4. Range over `client.ReceiveResponse(ctx)` — type-switch on messages
5. For each message, serialize to enriched WS protocol and broadcast
6. When user sends input via WS, call `client.Query(ctx, userInput)` for next turn
7. `ResultMessage` marks turn end — broadcast `turn_end` with cost data

**Non-interactive agents (SpawnDeveloper, SpawnReviewer):**
1. Use `claude.Query(ctx, prompt, opts)` one-shot function
2. Range over returned channel for typed messages
3. Log messages, track quota from `ResultMessage`
4. No WS broadcast needed (these run headless)

### WS Protocol Extension

New message types (all server → client):
```json
{"type": "turn_start", "turn_id": "uuid"}
{"type": "text", "text": "...", "turn_id": "uuid"}
{"type": "tool_use", "tool_name": "Read", "tool_id": "toolu_xxx", "input": {...}, "turn_id": "uuid"}
{"type": "thinking", "thinking": "...", "turn_id": "uuid"}
{"type": "turn_end", "turn_id": "uuid", "cost_usd": 0.034, "usage": {...}}
{"type": "system", "subtype": "init|warning|error", "data": {...}}
```

Existing types preserved for backward compatibility:
```json
{"type": "input", "text": "..."}     // client → server (unchanged)
{"type": "status", "status": "..."}  // server → client (unchanged)
{"type": "error", "message": "..."}  // server → client (unchanged)
```

### Quota Tracker Integration

Replace `ParseStreamLine` → `StreamEvent` with SDK `ResultMessage`:
- `ResultMessage.TotalCostUSD` → rate limit tracking
- `ResultMessage.Usage` map → token counting
- `ResultMessage.SessionID` → session correlation

---

_Generated by kf-architect from prompt: "Migrate agent spawner from raw exec.Command to Claude Agent SDK"_
