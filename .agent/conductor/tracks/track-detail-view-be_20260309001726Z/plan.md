# Implementation Plan: Track Detail View API

**Track ID:** track-detail-view-be_20260309001726Z

## Phase 1: Domain and Service

- [ ] Task 1.1: Add `TrackDetail` struct to track service with fields: ID, Title, Status, Type, Spec (string), Plan (string), Phases (total/completed), Tasks (total/completed), CreatedAt, UpdatedAt
- [ ] Task 1.2: Add `GetTrackDetail(conductorDir, trackID string) (*TrackDetail, error)` method — reads metadata.json, spec.md, plan.md from `{conductorDir}/tracks/{trackID}/`
- [ ] Task 1.3: Handle missing files gracefully — metadata.json missing falls back to TrackEntry data; spec.md/plan.md missing return empty strings

## Phase 2: REST Endpoint

- [ ] Task 2.1: Add `GetTrackDetail` handler method — resolve project path, call service, return JSON
- [ ] Task 2.2: Register `GET /api/tracks/{trackId}` route with project query param
- [ ] Task 2.3: Return 404 with error message if track directory not found

## Phase 3: OpenAPI and Code Generation

- [ ] Task 3.1: Add `TrackDetail` schema to openapi.yaml with all fields (id, title, status, type, spec, plan, phases, tasks, created_at, updated_at)
- [ ] Task 3.2: Add `GET /api/tracks/{trackId}` path to openapi.yaml with project query parameter and TrackDetail response
- [ ] Task 3.3: Run code generation to update generated types

## Phase 4: Verification

- [ ] Task 4.1: `make build` succeeds with no errors
- [ ] Task 4.2: Manual verification — start server, call endpoint, confirm response structure
