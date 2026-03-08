---
name: conductor-parallel
description: "DEPRECATED: Use /conductor-track-generator to create tracks or /conductor-developer <track-id> to implement them. This skill now redirects to the appropriate role."
metadata:
  argument-hint: "[track-generator | developer <track-id>]"
---

# Conductor Parallel (Redirector)

This skill has been split into two explicit roles. Use them directly:

## Track Generation (was: Coordinator)

```
/conductor-track-generator <prompt>
```

Research the codebase and generate tracks from a feature request or change description. Handles scoping, BE/FE splitting, and approval before creating track artifacts.

**Workflow:** prompt => codebase research => track generation => review => approval => commit

## Track Implementation (was: Worker)

```
/conductor-developer <track-id>
```

Claim and implement a specific track in a parallel worktree. Validates the track is active and unclaimed, then runs the full implementation cycle: branch, implement, verify, merge.

**Workflow:** validate track => branch => implement => verify => pause => merge

---

## If invoked directly

If the user invokes `/conductor-parallel` without specifying a role:

```
conductor-parallel has been split into two explicit roles:

  /conductor-track-generator <prompt>   — Generate tracks from a feature description
  /conductor-developer <track-id>       — Implement an existing track

Which would you like to use?
```

If an argument looks like a track ID (contains `_` and a timestamp pattern), suggest `/conductor-developer`.
If an argument is free-form text, suggest `/conductor-track-generator`.
