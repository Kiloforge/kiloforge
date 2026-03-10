package rest

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/adapter/badge"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/dashboard"
	gitadapter "kiloforge/internal/adapter/git"
	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/persistence/sqlite"
	"kiloforge/internal/adapter/rest/gen"
	"kiloforge/internal/adapter/tracing"
	wsAdapter "kiloforge/internal/adapter/ws"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
	"kiloforge/internal/core/service"
)

// ShutdownTimeout is how long to wait for agents to exit before force-killing.
const ShutdownTimeout = 10 * time.Second

// defaultTracer is the tracer used when no tracer is configured.
var defaultTracer port.Tracer = port.NoopTracer{}

// ServerOption configures optional features on the orchestrator server.
type ServerOption func(*Server)

// WithDashboard enables dashboard routes on the unified server.
func WithDashboard(agents dashboard.AgentLister, quota dashboard.QuotaReader, giteaURL string, projects dashboard.ProjectLister) ServerOption {
	return func(s *Server) {
		hub := dashboard.NewSSEHub()
		d := dashboard.New(0, agents, quota, giteaURL, projects, hub)
		d.SetTrackReader(service.NewTrackReader())
		s.dashboard = d
		s.quotaReader = quota
		s._projects = projects
	}
}

// WithTracing enables trace store for the trace API endpoints.
func WithTracing(store tracing.TraceReader) ServerOption {
	return func(s *Server) {
		s.traceStore = store
	}
}

// WithBoardService enables native board API endpoints.
func WithBoardService(svc port.BoardService) ServerOption {
	return func(s *Server) {
		s.boardSvc = svc
	}
}

// SessionEndCallbackSetter can register a callback for session end events.
type SessionEndCallbackSetter interface {
	SetSessionEndCallback(fn agent.SessionEndCallback)
}

// WithInteractiveSpawner enables interactive agent spawning with WebSocket support.
func WithInteractiveSpawner(spawner InteractiveSpawner) ServerOption {
	return func(s *Server) {
		s.interSpawner = spawner
		s.wsSessions = wsAdapter.NewSessionManager()
		// Wire session-end callback for automatic bridge cleanup.
		if setter, ok := spawner.(SessionEndCallbackSetter); ok {
			setter.SetSessionEndCallback(func(agentID string) {
				s.wsSessions.UnregisterBridge(agentID)
			})
		}
	}
}

// WithConsent enables agent permissions consent checking.
func WithConsent(checker ConsentChecker) ServerOption {
	return func(s *Server) {
		s.consent = checker
	}
}

// WithTourStore enables guided tour API endpoints.
func WithTourStore(store *sqlite.TourStore) ServerOption {
	return func(s *Server) {
		s.tourStore = store
	}
}

// WithQueueService enables the work queue API endpoints.
func WithQueueService(svc QueueServicer) ServerOption {
	return func(s *Server) {
		s.queueSvc = svc
	}
}

// WithAnalytics sets the analytics tracker for product telemetry.
func WithAnalytics(t port.AnalyticsTracker) ServerOption {
	return func(s *Server) {
		s.analytics = t
	}
}

// WithTracer sets the distributed tracer for webhook trace continuation.
func WithTracer(t port.Tracer) ServerOption {
	return func(s *Server) {
		s.tracer = t
	}
}

// Server handles the orchestrator REST API.
type Server struct {
	cfg          *config.Config
	registry     port.ProjectStore
	store        port.AgentStore
	prTracker    port.PRTrackingStore
	spawner      port.AgentSpawner
	prService    *service.PRService
	logger       *log.Logger
	port         int
	dashboard    *dashboard.Server
	quotaReader  QuotaReader
	_projects    dashboard.ProjectLister
	traceStore   tracing.TraceReader
	boardSvc     port.BoardService
	tracer       port.Tracer
	interSpawner InteractiveSpawner
	wsSessions   *wsAdapter.SessionManager
	consent      ConsentChecker
	tourStore    *sqlite.TourStore
	queueSvc     QueueServicer
	analytics    port.AnalyticsTracker
}

