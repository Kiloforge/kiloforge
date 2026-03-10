package rest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/adapter/config"
	gitadapter "kiloforge/internal/adapter/git"
	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/persistence/sqlite"
	"kiloforge/internal/adapter/rest/gen"
	"kiloforge/internal/adapter/tracing"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
	"kiloforge/internal/core/service"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func newTestBoardStore(dir string) port.BoardStore {
	db, err := sqlite.Open(dir)
	if err != nil {
		panic(fmt.Sprintf("open test db: %v", err))
	}
	return sqlite.NewBoardStore(db)
}

// stubProjectLister implements ProjectLister for testing.
type stubProjectLister struct {
	projects []domain.Project
}

func (s *stubProjectLister) List() []domain.Project { return s.projects }

// stubAgentLister implements AgentLister for testing.
type stubAgentLister struct {
	agents []domain.AgentInfo
}

func (s *stubAgentLister) Agents() []domain.AgentInfo { return s.agents }
func (s *stubAgentLister) Load() error                { return nil }
func (s *stubAgentLister) ListAgents(_ domain.PageOpts, _ ...string) (domain.Page[domain.AgentInfo], error) {
	return domain.Page[domain.AgentInfo]{Items: s.agents, TotalCount: len(s.agents)}, nil
}
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
	total        agent.TotalUsage
	agentUsage   map[string]*agent.AgentUsage
	rateLimited  bool
	tokensPerMin float64
	costPerHour  float64
}

