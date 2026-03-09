# Implementation Plan: Migrate Agent Spawner to Claude Agent SDK (Backend)

**Track ID:** sdk-agent-migration-be_20260310014147Z

## Phase 1: WS Protocol Extension

### Task 1.1: Define enriched WS message types
- Add new message type constants to `ws/message.go`: `MsgTurnStart`, `MsgText`, `MsgToolUse`, `MsgThinking`, `MsgTurnEnd`, `MsgSystem`
- Add struct fields for turn-based data (turn_id, tool_name, tool_id, input, thinking, cost_usd, usage, subtype, data)
- Add constructor functions: `TurnStartMsg()`, `TextMsg()`, `ToolUseMsg()`, `ThinkingMsg()`, `TurnEndMsg()`, `SystemMsg()`
- Keep existing `OutputMsg`, `StatusMsg`, `ErrorMsg` for backward compat

### Task 1.2: Verify WS message serialization
- Write tests for each new message constructor
- Verify JSON roundtrip for all message types
- Verify backward compatibility: old `OutputMsg` still works

## Phase 2: SDK Integration for Interactive Agents

### Task 2.1: Create SDK adapter layer
- Create `backend/internal/adapter/agent/sdk_client.go`
- Define `SDKSession` struct wrapping `claude.Client` with agent metadata
- Implement `NewSDKSession(ctx, opts)` that configures `types.ClaudeAgentOptions` from spawner config
- Handle: model, CWD, dangerously-skip-permissions, environment vars
- Map existing `SpawnInteractiveOpts` to SDK options

### Task 2.2: Replace SpawnInteractive with SDK Client
- Rewrite `SpawnInteractive` to create `SDKSession` instead of `exec.Command`
- Call `client.Connect(ctx)` to start the session
- If `opts.Prompt` set, call `client.Query(ctx, prompt)` for initial turn
- Return `InteractiveAgent` with adapted IO channels
- Update `InteractiveAgent` struct: replace `Stdin io.WriteCloser` with method-based input (`SendInput(text)`)

### Task 2.3: Implement message relay goroutine
- Create `relaySDKMessages(session, output chan)` goroutine
- Range over `client.ReceiveResponse(ctx)` channel
- Type-switch on SDK messages:
  - `*types.AssistantMessage` → iterate content blocks → emit `TextMsg`/`ToolUseMsg`/`ThinkingMsg`
  - `*types.SystemMessage` → emit `SystemMsg`
  - `*types.ResultMessage` → emit `TurnEndMsg` with cost/usage, update quota tracker
- Generate turn_id (UUID) per turn, emit `TurnStartMsg` at start

### Task 2.4: Implement turn-based input
- When user sends WS `input` message, call `client.Query(ctx, text)` to start a new turn
- Start new `relaySDKMessages` goroutine for the response
- Handle concurrent input rejection (one turn at a time)

### Task 2.5: Update Bridge and SessionManager
- Update `ws/session.go` `StartOutputRelay` to handle structured messages (already `[]byte`, so JSON marshaled messages work)
- Ensure ring buffer stores structured messages for reconnection replay

### Task 2.6: Verify interactive agent lifecycle
- Test: spawn interactive agent, receive typed messages over WS
- Test: send user input, receive new turn response
- Test: agent completion emits proper `turn_end` + `status` messages
- Test: reconnection replays structured messages from ring buffer

## Phase 3: SDK Integration for Non-Interactive Agents

### Task 3.1: Replace SpawnDeveloper with SDK Query
- Rewrite `SpawnDeveloper` to use `claude.Query(ctx, prompt, opts)` one-shot function
- Configure SDK options: model, CWD, dangerously-skip-permissions, session-id
- Range over returned message channel in monitor goroutine

### Task 3.2: Replace SpawnReviewer with SDK Query
- Same pattern as SpawnDeveloper but with reviewer-specific config
- Maintain existing prompt construction (`/kf-reviewer {prURL}`)

### Task 3.3: Update monitor goroutines
- Replace `monitorAgent` line-by-line scanning with SDK message processing
- Type-switch on messages for logging and quota tracking
- Extract cost/usage from `ResultMessage` for tracer spans
- Log structured messages to agent log files (JSON format)

### Task 3.4: Verify non-interactive agent lifecycle
- Test: spawn developer agent, verify completion callback fires
- Test: spawn reviewer agent with PR URL, verify proper lifecycle
- Test: quota tracker receives usage data from SDK ResultMessage

## Phase 4: Cleanup and Integration

### Task 4.1: Update quota tracker
- Modify `QuotaTracker.RecordEvent` to accept SDK `types.ResultMessage` (or adapt interface)
- Extract `TotalCostUSD`, `Usage` map fields for rate limit tracking
- Remove or deprecate `ParseStreamLine`-based recording path

### Task 4.2: Remove deprecated code
- Remove `CleanClaudeEnv()` from spawner (SDK handles this)
- Remove or deprecate `ExtractText()` and `ParseStreamLine()` from parser.go
- Remove manual CLI arg construction
- Clean up unused imports

### Task 4.3: Update api_handler.go callers
- Update `SpawnInteractiveAgent` handler if `InteractiveAgent` struct changed
- Update Bridge creation if `Stdin` replaced with method-based input
- Ensure WS handler calls `session.SendInput()` instead of writing to stdin pipe

### Task 4.4: Update agent tracing
- Update span attributes to use SDK message fields
- Set token counts from `ResultMessage.Usage` map
- Set cost from `ResultMessage.TotalCostUSD`

### Task 4.5: Full integration test
- Verify all 3 agent types spawn and complete successfully
- Verify WS protocol delivers structured messages
- Verify quota tracking works end-to-end
- Verify agent log files contain structured JSON
- Verify tracing spans have correct attributes

## Phase 5: Documentation

### Task 5.1: Update API documentation
- Update AsyncAPI schema for new WS message types
- Document the enriched WS protocol in code comments
- Update any internal docs referencing the old spawner pattern
