package rest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/rest/gen"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
	"kiloforge/internal/core/service"
)

// --- Consent stubs ---

type stubConsentChecker struct {
	consented bool
	recordErr error
}

func (s *stubConsentChecker) HasAgentPermissionsConsent() bool { return s.consented }
func (s *stubConsentChecker) RecordAgentPermissionsConsent() error {
	if s.recordErr != nil {
		return s.recordErr
	}
	s.consented = true
	return nil
}

// --- Track reader stub ---

type stubTrackReader struct {
	initialized map[string]bool
}

func (s *stubTrackReader) DiscoverTracks(dir string) ([]port.TrackEntry, error) {
	return nil, nil
}
func (s *stubTrackReader) DiscoverTracksPaginated(dir string, opts domain.PageOpts, statuses ...string) (domain.Page[port.TrackEntry], error) {
	return domain.Page[port.TrackEntry]{}, nil
}
func (s *stubTrackReader) GetTrackDetail(dir, trackID string) (*port.TrackDetail, error) {
	return nil, fmt.Errorf("track not found: %s", trackID)
}
func (s *stubTrackReader) IsInitialized(dir string) bool {
	if s.initialized == nil {
		return false
	}
	return s.initialized[dir]
}
func (s *stubTrackReader) RemoveTrack(dir, trackID string) error { return nil }

// --- Board service stub ---

type stubBoardService struct {
	board      *domain.BoardState
	syncResult *port.BoardSyncResult
	syncErr    error
}

func (s *stubBoardService) GetBoard(_ string) (*domain.BoardState, error) {
	if s.board == nil {
		return nil, fmt.Errorf("no board")
	}
	return s.board, nil
}
func (s *stubBoardService) MoveCard(_, _, _ string) (*port.BoardMoveCardResult, error) {
	return &port.BoardMoveCardResult{}, nil
}
func (s *stubBoardService) SyncFromTracks(_ string, _ []port.TrackEntry, _ map[string]string) (*port.BoardSyncResult, error) {
	if s.syncErr != nil {
		return nil, s.syncErr
	}
	if s.syncResult != nil {
		return s.syncResult, nil
	}
	return &port.BoardSyncResult{}, nil
}
func (s *stubBoardService) UpdateCardAgent(_, _, _, _ string) error { return nil }
func (s *stubBoardService) StoreTraceID(_, _, _ string) error       { return nil }
func (s *stubBoardService) GetTraceID(_, _ string) (string, bool)   { return "", false }
func (s *stubBoardService) RemoveCard(_, _ string) (bool, error)    { return false, nil }

// --- Consent Tests ---

func TestGetAgentPermissionsConsent_NilStore(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})

	resp, err := h.GetAgentPermissionsConsent(context.Background(), gen.GetAgentPermissionsConsentRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetAgentPermissionsConsent200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if r.Consented {
		t.Error("expected consented=false when consent store is nil")
	}
}

func TestGetAgentPermissionsConsent_WithConsent(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
		Consent:    &stubConsentChecker{consented: true},
	})

	resp, _ := h.GetAgentPermissionsConsent(context.Background(), gen.GetAgentPermissionsConsentRequestObject{})
	r := resp.(gen.GetAgentPermissionsConsent200JSONResponse)
	if !r.Consented {
		t.Error("expected consented=true")
	}
}

func TestRecordAgentPermissionsConsent_NilStore(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})

	resp, err := h.RecordAgentPermissionsConsent(context.Background(), gen.RecordAgentPermissionsConsentRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.RecordAgentPermissionsConsent500JSONResponse); !ok {
		t.Fatalf("expected 500 when consent store is nil, got %T", resp)
	}
}

func TestRecordAgentPermissionsConsent_Success(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
		Consent:    &stubConsentChecker{consented: false},
	})

	resp, err := h.RecordAgentPermissionsConsent(context.Background(), gen.RecordAgentPermissionsConsentRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.RecordAgentPermissionsConsent200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if !r.Consented {
		t.Error("expected consented=true after recording")
	}
}

// --- checkConsent Tests ---

func TestCheckConsent_NilStore(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{})
	if msg := h.checkConsent(); msg != "" {
		t.Errorf("expected empty when no consent store, got %q", msg)
	}
}

func TestCheckConsent_Consented(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Consent: &stubConsentChecker{consented: true},
	})
	if msg := h.checkConsent(); msg != "" {
		t.Errorf("expected empty when consented, got %q", msg)
	}
}

