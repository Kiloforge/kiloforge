package rest

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/rest/gen"
	"kiloforge/internal/adapter/tracing"
	"kiloforge/internal/core/domain"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// stubProjectLister implements ProjectLister for testing.
type stubProjectLister struct {
	projects []domain.Project
}

func (s *stubProjectLister) List() []domain.Project { return s.projects }

// stubAgentLister implements AgentLister for testing.
type stubAgentLister struct {
	agents []domain.AgentInfo
}

func (s *stubAgentLister) Agents() []domain.AgentInfo     { return s.agents }
func (s *stubAgentLister) Load() error                    { return nil }
func (s *stubAgentLister) FindAgent(id string) (*domain.AgentInfo, error) {
	for i := range s.agents {
		if s.agents[i].ID == id || len(id) <= len(s.agents[i].ID) && s.agents[i].ID[:len(id)] == id {
			return &s.agents[i], nil
		}
	}
	return nil, domain.ErrAgentNotFound
}

// stubQuotaReader implements QuotaReader for testing.
type stubQuotaReader struct {
	total       agent.TotalUsage
	agentUsage  map[string]*agent.AgentUsage
	rateLimited bool
}

func (s *stubQuotaReader) GetTotalUsage() agent.TotalUsage { return s.total }
func (s *stubQuotaReader) IsRateLimited() bool             { return s.rateLimited }
func (s *stubQuotaReader) RetryAfter() time.Duration       { return 0 }
func (s *stubQuotaReader) GetAgentUsage(id string) *agent.AgentUsage {
	return s.agentUsage[id]
}

func newTestHandler(agents []domain.AgentInfo) *APIHandler {
	return NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{agents: agents},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		Projects:   &stubProjectLister{projects: []domain.Project{{Slug: "proj-1"}, {Slug: "proj-2"}}},
		GiteaURL:   "http://localhost:3000",
		SSEClients: func() int { return 0 },
	})
}

