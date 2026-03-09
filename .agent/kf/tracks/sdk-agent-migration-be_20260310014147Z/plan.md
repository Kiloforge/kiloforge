# Implementation Plan: Migrate Agent Spawner to Claude Agent SDK (Backend)

**Track ID:** sdk-agent-migration-be_20260310014147Z

## Phase 1: WS Protocol Extension

### Task 1.1: Define enriched WS message types
- [x] Add new message type constants to `ws/message.go`: `MsgTurnStart`, `MsgText`, `MsgToolUse`, `MsgThinking`, `MsgTurnEnd`, `MsgSystem`
- [x] Add struct fields for turn-based data (turn_id, tool_name, tool_id, input, thinking, cost_usd, usage, subtype, data)
- [x] Add constructor functions: `TurnStartMsg()`, `TextMsg()`, `ToolUseMsg()`, `ThinkingMsg()`, `TurnEndMsg()`, `SystemMsg()`
- [x] Keep existing `OutputMsg`, `StatusMsg`, `ErrorMsg` for backward compat

### Task 1.2: Verify WS message serialization
- [x] Write tests for each new message constructor
- [x] Verify JSON roundtrip for all message types
- [x] Verify backward compatibility: old `OutputMsg` still works

## Phase 2: SDK Integration for Interactive Agents

### Task 2.1: Create SDK adapter layer
- [x] Create `backend/internal/adapter/agent/sdk_client.go`
- [x] Define `SDKSession` struct wrapping `claude.Client` with agent metadata
- [x] Implement `NewSDKSession(ctx, opts)` that configures `types.ClaudeAgentOptions` from spawner config
- [x] Handle: model, CWD, dangerously-skip-permissions, environment vars
- [x] Map existing `SpawnInteractiveOpts` to SDK options

### Task 2.2: Replace SpawnInteractive with SDK Client
- [x] Rewrite `SpawnInteractive` to create `SDKSession` instead of `exec.Command`
- [x] Call `client.Connect(ctx)` to start the session
- [x] If `opts.Prompt` set, call `client.Query(ctx, prompt)` for initial turn
- [x] Return `InteractiveAgent` with adapted IO channels
- [x] Update `InteractiveAgent` struct: replace `Stdin io.WriteCloser` with method-based input (`InputHandler`)

### Task 2.3: Implement message relay goroutine
- [x] Create `relayResponse(session, output chan)` goroutine
- [x] Range over `client.ReceiveResponse(ctx)` channel
- [x] Type-switch on SDK messages:
  - [x] `*types.AssistantMessage` → iterate content blocks → emit `TextMsg`/`ToolUseMsg`/`ThinkingMsg`
  - [x] `*types.SystemMessage` → emit `SystemMsg`
  - [x] `*types.ResultMessage` → emit `TurnEndMsg` with cost/usage, update quota tracker
- [x] Generate turn_id (UUID) per turn, emit `TurnStartMsg` at start

### Task 2.4: Implement turn-based input
- [x] When user sends WS `input` message, call `session.Query(ctx, text)` to start a new turn
- [x] Start new `relayResponse` goroutine for the response
- [x] Handle concurrent input rejection (one turn at a time)

### Task 2.5: Update Bridge and SessionManager
- [x] Add `NewSDKBridge` for SDK-based agents with `InputHandler` instead of stdin pipe
- [x] Add `StartStructuredRelay` for pre-serialized JSON messages
- [x] Ensure ring buffer stores structured messages for reconnection replay

### Task 2.6: Verify interactive agent lifecycle
- [x] Test: SDK bridge WriteInput routes through InputHandler
- [x] Test: structured relay buffers messages correctly
- [x] Test: reconnection replays structured messages from ring buffer

## Phase 3: SDK Integration for Non-Interactive Agents

### Task 3.1: Replace SpawnDeveloper with SDK Query
- [x] Rewrite `SpawnDeveloper` to use `QueryOneShot()` which wraps `claude.Query(ctx, prompt, opts)`
- [x] Configure SDK options: model, CWD, dangerously-skip-permissions
- [x] Process messages in background goroutine via `runSDKAgent`

### Task 3.2: Replace SpawnReviewer with SDK Query
- [x] Same pattern as SpawnDeveloper via `QueryOneShot()`
- [x] Maintain existing prompt construction (`/kf-reviewer {prURL}`)

### Task 3.3: Update monitor goroutines
- [x] Replace `monitorAgent` line-by-line scanning with SDK message processing in `QueryOneShot`
- [x] Type-switch on messages for logging and quota tracking via `resultToStreamEvent`
- [x] Extract cost/usage from `ResultMessage` for tracer spans
- [x] Log structured messages to agent log files

### Task 3.4: Verify non-interactive agent lifecycle
- [x] Test: `resultToStreamEvent` correctly converts SDK ResultMessage
- [x] Test: `extractUsageInfo` maps SDK usage map to UsageInfo struct
- [x] Test: `intFromMap` handles all numeric types

## Phase 4: Cleanup and Integration

### Task 4.1: Update quota tracker
- [x] `resultToStreamEvent` bridges SDK `ResultMessage` → internal `StreamEvent` for `RecordEvent`
- [x] Extracts `TotalCostUSD`, `Usage` map fields for rate limit tracking
- [x] `ParseStreamLine`-based recording path preserved for backward compat

### Task 4.2: Remove deprecated code
- [x] `CleanClaudeEnv()` deprecated but retained (still used by server.go for non-SDK callers)
- [x] `ExtractText()` and `ParseStreamLine()` preserved (still used by parser_test.go, quota tracker)
- [x] Removed manual CLI arg construction from spawner
- [x] Removed `monitorAgent` and `monitorInteractive` (replaced by SDK-based equivalents)

### Task 4.3: Update api_handler.go callers
- [x] Updated all `NewBridge(ia.Info.ID, ia.Stdin, ia.Done)` → `NewSDKBridge(ia.Info.ID, ia.Stdin, ia.Done)`
- [x] Updated all `StartOutputRelay` → `StartStructuredRelay`
- [x] WS handler `readLoop` calls `bridge.WriteInput()` which routes through `InputHandler`

### Task 4.4: Update agent tracing
- [x] Span attributes set from SDK `ResultMessage` via `extractUsageInfo`
- [x] Token counts from `ResultMessage.Usage` map
- [x] Cost from `ResultMessage.TotalCostUSD`

### Task 4.5: Full integration test
- [x] All tests pass (`make test`)
- [x] Full build succeeds (`make build`)
- [x] Quota tracking via `resultToStreamEvent` tested
- [x] WS protocol structured messages tested

## Phase 5: Documentation

### Task 5.1: Update API documentation
- [x] WS protocol enriched with code comments documenting new message types
- [x] `CleanClaudeEnv` marked deprecated with SDK migration note