func TestCheckConsent_NotConsented(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Consent: &stubConsentChecker{consented: false},
	})
	msg := h.checkConsent()
	if msg == "" {
		t.Error("expected non-empty error when not consented")
	}
}

// --- isSetupRequired Tests ---

func TestIsSetupRequired_NilProjects(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{})
	if h.isSetupRequired() {
		t.Error("expected false with nil projects")
	}
}

func TestIsSetupRequired_AllInitialized(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	h := NewAPIHandler(APIHandlerOpts{
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "a", ProjectDir: dir}}},
		TrackReader: &stubTrackReader{initialized: map[string]bool{dir: true}},
	})
	if h.isSetupRequired() {
		t.Error("expected false when all projects initialized")
	}
}

func TestIsSetupRequired_OneUninitialized(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	h := NewAPIHandler(APIHandlerOpts{
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "a", ProjectDir: dir}}},
		TrackReader: &stubTrackReader{initialized: map[string]bool{dir: false}},
	})
	if !h.isSetupRequired() {
		t.Error("expected true when a project is not initialized")
	}
}

// --- checkSetup Tests ---

func TestCheckSetup_EmptySlug(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{})
	if slug := h.checkSetup(""); slug != "" {
		t.Errorf("expected empty for empty slug, got %q", slug)
	}
}

func TestCheckSetup_ProjectNotFound(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "other"}}},
		TrackReader: &stubTrackReader{},
	})
	if slug := h.checkSetup("nonexistent"); slug != "" {
		t.Errorf("expected empty for nonexistent project, got %q", slug)
	}
}

func TestCheckSetup_Initialized(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	h := NewAPIHandler(APIHandlerOpts{
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "myapp", ProjectDir: dir}}},
		TrackReader: &stubTrackReader{initialized: map[string]bool{dir: true}},
	})
	if slug := h.checkSetup("myapp"); slug != "" {
		t.Errorf("expected empty for initialized project, got %q", slug)
	}
}

func TestCheckSetup_NotInitialized(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	h := NewAPIHandler(APIHandlerOpts{
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "myapp", ProjectDir: dir}}},
		TrackReader: &stubTrackReader{initialized: map[string]bool{dir: false}},
	})
	if slug := h.checkSetup("myapp"); slug != "myapp" {
		t.Errorf("expected 'myapp' for uninitialized project, got %q", slug)
	}
}

// --- ListProjects Tests ---

func TestListProjects_NilProjects(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})

	resp, err := h.ListProjects(context.Background(), gen.ListProjectsRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.ListProjects200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if len(r) != 0 {
		t.Errorf("expected 0 projects, got %d", len(r))
	}
}

func TestListProjects_WithProjects(t *testing.T) {
	t.Parallel()
	remote := "git@github.com:user/repo.git"
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
		Projects: &stubProjectLister{projects: []domain.Project{
			{Slug: "proj-1", Active: true, OriginRemote: remote},
			{Slug: "proj-2", Active: false},
		}},
	})

	resp, _ := h.ListProjects(context.Background(), gen.ListProjectsRequestObject{})
	r := resp.(gen.ListProjects200JSONResponse)
	if len(r) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(r))
	}
	if r[0].Slug != "proj-1" {
		t.Errorf("first slug = %q, want %q", r[0].Slug, "proj-1")
	}
	if r[0].OriginRemote == nil || *r[0].OriginRemote != remote {
		t.Errorf("first origin = %v, want %q", r[0].OriginRemote, remote)
	}
	if r[1].Active {
		t.Error("second project should be inactive")
	}
}

// --- Queue/Swarm Tests ---

func TestGetQueue_NilService(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{MaxSwarmSize: 5}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
		Cfg:        cfg,
	})

	resp, err := h.GetQueue(context.Background(), gen.GetQueueRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetQueue200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if r.Running {
		t.Error("expected Running=false when queueSvc is nil")
	}
	if r.MaxWorkers != 5 {
		t.Errorf("MaxWorkers = %d, want 5", r.MaxWorkers)
	}
}

func TestGetSwarmCapacity_NilSpawner(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{MaxSwarmSize: 8}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
		Cfg:        cfg,
	})

	resp, err := h.GetSwarmCapacity(context.Background(), gen.GetSwarmCapacityRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetSwarmCapacity200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if r.Max != 8 {
		t.Errorf("Max = %d, want 8", r.Max)
	}
	if r.Active != 0 {
		t.Errorf("Active = %d, want 0", r.Active)
	}
	if r.Available != 8 {
		t.Errorf("Available = %d, want 8", r.Available)
	}
}

