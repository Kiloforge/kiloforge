# Prioritized Roadmap

Recommendations organized into three tiers based on impact, effort, and dependencies.

---

## Tier 1: Immediate (Next 1-2 Weeks)

High-impact, low-effort improvements that should be done first.

| # | Item | Type | Effort | Impact | Rationale |
|---|------|------|--------|--------|-----------|
| 1 | API pagination + filtering (R1, R2) | Improvement | M | 5 | Unblocks scale. Every other feature benefits from this. |
| 2 | E2E tests in CI (R16) | Improvement | S | 4 | Tests exist but aren't run. Prevents regressions. |
| 3 | Generate API docs from OpenAPI (R18) | Improvement | S | 4 | Spec exists; just needs tooling. Low-hanging fruit. |
| 4 | Shell completions (R5) | Improvement | S | 3 | Nearly free with Cobra. Improves CLI discoverability. |
| 5 | Remove deprecated kf-parallel (R14) | Improvement | S | 2 | Cleanup. Reduces confusion. |
| 6 | Agent timeout enforcement (R11) | Improvement | S | 4 | Prevents runaway agents. Simple timer check. |
| 7 | CLI --json output (R4) | Improvement | M | 4 | Enables scripting and automation of kiloforge. |

**Estimated effort:** ~1-2 weeks of agent work (7-12 tracks)

---

## Tier 2: Near-Term (Weeks 3-6)

Significant capability additions that build on Tier 1.

| # | Item | Type | Effort | Impact | Rationale |
|---|------|------|--------|--------|-----------|
| 8 | Notification system (F1) | New Feature | L | 5 | #1 UX gap. Agents are autonomous — users need push notifications. |
| 9 | Webhook extensibility (F5) | New Feature | M | 4 | Enables Slack/Discord/PagerDuty integration. Pairs with notifications. |
| 10 | Agent retry/recovery (R12) | Improvement | M | 4 | Enables unattended operation. Critical for queue reliability. |
| 11 | Keyboard shortcuts + command palette (R7) | Improvement | M | 4 | Power user essential. |
| 12 | Enhanced TrackDetail page (R9) | Improvement | M | 3 | Progress viz, agent assignment, inline editing. |
| 13 | Trace visualization improvements (R13) | Improvement | M | 3 | Filter, search, zoom in span timeline. |
| 14 | User guide (R19) | Documentation | M | 4 | Comprehensive docs for onboarding. |
| 15 | Bulk operations (R3) | Improvement | S | 3 | Multi-select in dashboard, bulk API. |

**Estimated effort:** ~3-4 weeks of agent work (15-25 tracks)

---

## Tier 3: Strategic (Weeks 7+)

Transformative features that change the product's nature or addressable market.

| # | Item | Type | Effort | Impact | Rationale |
|---|------|------|--------|--------|-----------|
| 16 | Dynamic skill loading (F2) | New Feature | L | 4 | Platform play. Enables community ecosystem. |
| 17 | Agent analytics dashboard (F4) | New Feature | M | 4 | Cost visibility and optimization. |
| 18 | Agent scheduling (F6) | New Feature | M | 3 | Cron-like automation for recurring tasks. |
| 19 | Responsive/mobile dashboard (R8) | Improvement | L | 3 | Mobile monitoring. |
| 20 | GitHub forge support (F7) | New Feature | XL | 4 | Removes biggest adoption barrier. |
| 21 | Multi-user support (F3) | New Feature | XL | 4 | Team collaboration. Highest complexity. |
| 22 | TUI mode (F8) | New Feature | L | 3 | Terminal-centric alternative to browser dashboard. |

**Estimated effort:** 6-12 weeks of agent work (30-60 tracks)

---

## Dependency Graph

```
Tier 1: API pagination (R1) ──┐
                               ├── R10 (search/filtering in dashboard)
Tier 1: API filtering (R2) ───┘

Tier 1: Agent timeout (R11) ──── Tier 2: Agent retry (R12)

Tier 2: Notifications (F1) ──── Tier 2: Webhooks (F5)
                               └── Tier 3: Scheduling (F6)

Tier 2: Bulk ops (R3) ──── Tier 2: Keyboard shortcuts (R7)

Tier 3: Dynamic skills (F2) ──── Tier 3: Multi-user (F3)
                                          └── API auth (prerequisite)
```

---

## Total Estimated Impact

If all Tier 1 + Tier 2 items are completed:
- **Scale:** Product can handle 1000+ agents/tracks without performance issues
- **UX:** Users notified of all events, keyboard-driven workflow, rich search
- **Reliability:** Agent timeout + retry enables true unattended operation
- **Adoption:** API docs + user guide lower the onboarding barrier
- **Integration:** Webhook extensibility connects kiloforge to the broader toolchain

Tier 3 items transform kiloforge from a **power-user tool** into a **team platform** with community ecosystem potential.