func (s *stubQuotaReader) GetTotalUsage() agent.TotalUsage      { return s.total }
func (s *stubQuotaReader) IsRateLimited() bool                  { return s.rateLimited }
func (s *stubQuotaReader) RetryAfter() time.Duration            { return 0 }
func (s *stubQuotaReader) TokensPerMin(_ time.Duration) float64 { return s.tokensPerMin }
func (s *stubQuotaReader) CostPerHour(_ time.Duration) float64  { return s.costPerHour }
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
	showAll := false
	resp, err := h.ListAgents(context.Background(), gen.ListAgentsRequestObject{
		Params: gen.ListAgentsParams{Active: &showAll},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.ListAgents200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if len(r.Items) != 2 {
		t.Errorf("expected 2 agents, got %d", len(r.Items))
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
	t.Run("nil quota", func(t *testing.T) {
		h := NewAPIHandler(APIHandlerOpts{
			Agents:   &stubAgentLister{},
			LockMgr:  lock.New(""),
			GiteaURL: "http://localhost:3000",
		})
		resp, err := h.GetQuota(context.Background(), gen.GetQuotaRequestObject{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r, ok := resp.(gen.GetQuota200JSONResponse)
		if !ok {
			t.Fatalf("expected 200, got %T", resp)
		}
		if r.EstimatedCostUsd != 0 {
			t.Errorf("expected EstimatedCostUsd 0, got %f", r.EstimatedCostUsd)
		}
	})

	t.Run("with usage", func(t *testing.T) {
		quota := &stubQuotaReader{
			total: agent.TotalUsage{
				TotalCostUSD:        4.23,
				InputTokens:         10000,
				OutputTokens:        5000,
				CacheReadTokens:     2000,
				CacheCreationTokens: 500,
				AgentCount:          2,
			},
			agentUsage: map[string]*agent.AgentUsage{
				"a1": {
					AgentID:             "a1",
					TotalCostUSD:        2.50,
					InputTokens:         6000,
					OutputTokens:        3000,
					CacheReadTokens:     1200,
					CacheCreationTokens: 300,
				},
			},
		}
		h := NewAPIHandler(APIHandlerOpts{
			Agents:   &stubAgentLister{agents: []domain.AgentInfo{{ID: "a1"}}},
			Quota:    quota,
			LockMgr:  lock.New(""),
			GiteaURL: "http://localhost:3000",
		})
		resp, err := h.GetQuota(context.Background(), gen.GetQuotaRequestObject{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r, ok := resp.(gen.GetQuota200JSONResponse)
		if !ok {
			t.Fatalf("expected 200, got %T", resp)
		}
		if r.EstimatedCostUsd != 4.23 {
			t.Errorf("EstimatedCostUsd: want 4.23, got %f", r.EstimatedCostUsd)
		}
		if r.InputTokens != 10000 {
			t.Errorf("InputTokens: want 10000, got %d", r.InputTokens)
		}
		if r.CacheReadTokens != 2000 {
			t.Errorf("CacheReadTokens: want 2000, got %d", r.CacheReadTokens)
		}
		if r.CacheCreationTokens != 500 {
			t.Errorf("CacheCreationTokens: want 500, got %d", r.CacheCreationTokens)
		}
		if r.AgentCount != 2 {
			t.Errorf("AgentCount: want 2, got %d", r.AgentCount)
		}
		if r.Agents == nil || len(*r.Agents) != 1 {
			t.Fatal("expected 1 agent in breakdown")
		}
		au := (*r.Agents)[0]
		if au.EstimatedCostUsd != 2.50 {
			t.Errorf("agent EstimatedCostUsd: want 2.50, got %f", au.EstimatedCostUsd)
		}
		if au.CacheReadTokens != 1200 {
			t.Errorf("agent CacheReadTokens: want 1200, got %d", au.CacheReadTokens)
		}
	})

	t.Run("with rate metrics", func(t *testing.T) {
		quota := &stubQuotaReader{
			total:        agent.TotalUsage{TotalCostUSD: 2.00, InputTokens: 5000, OutputTokens: 2500, AgentCount: 1},
			tokensPerMin: 750.0,
			costPerHour:  1.80,
		}
		h := NewAPIHandler(APIHandlerOpts{
			Agents:  &stubAgentLister{},
			Quota:   quota,
			LockMgr: lock.New(""),
		})
		resp, err := h.GetQuota(context.Background(), gen.GetQuotaRequestObject{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := resp.(gen.GetQuota200JSONResponse)
		if r.RateTokensPerMin == nil || *r.RateTokensPerMin != 750.0 {
			t.Errorf("RateTokensPerMin: want 750, got %v", r.RateTokensPerMin)
		}
		if r.RateCostPerHour == nil || *r.RateCostPerHour != 1.80 {
			t.Errorf("RateCostPerHour: want 1.80, got %v", r.RateCostPerHour)
		}
		if r.BudgetUsd != nil {
			t.Errorf("BudgetUsd should be nil when no budget set")
		}
	})

	t.Run("with budget", func(t *testing.T) {
		quota := &stubQuotaReader{
			total:       agent.TotalUsage{TotalCostUSD: 5.00, AgentCount: 1},
			costPerHour: 2.00,
		}
		cfg := &config.Config{BudgetUSD: 20.0}
		h := NewAPIHandler(APIHandlerOpts{
			Agents:  &stubAgentLister{},
			Quota:   quota,
			Cfg:     cfg,
			LockMgr: lock.New(""),
		})
		resp, err := h.GetQuota(context.Background(), gen.GetQuotaRequestObject{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := resp.(gen.GetQuota200JSONResponse)
		if r.BudgetUsd == nil || *r.BudgetUsd != 20.0 {
			t.Errorf("BudgetUsd: want 20.0, got %v", r.BudgetUsd)
		}
		if r.BudgetUsedPct == nil || *r.BudgetUsedPct != 25.0 {
			t.Errorf("BudgetUsedPct: want 25.0, got %v", r.BudgetUsedPct)
		}
		if r.TimeToBudgetMins == nil || *r.TimeToBudgetMins != 450.0 {
			t.Errorf("TimeToBudgetMins: want 450.0, got %v", r.TimeToBudgetMins)
		}
	})

	t.Run("budget zero omits fields", func(t *testing.T) {
		quota := &stubQuotaReader{
			total:       agent.TotalUsage{TotalCostUSD: 5.00},
			costPerHour: 2.00,
		}
		cfg := &config.Config{BudgetUSD: 0}
		h := NewAPIHandler(APIHandlerOpts{
			Agents:  &stubAgentLister{},
			Quota:   quota,
			Cfg:     cfg,
			LockMgr: lock.New(""),
		})
		resp, err := h.GetQuota(context.Background(), gen.GetQuotaRequestObject{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := resp.(gen.GetQuota200JSONResponse)
		if r.BudgetUsd != nil {
			t.Errorf("BudgetUsd should be nil when budget is 0")
		}
		if r.BudgetUsedPct != nil {
			t.Errorf("BudgetUsedPct should be nil when budget is 0")
		}
		if r.TimeToBudgetMins != nil {
			t.Errorf("TimeToBudgetMins should be nil when budget is 0")
		}
	})
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
	if len(r.Items) != 0 {
		t.Errorf("expected 0 traces, got %d", len(r.Items))
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
	if len(r.Items) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(r.Items))
	}
	if r.Items[0].SpanCount != 2 {
		t.Errorf("expected 2 spans, got %d", r.Items[0].SpanCount)
	}
	if r.Items[0].RootName != "track/abc" {
		t.Errorf("expected root name 'track/abc', got %q", r.Items[0].RootName)
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
	if len(traces.Items) == 0 {
		t.Fatal("expected at least 1 trace")
	}
	traceID := traces.Items[0].TraceId

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
		Agents:   &stubAgentLister{},
		Quota:    &stubQuotaReader{},
		LockMgr:  lock.New(""),
		GiteaURL: "http://localhost:3000",
	})
	resp, err := h.ListTraces(context.Background(), gen.ListTracesRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := resp.(gen.ListTraces200JSONResponse)
	if len(r.Items) != 0 {
		t.Errorf("expected 0 traces, got %d", len(r.Items))
	}
}

// stubProjectManager implements ProjectManager for testing.
type stubProjectManager struct {
	addResult      *domain.AddProjectResult
	addErr         error
	createResult   *domain.AddProjectResult
	createErr      error
	removeErr      error
	removedSlug    string
	removedCleanup bool
}

func (m *stubProjectManager) AddProject(_ context.Context, remoteURL, name string, _ ...domain.AddProjectOpts) (*domain.AddProjectResult, error) {
	if m.addErr != nil {
		return nil, m.addErr
	}
	return m.addResult, nil
}

func (m *stubProjectManager) CreateProject(_ context.Context, name string) (*domain.AddProjectResult, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.createResult, nil
}

func (m *stubProjectManager) RemoveProject(_ context.Context, slug string, cleanup bool) error {
	m.removedSlug = slug
	m.removedCleanup = cleanup
	return m.removeErr
}

func TestAddProject_Success(t *testing.T) {
	t.Parallel()

	mgr := &stubProjectManager{
		addResult: &domain.AddProjectResult{
			Project: domain.Project{
				Slug:         "myapp",
				RepoName:     "myapp",
				OriginRemote: "git@github.com:user/myapp.git",
				Active:       true,
			},
		},
	}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		Projects:   &stubProjectLister{},
		ProjectMgr: mgr,
		GiteaURL:   "http://localhost:3000",
	})

	resp, err := h.AddProject(context.Background(), gen.AddProjectRequestObject{
		Body: &gen.AddProjectJSONRequestBody{
			RemoteUrl: strPtr("git@github.com:user/myapp.git"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.AddProject201JSONResponse)
	if !ok {
		t.Fatalf("expected 201, got %T", resp)
	}
	if r.Slug != "myapp" {
		t.Errorf("expected slug 'myapp', got %q", r.Slug)
	}
	if r.Active != true {
		t.Error("expected active true")
	}
}

func TestAddProject_Duplicate(t *testing.T) {
	t.Parallel()

	mgr := &stubProjectManager{
		addErr: fmt.Errorf("project myapp: %w", domain.ErrProjectExists),
	}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		ProjectMgr: mgr,
		GiteaURL:   "http://localhost:3000",
	})

	resp, err := h.AddProject(context.Background(), gen.AddProjectRequestObject{
		Body: &gen.AddProjectJSONRequestBody{
			RemoteUrl: strPtr("git@github.com:user/myapp.git"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.AddProject409JSONResponse); !ok {
		t.Fatalf("expected 409, got %T", resp)
	}
}

func TestAddProject_BadURL(t *testing.T) {
	t.Parallel()

	mgr := &stubProjectManager{
		addErr: fmt.Errorf("invalid remote URL: /local/path"),
	}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		ProjectMgr: mgr,
		GiteaURL:   "http://localhost:3000",
	})

	resp, err := h.AddProject(context.Background(), gen.AddProjectRequestObject{
		Body: &gen.AddProjectJSONRequestBody{
			RemoteUrl: strPtr("/local/path"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.AddProject400JSONResponse); !ok {
		t.Fatalf("expected 400, got %T", resp)
	}
}

func TestAddProject_MissingURL(t *testing.T) {
	t.Parallel()

	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		ProjectMgr: &stubProjectManager{},
		GiteaURL:   "http://localhost:3000",
	})

	resp, err := h.AddProject(context.Background(), gen.AddProjectRequestObject{
		Body: &gen.AddProjectJSONRequestBody{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.AddProject400JSONResponse); !ok {
		t.Fatalf("expected 400, got %T", resp)
	}
}

func TestRemoveProject_Success(t *testing.T) {
	t.Parallel()

	mgr := &stubProjectManager{}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		ProjectMgr: mgr,
		GiteaURL:   "http://localhost:3000",
	})

	resp, err := h.RemoveProject(context.Background(), gen.RemoveProjectRequestObject{
		Slug:   "myapp",
		Params: gen.RemoveProjectParams{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.RemoveProject204Response); !ok {
		t.Fatalf("expected 204, got %T", resp)
	}
	if mgr.removedSlug != "myapp" {
		t.Errorf("expected slug 'myapp', got %q", mgr.removedSlug)
	}
}

func TestRemoveProject_WithCleanup(t *testing.T) {
	t.Parallel()

	mgr := &stubProjectManager{}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		ProjectMgr: mgr,
		GiteaURL:   "http://localhost:3000",
	})

	cleanup := true
	resp, err := h.RemoveProject(context.Background(), gen.RemoveProjectRequestObject{
		Slug:   "myapp",
		Params: gen.RemoveProjectParams{Cleanup: &cleanup},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.RemoveProject204Response); !ok {
		t.Fatalf("expected 204, got %T", resp)
	}
	if !mgr.removedCleanup {
		t.Error("expected cleanup=true to be passed through")
	}
}

func TestRemoveProject_NotFound(t *testing.T) {
	t.Parallel()

	mgr := &stubProjectManager{
		removeErr: fmt.Errorf("project nope: %w", domain.ErrProjectNotFound),
	}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		ProjectMgr: mgr,
		GiteaURL:   "http://localhost:3000",
	})

	resp, err := h.RemoveProject(context.Background(), gen.RemoveProjectRequestObject{
		Slug:   "nope",
		Params: gen.RemoveProjectParams{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.RemoveProject404JSONResponse); !ok {
		t.Fatalf("expected 404, got %T", resp)
	}
}

// spyEventBus records published events for testing.
type spyEventBus struct {
	events []domain.Event
}

func (s *spyEventBus) Publish(event domain.Event)        { s.events = append(s.events, event) }
func (s *spyEventBus) Subscribe() <-chan domain.Event    { return make(chan domain.Event) }
func (s *spyEventBus) Unsubscribe(_ <-chan domain.Event) {}
func (s *spyEventBus) ClientCount() int                  { return 0 }

func TestGetBoard_AutoSyncOnEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	boardSvc := service.NewNativeBoardService(newTestBoardStore(dir))

	// Create a project directory with kf tracks.yaml containing a track.
	projectDir := t.TempDir()
	kfDir := filepath.Join(projectDir, ".agent", "kf")
	if err := os.MkdirAll(kfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tracksYaml := `my-track_20260310Z: {"title":"My Test Track","status":"pending","type":"feature","created":"2026-03-10","updated":"2026-03-10"}
`
	if err := os.WriteFile(filepath.Join(kfDir, "tracks.yaml"), []byte(tracksYaml), 0o644); err != nil {
		t.Fatal(err)
	}

	spy := &spyEventBus{}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:      &stubAgentLister{},
		Quota:       &stubQuotaReader{},
		LockMgr:     lock.New(""),
		BoardSvc:    boardSvc,
		TrackReader: service.NewTrackReader(),
		EventBus:    spy,
		Projects: &stubProjectLister{projects: []domain.Project{
			{Slug: "proj", ProjectDir: projectDir},
		}},
		GiteaURL: "http://localhost:3000",
	})

	// First call: board is empty, should auto-sync.
	resp, err := h.GetBoard(context.Background(), gen.GetBoardRequestObject{Project: "proj"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetBoard200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp)
	}
	if len(r.Cards) == 0 {
		t.Error("expected auto-synced cards, got empty board")
	}
	if _, found := r.Cards["my-track_20260310Z"]; !found {
		t.Error("expected card my-track_20260310Z to be auto-synced")
	}
	if len(spy.events) != 1 || spy.events[0].Type != domain.EventBoardUpdate {
		t.Errorf("expected 1 board_update event, got %d events", len(spy.events))
	}

	// Second call: board is not empty, should NOT re-sync.
	spy.events = nil
	resp2, err := h.GetBoard(context.Background(), gen.GetBoardRequestObject{Project: "proj"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r2, ok := resp2.(gen.GetBoard200JSONResponse)
	if !ok {
		t.Fatalf("unexpected response type: %T", resp2)
	}
	if len(r2.Cards) == 0 {
		t.Error("expected non-empty board on second call")
	}
	if len(spy.events) != 0 {
		t.Errorf("expected 0 events on second call (no re-sync), got %d", len(spy.events))
	}
}

func TestMoveCard_EmitsBoardUpdate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	boardSvc := service.NewNativeBoardService(newTestBoardStore(dir))

	// Create a board with a card.
	_, _ = boardSvc.SyncFromTracks("proj", []port.TrackEntry{
		{ID: "track-1", Title: "Test Track", Status: "pending"},
	}, nil)

	spy := &spyEventBus{}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:   &stubAgentLister{},
		Quota:    &stubQuotaReader{},
		LockMgr:  lock.New(""),
		BoardSvc: boardSvc,
		EventBus: spy,
		GiteaURL: "http://localhost:3000",
	})

	_, err := h.MoveCard(context.Background(), gen.MoveCardRequestObject{
		Project: "proj",
		Body:    &gen.MoveCardJSONRequestBody{TrackId: "track-1", ToColumn: gen.MoveCardRequestToColumnApproved},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spy.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(spy.events))
	}
	if spy.events[0].Type != domain.EventBoardUpdate {
		t.Errorf("expected board_update, got %s", spy.events[0].Type)
	}
}

func TestLockAcquireRelease_EmitsEvents(t *testing.T) {
	t.Parallel()
	spy := &spyEventBus{}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:   &stubAgentLister{},
		Quota:    &stubQuotaReader{},
		LockMgr:  lock.New(""),
		EventBus: spy,
		GiteaURL: "http://localhost:3000",
	})

	ttl := 60
	_, _ = h.AcquireLock(context.Background(), gen.AcquireLockRequestObject{
		Scope: "test-scope",
		Body:  &gen.LockAcquireRequest{Holder: "w1", TtlSeconds: &ttl},
	})
	_, _ = h.ReleaseLock(context.Background(), gen.ReleaseLockRequestObject{
		Scope: "test-scope",
		Body:  &gen.LockReleaseRequest{Holder: "w1"},
	})

	if len(spy.events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(spy.events))
	}
	if spy.events[0].Type != domain.EventLockUpdate {
		t.Errorf("expected lock_update, got %s", spy.events[0].Type)
	}
	if spy.events[1].Type != domain.EventLockReleased {
		t.Errorf("expected lock_released, got %s", spy.events[1].Type)
	}
}

func TestAddProject_EmitsProjectUpdate(t *testing.T) {
	t.Parallel()
	spy := &spyEventBus{}
	mgr := &stubProjectManager{
		addResult: &domain.AddProjectResult{
			Project: domain.Project{Slug: "myapp", RepoName: "myapp", Active: true},
		},
	}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		ProjectMgr: mgr,
		EventBus:   spy,
		GiteaURL:   "http://localhost:3000",
	})

	_, err := h.AddProject(context.Background(), gen.AddProjectRequestObject{
		Body: &gen.AddProjectJSONRequestBody{RemoteUrl: strPtr("git@github.com:user/myapp.git")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spy.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(spy.events))
	}
	if spy.events[0].Type != domain.EventProjectUpdate {
		t.Errorf("expected project_update, got %s", spy.events[0].Type)
	}
}

func TestRemoveProject_EmitsProjectRemoved(t *testing.T) {
	t.Parallel()
	spy := &spyEventBus{}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		Quota:      &stubQuotaReader{},
		LockMgr:    lock.New(""),
		ProjectMgr: &stubProjectManager{},
		EventBus:   spy,
		GiteaURL:   "http://localhost:3000",
	})

	_, err := h.RemoveProject(context.Background(), gen.RemoveProjectRequestObject{
		Slug:   "myapp",
		Params: gen.RemoveProjectParams{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spy.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(spy.events))
	}
	if spy.events[0].Type != domain.EventProjectRemoved {
		t.Errorf("expected project_removed, got %s", spy.events[0].Type)
	}
}

func TestAddProject_NilManager(t *testing.T) {
	t.Parallel()

	h := NewAPIHandler(APIHandlerOpts{
		Agents:   &stubAgentLister{},
		Quota:    &stubQuotaReader{},
		LockMgr:  lock.New(""),
		GiteaURL: "http://localhost:3000",
	})

	resp, err := h.AddProject(context.Background(), gen.AddProjectRequestObject{
		Body: &gen.AddProjectJSONRequestBody{RemoteUrl: strPtr("git@github.com:user/repo.git")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.AddProject500JSONResponse); !ok {
		t.Fatalf("expected 500, got %T", resp)
	}
}

func TestAddProject_WithSSHKey(t *testing.T) {
	t.Parallel()

	var capturedOpts []domain.AddProjectOpts
	mgr := &stubProjectManager{
		addResult: &domain.AddProjectResult{
			Project: domain.Project{
				Slug:     "myapp",
				RepoName: "myapp",
				Active:   true,
			},
		},
	}
	// Override AddProject to capture opts.
	origAdd := mgr.addResult
	_ = origAdd

	h := NewAPIHandler(APIHandlerOpts{
		Agents:     &stubAgentLister{},
		LockMgr:    lock.New(t.TempDir()),
		ProjectMgr: mgr,
		GiteaURL:   "http://localhost:3000",
	})

	sshKeyPath := "/home/user/.ssh/id_ed25519"
	resp, err := h.AddProject(context.Background(), gen.AddProjectRequestObject{
		Body: &gen.AddProjectJSONRequestBody{
			RemoteUrl: strPtr("git@github.com:user/myapp.git"),
			SshKey:    &sshKeyPath,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.AddProject201JSONResponse); !ok {
		t.Fatalf("expected 201, got %T", resp)
	}
	_ = capturedOpts // opts captured via interface are not directly inspectable with stub
}

func TestListSSHKeys_Returns200(t *testing.T) {
	t.Parallel()

	h := NewAPIHandler(APIHandlerOpts{
		Agents:  &stubAgentLister{},
		LockMgr: lock.New(t.TempDir()),
	})

	resp, err := h.ListSSHKeys(context.Background(), gen.ListSSHKeysRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.ListSSHKeys200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	// Should return a non-nil keys slice (may be empty depending on the host).
	if r.Keys == nil {
		t.Fatal("expected non-nil keys slice")
	}
}

func TestGetConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &config.Config{DataDir: dir}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:  &stubAgentLister{},
		LockMgr: lock.New(t.TempDir()),
		Cfg:     cfg,
	})

	resp, err := h.GetConfig(context.Background(), gen.GetConfigRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetConfig200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	// Default: dashboard=true.
	if !r.DashboardEnabled {
		t.Error("expected DashboardEnabled=true by default")
	}
}

func TestUpdateConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &config.Config{DataDir: dir}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:  &stubAgentLister{},
		LockMgr: lock.New(t.TempDir()),
		Cfg:     cfg,
	})

	f := false
	resp, err := h.UpdateConfig(context.Background(), gen.UpdateConfigRequestObject{
		Body: &gen.UpdateConfigJSONRequestBody{
			DashboardEnabled: &f,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.UpdateConfig200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if r.DashboardEnabled {
		t.Error("expected DashboardEnabled=false after update")
	}
	// Verify persisted.
	if cfg.DashboardEnabled == nil || *cfg.DashboardEnabled != false {
		t.Error("expected cfg.DashboardEnabled to be set to false")
	}
}

func TestUpdateConfig_NilBody(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &config.Config{DataDir: dir}
	h := NewAPIHandler(APIHandlerOpts{
		Agents:  &stubAgentLister{},
		LockMgr: lock.New(t.TempDir()),
		Cfg:     cfg,
	})

	resp, err := h.UpdateConfig(context.Background(), gen.UpdateConfigRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.UpdateConfig400JSONResponse); !ok {
		t.Fatalf("expected 400, got %T", resp)
	}
}

func TestConfigAPI_AgentMaxDuration_Roundtrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := &config.Config{DataDir: dir}
	h := NewAPIHandler(APIHandlerOpts{Agents: &stubAgentLister{}, LockMgr: lock.New(t.TempDir()), Cfg: cfg})
	resp, _ := h.GetConfig(context.Background(), gen.GetConfigRequestObject{})
	r := resp.(gen.GetConfig200JSONResponse)
	if r.AgentMaxDuration != nil {
		t.Errorf("expected nil initially, got %q", *r.AgentMaxDuration)
	}
	dur := "30m"
	updateResp, _ := h.UpdateConfig(context.Background(), gen.UpdateConfigRequestObject{
		Body: &gen.UpdateConfigJSONRequestBody{AgentMaxDuration: &dur},
	})
	ur := updateResp.(gen.UpdateConfig200JSONResponse)
	if ur.AgentMaxDuration == nil || *ur.AgentMaxDuration != "30m" {
		t.Errorf("expected 30m, got %v", ur.AgentMaxDuration)
	}
}

func TestGetProjectDiff(t *testing.T) {
	t.Run("project not found", func(t *testing.T) {
		h := newTestHandler(nil)
		h.diffProv = gitadapter.New()
		resp, err := h.GetProjectDiff(context.Background(), gen.GetProjectDiffRequestObject{
			Slug:   "nonexistent",
			Params: gen.GetProjectDiffParams{Branch: "some-branch"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := resp.(gen.GetProjectDiff404JSONResponse); !ok {
			t.Fatalf("expected 404, got %T", resp)
		}
	})

	t.Run("diff provider not configured", func(t *testing.T) {
		h := newTestHandler(nil)
		// diffProv is nil by default in newTestHandler
		resp, err := h.GetProjectDiff(context.Background(), gen.GetProjectDiffRequestObject{
			Slug:   "proj-1",
			Params: gen.GetProjectDiffParams{Branch: "some-branch"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := resp.(gen.GetProjectDiff500JSONResponse); !ok {
			t.Fatalf("expected 500, got %T", resp)
		}
	})
}

func TestGetProjectBranches(t *testing.T) {
	t.Run("project not found", func(t *testing.T) {
		h := newTestHandler(nil)
		resp, err := h.GetProjectBranches(context.Background(), gen.GetProjectBranchesRequestObject{
			Slug: "nonexistent",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := resp.(gen.GetProjectBranches404JSONResponse); !ok {
			t.Fatalf("expected 404, got %T", resp)
		}
	})

	t.Run("returns branches from agents", func(t *testing.T) {
		agents := []domain.AgentInfo{
			{ID: "agent-1", Role: "developer", Ref: "feature/track-1", Status: "running", WorktreeDir: "/some/path"},
			{ID: "agent-2", Role: "developer", Ref: "feature/track-2", Status: "completed", WorktreeDir: "/other/path"},
			{ID: "agent-3", Role: "interactive", Ref: "interactive", Status: "running"}, // no worktree
		}
		h := newTestHandler(agents)
		resp, err := h.GetProjectBranches(context.Background(), gen.GetProjectBranchesRequestObject{
			Slug: "proj-1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r, ok := resp.(gen.GetProjectBranches200JSONResponse)
		if !ok {
			t.Fatalf("expected 200, got %T", resp)
		}
		// Only agents with worktree dirs should be included
		if len(r) != 2 {
			t.Fatalf("expected 2 branches, got %d", len(r))
		}
		if r[0].Branch != "feature/track-1" {
			t.Errorf("branch[0] = %q, want %q", r[0].Branch, "feature/track-1")
		}
		if r[0].Status != "running" {
			t.Errorf("status[0] = %q, want %q", r[0].Status, "running")
		}
	})

	t.Run("empty when no agents", func(t *testing.T) {
		h := newTestHandler(nil)
		resp, err := h.GetProjectBranches(context.Background(), gen.GetProjectBranchesRequestObject{
			Slug: "proj-1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r, ok := resp.(gen.GetProjectBranches200JSONResponse)
		if !ok {
			t.Fatalf("expected 200, got %T", resp)
		}
		if len(r) != 0 {
			t.Errorf("expected 0 branches, got %d", len(r))
		}
	})
}

func TestGetProjectMetadata_HappyPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create a project directory with kf metadata.
	projDir := filepath.Join(dir, "myproject")
	kfDir := filepath.Join(projDir, ".agent", "kf")
	os.MkdirAll(filepath.Join(kfDir, "code_styleguides"), 0o755)

	os.WriteFile(filepath.Join(kfDir, "product.md"), []byte("# My Product"), 0o644)
	os.WriteFile(filepath.Join(kfDir, "product-guidelines.md"), []byte("# Guidelines"), 0o644)
	os.WriteFile(filepath.Join(kfDir, "tech-stack.md"), []byte("# Tech Stack\nGo 1.24"), 0o644)
	os.WriteFile(filepath.Join(kfDir, "workflow.md"), []byte("# Workflow"), 0o644)
	os.WriteFile(filepath.Join(kfDir, "quick-links.md"), []byte("- [Product](./product.md)\n- [Tech](./tech-stack.md)"), 0o644)
	os.WriteFile(filepath.Join(kfDir, "code_styleguides", "go.md"), []byte("# Go Style"), 0o644)
	os.WriteFile(filepath.Join(kfDir, "tracks.yaml"), []byte("t1: {\"title\":\"Track 1\",\"status\":\"completed\",\"type\":\"feature\",\"created\":\"2026-01-01\",\"updated\":\"2026-01-01\"}\nt2: {\"title\":\"Track 2\",\"status\":\"pending\",\"type\":\"chore\",\"created\":\"2026-01-01\",\"updated\":\"2026-01-01\"}\n"), 0o644)

	h := NewAPIHandler(APIHandlerOpts{
		Projects: &stubProjectLister{projects: []domain.Project{
			{Slug: "myproject", ProjectDir: projDir},
		}},
	})

	resp, err := h.GetProjectMetadata(context.Background(), gen.GetProjectMetadataRequestObject{Slug: "myproject"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetProjectMetadata200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if r.Product != "# My Product" {
		t.Errorf("product = %q, want %q", r.Product, "# My Product")
	}
	if r.ProductGuidelines == nil || *r.ProductGuidelines != "# Guidelines" {
		t.Errorf("product_guidelines = %v, want %q", r.ProductGuidelines, "# Guidelines")
	}
	if r.TechStack != "# Tech Stack\nGo 1.24" {
		t.Errorf("tech_stack = %q", r.TechStack)
	}
	if r.Workflow == nil || *r.Workflow != "# Workflow" {
		t.Errorf("workflow = %v", r.Workflow)
	}
	if len(r.QuickLinks) != 2 {
		t.Errorf("quick_links count = %d, want 2", len(r.QuickLinks))
	}
	if r.StyleGuides == nil || len(*r.StyleGuides) != 1 {
		t.Fatalf("style_guides count = %v, want 1", r.StyleGuides)
	}
	if (*r.StyleGuides)[0].Name != "go" {
		t.Errorf("style_guides[0].name = %q, want %q", (*r.StyleGuides)[0].Name, "go")
	}
	if r.TrackSummary.Total != 2 {
		t.Errorf("track_summary.total = %d, want 2", r.TrackSummary.Total)
	}
	if r.TrackSummary.Completed != 1 {
		t.Errorf("track_summary.completed = %d, want 1", r.TrackSummary.Completed)
	}
	if r.TrackSummary.Pending != 1 {
		t.Errorf("track_summary.pending = %d, want 1", r.TrackSummary.Pending)
	}
}

func TestGetProjectMetadata_ProjectNotFound(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Projects: &stubProjectLister{projects: []domain.Project{{Slug: "other"}}},
	})

	resp, err := h.GetProjectMetadata(context.Background(), gen.GetProjectMetadataRequestObject{Slug: "nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.GetProjectMetadata404JSONResponse); !ok {
		t.Fatalf("expected 404, got %T", resp)
	}
}

func TestGetProjectMetadata_KFNotInitialized(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	projDir := filepath.Join(dir, "emptyproject")
	os.MkdirAll(projDir, 0o755)

	h := NewAPIHandler(APIHandlerOpts{
		Projects: &stubProjectLister{projects: []domain.Project{
			{Slug: "emptyproject", ProjectDir: projDir},
		}},
	})

	resp, err := h.GetProjectMetadata(context.Background(), gen.GetProjectMetadataRequestObject{Slug: "emptyproject"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetProjectMetadata404JSONResponse)
	if !ok {
		t.Fatalf("expected 404, got %T", resp)
	}
	if r.Error == "" {
		t.Error("expected error message for uninitialized kf")
	}
}

func TestGetProjectSettings_Defaults(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	projDir := filepath.Join(dir, "proj")
	kfDir := filepath.Join(projDir, ".agent", "kf")
	os.MkdirAll(kfDir, 0o755)
	// No config.yaml — should return defaults.

	h := NewAPIHandler(APIHandlerOpts{
		Projects: &stubProjectLister{projects: []domain.Project{
			{Slug: "proj", ProjectDir: projDir},
		}},
	})

	resp, err := h.GetProjectSettings(context.Background(), gen.GetProjectSettingsRequestObject{Slug: "proj"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetProjectSettings200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if r.PrimaryBranch != "main" {
		t.Errorf("primary_branch = %q, want %q", r.PrimaryBranch, "main")
	}
	if !r.EnforceDepOrdering {
		t.Error("enforce_dep_ordering should default to true")
	}
}

func TestGetProjectSettings_WithConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	projDir := filepath.Join(dir, "proj")
	kfDir := filepath.Join(projDir, ".agent", "kf")
	os.MkdirAll(kfDir, 0o755)
	os.WriteFile(filepath.Join(kfDir, "config.yaml"), []byte("primary_branch: develop\nenforce_dep_ordering: false\n"), 0o644)

	h := NewAPIHandler(APIHandlerOpts{
		Projects: &stubProjectLister{projects: []domain.Project{
			{Slug: "proj", ProjectDir: projDir},
		}},
	})

	resp, err := h.GetProjectSettings(context.Background(), gen.GetProjectSettingsRequestObject{Slug: "proj"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.GetProjectSettings200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if r.PrimaryBranch != "develop" {
		t.Errorf("primary_branch = %q, want %q", r.PrimaryBranch, "develop")
	}
	if r.EnforceDepOrdering {
		t.Error("enforce_dep_ordering should be false")
	}
}

func TestGetProjectSettings_ProjectNotFound(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Projects: &stubProjectLister{projects: []domain.Project{{Slug: "other"}}},
	})

	resp, err := h.GetProjectSettings(context.Background(), gen.GetProjectSettingsRequestObject{Slug: "nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.GetProjectSettings404JSONResponse); !ok {
		t.Fatalf("expected 404, got %T", resp)
	}
}

func TestUpdateProjectSettings_PartialUpdate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	projDir := filepath.Join(dir, "proj")
	kfDir := filepath.Join(projDir, ".agent", "kf")
	os.MkdirAll(kfDir, 0o755)
	// Start with defaults (no config.yaml).

	spy := &spyEventBus{}
	h := NewAPIHandler(APIHandlerOpts{
		Projects: &stubProjectLister{projects: []domain.Project{
			{Slug: "proj", ProjectDir: projDir},
		}},
		EventBus: spy,
	})

	// Update only primary_branch.
	branch := "develop"
	resp, err := h.UpdateProjectSettings(context.Background(), gen.UpdateProjectSettingsRequestObject{
		Slug: "proj",
		Body: &gen.UpdateProjectSettingsJSONRequestBody{
			PrimaryBranch: &branch,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(gen.UpdateProjectSettings200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if r.PrimaryBranch != "develop" {
		t.Errorf("primary_branch = %q, want %q", r.PrimaryBranch, "develop")
	}
	// enforce_dep_ordering should remain default (true).
	if !r.EnforceDepOrdering {
		t.Error("enforce_dep_ordering should remain true")
	}

	// Verify SSE event emitted.
	if len(spy.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(spy.events))
	}
	if spy.events[0].Type != domain.EventProjectSettingsUpdate {
		t.Errorf("expected project_settings_update, got %s", spy.events[0].Type)
	}

	// Verify persisted — read back.
	resp2, _ := h.GetProjectSettings(context.Background(), gen.GetProjectSettingsRequestObject{Slug: "proj"})
	r2, ok := resp2.(gen.GetProjectSettings200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 on re-read, got %T", resp2)
	}
	if r2.PrimaryBranch != "develop" {
		t.Errorf("re-read primary_branch = %q, want %q", r2.PrimaryBranch, "develop")
	}
}

func TestUpdateProjectSettings_ProjectNotFound(t *testing.T) {
	t.Parallel()
	h := NewAPIHandler(APIHandlerOpts{
		Projects: &stubProjectLister{projects: []domain.Project{{Slug: "other"}}},
	})

	branch := "develop"
	resp, err := h.UpdateProjectSettings(context.Background(), gen.UpdateProjectSettingsRequestObject{
		Slug: "nonexistent",
		Body: &gen.UpdateProjectSettingsJSONRequestBody{
			PrimaryBranch: &branch,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(gen.UpdateProjectSettings404JSONResponse); !ok {
		t.Fatalf("expected 404, got %T", resp)
	}
}
