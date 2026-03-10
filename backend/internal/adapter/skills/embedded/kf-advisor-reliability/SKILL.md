---
name: kf-advisor-reliability
description: Audit codebase reliability — testing coverage, linting strictness, type safety, build checks, CI gates, and dependency security. Produces a structured report with severity ratings and generates kiloforge tracks for improvements.
metadata:
  argument-hint: "[--full | --testing | --linting | --security | --ci] [--fix-small] [--generate-tracks]"
---

# Kiloforge Reliability Advisor

Systematic reliability audit for the kiloforge project. Identifies gaps in testing, linting, type safety, build-time checks, CI gates, and dependency security. Produces a structured report and optionally generates improvement tracks.

## Use this skill when

- After major features land and you want to verify reliability coverage
- When reliability concerns arise (test failures, missed regressions, type errors in production)
- Periodically (e.g., weekly) to maintain quality baselines
- When onboarding a new project to establish reliability standards

## Do not use this skill when

- You need to implement fixes directly (use `/kf-developer` with generated tracks)
- The project has no Kiloforge artifacts (use `/kf-setup` first)

---

## After Compaction

When entering the reliability advisor role, output this anchor line exactly:

```
ACTIVE ROLE: kf-reliability — skill at ~/.claude/skills/kf-reliability/SKILL.md
```

---

## Arguments

| Flag | Effect |
|------|--------|
| (none) | Full audit across all dimensions |
| `--testing` | Audit only testing dimensions |
| `--linting` | Audit only linting and static analysis |
| `--security` | Audit only dependency security and secrets |
| `--ci` | Audit only CI/CD pipeline gaps |
| `--full` | Full audit (same as no flags) |
| `--fix-small` | Apply small fixes directly (add missing lint config, etc.) |
| `--generate-tracks` | Create kiloforge tracks for larger improvements via kf-architect |

---

## Phase 1: Pre-flight

### Step 1 — Verify project context

```bash
# Check kiloforge is initialized
ls .agent/kf/product.md .agent/kf/tech-stack.md .agent/kf/workflow.md 2>/dev/null
```

If missing, suggest `/kf-setup` and **HALT**.

### Step 2 — Load project context

Read:
- `.agent/kf/product.md` — understand the product
- `.agent/kf/tech-stack.md` — know the tech stack and tools
- `.agent/kf/workflow.md` — understand verification commands and TDD policy
- `.agent/kf/code_styleguides/` — existing style conventions

### Step 3 — Check audit history

```bash
ls .agent/kf/_reports/reliability-*.md 2>/dev/null
```

If previous audits exist, note the last audit date and key findings for trend comparison.

---

## Phase 2: Audit

Systematically check each reliability dimension. For each check, record:
- **Status**: PASS / WARN / FAIL
- **Severity**: critical / high / medium / low
- **Finding**: What was found
- **Recommendation**: What should be done

### Dimension 1: Testing Coverage

#### 1a. Go test tiers

```bash
# Count test files by tier
find backend -name '*_test.go' | wc -l
grep -rl '//go:build e2e' backend/ | wc -l
grep -rl '//go:build integration' backend/ | wc -l

# Check for packages without tests
for pkg in $(find backend -name '*.go' ! -name '*_test.go' -exec dirname {} \; | sort -u); do
  if ! ls "$pkg"/*_test.go >/dev/null 2>&1; then
    echo "NO TESTS: $pkg"
  fi
done
```

Check:
- [ ] Unit tests exist for all packages with business logic
- [ ] E2E tests cover critical user flows
- [ ] Integration tests cover adapter interactions
- [ ] Table-driven tests used for multi-case scenarios
- [ ] Error paths tested (not just happy paths)

#### 1b. Frontend tests

```bash
# Count test files
find frontend/src -name '*.test.*' -o -name '*.spec.*' | wc -l
find frontend/e2e -name '*.spec.*' | wc -l

# Check for components without tests
for comp in $(find frontend/src -name '*.tsx' ! -name '*.test.*' ! -name '*.spec.*' -exec dirname {} \; | sort -u); do
  if ! ls "$comp"/*.test.* "$comp"/*.spec.* >/dev/null 2>&1; then
    echo "NO TESTS: $comp"
  fi
done
```

