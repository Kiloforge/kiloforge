# Implementation Plan: Add Apache 2.0 License and Upstream Attribution

**Track ID:** licensing_20260309113000Z

## Phase 1: License Files

- [ ] Task 1.1: Create `LICENSE` at repo root with full Apache License 2.0 text, correct copyright year and holder
- [ ] Task 1.2: Create `NOTICE` at repo root with Kiloforge copyright and gemini-conductor MIT attribution (include full MIT license text)

## Phase 2: Update References

- [ ] Task 2.1: Update `README.md` — replace "MIT" license section with Apache 2.0 reference
- [ ] Task 2.2: Add `"license": "Apache-2.0"` to `frontend/package.json`

## Phase 3: Verification

- [ ] Task 3.1: Verify LICENSE file is valid Apache 2.0 text
- [ ] Task 3.2: Verify NOTICE file includes complete gemini-conductor MIT license text
- [ ] Task 3.3: Verify `make build` succeeds (no broken references)
