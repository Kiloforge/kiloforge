# Gap Analysis

Missing capabilities organized by category, with severity ratings.

## Severity Scale

| Level | Meaning |
|-------|---------|
| Critical | Blocks scale or adoption; must fix |
| High | Significant UX or reliability gap |
| Medium | Notable missing capability |
| Low | Nice-to-have improvement |

---

## 1. Scale and Performance

| Gap | Severity | Description |
|-----|----------|-------------|
| No API pagination | Critical | All list endpoints return unbounded results. With 172+ tracks and growing agent history, this will cause performance degradation and potential OOM. |
| No API filtering/sorting | High | Cannot filter agents by status, traces by time range, or tracks by type. Forces client-side filtering which breaks with pagination. |
| No database indexing strategy | Medium | SQLite indexes not audited for query patterns. As data grows, slow queries will emerge. |
| No data retention/archival | Medium | No automatic pruning of old agent logs, traces, or completed track data. Storage grows unbounded. |

## 2. Observability and Notifications

| Gap | Severity | Description |
|-----|----------|-------------|
| No notification system | Critical | Users get no alerts when agents complete, fail, escalate, or require attention. Must actively monitor the dashboard. For long-running agents (hours), this is a major UX failure. |
| No log search/aggregation | High | Agent logs are per-agent only. Cannot search across all agents for errors, patterns, or keywords. |
| No agent metrics/analytics | Medium | No aggregate views of agent success rates, average duration, token efficiency trends. |
| No alerting rules | Medium | No configurable thresholds (e.g., alert if agent runs > 30min, cost exceeds $X). |

## 3. Agent Management

| Gap | Severity | Description |
|-----|----------|-------------|
| No agent timeout enforcement | High | Agents can run indefinitely. No configurable max duration. |
| No automatic retry/recovery | High | Failed agents require manual restart. No configurable retry policy. |
| No agent templates/presets | Medium | Cannot save and reuse agent configurations (skill + project + branch combos). |
| No agent scheduling | Medium | Cannot schedule agent runs (e.g., nightly review of all open PRs). |
| No agent resource limits | Low | Cannot limit concurrent agents per project or globally beyond pool size. |

## 4. Project Management

| Gap | Severity | Description |
|-----|----------|-------------|
| No multi-forge support | Medium | Only Gitea is supported. GitHub, GitLab, and Forgejo would expand adoption. |
| No branch protection rules | Medium | No configurable rules about which branches agents can push to. |
| No project templates | Low | Cannot create project configurations from templates. |
| No project health dashboard | Low | No per-project metrics (agents run, tracks completed, cost). |

## 5. User Experience

| Gap | Severity | Description |
|-----|----------|-------------|
| No responsive/mobile design | High | Dashboard is desktop-only. Cannot monitor agents from mobile. |
| No keyboard shortcuts | High | No command palette, no hotkeys for common operations. Power users blocked. |
| No dark/light theme | Medium | No theme preferences. May affect usability in different environments. |
| No CLI shell completion | Medium | No bash/zsh/fish completion generation. Reduces CLI discoverability. |
| No CLI `--json` output | Medium | No machine-readable output for scripting and automation. |
| No onboarding tour/wizard | Low | New users must read docs to understand the workflow. |

## 6. Extensibility

| Gap | Severity | Description |
|-----|----------|-------------|
| No dynamic skill loading | High | Skills are embedded at compile time. Users cannot create custom skills without forking. |
| No webhook extensibility | High | Cannot send events to external systems (Slack, Discord, PagerDuty). |
| No plugin architecture | Medium | No extension points for custom adapters, stores, or services. |
| No API authentication | Medium | REST API has no auth. Fine for local single-user, but blocks multi-user and remote access. |
| No API versioning | Low | No version prefix on API routes. Breaking changes have no migration path. |

## 7. Documentation

| Gap | Severity | Description |
|-----|----------|-------------|
| No API reference docs | High | 49 operations with no generated reference. OpenAPI spec exists but isn't published as docs. |
| No user guide | High | Getting-started.md exists (138 lines) but no comprehensive user guide for the full feature set. |
| No skill authoring guide | Medium | No documentation on how skills work, their lifecycle, or how to write effective skill prompts. |
| No architecture decision records | Low | docs/architecture.md exists but no ADRs for key decisions. |
| No changelog | Low | No CHANGELOG.md tracking releases and breaking changes. |

## 8. Testing Infrastructure

| Gap | Severity | Description |
|-----|----------|-------------|
| E2E tests not in CI | High | 21 E2E specs exist but aren't run in the CI pipeline. UI regressions can land undetected. |
| No load/stress testing | Medium | No testing for concurrent agent workloads or API throughput. |
| No frontend snapshot testing | Low | Component renders not snapshotted for visual regression detection. |
| No mutation testing | Low | Test quality not verified via mutation analysis. |

## 9. Security

| Gap | Severity | Description |
|-----|----------|-------------|
| No API authentication | Medium | REST API is completely unauthenticated. Acceptable for local single-user but blocks any network exposure. |
| No audit logging | Medium | No record of who performed what operations and when. |
| No secret scanning | Low | No pre-commit hooks or CI checks for accidental secret commits. |

---

## Summary by Severity

| Severity | Count |
|----------|-------|
| Critical | 2 |
| High | 12 |
| Medium | 16 |
| Low | 9 |
| **Total** | **39** |
