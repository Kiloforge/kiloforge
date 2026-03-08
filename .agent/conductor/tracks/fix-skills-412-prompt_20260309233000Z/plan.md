# Implementation Plan: Handle 412 Skills Missing with Install Prompt Instead of Error

**Track ID:** fix-skills-412-prompt_20260309233000Z

## Phase 1: Skills Install Prompt Hook

- [ ] Task 1.1: Create `useSkillsPrompt` hook (or extend `useSkillsStatus`) with `requestInstall(onComplete)` pattern — shows dialog, triggers update, calls callback on success
- [ ] Task 1.2: Create `SkillsInstallDialog` component — modal prompting user to install missing skills with Install/Cancel buttons and progress state

## Phase 2: Wire 412 Handling

- [ ] Task 2.1: Update `App.tsx` `spawnMutation.onError` to handle 412 — call `skillsPrompt.requestInstall(() => spawnMutation.mutate())`
- [ ] Task 2.2: Update `ProjectPage.tsx` `generateMutation.onError` to handle 412 — same pattern
- [ ] Task 2.3: Render `SkillsInstallDialog` in App.tsx (alongside ConsentDialog)

## Phase 3: Verification

- [ ] Task 3.1: `npm run build` succeeds
- [ ] Task 3.2: 412 shows install prompt instead of error toast
- [ ] Task 3.3: After installing, original action retries automatically