func TestGetHealth(t *testing.T) {
	h := newTestHandler(nil)
	resp, err := h.GetHealth(context.Background(), gen.GetHealthRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetHealth200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if r.Status != "ok" {
		t.Errorf("expected status ok, got %s", r.Status)
	}
	if r.Projects != 2 {
		t.Errorf("expected 2 projects, got %d", r.Projects)
	}
}

func TestListAgents(t *testing.T) {
	agents := []domain.AgentInfo{
		{ID: "agent-1", Role: "developer", Status: "running", StartedAt: time.Now()},
		{ID: "agent-2", Role: "reviewer", Status: "completed", StartedAt: time.Now()},
	}
	h := newTestHandler(agents)
	resp, err := h.ListAgents(context.Background(), gen.ListAgentsRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.ListAgents200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if len(r) != 2 {
		t.Errorf("expected 2 agents, got %d", len(r))
	}
}

func TestGetAgent(t *testing.T) {
	agents := []domain.AgentInfo{
		{ID: "agent-abc123", Role: "developer", Status: "running"},
	}
	h := newTestHandler(agents)

	t.Run("found", func(t *testing.T) {
		resp, err := h.GetAgent(context.Background(), gen.GetAgentRequestObject{Id: "agent-abc123"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := resp.(gen.GetAgent200JSONResponse); !ok {
			t.Fatalf("expected 200, got %T", resp)
		}
	})

	t.Run("not found", func(t *testing.T) {
		resp, err := h.GetAgent(context.Background(), gen.GetAgentRequestObject{Id: "nonexistent"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := resp.(gen.GetAgent404JSONResponse); !ok {
			t.Fatalf("expected 404, got %T", resp)
		}
	})
}

func TestGetAgentLog(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")
	_ = os.WriteFile(logFile, []byte("line1\nline2\nline3\n"), 0o644)

	agents := []domain.AgentInfo{
		{ID: "agent-log1", Role: "developer", Status: "running", LogFile: logFile},
		{ID: "agent-nolog", Role: "developer", Status: "running"},
	}
	h := newTestHandler(agents)

	t.Run("with log", func(t *testing.T) {
		lines := 2
		resp, err := h.GetAgentLog(context.Background(), gen.GetAgentLogRequestObject{
			Id:     "agent-log1",
			Params: gen.GetAgentLogParams{Lines: &lines},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r, ok := resp.(gen.GetAgentLog200JSONResponse)
		if !ok {
			t.Fatalf("expected 200, got %T", resp)
		}
		if len(r.Lines) != 2 {
			t.Errorf("expected 2 lines, got %d", len(r.Lines))
		}
		if r.Total != 3 {
			t.Errorf("expected 3 total, got %d", r.Total)
		}
	})

	t.Run("no log file", func(t *testing.T) {
		resp, err := h.GetAgentLog(context.Background(), gen.GetAgentLogRequestObject{
			Id:     "agent-nolog",
			Params: gen.GetAgentLogParams{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := resp.(gen.GetAgentLog404JSONResponse); !ok {
			t.Fatalf("expected 404, got %T", resp)
		}
	})
}

func TestLockOperations(t *testing.T) {
	h := newTestHandler(nil)

	t.Run("acquire and release", func(t *testing.T) {
		// Acquire
		ttl := 60
		resp, err := h.AcquireLock(context.Background(), gen.AcquireLockRequestObject{
			Scope: "test",
			Body:  &gen.LockAcquireRequest{Holder: "worker-1", TtlSeconds: &ttl},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := resp.(gen.AcquireLock200JSONResponse); !ok {
			t.Fatalf("expected 200, got %T", resp)
		}

		// List
		listResp, err := h.ListLocks(context.Background(), gen.ListLocksRequestObject{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		locks, ok := listResp.(gen.ListLocks200JSONResponse)
		if !ok {
			t.Fatalf("expected 200, got %T", listResp)
		}
		if len(locks) != 1 {
			t.Errorf("expected 1 lock, got %d", len(locks))
		}

		// Release
		releaseResp, err := h.ReleaseLock(context.Background(), gen.ReleaseLockRequestObject{
			Scope: "test",
			Body:  &gen.LockReleaseRequest{Holder: "worker-1"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := releaseResp.(gen.ReleaseLock200JSONResponse); !ok {
			t.Fatalf("expected 200, got %T", releaseResp)
		}
	})

	t.Run("heartbeat", func(t *testing.T) {
		// Acquire first
		ttl := 30
		_, _ = h.AcquireLock(context.Background(), gen.AcquireLockRequestObject{
			Scope: "hb-test",
			Body:  &gen.LockAcquireRequest{Holder: "worker-1", TtlSeconds: &ttl},
		})

		// Heartbeat
		hbTTL := 120
		resp, err := h.HeartbeatLock(context.Background(), gen.HeartbeatLockRequestObject{
			Scope: "hb-test",
			Body:  &gen.LockHeartbeatRequest{Holder: "worker-1", TtlSeconds: &hbTTL},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := resp.(gen.HeartbeatLock200JSONResponse); !ok {
			t.Fatalf("expected 200, got %T", resp)
		}
	})

	t.Run("acquire conflict", func(t *testing.T) {
		ttl := 60
		_, _ = h.AcquireLock(context.Background(), gen.AcquireLockRequestObject{
			Scope: "conflict",
			Body:  &gen.LockAcquireRequest{Holder: "worker-a", TtlSeconds: &ttl},
		})

		// Second acquire with zero timeout should conflict
		timeout := 0
		resp, err := h.AcquireLock(context.Background(), gen.AcquireLockRequestObject{
			Scope: "conflict",
			Body:  &gen.LockAcquireRequest{Holder: "worker-b", TtlSeconds: &ttl, TimeoutSeconds: &timeout},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		conflict, ok := resp.(gen.AcquireLock409JSONResponse)
		if !ok {
			t.Fatalf("expected 409, got %T", resp)
		}
		if conflict.CurrentHolder == nil || *conflict.CurrentHolder != "worker-a" {
			t.Errorf("expected current_holder worker-a, got %v", conflict.CurrentHolder)
		}
	})
}

func TestGetQuota(t *testing.T) {
	h := newTestHandler(nil)
	resp, err := h.GetQuota(context.Background(), gen.GetQuotaRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.GetQuota200JSONResponse); !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
}

func TestGetStatus(t *testing.T) {
	agents := []domain.AgentInfo{
		{ID: "a1", Status: "running"},
		{ID: "a2", Status: "running"},
		{ID: "a3", Status: "completed"},
	}
	h := newTestHandler(agents)
	resp, err := h.GetStatus(context.Background(), gen.GetStatusRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetStatus200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if r.TotalAgents != 3 {
		t.Errorf("expected 3 agents, got %d", r.TotalAgents)
	}
	if r.AgentCounts["running"] != 2 {
		t.Errorf("expected 2 running, got %d", r.AgentCounts["running"])
	}
}

func TestGetSkillsStatus_NoRepo(t *testing.T) {
	h := newTestHandler(nil)
	resp, err := h.GetSkillsStatus(context.Background(), gen.GetSkillsStatusRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetSkillsStatus200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if r.InstalledVersion != "" {
		t.Errorf("expected empty version, got %s", r.InstalledVersion)
	}
	if r.UpdateAvailable {
		t.Error("expected update_available false")
	}
}

func TestGetSkillsStatus_WithSkills(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a skill directory with SKILL.md.
	skillDir := filepath.Join(tmpDir, "test-skill")
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Test Skill"), 0o644)

	cfg := &config.Config{
		SkillsRepo:    "owner/repo",
		SkillsVersion: "v1.0.0",
		SkillsDir:     tmpDir,
	}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		Projects:   &stubProjectLister{},
		GiteaURL:   "http://localhost:3000",
		SSEClients: func() int { return 0 },
		Cfg:        cfg,
	})

	resp, err := h.GetSkillsStatus(context.Background(), gen.GetSkillsStatusRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetSkillsStatus200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if r.InstalledVersion != "v1.0.0" {
		t.Errorf("expected v1.0.0, got %s", r.InstalledVersion)
	}
	if len(r.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(r.Skills))
	}
	if r.Skills[0].Name != "test-skill" {
		t.Errorf("expected test-skill, got %s", r.Skills[0].Name)
	}
}

func TestUpdateSkills_NoRepo(t *testing.T) {
	h := newTestHandler(nil)
	resp, err := h.UpdateSkills(context.Background(), gen.UpdateSkillsRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.UpdateSkills400JSONResponse); !ok {
		t.Fatalf("expected 400, got %T", resp)
	}
}

func newTestHandlerWithTraces() (*APIHandler, *tracing.Store) {
	store := tracing.NewStore()
	proc := tracing.NewStoreProcessor(store)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(proc))
	otel.SetTracerProvider(tp)

	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		Projects:   &stubProjectLister{},
		TraceStore: store,
		GiteaURL:   "http://localhost:3000",
		SSEClients: func() int { return 0 },
	})
	return h, store
}

func TestListTraces_Empty(t *testing.T) {
	h, _ := newTestHandlerWithTraces()
	resp, err := h.ListTraces(context.Background(), gen.ListTracesRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.ListTraces200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if len(r) != 0 {
		t.Errorf("expected 0 traces, got %d", len(r))
	}
}

func TestListTraces_WithSpans(t *testing.T) {
	h, _ := newTestHandlerWithTraces()
	tracer := otel.Tracer("test")

	ctx, parent := tracer.Start(context.Background(), "track/abc")
	_, child := tracer.Start(ctx, "phase/1")
	child.End()
	parent.End()

	resp, err := h.ListTraces(context.Background(), gen.ListTracesRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := resp.(gen.ListTraces200JSONResponse)
	if len(r) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(r))
	}
	if r[0].SpanCount != 2 {
		t.Errorf("expected 2 spans, got %d", r[0].SpanCount)
	}
	if r[0].RootName != "track/abc" {
		t.Errorf("expected root name 'track/abc', got %q", r[0].RootName)
	}
}

func TestGetTrace_NotFound(t *testing.T) {
	h, _ := newTestHandlerWithTraces()
	resp, err := h.GetTrace(context.Background(), gen.GetTraceRequestObject{TraceId: "nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.GetTrace404JSONResponse); !ok {
		t.Errorf("expected 404, got %T", resp)
	}
}

func TestGetTrace_WithSpans(t *testing.T) {
	h, _ := newTestHandlerWithTraces()
	tracer := otel.Tracer("test")

	_, span := tracer.Start(context.Background(), "agent/developer")
	span.End()

	// Get the trace ID from list.
	listResp, _ := h.ListTraces(context.Background(), gen.ListTracesRequestObject{})
	traces := listResp.(gen.ListTraces200JSONResponse)
	if len(traces) == 0 {
		t.Fatal("expected at least 1 trace")
	}
	traceID := traces[0].TraceId

	resp, err := h.GetTrace(context.Background(), gen.GetTraceRequestObject{TraceId: traceID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetTrace200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if len(r.Spans) != 1 {
		t.Errorf("expected 1 span, got %d", len(r.Spans))
	}
	if r.Spans[0].Name != "agent/developer" {
		t.Errorf("expected span name 'agent/developer', got %q", r.Spans[0].Name)
	}
}

func TestListTraces_NilStore(t *testing.T) {
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		GiteaURL:   "http://localhost:3000",
	})
	resp, err := h.ListTraces(context.Background(), gen.ListTracesRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := resp.(gen.ListTraces200JSONResponse)
	if len(r) != 0 {
		t.Errorf("expected 0 traces, got %d", len(r))
	}
}