func TestUpdateSwarmSettings_InvalidValue(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgFile, []byte("max_swarm_size: 3\n"), 0o644)
	cfg := &config.Config{MaxSwarmSize: 3, DataDir: dir}

	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
		Cfg:        cfg,
	})

	val := 0
	resp, err := h.UpdateSwarmSettings(context.Background(), gen.UpdateSwarmSettingsRequestObject{
		Body: &gen.UpdateSwarmSettingsJSONRequestBody{MaxSwarmSize: &val},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.UpdateSwarmSettings400JSONResponse); !ok {
		t.Fatalf("expected 400, got %T", resp)
	}
}

func TestUpdateQueueSettings_InvalidValue(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{MaxSwarmSize: 3}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
		Cfg:        cfg,
	})

	val := 0
	resp, err := h.UpdateQueueSettings(context.Background(), gen.UpdateQueueSettingsRequestObject{
		Body: &gen.UpdateQueueSettingsJSONRequestBody{MaxWorkers: &val},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.UpdateQueueSettings400JSONResponse); !ok {
		t.Fatalf("expected 400, got %T", resp)
	}
}

// --- StopAgent Tests ---

func TestStopAgent_NilSpawner(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})

	resp, err := h.StopAgent(context.Background(), gen.StopAgentRequestObject{Id: "agent-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.StopAgent409JSONResponse); !ok {
		t.Fatalf("expected 409 when spawner is nil, got %T", resp)
	}
}

// --- ResumeAgent Tests ---

func TestResumeAgent_NilSpawner(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})

	resp, err := h.ResumeAgent(context.Background(), gen.ResumeAgentRequestObject{Id: "agent-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.ResumeAgent409JSONResponse); !ok {
		t.Fatalf("expected 409 when spawner is nil, got %T", resp)
	}
}

// --- DeleteAgent Tests ---

func TestDeleteAgent_NilRemover(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})

	resp, err := h.DeleteAgent(context.Background(), gen.DeleteAgentRequestObject{Id: "agent-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.DeleteAgent409JSONResponse); !ok {
		t.Fatalf("expected 409 when remover is nil, got %T", resp)
	}
}

// --- SyncBoard Tests ---

func TestSyncBoard_NilBoardSvc(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})

	resp, err := h.SyncBoard(context.Background(), gen.SyncBoardRequestObject{Project: "proj"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.SyncBoard500JSONResponse); !ok {
		t.Fatalf("expected 500 when boardSvc is nil, got %T", resp)
	}
}

func TestSyncBoard_NilProjects(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
		BoardSvc:   &stubBoardService{},
	})

	resp, err := h.SyncBoard(context.Background(), gen.SyncBoardRequestObject{Project: "proj"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.SyncBoard400JSONResponse); !ok {
		t.Fatalf("expected 400 when projects is nil, got %T", resp)
	}
}

// --- GetTrackDetail Tests ---

func TestGetTrackDetail_NilProjects(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})

	proj := "myapp"
	resp, err := h.GetTrackDetail(context.Background(), gen.GetTrackDetailRequestObject{
		TrackId: "track-1",
		Params:  gen.GetTrackDetailParams{Project: proj},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.GetTrackDetail404JSONResponse); !ok {
		t.Fatalf("expected 404, got %T", resp)
	}
}

func TestGetTrackDetail_ProjectNotFound(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:      &stubAgentLister{},
		Quota:       &stubQuotaReader{},
		LockMgr:     lock.New(""),
		SSEClients:  func() int { return 0 },
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "other"}}},
		TrackReader: &stubTrackReader{},
	})

	proj := "nonexistent"
	resp, err := h.GetTrackDetail(context.Background(), gen.GetTrackDetailRequestObject{
		TrackId: "track-1",
		Params:  gen.GetTrackDetailParams{Project: proj},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.GetTrackDetail404JSONResponse); !ok {
		t.Fatalf("expected 404, got %T", resp)
	}
}

// --- DeleteTrack Tests ---

func TestDeleteTrack_ProjectNotFound(t *testing.T) {
	t.Parallel()
	proj := "nonexistent"
	h := NewAPIHandler(APIHandlerOpts{
		Agents:      &stubAgentLister{},
		Quota:       &stubQuotaReader{},
		LockMgr:     lock.New(""),
		SSEClients:  func() int { return 0 },
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "other"}}},
		TrackReader: &stubTrackReader{},
	})

	resp, err := h.DeleteTrack(context.Background(), gen.DeleteTrackRequestObject{
		TrackId: "track-1",
		Params:  gen.DeleteTrackParams{Project: &proj},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.DeleteTrack404JSONResponse); !ok {
		t.Fatalf("expected 404, got %T", resp)
	}
}

