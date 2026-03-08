---
name: kf-validate
description: Validates Kiloforge project artifacts for completeness,
  consistency, and correctness. Use after setup, when diagnosing issues, or
  before implementation to verify project context.
allowed-tools: Read Glob Grep Bash
metadata:
  model: opus
  color: cyan
---

# Check if kiloforge directory exists

ls -la .agent/conductor/

# Find all track directories

ls -la .agent/conductor/tracks/

# Check for required files

ls .agent/conductor/index.md .agent/conductor/product.md .agent/conductor/tech-stack.md .agent/conductor/workflow.md .agent/conductor/tracks.md

```

## Use this skill when

- Working on check if kiloforge directory exists tasks or workflows
- Needing guidance, best practices, or checklists for check if kiloforge directory exists

## Do not use this skill when

- The task is unrelated to check if kiloforge directory exists
- You need a different domain or tool outside this scope

## Instructions

- Clarify goals, constraints, and required inputs.
- Apply relevant best practices and validate outcomes.
- Provide actionable steps and verification.
- If detailed examples are required, open `resources/implementation-playbook.md`.

## Pattern Matching

**Status markers in tracks.md:**

```

- [ ] Track Name # Not started
- [~] Track Name # In progress
- [x] Track Name # Complete

```

**Task markers in plan.md:**

```

- [ ] Task description # Pending
- [~] Task description # In progress
- [x] Task description # Complete

```

**Track ID pattern:**

```

<type>_<name>_<YYYYMMDDHHmmssZ>
Example: feature_user_auth_20250115100000Z

```

```
