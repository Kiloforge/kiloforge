# Implementation Plan: E2E Tests — Quota and Token Usage

**Track ID:** e2e-quota-usage_20260309194841Z

## Phase 1: Stat Card Tests

- [ ] Task 1.1: Token count display — seed quota data with known input_tokens (1500), output_tokens (800), cache_read_tokens (200), navigate to dashboard, verify each stat card displays the correct token count with appropriate formatting
- [ ] Task 1.2: Cost estimate format — seed quota data with `total_cost_usd: 1.23`, navigate to dashboard, verify cost stat card shows "$1.23" in USD format; repeat with `0.00`, `0.0012`, and `1234.56` to verify formatting across ranges
- [ ] Task 1.3: Initial load from API — navigate to dashboard with no cache, verify stat cards load token counts and cost from `GET /api/quota` on initial page load, verify a loading state is shown briefly before data appears

## Phase 2: Rate Limit Tests

- [ ] Task 2.1: Rate limit badge visible — seed quota data with `rate_limited: true` and `rate_limit_retry_after: 30`, navigate to dashboard, verify a rate limit badge/indicator is visible with retry countdown text
- [ ] Task 2.2: Countdown timer — seed rate-limited quota with `retry_after: 10`, navigate to dashboard, verify countdown decrements over time (check value at T+0 and T+2 seconds), verify countdown text updates
- [ ] Task 2.3: Badge hides when cleared — seed rate-limited quota, navigate to dashboard, verify badge is visible; send SSE `quota_update` with `rate_limited: false`, verify badge disappears without page refresh

## Phase 3: SSE Update Tests

- [ ] Task 3.1: quota_update refreshes cards — open dashboard in Playwright, verify initial token counts, send SSE `quota_update` event with new token counts (input_tokens: 2500, output_tokens: 1200), verify stat cards update to new values without page refresh
- [ ] Task 3.2: Rapid updates coalesce — open dashboard, send 10 `quota_update` SSE events in quick succession (50ms apart) with incrementing token counts, verify final displayed values match the last event, verify no visual flickering or intermediate renders
- [ ] Task 3.3: Incremental updates — open dashboard with initial quota (input: 1000), spawn a mock agent via API that reports usage (input: 500), verify the stat cards update to show aggregated total (input: 1500) after the `quota_update` event arrives

## Phase 4: Aggregation Tests

- [ ] Task 4.1: Multi-agent totals — seed quota from two mock agents (agent-A: 1000 input, agent-B: 2000 input), navigate to dashboard, verify stat cards show aggregated totals (3000 input), verify cost is the sum of both agents' costs
- [ ] Task 4.2: Per-agent breakdown — if the UI provides a per-agent breakdown view, seed two agents with different usage, navigate to breakdown, verify individual agent usage is displayed correctly; if no breakdown view, verify the aggregate display is correct
- [ ] Task 4.3: History view — if the UI provides a quota history or timeline, seed multiple quota snapshots at different timestamps, verify the history displays data points in chronological order; if no history view, verify the current snapshot display is correct

## Phase 5: Edge and Failure Cases

- [ ] Task 5.1: Large and zero values — seed quota with very large token counts (1,000,000+ input tokens, $10,000+ cost), verify stat cards format with commas or abbreviations without overflow; seed zero values (0 tokens, $0.00 cost), verify cards display zero gracefully
- [ ] Task 5.2: Zero and small costs — seed quota with `total_cost_usd: 0`, verify "$0.00" display; seed `total_cost_usd: 0.0001`, verify the UI shows appropriate precision without rounding to zero
- [ ] Task 5.3: API errors and malformed events — simulate `GET /api/quota` returning 500 error, verify error state displayed in dashboard (error message or retry prompt); send malformed SSE `quota_update` event (invalid JSON), verify UI does not crash and previous values remain displayed