// --- StartQueue Tests ---

func TestStartQueue_NilService(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})

	resp, err := h.StartQueue(context.Background(), gen.StartQueueRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.StartQueue500JSONResponse); !ok {
		t.Fatalf("expected 500 when queueSvc is nil, got %T", resp)
	}
}

// --- StopQueue Tests ---

func TestStopQueue_NilService(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})

	resp, err := h.StopQueue(context.Background(), gen.StopQueueRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.StopQueue409JSONResponse); !ok {
		t.Fatalf("expected 409 when queueSvc is nil, got %T", resp)
	}
}

// --- SpawnInteractiveAgent Tests ---

func TestSpawnInteractiveAgent_NilSpawner(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})

	resp, err := h.SpawnInteractiveAgent(context.Background(), gen.SpawnInteractiveAgentRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.SpawnInteractiveAgent500JSONResponse); !ok {
		t.Fatalf("expected 500 when spawner is nil, got %T", resp)
	}
}

// --- GetProjectSetupStatus Tests ---

func TestGetProjectSetupStatus_ProjectNotFound(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
		Projects:   &stubProjectLister{projects: []domain.Project{{Slug: "other"}}},
	})

	resp, err := h.GetProjectSetupStatus(context.Background(), gen.GetProjectSetupStatusRequestObject{Slug: "nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.GetProjectSetupStatus404JSONResponse); !ok {
		t.Fatalf("expected 404, got %T", resp)
	}
}

func TestGetProjectSetupStatus_Initialized(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:      &stubAgentLister{},
		Quota:       &stubQuotaReader{},
		LockMgr:     lock.New(""),
		SSEClients:  func() int { return 0 },
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "myapp", ProjectDir: dir}}},
		TrackReader: &stubTrackReader{initialized: map[string]bool{dir: true}},
	})

	resp, err := h.GetProjectSetupStatus(context.Background(), gen.GetProjectSetupStatusRequestObject{Slug: "myapp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetProjectSetupStatus200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if !r.SetupComplete {
		t.Error("expected SetupComplete=true")
	}
}

// --- checkSkillsForRole Tests ---

func TestCheckSkillsForRole_NilCfg(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{})
	result := h.checkSkillsForRole("interactive", "")
	if result != nil {
		t.Error("expected nil when cfg is nil")
	}
}

// --- ListTracks Tests ---

func TestListTracks_NilProjects(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})
	resp, err := h.ListTracks(context.Background(), gen.ListTracksRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.ListTracks200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if len(r.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(r.Items))
	}
}

func TestListTracks_WithProjects(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:      &stubAgentLister{},
		Quota:       &stubQuotaReader{},
		LockMgr:     lock.New(""),
		SSEClients:  func() int { return 0 },
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "myapp", ProjectDir: dir}}},
		TrackReader: &stubTrackReader{},
	})
	resp, err := h.ListTracks(context.Background(), gen.ListTracksRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.ListTracks200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if r.TotalCount != 0 {
		t.Errorf("expected 0 tracks from empty reader, got %d", r.TotalCount)
	}
}

func TestListTracks_WithStatusFilter(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	status := "pending"
	h := NewAPIHandler(APIHandlerOpts{
		Agents:      &stubAgentLister{},
		Quota:       &stubQuotaReader{},
		LockMgr:     lock.New(""),
		SSEClients:  func() int { return 0 },
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "myapp", ProjectDir: dir}}},
		TrackReader: &stubTrackReader{},
	})
	resp, err := h.ListTracks(context.Background(), gen.ListTracksRequestObject{
		Params: gen.ListTracksParams{Status: &status},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.ListTracks200JSONResponse); !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
}

