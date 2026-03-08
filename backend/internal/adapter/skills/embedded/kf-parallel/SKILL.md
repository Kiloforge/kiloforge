---
name: kf-parallel
description: "DEPRECATED: Use /kf-track-generator to create tracks or /kf-developer <track-id> to implement them. This skill now redirects to the appropriate role."
metadata:
  argument-hint: "[track-generator | developer <track-id>]"
---

# Kiloforge Parallel (Redirector)

This skill has been split into two explicit roles. Use them directly:

## Track Generation (was: Coordinator)

```
/kf-track-generator <prompt>
```

Research the codebase and generate tracks from a feature request or change description. Handles scoping, BE/FE splitting, and approval before creating track artifacts.

**Workflow:** prompt => codebase research => track generation => review => approval => commit

## Track Implementation (was: Worker)

```
/kf-developer <track-id>
```

Claim and implement a specific track in a parallel worktree. Validates the track is active and unclaimed, then runs the full implementation cycle: branch, implement, verify, merge.

**Workflow:** validate track => branch => implement => verify => pause => merge

---

## If invoked directly

If the user invokes `/kf-parallel` without specifying a role:

```
kf-parallel has been split into two explicit roles:

  /kf-track-generator <prompt>   — Generate tracks from a feature description
  /kf-developer <track-id>       — Implement an existing track

Which would you like to use?
```

If an argument looks like a track ID (contains `_` and a timestamp pattern), suggest `/kf-developer`.
If an argument is free-form text, suggest `/kf-track-generator`.
