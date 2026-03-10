# Improvement Recommendations

Enhancements to existing features. Each recommendation includes effort estimate and impact rating.

## Effort Scale

| Size | Meaning |
|------|---------|
| S | < 1 day (1-2 tracks) |
| M | 1-3 days (2-4 tracks) |
| L | 3-7 days (4-8 tracks) |
| XL | 1-2 weeks (8+ tracks) |

## Impact Scale: 1 (minimal) to 5 (transformative)

---

## API Improvements

### R1. Add pagination to all list endpoints
- **Effort:** M | **Impact:** 5
- **Current:** All list endpoints return unbounded results
- **Proposed:** Add cursor-based pagination with `?cursor=X&limit=N` pattern. Return `next_cursor` in response. Default limit: 50, max: 200.
- **Endpoints affected:** listAgents, listProjects, listTracks, listTraces, listLocks, listSSHKeys
- **Notes:** Schema-first — update OpenAPI first, regenerate, then implement in store layer. Consider adding `?status=`, `?project=`, `?since=` filters at the same time.

### R2. Add filtering and sorting to list endpoints
- **Effort:** M | **Impact:** 4
- **Current:** No server-side filtering or sorting
- **Proposed:** Add query parameters: `?status=running`, `?project=myproject`, `?sort=created_at&order=desc`, `?since=2026-01-01`
- **Notes:** Pair with R1 (pagination). Define filtering schema in OpenAPI.

### R3. Add bulk operations
- **Effort:** S | **Impact:** 3
- **Current:** Operations are per-entity only
- **Proposed:** `POST /-/api/agents/bulk-stop`, `DELETE /-/api/tracks/bulk`, etc. Accept array of IDs.
- **Notes:** Enables dashboard multi-select UX.

---

## CLI Improvements

### R4. Add `--json` output flag to all commands
- **Effort:** M | **Impact:** 4
- **Current:** Human-readable output only
- **Proposed:** Global `--json` flag outputs structured JSON for all commands. Enables scripting and piping.
- **Notes:** Could also add `--format` flag with options: `table`, `json`, `yaml`.

### R5. Generate shell completions
- **Effort:** S | **Impact:** 3
- **Current:** No shell completion support
- **Proposed:** Add `kf completion bash|zsh|fish` command. Cobra has built-in support via `GenBashCompletionV2()`.
- **Notes:** Nearly free with Cobra. Just needs a root command addition.

### R6. Add `--help` examples to all commands
- **Effort:** S | **Impact:** 2
- **Current:** Help text shows flags but no usage examples
- **Proposed:** Add `Example:` field to Cobra command definitions with common use cases.

---

## Dashboard Improvements

### R7. Add keyboard shortcuts and command palette
- **Effort:** M | **Impact:** 4
- **Current:** Mouse-only navigation
- **Proposed:** Global command palette (Cmd+K), keyboard shortcuts for common actions (new agent, stop agent, navigate pages). Use a library like `cmdk` or build custom.
- **Notes:** High value for power users who spend significant time in the dashboard.

### R8. Add responsive/mobile layout
- **Effort:** L | **Impact:** 3
- **Current:** Desktop-only CSS
- **Proposed:** Responsive breakpoints for tablet and mobile. Key: ability to monitor agent status and read logs from mobile.
- **Notes:** Focus on read-only monitoring first. Agent spawning can remain desktop-only.

### R9. Enhance TrackDetail page
- **Effort:** M | **Impact:** 3
- **Current:** Read-only spec/plan display
- **Proposed:** Add progress visualization (task completion bar), agent assignment display, time estimates, and inline editing of spec/plan fields.
- **Notes:** Makes the dashboard a full track management interface.

### R10. Add search/filtering across all list views
- **Effort:** M | **Impact:** 4
- **Current:** Limited filtering
- **Proposed:** Global search bar, per-column filtering in tables, saved filter presets.
- **Notes:** Depends on R1/R2 (API pagination/filtering) for server-side support.

---

## Orchestration Improvements

### R11. Add agent timeout enforcement
- **Effort:** S | **Impact:** 4
- **Current:** Agents run indefinitely
- **Proposed:** Configurable `max_duration` per agent spawn. Default: 2 hours. Auto-stop with notification on timeout.
- **Notes:** Prevents runaway agents from consuming tokens and resources.

### R12. Add automatic retry for failed agents
- **Effort:** M | **Impact:** 4
- **Current:** Failed agents require manual restart
- **Proposed:** Configurable retry policy: `max_retries`, `backoff`, `retry_on` (specific exit codes). Queue re-enqueues failed work items.
- **Notes:** Critical for unattended operation. Exponential backoff prevents API rate limit issues.

### R13. Improve trace visualization
- **Effort:** M | **Impact:** 3
- **Current:** Basic span timeline
- **Proposed:** Filterable span tree, search within spans, zoom/pan, span detail panel, duration breakdown by service.
- **Notes:** OpenTelemetry data is already collected; this is a visualization improvement.

---

## Skills Improvements

### R14. Remove deprecated kf-parallel skill
- **Effort:** S | **Impact:** 2
- **Current:** Deprecated skill still ships, redirects to kf-architect/kf-developer
- **Proposed:** Remove the skill entirely. Update any references.

### R15. Add skill versioning
- **Effort:** M | **Impact:** 3
- **Current:** All skills update atomically
- **Proposed:** Version tag per skill. Projects can pin skill versions. `kf skills update --skill=kf-developer --version=2.1` pattern.
- **Notes:** Enables safe rollouts and per-project customization.

---

## Testing Improvements

### R16. Add E2E tests to CI pipeline
- **Effort:** S | **Impact:** 4
- **Current:** 21 E2E specs exist but aren't in CI
- **Proposed:** Add headless Playwright step to CI. Use the mock agent binary for deterministic behavior.
- **Notes:** The mock agent infrastructure already exists. Just needs a CI job definition and Playwright setup action.

### R17. Add Go test coverage reporting
- **Effort:** S | **Impact:** 3
- **Current:** Coverage generated but not tracked over time
- **Proposed:** Upload coverage to a service (Codecov, Coveralls) or track in CI artifacts. Set minimum thresholds.

---

## Documentation Improvements

### R18. Generate API reference from OpenAPI spec
- **Effort:** S | **Impact:** 4
- **Current:** OpenAPI spec exists but no published docs
- **Proposed:** Use Redoc, Swagger UI, or Scalar to generate browseable API docs. Can be embedded in the dashboard or served as a static page.
- **Notes:** The spec already exists and is verified in CI. This is low-hanging fruit.

### R19. Write comprehensive user guide
- **Effort:** M | **Impact:** 4
- **Current:** Getting-started.md (138 lines) + scattered docs
- **Proposed:** Full user guide covering: installation, first project, agent workflow, dashboard usage, skill customization, troubleshooting.
- **Notes:** Can be auto-generated from CLI help + API spec + skill descriptions as a starting point.

### R20. Add changelog
- **Effort:** S | **Impact:** 2
- **Current:** No changelog
- **Proposed:** CHANGELOG.md following Keep a Changelog format. Can be auto-generated from conventional commits.