func TestListTracks_WithProjectFilter(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	proj := "other"
	h := NewAPIHandler(APIHandlerOpts{
		Agents:      &stubAgentLister{},
		Quota:       &stubQuotaReader{},
		LockMgr:     lock.New(""),
		SSEClients:  func() int { return 0 },
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "myapp", ProjectDir: dir}}},
		TrackReader: &stubTrackReader{},
	})
	resp, err := h.ListTracks(context.Background(), gen.ListTracksRequestObject{
		Params: gen.ListTracksParams{Project: &proj},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.ListTracks200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	// "other" doesn't match "myapp", so 0 tracks.
	if len(r.Items) != 0 {
		t.Errorf("expected 0 items for non-matching project, got %d", len(r.Items))
	}
}

// --- SyncBoard full-flow Tests ---

func TestSyncBoard_ProjectNotFound(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:      &stubAgentLister{},
		Quota:       &stubQuotaReader{},
		LockMgr:     lock.New(""),
		SSEClients:  func() int { return 0 },
		BoardSvc:    &stubBoardService{},
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "alpha", ProjectDir: "/tmp"}}},
		TrackReader: &stubTrackReader{},
	})
	resp, err := h.SyncBoard(context.Background(), gen.SyncBoardRequestObject{Project: "nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.SyncBoard400JSONResponse); !ok {
		t.Fatalf("expected 400 for unknown project, got %T", resp)
	}
}

func TestSyncBoard_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:      &stubAgentLister{},
		Quota:       &stubQuotaReader{},
		LockMgr:     lock.New(""),
		SSEClients:  func() int { return 0 },
		BoardSvc:    &stubBoardService{syncResult: &port.BoardSyncResult{Created: 2, Updated: 1}},
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "myapp", ProjectDir: dir}}},
		TrackReader: &stubTrackReader{},
	})
	resp, err := h.SyncBoard(context.Background(), gen.SyncBoardRequestObject{Project: "myapp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.SyncBoard200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if r.Created != 2 {
		t.Errorf("Created = %d, want 2", r.Created)
	}
}

func TestSyncBoard_SyncError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:      &stubAgentLister{},
		Quota:       &stubQuotaReader{},
		LockMgr:     lock.New(""),
		SSEClients:  func() int { return 0 },
		BoardSvc:    &stubBoardService{syncErr: fmt.Errorf("sync failed")},
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "myapp", ProjectDir: dir}}},
		TrackReader: &stubTrackReader{},
	})
	resp, err := h.SyncBoard(context.Background(), gen.SyncBoardRequestObject{Project: "myapp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.SyncBoard500JSONResponse); !ok {
		t.Fatalf("expected 500 on sync error, got %T", resp)
	}
}

// --- filterActiveAgents Tests ---

func TestFilterActiveAgents(t *testing.T) {
	t.Parallel()
	now := time.Now()
	cutoff := now.Add(-30 * time.Minute)

	recentFinish := now.Add(-10 * time.Minute)
	oldFinish := now.Add(-60 * time.Minute)

	agents := []domain.AgentInfo{
		{ID: "running", Status: "running"},
		{ID: "recent-done", Status: "completed", FinishedAt: &recentFinish},
		{ID: "old-done", Status: "completed", FinishedAt: &oldFinish},
	}

	result := filterActiveAgents(agents, cutoff)
	if len(result) != 2 {
		t.Fatalf("expected 2 active agents, got %d", len(result))
	}
	if result[0].ID != "running" {
		t.Errorf("first should be running, got %s", result[0].ID)
	}
	if result[1].ID != "recent-done" {
		t.Errorf("second should be recent-done, got %s", result[1].ID)
	}
}

// --- domainBoardToGen Tests ---

func TestDomainBoardToGen(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	board := &domain.BoardState{
		Columns: []string{"pending", "done"},
		Cards: map[string]domain.BoardCard{
			"t-1": {
				TrackID:        "t-1",
				Title:          "Test Card",
				Type:           "feature",
				Column:         "pending",
				Position:       0,
				AgentID:        "agent-1",
				AgentStatus:    "running",
				AssignedWorker: "worker-1",
				PRNumber:       42,
				CreatedAt:      now,
			},
			"t-2": {
				TrackID:   "t-2",
				Title:     "Minimal Card",
				Column:    "done",
				Position:  1,
				CreatedAt: now,
			},
		},
	}

	result := domainBoardToGen(board)
	if len(result.Cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(result.Cards))
	}
	card1 := result.Cards["t-1"]
	if card1.TrackId != "t-1" {
		t.Errorf("TrackId = %q, want t-1", card1.TrackId)
	}
	if card1.AgentId == nil || *card1.AgentId != "agent-1" {
		t.Error("AgentId should be set to agent-1")
	}
	if card1.PrNumber == nil || *card1.PrNumber != 42 {
		t.Error("PrNumber should be 42")
	}
	if card1.Type == nil || *card1.Type != "feature" {
		t.Error("Type should be feature")
	}
	if card1.AssignedWorker == nil || *card1.AssignedWorker != "worker-1" {
		t.Error("AssignedWorker should be worker-1")
	}

	// Minimal card should have nil optionals.
	card2 := result.Cards["t-2"]
	if card2.AgentId != nil {
		t.Error("minimal card AgentId should be nil")
	}
	if card2.PrNumber != nil {
		t.Error("minimal card PrNumber should be nil")
	}

	if len(result.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(result.Columns))
	}
}

