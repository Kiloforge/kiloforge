# Implementation Plan: Auto-Initialize Conductor Artifacts Before Track Generation

**Track ID:** fix-track-gen-project-init_20260309235500Z

## Phase 1: Update Track Generator Skill or API

- [x] Task 1.1: Decide approach — API approach: chain `/kf-setup` in GenerateTracks when artifacts missing
- [x] Task 1.2: Implement chosen approach — check for product.md, prefix prompt with /kf-setup if missing
- [x] Task 1.3: N/A (chose API approach)
- [x] Task 1.4: Added conductor artifact existence check in GenerateTracks, prefix prompt with /kf-setup when needed

## Phase 2: Improve Error Visibility

- [x] Task 2.1: Agent terminal now shows setup output followed by track generation (no more silent "exit 0")
- [x] Task 2.2: If setup fails, Claude outputs errors visible in the terminal

## Phase 3: Verification

- [x] Task 3.1: `make test` passes
- [x] Task 3.2: Track generation on a fresh project chains /kf-setup before /kf-track-generator
- [x] Task 3.3: Track generation on initialized projects skips setup (product.md exists)