Check:
- [ ] Component tests exist for UI logic
- [ ] Hook tests exist for custom hooks
- [ ] E2E tests cover critical user journeys
- [ ] API integration tests validate response shapes

#### 1c. Coverage thresholds

Check:
- [ ] Go test coverage reporting enabled (`go test -coverprofile`)
- [ ] Frontend coverage reporting enabled (vitest coverage)
- [ ] Coverage thresholds enforced in CI
- [ ] Critical packages have >80% coverage

#### 1d. API contract tests

Check:
- [ ] OpenAPI schema validated against actual responses
- [ ] `make verify-codegen` catches schema drift
- [ ] Request/response validation in tests

### Dimension 2: Linting and Static Analysis

#### 2a. Go linting

```bash
# Check for golangci-lint config
ls .golangci.yaml .golangci.yml backend/.golangci.yaml 2>/dev/null

# Check which linters are enabled
cat .golangci.yaml 2>/dev/null || echo "NO CONFIG"
```

Check:
- [ ] `.golangci.yaml` exists with explicit linter configuration
- [ ] Security linters enabled (gosec)
- [ ] Error handling linters enabled (errorlint, wrapcheck)
- [ ] Style linters enabled (gocritic, goimports, revive)
- [ ] Cyclomatic complexity limits configured
- [ ] golangci-lint runs in CI

#### 2b. Frontend linting

```bash
# Check ESLint config
cat frontend/.eslintrc* frontend/eslint.config* 2>/dev/null | head -50
cat frontend/package.json | grep -A5 '"eslint'
```

Check:
- [ ] ESLint configured with TypeScript plugin
- [ ] React hooks linting enabled
- [ ] Import ordering enforced
- [ ] No unused variables/imports rule enabled
- [ ] Accessibility linting (eslint-plugin-jsx-a11y)
- [ ] ESLint runs in CI

#### 2c. Type safety

Check:
- [ ] TypeScript strict mode enabled (`"strict": true`)
- [ ] No `any` types in business logic
- [ ] Go interfaces used for dependency injection
- [ ] Generated types used for API responses (not manual types)

### Dimension 3: Build-time Checks

```bash
# Check Makefile targets
grep -E '^[a-zA-Z_-]+:' Makefile backend/Makefile 2>/dev/null
```

Check:
- [ ] `make build` catches all compilation errors
- [ ] `make verify-codegen` detects OpenAPI drift
- [ ] Frontend embed validation prevents stale bundles
- [ ] VCS stamping works in worktree environments
- [ ] Build fails on warnings (not just errors)

### Dimension 4: CI/CD Gates

```bash
# Check CI configuration
cat .github/workflows/*.yml 2>/dev/null | head -100
```

Check:
- [ ] Tests run on PR and push
- [ ] Multiple OS matrix (ubuntu, macOS)
- [ ] Go race detector enabled (`-race`)
- [ ] Build verification included
- [ ] Lint step included
- [ ] Coverage reporting
- [ ] Dependency audit step (`go mod verify`, `npm audit`)
- [ ] Branch protection rules configured

### Dimension 5: Dependency Security

```bash
# Go dependencies
cd backend && go list -m all | wc -l
go mod verify 2>&1

# Frontend dependencies
cd frontend && npm audit --audit-level=moderate 2>&1 | tail -5
```

Check:
- [ ] `go mod verify` passes (no tampered modules)
- [ ] `npm audit` has no critical/high vulnerabilities
- [ ] Dependencies are pinned (go.sum, package-lock.json)
- [ ] No known-vulnerable dependencies
- [ ] License compliance checked

### Dimension 6: Error Handling and Resilience

Check (via code sampling):
- [ ] Errors wrapped with context (`fmt.Errorf("%w", err)`)
- [ ] No swallowed errors (empty error checks)
- [ ] Timeouts on HTTP calls and database operations
- [ ] Graceful shutdown handling
- [ ] Resource cleanup (defer close, context cancellation)

---

## Phase 3: Analysis

### Step 1 — Compile findings

Aggregate all findings from Phase 2 into a structured report:

```markdown
# Reliability Audit Report

**Date:** {YYYY-MM-DD}
**Project:** {project name}
**Auditor:** kf-reliability

## Summary

| Dimension | Checks | Pass | Warn | Fail |
|-----------|--------|------|------|------|
| Testing   | N      | N    | N    | N    |
| Linting   | N      | N    | N    | N    |
| Build     | N      | N    | N    | N    |
| CI/CD     | N      | N    | N    | N    |
| Security  | N      | N    | N    | N    |
| Errors    | N      | N    | N    | N    |

**Overall Score:** {X}/{total} ({percentage}%)
**Previous Score:** {if available}
**Trend:** {improving/stable/declining}

## Critical Findings

{List FAIL items with severity=critical or high}

## Warnings

{List WARN items}

## Passing

{List PASS items — what's already good}

## Recommendations

{Prioritized list of improvements, ordered by impact/effort ratio}
```

### Step 2 — Prioritize by impact

Rank findings by:
1. **Critical** — Security vulnerabilities, missing error handling in core paths
2. **High** — Missing tests for core business logic, no lint config
3. **Medium** — Missing CI gates, incomplete coverage
4. **Low** — Style improvements, nice-to-have checks

### Step 3 — Group related items

Combine related findings into logical improvement units that map to tracks:
- "Add golangci-lint strict configuration" (linting findings)
- "Add test coverage for service layer" (testing findings)
- "Harden CI pipeline" (CI/CD findings)

---

## Phase 4: Recommendations

### Step 1 — Classify actions

For each grouped finding, classify as:

| Size | Action | Example |
|------|--------|---------|
| Small | Fix directly | Add `.golangci.yaml`, add missing `defer Close()` |
| Medium | Generate track | Add tests for untested packages |
| Large | Generate track | Implement API contract testing framework |

### Step 2 — Apply small fixes (if `--fix-small`)

If the `--fix-small` flag was provided, apply small fixes directly:
- Create/update lint config files
- Add missing `defer` cleanup calls
- Fix obvious error handling gaps
- Add missing build tags

Commit each fix:
```bash
git add <files>
git commit -m "fix(<scope>): <description from finding>"
```

### Step 3 — Generate tracks (if `--generate-tracks`)

If the `--generate-tracks` flag was provided, create kiloforge tracks for medium/large items:

For each improvement group, invoke track creation:
```bash
.agent/kf/bin/kf-track create --type chore --title "<improvement title>"
```

Or describe the tracks and suggest the user run `/kf-architect` with the recommendations.

---

## Phase 5: Report Output

### Step 1 — Write report

Save the structured report:

```bash
mkdir -p .agent/kf/_reports
```

Write to `.agent/kf/_reports/reliability-{YYYY-MM-DD}.md`.

### Step 2 — Display summary

Output a condensed summary to the terminal:

```
================================================================================
                    RELIABILITY AUDIT COMPLETE
================================================================================

Score:     {X}/{total} ({percentage}%)
Trend:     {improving/stable/declining/first audit}

Critical:  {count} findings
High:      {count} findings
Medium:    {count} findings
Low:       {count} findings

Top 3 Recommendations:
  1. {highest impact recommendation}
  2. {second highest}
  3. {third highest}

Report:    .agent/kf/_reports/reliability-{date}.md
{if --fix-small}  Fixes Applied: {count}
{if --generate-tracks}  Tracks Created: {count}
================================================================================
```

---

## Audit Dimension Reference

Quick reference for all checked items across dimensions:

### Testing
- Go unit test coverage per package
- Go E2E test coverage for critical flows
- Go integration test coverage for adapters
- Frontend component test coverage
- Frontend E2E test coverage
- Coverage thresholds in CI
- API contract validation

### Linting
- golangci-lint configuration and strictness
- ESLint configuration and plugins
- TypeScript strict mode
- Import ordering enforcement
- Cyclomatic complexity limits

### Build
- Compilation error detection
- OpenAPI codegen drift detection
- Frontend embed validation
- VCS stamping in worktrees

### CI/CD
- Test execution on PR/push
- Multi-OS matrix
- Race detector
- Coverage reporting
- Dependency auditing
- Branch protection

### Security
- Go module verification
- npm audit status
- Dependency pinning
- License compliance
- Secrets detection (no .env committed)

### Error Handling
- Error wrapping with context
- No swallowed errors
- Timeouts on external calls
- Graceful shutdown
- Resource cleanup patterns