// --- domainAgentToGen Tests ---

func TestDomainAgentToGen_AllFields(t *testing.T) {
	t.Parallel()
	now := time.Now()
	finished := now.Add(time.Hour)
	suspended := now.Add(30 * time.Minute)
	a := domain.AgentInfo{
		ID:             "agent-1",
		Role:           "developer",
		Ref:            "ref-abc",
		Status:         "completed",
		PID:            12345,
		SessionID:      "sess-1",
		WorktreeDir:    "/tmp/wt",
		LogFile:        "/tmp/log",
		SuspendedAt:    &suspended,
		FinishedAt:     &finished,
		ShutdownReason: "done",
		Model:          "claude-4",
		StartedAt:      now,
		UpdatedAt:      now,
	}

	g := domainAgentToGen(a, nil)
	if g.Id != "agent-1" {
		t.Errorf("Id = %q, want agent-1", g.Id)
	}
	if g.SessionId == nil || *g.SessionId != "sess-1" {
		t.Error("SessionId should be set")
	}
	if g.WorktreeDir == nil || *g.WorktreeDir != "/tmp/wt" {
		t.Error("WorktreeDir should be set")
	}
	if g.LogFile == nil || *g.LogFile != "/tmp/log" {
		t.Error("LogFile should be set")
	}
	if g.SuspendedAt == nil {
		t.Error("SuspendedAt should be set")
	}
	if g.FinishedAt == nil {
		t.Error("FinishedAt should be set")
	}
	if g.ShutdownReason == nil || *g.ShutdownReason != "done" {
		t.Error("ShutdownReason should be set")
	}
	if g.Model == nil || *g.Model != "claude-4" {
		t.Error("Model should be set")
	}
	if g.UptimeSeconds == nil {
		t.Error("UptimeSeconds should be set")
	}
}

// --- toGenQueueStatus Tests ---

func TestToGenQueueStatus(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	assigned := now.Add(time.Minute)
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})

	s := &service.QueueStatus{
		Running:       true,
		MaxWorkers:    4,
		ActiveWorkers: 2,
		Items: []domain.QueueItem{
			{TrackID: "t-1", ProjectSlug: "proj", Status: "queued", EnqueuedAt: now},
			{TrackID: "t-2", ProjectSlug: "proj", Status: "assigned", EnqueuedAt: now, AgentID: "agent-1", AssignedAt: &assigned},
		},
	}

	result := h.toGenQueueStatus(s)
	if !result.Running {
		t.Error("Running should be true")
	}
	if result.MaxWorkers != 4 {
		t.Errorf("MaxWorkers = %d, want 4", result.MaxWorkers)
	}
	if result.TotalItems != 2 {
		t.Errorf("TotalItems = %d, want 2", result.TotalItems)
	}
	if result.Items[1].AgentId == nil || *result.Items[1].AgentId != "agent-1" {
		t.Error("second item AgentId should be agent-1")
	}
	if result.Items[0].AgentId != nil {
		t.Error("first item AgentId should be nil")
	}
}

// --- GetPreflight Tests ---

func TestGetPreflight_NilConsent(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})
	resp, err := h.GetPreflight(context.Background(), gen.GetPreflightRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetPreflight200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	// With nil consent, consent should default to true.
	if !r.ConsentGiven {
		t.Error("ConsentGiven should be true when consent store is nil")
	}
	if !r.SkillsOk {
		t.Error("SkillsOk should be true when cfg is nil")
	}
}

func TestGetPreflight_ConsentNotGiven(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
		Consent:    &stubConsentChecker{consented: false},
	})
	resp, err := h.GetPreflight(context.Background(), gen.GetPreflightRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := resp.(gen.GetPreflight200JSONResponse)
	if r.ConsentGiven {
		t.Error("ConsentGiven should be false when consent not given")
	}
}

// --- GetProjectDiff nil provider ---

func TestGetProjectDiff_NilProvider(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})
	resp, err := h.GetProjectDiff(context.Background(), gen.GetProjectDiffRequestObject{Slug: "proj"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.GetProjectDiff500JSONResponse); !ok {
		t.Fatalf("expected 500 when diffProv is nil, got %T", resp)
	}
}

