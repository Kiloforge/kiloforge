# Implementation Plan: Move Service-Local Interfaces to Port Layer

**Track ID:** refactor-port-interfaces_20260310041002Z

## Phase 1: Move Existing Service-Local Interfaces to Port

### Task 1.1: Move ProjectGiteaClient to port/
- Create `port/gitea_client.go` (or extend existing `port/gitea_client.go`)
- Move `ProjectGiteaClient` interface from `service/project_service.go` to `port/`
- Update `service/project_service.go` to import from `port`

### Task 1.2: Move ProjectStoreWriter to port/
- Move `ProjectStoreWriter` interface to `port/project_store.go` (merge with existing ProjectStore or create separate)
- Update service imports

### Task 1.3: Move NativeBoardStore to port/
- Create `port/board_store.go`
- Move `NativeBoardStore` interface from `service/board_service.go` to `port/`
- Update service imports

### Task 1.4: Verify Phase 1
- `go test ./internal/core/... -race` passes
- No service files define local interfaces

## Phase 2: Create Port Interfaces for REST Handler Dependencies

### Task 2.1: Create BoardService port interface
- Define `port.BoardService` interface covering methods used by `api_handler.go`
- Ensure `service.NativeBoardService` satisfies the interface
- Add compile-time check: `var _ port.BoardService = (*service.NativeBoardService)(nil)`

### Task 2.2: Create TrackReader port interface
- Define `port.TrackReader` interface for `DiscoverTracks` and `GetTrackDetail`
- Used by both `api_handler.go` and `dashboard/watcher.go`
- Add compile-time check

### Task 2.3: Move AddProjectOpts to domain
- Move `service.AddProjectOpts` to `domain.AddProjectOpts` (it's a value object)
- Update all references

### Task 2.4: Verify Phase 2
- `go test ./... -race` passes
- New port interfaces have compile-time satisfaction checks

## Phase 3: Update Adapters to Use Port Interfaces

### Task 3.1: Refactor api_handler.go to use port interfaces
- Change `APIHandler` struct fields from concrete service types to port interfaces
- Remove `import "service"` from `api_handler.go`
- Update `NewAPIHandler()` constructor

### Task 3.2: Refactor dashboard/watcher.go to use port interface
- Change `watcher` to depend on `port.TrackReader` instead of `service.DiscoverTracks`
- Remove `import "service"` from `watcher.go`

### Task 3.3: Update server.go wiring
- `server.go` still imports `service` for construction — this is acceptable (composition root)
- Ensure it passes port-typed values to handlers

### Task 3.4: Verify Phase 3
- Full test suite passes: `make test`
- `api_handler.go` and `watcher.go` have no `service` import