// NewServer creates an orchestrator server with multi-project routing via the registry.
func NewServer(cfg *config.Config, registry port.ProjectStore, store port.AgentStore, prTracker port.PRTrackingStore, port int, opts ...ServerOption) *Server {
	logger := log.New(log.Writer(), "[orchestrator] ", log.LstdFlags)
	s := &Server{
		cfg:       cfg,
		registry:  registry,
		store:     store,
		prTracker: prTracker,
		spawner:   &defaultSpawner{},
		prService: service.NewPRService(&defaultSpawner{}, logger),
		logger:    logger,
		port:      port,
		tracer:    defaultTracer,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// defaultSpawner implements port.AgentSpawner using real claude commands.
type defaultSpawner struct{}

func (d *defaultSpawner) SpawnReviewer(ctx context.Context, opts port.ReviewerOpts) (*domain.AgentInfo, error) {
	args := []string{"-p", fmt.Sprintf("/kf-reviewer %s", opts.PRURL), "--output-format", "stream-json", "--verbose"}
	if opts.Model != "" {
		args = append([]string{"--model", opts.Model}, args...)
	}
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = opts.WorkDir
	cmd.Env = agent.CleanClaudeEnv()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start reviewer: %w", err)
	}
	return &domain.AgentInfo{
		ID:     fmt.Sprintf("reviewer-%d", cmd.Process.Pid),
		Role:   "reviewer",
		Ref:    fmt.Sprintf("PR #%d", opts.PRNumber),
		Status: "running",
		PID:    cmd.Process.Pid,
		Model:  opts.Model,
	}, nil
}

func (d *defaultSpawner) ResumeDeveloper(ctx context.Context, sessionID, workDir string) error {
	cmd := exec.CommandContext(ctx, "claude", "--resume", sessionID)
	cmd.Dir = workDir
	cmd.Env = agent.CleanClaudeEnv()
	return cmd.Start()
}

// Run starts the HTTP server and blocks until the context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()

	// Trace seeding endpoint for E2E tests.
	if writer, ok := s.traceStore.(tracing.TraceWriter); ok {
		mux.HandleFunc("POST /api/traces", handleSeedTrace(writer))
	}

	// Track seeding endpoint for E2E tests.
	if s._projects != nil {
		mux.HandleFunc("POST /api/tracks/seed", handleSeedTracks(s._projects))
	}

	// Lock service (shared with generated API handler).
	lockMgr := lock.New(s.cfg.DataDir)
	lockMgr.StartReaper(ctx)

	// Agent timeout reaper — halts agents exceeding configured max duration.
	var eventBusForReaper port.EventBus
	if s.dashboard != nil {
		eventBusForReaper = s.dashboard.EventBus()
	}
	timeoutReaper := agent.NewTimeoutReaper(s.store, s.cfg, eventBusForReaper)
	timeoutReaper.Start(ctx)

	// Wire generated OpenAPI routes (health, agents, quota, tracks, status, locks).
	var sseClients func() int
	if s.dashboard != nil {
		sseClients = s.dashboard.SSEClientCount
	}
	var eventBus port.EventBus
	if s.dashboard != nil {
		eventBus = s.dashboard.EventBus()
	}
	projectSvc := service.NewProjectService(s.registry, service.ProjectServiceConfig{
		DataDir:          s.cfg.DataDir,
		OrchestratorPort: s.cfg.OrchestratorPort,
	})

	gitSync := gitadapter.New()
	apiHandler := NewAPIHandler(APIHandlerOpts{
		Agents:       s.store,
		Quota:        s.quotaReader,
		LockMgr:      lockMgr,
		Projects:     s._projects,
		ProjectMgr:   projectSvc,
		GitSync:      gitSync,
		DiffProvider: gitSync,
		TraceStore:   s.traceStore,
		BoardSvc:     s.boardSvc,
		TrackReader:  service.NewTrackReader(),
		EventBus:     eventBus,
		SSEClients:   sseClients,
		Cfg:          s.cfg,
		InterSpawner: s.interSpawner,
		WSSessions:   s.wsSessions,
		Consent:      s.consent,
		AgentRemover: s.store,
		QueueSvc:     s.queueSvc,
		Analytics:    s.analytics,
	})
	strictHandler := gen.NewStrictHandler(apiHandler, nil)
	gen.HandlerFromMux(strictHandler, mux)

	// WebSocket routes for interactive agents.
	if s.wsSessions != nil {
		wsHandler := wsAdapter.NewHandler(s.wsSessions, nil)
		wsHandler.RegisterRoutes(mux)
	}

	// Badge endpoints (SVG, not JSON — stays manual).
	prLoader := func(slug string) (*domain.PRTracking, error) {
		return s.prTracker.LoadPRTracking(slug)
	}
	badgeHandler := badge.NewHandler(s.store, prLoader)
	badgeHandler.RegisterRoutes(mux)

	// Guided tour endpoints (manual routes, not OpenAPI).
	if s.tourStore != nil {
		tourHandler := NewTourHandler(s.tourStore)
		tourHandler.RegisterRoutes(mux)
	}

	// Mount dashboard non-API routes (SSE, HTML pages, SPA static).
	if s.dashboard != nil {
		if s.traceStore != nil {
			s.dashboard.SetTraceStore(s.traceStore)
		}
		s.dashboard.RegisterNonAPIRoutes(mux)
		s.dashboard.StartWatcher(ctx)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	// Graceful agent shutdown on orchestrator stop.
	running := s.store.AgentsByStatus("running", "waiting")
	if len(running) > 0 {
		s.logger.Printf("Shutting down %d agent(s)...", len(running))
		sm := agent.NewShutdownManager(s.store)
		result := sm.ShutdownAll(ShutdownTimeout)
		if len(result.Suspended) > 0 {
			s.logger.Printf("%d agent(s) suspended", len(result.Suspended))
		}
		if len(result.ForceKilled) > 0 {
			s.logger.Printf("%d agent(s) force-killed", len(result.ForceKilled))
		}
	}

	return nil
}
