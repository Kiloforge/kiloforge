# Implementation Plan: Auto-Initialize Conductor Artifacts Before Track Generation

**Track ID:** fix-track-gen-project-init_20260309235500Z

## Phase 1: Update Track Generator Skill or API

- [ ] Task 1.1: Decide approach — modify skill SKILL.md to auto-setup, OR modify GenerateTracks API to chain `/kf-setup` when artifacts missing
- [ ] Task 1.2: Implement chosen approach
- [ ] Task 1.3: If modifying skill: change Step 1 from HALT to auto-initialize with minimal artifacts
- [ ] Task 1.4: If modifying API: add conductor artifact existence check before spawning, prefix prompt with setup if needed

## Phase 2: Improve Error Visibility

- [ ] Task 2.1: Ensure the agent terminal shows meaningful output (not just "exit 0")
- [ ] Task 2.2: If the agent fails setup, surface the error in the terminal output

## Phase 3: Verification

- [ ] Task 3.1: `make test` passes
- [ ] Task 3.2: Track generation on a fresh project (no conductor artifacts) works — setup runs automatically, then track generation proceeds
- [ ] Task 3.3: Track generation on an initialized project still works normally
