# Research: Claude Code Quota Monitoring and Graceful Degradation

**Track ID:** research-cc-quota-monitoring_20260307150000Z
**Date:** 2026-03-07
**CC Version Tested:** 2.1.71

---

## 1. CC Behavior Under Rate Limiting (429/529)

### 1.1 What happens when CC hits a 429

Claude Code **retries automatically** on 429 errors. The Anthropic API returns a `retry-after` header specifying the number of seconds to wait. CC internally handles this retry loop.

However, there are documented failure modes:

- **Persistent 429 lockout**: CC can become "completely locked out" when the backend erroneously flags accounts as rate-limited even after quota reset (12+ hours). No automatic recovery occurs — manual intervention required.
- **Stuck stop hooks**: CC may hang on "running stop hooks..." when OTEL telemetry export itself encounters 429 rate limiting, creating a retry loop on exit.
- **Subagent hang on exit**: Subagent processes can hang indefinitely due to OTEL telemetry export retries hitting 429s (GitHub issue #30378).

### 1.2 Error messages observed

```
HTTP 429: rate_limit_error: This request would exceed your account's rate limit.
Please try again later. (request_id: req_...)
```

```
API Error: Rate limit reached
```

### 1.3 Exit vs. wait vs. retry

| Scenario | CC Behavior |
|----------|-------------|
| Transient 429 (under daily limit) | Auto-retry with backoff using `retry-after` header |
| Persistent 429 (backend desync) | Hangs or fails immediately on launch, no recovery |
| 529 (overloaded) | Similar retry behavior to 429 |
| Budget exceeded (`--max-budget-usd`) | Exits with result subtype `error_max_budget_usd` |

### 1.4 Key finding

CC does **not** exit cleanly on persistent rate limits. It either retries indefinitely or hangs. There is no configurable retry limit or timeout for rate-limit retries. This is a significant concern for kiloforge's multi-agent spawning — a single rate-limited agent could become a zombie process consuming system resources.

---

## 2. Stream-JSON Output Format

### 2.1 Event types

CC's `--output-format stream-json` emits newline-delimited JSON (NDJSON). Each line is a JSON object with the following event types:

| Type | Description |
|------|-------------|
| `init` | Session initialization (session_id, timestamp) |
| `message` | Assistant or user message content |
| `tool_use` | Tool invocation with parameters |
| `tool_result` | Tool execution result |
| `result` | Final completion status |

With `--verbose --include-partial-messages`, streaming text deltas are also emitted as `stream_event` types.

### 2.2 Message schema

```typescript
interface StreamMessage {
  type: 'init' | 'message' | 'tool_use' | 'tool_result' | 'result';
  timestamp?: string;
  session_id?: string;
  role?: 'assistant' | 'user';
  content?: Array<{
    type: 'text' | 'tool_use';
    text?: string;
    name?: string;
    input?: any;
  }>;
  output?: string;
  status?: 'success' | 'error';
  duration_ms?: number;
}
```

### 2.3 Result subtypes

The final `result` event includes a `subtype` field:

| Subtype | Meaning |
|---------|---------|
| `success` | Completed normally |
| `error_max_turns` | Exceeded max conversation turns |
| `error_during_execution` | Runtime error during execution |
| `error_max_budget_usd` | Budget limit reached |
| `error_max_structured_output_retries` | Schema validation retries exhausted |

### 2.4 Usage data in result messages

The result message includes:

```json
{
  "type": "result",
  "subtype": "success",
  "total_cost_usd": 0.0342,
  "session_id": "...",
  "usage": {
    "input_tokens": 12500,
    "output_tokens": 3200,
    "cache_read_input_tokens": 8000,
    "cache_creation_input_tokens": 1500
  }
}
```

### 2.5 How errors appear in stream-json

- **Budget exceeded**: `{"type":"result","subtype":"error_max_budget_usd","total_cost_usd":...}`
- **Rate limit errors**: Not surfaced as distinct stream-json events. CC handles retries internally. If it gives up, the result event has `subtype: "error_during_execution"`.
- **429 during OTEL export**: Not visible in stream-json output (internal to CC process lifecycle).

### 2.6 Known issue

CC CLI can hang indefinitely after sending the final `{"type":"result","subtype":"success"}` event in stream-json mode — the process remains running with stdout open, never exiting cleanly (GitHub issue #25629). This must be handled with a timeout in kiloforge's agent monitor goroutine.

---

## 3. OpenTelemetry Metrics

### 3.1 Enabling telemetry

```bash
export CLAUDE_CODE_ENABLE_TELEMETRY=1
export OTEL_METRICS_EXPORTER=otlp          # otlp | prometheus | console
export OTEL_LOGS_EXPORTER=otlp             # otlp | console
export OTEL_EXPORTER_OTLP_PROTOCOL=grpc    # grpc | http/json | http/protobuf
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

### 3.2 Available metrics

| Metric | Description | Unit | Key Attributes |
|--------|-------------|------|----------------|
| `claude_code.session.count` | Sessions started | count | standard |
| `claude_code.token.usage` | Tokens used | tokens | `type` (input/output/cacheRead/cacheCreation), `model` |
| `claude_code.cost.usage` | Session cost | USD | `model` |
| `claude_code.lines_of_code.count` | Lines modified | count | `type` (added/removed) |
| `claude_code.commit.count` | Commits created | count | standard |
| `claude_code.pull_request.count` | PRs created | count | standard |
| `claude_code.code_edit_tool.decision` | Edit permission decisions | count | `tool_name`, `decision`, `source` |
| `claude_code.active_time.total` | Active time | seconds | `type` (user/cli) |

### 3.3 Available events (via OTEL logs protocol)

| Event Name | Description | Key Attributes |
|------------|-------------|----------------|
| `claude_code.user_prompt` | User submitted a prompt | `prompt_length`, `prompt` (if opted in) |
| `claude_code.tool_result` | Tool completed | `tool_name`, `success`, `duration_ms`, `error` |
| `claude_code.api_request` | API call made | `model`, `cost_usd`, `duration_ms`, `input_tokens`, `output_tokens`, `cache_*_tokens` |
| `claude_code.api_error` | API call failed | `model`, `error`, `status_code`, `attempt` |
| `claude_code.tool_decision` | Tool permission decision | `tool_name`, `decision`, `source` |

### 3.4 Standard attributes on all metrics/events

| Attribute | Description |
|-----------|-------------|
| `session.id` | Unique session identifier |
| `organization.id` | Org UUID (when authenticated) |
| `user.account_uuid` | Account UUID |
| `user.id` | Anonymous device identifier |
| `terminal.type` | Terminal type |
| `prompt.id` | UUID linking events to a single prompt (events only) |

### 3.5 Propagation to child processes

OTel env vars (`OTEL_EXPORTER_OTLP_ENDPOINT`, etc.) are standard environment variables. When kiloforge spawns CC via `exec.CommandContext`, these env vars **can be passed through** to the child process via `cmd.Env`. Each spawned agent will independently export to the configured OTLP endpoint.

### 3.6 Per-worker collection feasibility

**Feasible via `OTEL_RESOURCE_ATTRIBUTES`:**

```bash
# Set per-worker identity
export OTEL_RESOURCE_ATTRIBUTES="worker.id=developer-1,track.id=feature_xyz,agent.role=developer"
```

This allows filtering/grouping metrics by worker, track, or role in the backend. Combined with `session.id`, provides full attribution.

### 3.7 Key finding: `claude_code.api_error` event

The `api_error` event includes `status_code` (e.g., "429") and `attempt` number. This is the **primary mechanism** for detecting rate limiting across workers. A centralized OTLP collector can aggregate `api_error` events with `status_code=429` across all workers and trigger alerts or backoff.

---

## 4. Feasibility of Parsing Rate Limit Info from Agent Output

### 4.1 Stream-JSON parsing

**Partially feasible.** The stream-json output provides:
- Total cost via `result.total_cost_usd`
- Token usage via `result.usage`
- Budget exhaustion via `result.subtype == "error_max_budget_usd"`

**Not available in stream-json:**
- Real-time token counts (only at session end)
- Rate limit headers (`anthropic-ratelimit-*-remaining`)
- Per-request cost breakdown (only cumulative total)

### 4.2 OTel events (preferred approach)

**Fully feasible.** The `api_request` event provides per-request:
- `cost_usd`, `input_tokens`, `output_tokens`, `cache_*_tokens`
- `duration_ms`, `model`

The `api_error` event provides:
- `status_code` (429 for rate limit)
- `attempt` number (retry count)
- `error` message

This is significantly richer than stream-json and provides **real-time** data (exported every 5s for events, 60s for metrics by default).

### 4.3 Recommendation

Use **OTel events as the primary data source** and stream-json `result` messages as a **secondary/fallback** for final cost/usage summary.

---

## 5. Architecture Proposal: Centralized Quota Tracker

### 5.1 Recommended architecture

```
                                    +-------------------+
                                    |   Grafana / Alert  |
                                    |    Dashboard       |
                                    +--------+----------+
                                             |
                                    +--------v----------+
                                    |  OTLP Collector   |
                                    |  (e.g., otel-col) |
                                    +--------+----------+
                                             ^
                    +------------------------+------------------------+
                    |                        |                        |
            +-------+-------+       +-------+-------+       +-------+-------+
            |  CC Agent     |       |  CC Agent     |       |  CC Agent     |
            |  (developer-1)|       |  (developer-2)|       |  (reviewer-1) |
            |               |       |               |       |               |
            | OTEL_RESOURCE |       | OTEL_RESOURCE |       | OTEL_RESOURCE |
            | worker.id=d1  |       | worker.id=d2  |       | worker.id=r1  |
            +---------------+       +---------------+       +---------------+
```

### 5.2 Components

1. **OTLP Collector** (lightweight, runs alongside Gitea in docker-compose)
   - Receives metrics and events from all CC agents
   - Options: OpenTelemetry Collector, VictoriaMetrics, or simple file-based collector
   - Minimal footprint: just needs to aggregate and expose data

2. **Quota Tracker Service** (new component in kiloforge)
   - Reads aggregated metrics from collector
   - Tracks per-worker and aggregate token/cost usage
   - Exposes simple API for relay server to query
   - Manages rate limit state machine (normal -> warning -> throttle -> pause)

3. **Relay Server Integration** (modify existing relay)
   - Before spawning new agent: check quota tracker
   - On `api_error` with 429: pause spawning, notify user
   - On budget threshold: warn user via CLI output

### 5.3 Data model

```go
type WorkerUsage struct {
    WorkerID    string
    TrackID     string
    SessionID   string
    Role        string    // developer, reviewer
    InputTokens int64
    OutputTokens int64
    CacheReadTokens int64
    CostUSD     float64
    StartedAt   time.Time
    UpdatedAt   time.Time
    RateLimitHits int
    LastRateLimitAt *time.Time
}

type AggregateUsage struct {
    TotalCostUSD    float64
    TotalTokens     int64
    ActiveWorkers   int
    RateLimitState  string // "normal", "warning", "throttle", "paused"
    WindowStart     time.Time
}
```

### 5.4 Simpler alternative: stream-json parsing only

If OTel infrastructure is too heavy, a lighter approach:

1. Parse stream-json output in the existing agent monitor goroutine (`spawner.go:87-101`)
2. Extract `result` messages for cost/usage at session end
3. Use `--max-budget-usd` per agent as a hard cap
4. Track cumulative cost in kiloforge's state store

**Tradeoffs:**
- No real-time visibility (only at session end)
- No rate-limit detection (only budget enforcement)
- Simpler to implement, no infrastructure dependencies
- Good enough for MVP

---

## 6. Graceful Degradation Strategy

### 6.1 Threshold levels

| Level | Trigger | Action |
|-------|---------|--------|
| **Normal** | < 70% of daily budget | No action |
| **Warning** | 70-85% of daily budget | Log warning, notify user via `kf status` |
| **Throttle** | 85-95% of daily budget | Delay new agent spawns, prefer Sonnet over Opus |
| **Pause** | > 95% of daily budget OR 3+ 429s in 5min | Pause all new spawns, notify user |
| **Emergency** | Persistent 429 across all workers | Kill non-essential agents, preserve only critical work |

### 6.2 Worker queuing/backoff

When in **Throttle** state:
- New `kf implement` commands are queued (FIFO)
- Queue depth limit: 5 tracks (reject with error beyond this)
- Existing agents continue running
- Check every 60s if budget has recovered (new billing window)

When in **Pause** state:
- All queued spawns held
- Existing agents continue but with `--max-budget-usd` reduced
- User must explicitly `kf resume` to unblock

### 6.3 User notification

```
$ kf status
QUOTA STATUS: WARNING (78% of daily budget consumed)

Active Workers:
  developer-1  track:auth_feature    cost:$2.34  tokens:145K
  developer-2  track:api_refactor    cost:$1.89  tokens:112K
  reviewer-1   track:auth_feature    cost:$0.45  tokens:28K

Daily Budget: $8.00 / $10.00 consumed
Rate Limit Hits: 0 in last 5min

Queued: 0 tracks waiting
```

### 6.4 Should relay pause new spawns vs. halt existing ones?

**Recommendation: Pause new spawns first, then reduce budgets on existing agents.**

Rationale:
- Killing an in-progress agent wastes all tokens already consumed
- Agents doing complex multi-step work may lose significant progress
- Pausing spawns is safer and immediately reduces future consumption
- Only halt existing agents as a last resort (emergency level)

---

## 7. Per-Track Cost Tracking Feasibility

### 7.1 Cost attribution model

| Level | Source | Feasibility |
|-------|--------|-------------|
| Per agent | `session.id` in OTel or `total_cost_usd` in stream-json | Easy |
| Per track | `OTEL_RESOURCE_ATTRIBUTES=track.id=...` or track ↔ session mapping in state | Easy |
| Per phase/task | Requires commit-aligned cost snapshots | Medium (parse stream-json at commit boundaries) |

### 7.2 Storage format

Extend existing `state.Store` with cost data:

```go
type TrackCost struct {
    TrackID     string
    TotalCostUSD float64
    TokenBreakdown struct {
        Input         int64
        Output        int64
        CacheRead     int64
        CacheCreation int64
    }
    AgentSessions []SessionCost
}

type SessionCost struct {
    SessionID  string
    AgentRole  string
    CostUSD    float64
    StartedAt  time.Time
    CompletedAt *time.Time
}
```

### 7.3 Reporting interface

```
$ kiloforge cost
Track                          Cost      Tokens   Status
auth_feature_20260307          $4.23     267K     in-progress
api_refactor_20260307          $1.89     112K     in-progress
review_cycle_20260306          $0.92      58K     complete
────────────────────────────────────────────────────
Total (today)                  $7.04     437K

$ kiloforge cost --track auth_feature_20260307
Track: auth_feature_20260307
Total Cost: $4.23

Sessions:
  developer-1  $2.34  145K tokens  running  (2h 15m)
  developer-1  $1.44   94K tokens  completed (1h 02m, earlier attempt)
  reviewer-1   $0.45   28K tokens  completed (12m)
```

---

## 8. Limitations and Unknowns

### 8.1 Cannot test inside nested CC session

CC prevents nested sessions (`CLAUDECODE` env var check). This means:
- Cannot empirically verify rate-limit retry behavior from within kiloforge's dev workflow
- Must rely on documentation, GitHub issues, and external testing
- Recommendation: create a standalone test script outside CC for empirical validation

### 8.2 No direct quota-checking API

- Anthropic does not expose a "check remaining quota" API endpoint
- Must infer quota state from 429 errors or OTel `api_error` events
- The `/api/oauth/usage` endpoint exists but has its own aggressive rate limiting (GitHub issue #31021)

### 8.3 CC stream-json hang bug

- CC can hang after emitting the final result event (issue #25629)
- Kiloforge's monitor goroutine must implement a timeout after receiving a `result` event
- Recommended: 30s timeout after result event, then SIGTERM → SIGKILL

### 8.4 OTel export retry loop on 429

- CC's OTEL exporter can enter an infinite retry loop when the export endpoint returns 429
- This causes subagent processes to hang on exit
- If running a local OTLP collector, ensure it doesn't rate-limit CC's exports

### 8.5 Colima/Docker networking

- OTLP collector in docker-compose needs to be accessible from host (where CC runs)
- Use `ports: ["4317:4317"]` for gRPC or `["4318:4318"]` for HTTP
- With Colima, verify network connectivity from host to container

---

## 9. Proposed Implementation Tracks

### Track A: MVP — Stream-JSON Cost Tracking (Small, No Infrastructure)

**Scope:**
- Parse stream-json `result` messages in spawner monitor goroutine
- Extract `total_cost_usd` and `usage` on agent completion
- Store per-track cost in state store
- Add `--max-budget-usd` flag to `kf implement` (passed through to CC)
- Add `kf cost` command for reporting
- Handle stream-json hang with post-result timeout

**Effort:** ~1-2 tracks (feature + tests)
**Dependencies:** None
**Blockers:** None

### Track B: Full — OTel-Based Quota Monitoring (Medium, Requires Infrastructure)

**Scope:**
- Add lightweight OTLP collector to docker-compose (OpenTelemetry Collector or VictoriaMetrics)
- Set `CLAUDE_CODE_ENABLE_TELEMETRY=1` and OTLP env vars when spawning agents
- Set per-worker `OTEL_RESOURCE_ATTRIBUTES` for attribution
- Build quota tracker service that reads from collector
- Implement threshold-based degradation (warning/throttle/pause)
- Real-time `kf status` with quota information
- Alert on 429 patterns via `api_error` events

**Effort:** ~3-4 tracks
**Dependencies:** Track A (stream-json parsing as fallback)
**Blockers:** None, but requires choosing an OTLP backend

### Recommendation

**Start with Track A** — it provides immediate value with zero infrastructure overhead. The stream-json result parsing fits naturally into the existing spawner goroutine. Track B can be layered on top later for organizations that need real-time monitoring and multi-worker coordination.

---

## Sources

- [Claude Code Headless/Programmatic Docs](https://code.claude.com/docs/en/headless)
- [Claude Code Monitoring Docs](https://code.claude.com/docs/en/monitoring-usage)
- [Claude Code Cost Management Docs](https://code.claude.com/docs/en/costs)
- [Agent SDK Cost Tracking](https://platform.claude.com/docs/en/agent-sdk/cost-tracking)
- [Anthropic Rate Limits](https://platform.claude.com/docs/en/api/rate-limits)
- [GitHub Issue #22876 — 429 despite available quota](https://github.com/anthropics/claude-code/issues/22876)
- [GitHub Issue #29650 — Persistent 429 / stuck stop hooks](https://github.com/anthropics/claude-code/issues/29650)
- [GitHub Issue #25629 — Stream-JSON hang on exit](https://github.com/anthropics/claude-code/issues/25629)
- [GitHub Issue #30378 — Subagent hang on OTEL 429](https://github.com/anthropics/claude-code/issues/30378)
- [Claude Code OTel Guide (SigNoz)](https://signoz.io/blog/claude-code-monitoring-with-opentelemetry/)
- [Claude Code OTel (ColeMurray)](https://github.com/ColeMurray/claude-code-otel)
- [Claude Code Monitoring Guide (Anthropic)](https://github.com/anthropics/claude-code-monitoring-guide)
