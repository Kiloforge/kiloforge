# Workflow

## TDD Policy

**Strict** — Tests must be written before implementation. Every task follows the red-green-refactor cycle:

1. Write a failing test that defines the expected behavior
2. Write the minimum implementation to make the test pass
3. Refactor while keeping tests green

## Commit Strategy

**Conventional Commits** — All commits must follow the conventional commits specification:

- `feat:` — New feature
- `fix:` — Bug fix
- `refactor:` — Code restructuring without behavior change
- `test:` — Adding or updating tests
- `docs:` — Documentation changes
- `chore:` — Build, tooling, or dependency changes

Format: `<type>(<optional scope>): <description>`

## Code Review

**Optional / Self-review OK** — Code review is not required for changes. Contributors may self-review and merge.

## Verification Commands

The following commands must pass at phase completion and before merge (post-rebase):

```bash
make test    # Go unit/integration tests
make build   # Full build (frontend + backend) — catches TS errors, embed failures, etc.
```

Both commands must succeed. A build failure is a blocking error.

## Verification Checkpoints

**Track completion only** — Manual verification is required only when an entire track is complete. Individual phases and tasks do not require manual sign-off.

## Task Lifecycle

1. **Pending** — Task defined but not started
2. **In Progress** — Actively being worked on
3. **Testing** — Implementation complete, tests being written/verified
4. **Complete** — All tests pass, code reviewed (if applicable)
5. **Blocked** — Cannot proceed, dependency or issue identified