// --- GetTrackDetail with projects and track found ---

type stubTrackReaderWithDetail struct {
	stubTrackReader
	detail *port.TrackDetail
}

func (s *stubTrackReaderWithDetail) GetTrackDetail(_, _ string) (*port.TrackDetail, error) {
	if s.detail != nil {
		return s.detail, nil
	}
	return nil, fmt.Errorf("track not found")
}

func TestGetTrackDetail_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	detail := &port.TrackDetail{
		ID:     "track-1",
		Title:  "Test Track",
		Status: "pending",
		Type:   "feature",
	}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:      &stubAgentLister{},
		Quota:       &stubQuotaReader{},
		LockMgr:     lock.New(""),
		SSEClients:  func() int { return 0 },
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "myapp", ProjectDir: dir}}},
		TrackReader: &stubTrackReaderWithDetail{detail: detail},
	})
	resp, err := h.GetTrackDetail(context.Background(), gen.GetTrackDetailRequestObject{
		TrackId: "track-1",
		Params:  gen.GetTrackDetailParams{Project: "myapp"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetTrackDetail200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if r.Id != "track-1" {
		t.Errorf("Id = %q, want track-1", r.Id)
	}
	if r.Title != "Test Track" {
		t.Errorf("Title = %q, want Test Track", r.Title)
	}
}

// --- DeleteTrack Tests ---

func TestDeleteTrack_WithProject(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	proj := "myapp"
	h := NewAPIHandler(APIHandlerOpts{
		Agents:      &stubAgentLister{},
		Quota:       &stubQuotaReader{},
		LockMgr:     lock.New(""),
		SSEClients:  func() int { return 0 },
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "myapp", ProjectDir: dir}}},
		TrackReader: &stubTrackReader{},
	})
	resp, err := h.DeleteTrack(context.Background(), gen.DeleteTrackRequestObject{
		TrackId: "track-1",
		Params:  gen.DeleteTrackParams{Project: &proj},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.DeleteTrack204Response); !ok {
		t.Fatalf("expected 204, got %T", resp)
	}
}

// --- intPtr helper ---

func TestIntPtr(t *testing.T) {
	t.Parallel()
	p := intPtr(42)
	if *p != 42 {
		t.Errorf("intPtr(42) = %d, want 42", *p)
	}
}

// --- GetTrackDetail full fields ---

func TestGetTrackDetail_AllFields(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	detail := &port.TrackDetail{
		ID:     "track-full",
		Title:  "Full Track",
		Status: "in-progress",
		Type:   "feature",
		Spec:   "some spec",
		Plan:   "some plan",
		Phases: struct {
			Total     int
			Completed int
		}{Total: 3, Completed: 1},
		Tasks: struct {
			Total     int
			Completed int
		}{Total: 10, Completed: 4},
		CreatedAt: "2025-01-01",
		UpdatedAt: "2025-01-02",
		Dependencies: []port.TrackDependency{
			{ID: "dep-1", Title: "Dep One", Status: "completed"},
		},
		Conflicts: []port.TrackConflict{
			{TrackID: "conf-1", TrackTitle: "Conflict One", Risk: "high", Note: "overlapping files"},
		},
	}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:      &stubAgentLister{},
		Quota:       &stubQuotaReader{},
		LockMgr:     lock.New(""),
		SSEClients:  func() int { return 0 },
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "myapp", ProjectDir: dir}}},
		TrackReader: &stubTrackReaderWithDetail{detail: detail},
	})
	resp, err := h.GetTrackDetail(context.Background(), gen.GetTrackDetailRequestObject{
		TrackId: "track-full",
		Params:  gen.GetTrackDetailParams{Project: "myapp"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetTrackDetail200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if r.Type == nil || *r.Type != "feature" {
		t.Error("Type should be feature")
	}
	if r.Spec == nil || *r.Spec != "some spec" {
		t.Error("Spec should be set")
	}
	if r.Plan == nil || *r.Plan != "some plan" {
		t.Error("Plan should be set")
	}
	if r.PhasesTotal == nil || *r.PhasesTotal != 3 {
		t.Error("PhasesTotal should be 3")
	}
	if r.TasksCompleted == nil || *r.TasksCompleted != 4 {
		t.Error("TasksCompleted should be 4")
	}
	if r.Dependencies == nil || len(*r.Dependencies) != 1 {
		t.Error("expected 1 dependency")
	}
	if r.Conflicts == nil || len(*r.Conflicts) != 1 {
		t.Error("expected 1 conflict")
	}
}

func TestGetTrackDetail_UnknownProject(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:      &stubAgentLister{},
		Quota:       &stubQuotaReader{},
		LockMgr:     lock.New(""),
		SSEClients:  func() int { return 0 },
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "alpha", ProjectDir: "/tmp"}}},
		TrackReader: &stubTrackReader{},
	})
	resp, err := h.GetTrackDetail(context.Background(), gen.GetTrackDetailRequestObject{
		TrackId: "t-1",
		Params:  gen.GetTrackDetailParams{Project: "nonexistent"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.GetTrackDetail404JSONResponse); !ok {
		t.Fatalf("expected 404, got %T", resp)
	}
}

// --- UpdateSkills Tests ---

func TestUpdateSkills_NilCfg(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})
	resp, err := h.UpdateSkills(context.Background(), gen.UpdateSkillsRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.UpdateSkills400JSONResponse); !ok {
		t.Fatalf("expected 400 when cfg is nil, got %T", resp)
	}
}

