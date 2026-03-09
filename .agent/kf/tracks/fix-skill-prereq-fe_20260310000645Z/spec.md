# Specification: Fix Skill Prerequisite Chain — Frontend Proactive Gating

**Track ID:** fix-skill-prereq-fe_20260310000645Z
**Type:** Bug
**Created:** 2026-03-10T00:06:45Z
**Status:** Draft

## Summary

The setup-gate-fe implementation only disables agent actions when setup is incomplete, but doesn't account for the skills prerequisite that comes before setup in the chain. The correct order is: **skills installed** → **setup complete** → **agent actions enabled**. The frontend should use the preflight endpoint to detect missing skills and show appropriate messaging — "Install skills first" when skills are missing, "Run kiloforge setup first" when setup is incomplete.

## Context

The prerequisite chain is enforced by the backend:
1. Auth (401) — Claude CLI authenticated
2. Consent (403) — User accepted agent permissions
3. Skills (412) — Required kf-* skills installed
4. Setup (428) — `/kf-setup` has been run for the project

The current setup-gate-fe implementation (merged in `3fa3872`) only checks step 4 via `GET /api/projects/{slug}/setup-status`. It does not proactively check step 3 (skills). Users see disabled buttons with "Run kiloforge setup first" when the real blocker might be missing skills — and running setup would itself fail because setup requires skills.

The backend provides `GET /api/preflight` which returns `skills_ok: boolean` and `skills_missing: string[]`. The frontend should consume this to proactively gate actions with the correct message.

## Codebase Analysis

### Current ProjectPage.tsx

- Lines 39-46: Queries `setupStatus` via `GET /api/projects/{slug}/setup-status`
- Line 48: `const setupIncomplete = setupStatus !== undefined && !setupStatus.setup_complete`
- Lines 211, 218: Buttons disabled when `setupIncomplete`
- Lines 170-185: Setup banner shows when setup incomplete

No preflight query. No skills check. The `disabled` tooltip always says "Run kiloforge setup first".

### Current AdminPanel.tsx

- Line 13: `disabled?: boolean` prop
- Line 85: `title={disabled ? "Run kiloforge setup first" : undefined}` — hardcoded message
- Lines 48-53: Handles 412 and 428 errors reactively via callbacks

### Frontend queryKeys.ts

No `preflight` key defined. Needs adding.

### Backend GET /api/preflight response

```json
{
  "claude_authenticated": true,
  "skills_ok": false,
  "skills_missing": ["kf-architect"],
  "consent_given": true,
  "setup_required": true
}
```

### SetupRequiredDialog

Already exists. Shows banner + "Run Setup" button + agent terminal.

### SkillsInstallDialog

Already exists. Shows skills install prompt.

## Acceptance Criteria

- [ ] Add `preflight` query key to `queryKeys.ts`
- [ ] ProjectPage queries `GET /api/preflight` to get `skills_ok` status
- [ ] Derive `actionsDisabled` from both `!preflight.skills_ok` and `!setupStatus.setup_complete`
- [ ] Derive `disabledReason` — "Install skills first" when skills missing, "Run kiloforge setup first" when setup incomplete
- [ ] All disabled buttons show the correct `disabledReason` as tooltip
- [ ] Setup banner shows skills-missing state when skills are the blocker (with Install button)
- [ ] Setup banner shows setup-incomplete state when skills OK but setup needed (existing behavior)
- [ ] AdminPanel receives `disabledReason` string prop instead of hardcoded tooltip
- [ ] After skills install or setup completes, relevant queries are invalidated and buttons re-enable
- [ ] Frontend builds without errors (`npm run build`)

## Dependencies

- **fix-skill-prereq-be_20260310000644Z** — Backend must correctly populate `skills_ok` in preflight response (fix the `kf-track-generator` → `kf-architect` rename and add `kf-setup` to required list)

## Blockers

None.

## Conflict Risk

- LOW — modifies `ProjectPage.tsx` (add preflight query, update disabled logic), `AdminPanel.tsx` (tooltip prop), `queryKeys.ts` (add key). No pending tracks touch these files.

## Out of Scope

- OverviewPage interactive agent button — has no project context, different gating rules
- Auth and consent proactive checking — these are already handled by existing dialog flows

## Technical Notes

### Preflight query

```tsx
// queryKeys.ts
preflight: ["preflight"] as const,

// ProjectPage.tsx
const { data: preflight } = useQuery({
  queryKey: queryKeys.preflight,
  queryFn: () => fetcher<{
    claude_authenticated: boolean;
    skills_ok: boolean;
    skills_missing?: string[];
    consent_given: boolean;
    setup_required: boolean;
  }>("/api/preflight"),
});
```

### Prerequisite state derivation

```tsx
const skillsMissing = preflight !== undefined && !preflight.skills_ok;
const setupIncomplete = !skillsMissing && setupStatus !== undefined && !setupStatus.setup_complete;
const actionsDisabled = skillsMissing || setupIncomplete;
const disabledReason = skillsMissing
  ? "Install skills first"
  : setupIncomplete
  ? "Run kiloforge setup first"
  : undefined;
```

### Setup banner update

```tsx
{skillsMissing && (
  <div className={styles.setupBanner}>
    <span>Required skills not installed.</span>
    <button onClick={() => skillsPrompt.requestInstall(() => {
      queryClient.invalidateQueries({ queryKey: queryKeys.preflight });
    })}>
      Install Skills
    </button>
  </div>
)}
{!skillsMissing && setupIncomplete && slug && (
  <div className={styles.setupBanner}>
    <span>Kiloforge setup required — run setup to configure this project.</span>
    <button onClick={() => setupPrompt.requestSetup(slug, () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.setupStatus(slug) });
    })}>
      Run Setup
    </button>
  </div>
)}
```

### AdminPanel tooltip

Change from hardcoded string to prop:

```tsx
interface Props {
  // ...existing
  disabledReason?: string;
}

// In button:
title={disabled ? disabledReason : undefined}
```

---

_Generated by kf-architect from prompt: "Fix skill prerequisite chain — frontend proactive gating"_
