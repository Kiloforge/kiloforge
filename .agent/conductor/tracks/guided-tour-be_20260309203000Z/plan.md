# Implementation Plan: Guided Tour State API and Demo Seed Data (Backend)

**Track ID:** guided-tour-be_20260309203000Z

## Phase 1: Tour State Persistence

- [x] Task 1.1: Create `backend/internal/adapter/persistence/sqlite/tour_store.go` — `TourState` struct, `GetTourState()`, `UpdateTourState()` methods on existing SQLite store
- [x] Task 1.2: Write tests for tour store — default pending state, update transitions, JSON round-trip
- [x] Task 1.3: Wire `TourStore` interface into port layer if needed (or keep as concrete store method)

## Phase 2: REST API Endpoints

- [x] Task 2.1: Add `GET /api/tour` handler — returns current tour state (default: `{"status":"pending"}`)
- [x] Task 2.2: Add `PUT /api/tour` handler — accepts state updates (accept, advance, dismiss, complete)
- [x] Task 2.3: Add `GET /api/tour/demo-board` handler — returns hardcoded simulated board with 3 demo tracks
- [x] Task 2.4: Register routes in server setup

## Phase 3: Verification

- [x] Task 3.1: `go test ./...` passes
- [x] Task 3.2: Manual test — curl endpoints, verify state persistence across restarts
