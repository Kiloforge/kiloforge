# Implementation Plan: Move Service-Local Interfaces to Port Layer

**Track ID:** refactor-port-interfaces_20260310041002Z

## Phase 1: Move Existing Service-Local Interfaces to Port

### Task 1.1: Move ProjectGiteaClient to port/
- [x] Create `port/project_gitea_client.go`
- [x] Move `ProjectGiteaClient` interface from `service/project_service.go` to `port/`
- [x] Update `service/project_service.go` to import from `port`

### Task 1.2: Move ProjectStoreWriter to port/
- [x] Update `ProjectService` to use `port.ProjectStore` (superset of `ProjectStoreWriter`)
- [x] Remove local `ProjectStoreWriter` definition
- [x] Update test mocks to satisfy `port.ProjectStore`

### Task 1.3: Move NativeBoardStore to port/
- [x] `port/board_store.go` already exists with identical interface
- [x] Update `service/board_service.go` to use `port.BoardStore`
- [x] Remove local `NativeBoardStore` definition

### Task 1.4: Verify Phase 1
- [x] `go test ./internal/core/... -race` passes
- [x] No service files define local interfaces

## Phase 2: Create Port Interfaces for REST Handler Dependencies

### Task 2.1: Create BoardService port interface
- [x] Define `port.BoardService` interface in `port/board_service.go`
- [x] Add compile-time check: `var _ port.BoardService = (*service.NativeBoardService)(nil)`
- [x] Move `MoveCardResult` and `SyncResult` to port as `BoardMoveCardResult` and `BoardSyncResult`

### Task 2.2: Create TrackReader port interface
- [x] Define `port.TrackReader` interface in `port/track_reader.go`
- [x] Move `TrackEntry`, `TrackDetail`, `ProgressCount` types to port
- [x] Create `service.TrackReaderImpl` satisfying port.TrackReader
- [x] Add compile-time check

### Task 2.3: Move AddProjectOpts to domain
- [x] Move `AddProjectOpts` and `AddProjectResult` to `domain/project.go`
- [x] Update all references in service, CLI, and REST adapter

### Task 2.4: Verify Phase 2
- [x] `make test` passes
- [x] New port interfaces have compile-time satisfaction checks

## Phase 3: Update Adapters to Use Port Interfaces

### Task 3.1: Refactor api_handler.go to use port interfaces
- [x] Change `boardSvc` field from `*service.NativeBoardService` to `port.BoardService`
- [x] Add `trackReader port.TrackReader` field
- [x] Replace `service.DiscoverTracks()` calls with `h.trackReader.DiscoverTracks()`
- [x] Replace `service.GetTrackDetail()` calls with `h.trackReader.GetTrackDetail()`
- [x] Replace `service.AddProjectOpts` with `domain.AddProjectOpts`

### Task 3.2: Refactor dashboard/watcher.go to use port interface
- [x] Add `trackReader port.TrackReader` field to dashboard Server
- [x] Replace `service.DiscoverTracks()` with `s.trackReader.DiscoverTracks()`
- [x] Remove `import "service"` from `watcher.go`

### Task 3.3: Update server.go wiring
- [x] `WithBoardService` accepts `port.BoardService` instead of `*service.NativeBoardService`
- [x] Wire `service.NewTrackReader()` at composition roots
- [x] Add `SetTrackReader()` method to dashboard Server

### Task 3.4: Verify Phase 3
- [x] `make test` passes
- [x] `make build` passes
- [x] `watcher.go` has no `service` import
- [x] `api_handler.go` only imports `service` for error types (deferred to error-standardization track)