func TestUpdateSkills_LocalInstall(t *testing.T) {
	t.Parallel()
	projectDir := t.TempDir()
	slug := "test-proj"
	h := NewAPIHandler(APIHandlerOpts{
		Agents:   &stubAgentLister{},
		Quota:    &stubQuotaReader{},
		LockMgr:  lock.New(""),
		Projects: &stubProjectLister{projects: []domain.Project{{Slug: slug, ProjectDir: projectDir}}},
		Cfg:      &config.Config{}, // no SkillsRepo → embedded install path
		SSEClients: func() int { return 0 },
	})

	body := gen.SkillUpdateRequest{ProjectSlug: &slug}
	resp, err := h.UpdateSkills(context.Background(), gen.UpdateSkillsRequestObject{Body: &body})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.UpdateSkills200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if r.InstalledCount == 0 {
		t.Error("expected at least 1 skill installed")
	}

	// Verify skills were installed to the project's local dir, not global.
	localSkillsDir := filepath.Join(projectDir, ".claude", "skills")
	entries, err := os.ReadDir(localSkillsDir)
	if err != nil {
		t.Fatalf("failed to read local skills dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected skills in local dir")
	}
}

func TestUpdateSkills_LocalInstall_ProjectNotFound(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:   &stubAgentLister{},
		Quota:    &stubQuotaReader{},
		LockMgr:  lock.New(""),
		Projects: &stubProjectLister{},
		Cfg:      &config.Config{},
		SSEClients: func() int { return 0 },
	})

	slug := "nonexistent"
	body := gen.SkillUpdateRequest{ProjectSlug: &slug}
	resp, err := h.UpdateSkills(context.Background(), gen.UpdateSkillsRequestObject{Body: &body})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.UpdateSkills400JSONResponse); !ok {
		t.Fatalf("expected 400 for unknown project, got %T", resp)
	}
}

// --- DeleteTrack without project (cross-project scan) ---

func TestDeleteTrack_NilProject_ScanAllProjects(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:      &stubAgentLister{},
		Quota:       &stubQuotaReader{},
		LockMgr:     lock.New(""),
		SSEClients:  func() int { return 0 },
		Projects:    &stubProjectLister{projects: []domain.Project{{Slug: "myapp", ProjectDir: dir}}},
		TrackReader: &stubTrackReader{},
	})
	// No project param, stub returns no tracks, so 404.
	resp, err := h.DeleteTrack(context.Background(), gen.DeleteTrackRequestObject{
		TrackId: "track-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.DeleteTrack404JSONResponse); !ok {
		t.Fatalf("expected 404 when track not found across projects, got %T", resp)
	}
}

// --- GenerateTracks nil check ---

func TestGenerateTracks_NilBody(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})
	resp, err := h.GenerateTracks(context.Background(), gen.GenerateTracksRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

// --- StartProjectSetup nil check ---

func TestStartProjectSetup_NilProjects(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})
	resp, err := h.StartProjectSetup(context.Background(), gen.StartProjectSetupRequestObject{Slug: "proj"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

// --- RunAdminOperation ---

func TestRunAdminOperation_NilHandler(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		SSEClients: func() int { return 0 },
	})
	resp, err := h.RunAdminOperation(context.Background(), gen.RunAdminOperationRequestObject{
		Body: &gen.AdminOperationRequest{Operation: "unknown"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}
