# Specification: Disable Agent Actions Until Setup Complete (Frontend)

**Track ID:** setup-gate-fe_20260310234711Z
**Type:** Feature
**Created:** 2026-03-10T23:47:11Z
**Status:** Draft

## Summary

Proactively disable all agent-requiring actions on the ProjectPage when kiloforge setup is incomplete. Currently the UI lets users click "Generate Tracks", admin operation buttons, and board sync — only to get a 428 error dialog after the fact. Instead, these buttons should be visually disabled with a tooltip explaining why, and missing 428 handlers should be added as a safety net.

## Context

The setup prerequisite system returns `setup_complete: false` from `GET /api/projects/{slug}/setup-status`, and shows a setup banner on ProjectPage. But the action buttons below the banner are still fully clickable. Users can trigger agent operations that will inevitably fail with 428. The UX should prevent the action proactively.

Additionally, the interactive agent spawn in `App.tsx` and admin operations in `AdminPanel.tsx` are missing 428 error handlers — they silently fail if setup is required.

## Codebase Analysis

### ProjectPage (`pages/ProjectPage.tsx`)

Already queries `setupStatus` (lines 39-46) and shows a banner when incomplete (lines 168-183). Action buttons that should be disabled:

1. **"Generate Tracks" button** (line 214-219) — opens prompt form
2. **"Sync" board button** (line 206-212) — triggers board sync
3. **Admin operation buttons** — rendered by `AdminPanel` component (line 267-271)

### AdminPanel (`components/AdminPanel.tsx`)

- Mutation on line 26: POSTs to `/api/admin/run`
- Error handler (line 38): Only handles 403 (consent), missing 412 (skills) and 428 (setup)
- Currently receives `projectSlug` and `running` props — no `disabled` prop

### App.tsx interactive agent spawn

- `spawnMutation` (lines 84-97): Handles 403 and 412 but NOT 428
- The spawn button is in OverviewPage, not ProjectPage — it doesn't have project context
- However, if project is later passed to spawn, 428 should be handled

### SetupRequiredDialog

Already exists and works well — used by GenerateTracks 428 handler on ProjectPage.

### useSetupPrompt hook

Already exists at `hooks/useSetupPrompt.ts` — provides `requestSetup()`, `startSetup()`, `handleSetupComplete()`, `cancel()`.

## Acceptance Criteria

- [ ] "Generate Tracks" button is disabled when `setup_complete === false` — shows tooltip "Run setup first"
- [ ] "Sync" board button is disabled when `setup_complete === false`
- [ ] Admin operation buttons are disabled when `setup_complete === false` — AdminPanel receives `disabled` prop
- [ ] Prompt textarea form area does not appear when setup is incomplete (Generate Tracks button shows tooltip instead of opening form)
- [ ] AdminPanel 428 error handler added — shows SetupRequiredDialog on 428 response
- [ ] AdminPanel 412 error handler added — shows SkillsInstallDialog on 412 response
- [ ] All disabled buttons have a visual disabled state (reduced opacity, no-cursor) and title/tooltip
- [ ] After setup completes (dialog closes), `setupStatus` query is invalidated and buttons become enabled
- [ ] Frontend builds without errors (`npm run build`)

## Dependencies

- **setup-gate-be_20260310234710Z** — Backend must return 428 from `RunAdminOperation` for the frontend handler to catch it

## Blockers

None.

## Conflict Risk

- LOW — modifies `ProjectPage.tsx` (button disabled states), `AdminPanel.tsx` (new props + error handlers). The pending `agent-display-ttl-fe` track modifies `OverviewPage.tsx` and `AgentHistogram.tsx`, not these files.

## Out of Scope

- Disabling the "Start Interactive Agent" button on OverviewPage — that button has no project context and interactive agents can run without a project
- Disabling project-level actions outside the ProjectPage (track list, etc.) — read-only views don't require setup

## Technical Notes

### ProjectPage disabled state

Derive a `setupIncomplete` boolean and pass it down:

```tsx
const setupIncomplete = setupStatus !== undefined && !setupStatus.setup_complete;
```

Apply to buttons:

```tsx
<button
  className={styles.generateBtn}
  onClick={() => !setupIncomplete && setShowPrompt((v) => !v)}
  disabled={setupIncomplete}
  title={setupIncomplete ? "Run kiloforge setup first" : undefined}
>
  Generate Tracks
</button>

<button
  className={styles.syncBtn}
  onClick={syncBoard}
  disabled={syncing || setupIncomplete}
  title={setupIncomplete ? "Run kiloforge setup first" : undefined}
>
  {syncing ? "Syncing..." : "Sync"}
</button>
```

### AdminPanel disabled prop

Add `disabled?: boolean` to AdminPanel props. Each operation button:

```tsx
<button disabled={disabled || running} title={disabled ? "Run kiloforge setup first" : undefined}>
```

Add 428 and 412 error handling to the mutation:

```tsx
onError: (err, op) => {
  if (err instanceof FetchError && err.status === 403) {
    consent.requestConsent(() => handleRun(op));
  } else if (err instanceof FetchError && err.status === 412) {
    skillsPrompt.requestInstall(() => handleRun(op));
  } else if (err instanceof FetchError && err.status === 428) {
    setupPrompt.requestSetup(projectSlug ?? "", () => handleRun(op));
  }
}
```

AdminPanel will need to receive the consent, skillsPrompt, and setupPrompt hook instances (or callbacks) from ProjectPage.

### Prevent prompt form when disabled

Guard `setShowPrompt` so clicking Generate Tracks while disabled is a no-op:

```tsx
onClick={() => { if (!setupIncomplete) setShowPrompt((v) => !v); }}
```

---

_Generated by kf-architect from prompt: "All the actions requiring agents in a project should be disabled if setup has not been done yet"_
