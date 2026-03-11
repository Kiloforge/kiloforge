package rest

import (
	"context"
	"fmt"
	"log"
	"log/slog"
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

// ServerOption configures optional features on the Cortex server.
type ServerOption func(*Server)

// WithDashboard enables dashboard routes on the unified server.
func WithDashboard(agents dashboard.AgentLister, quota QuotaReader, projects dashboard.ProjectLister) ServerOption {
	return func(s *Server) {
		hub := dashboard.NewSSEHub()
		d := dashboard.New(0, agents, quota, projects, hub)
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

// WithReliability enables the reliability metrics service.
func WithReliability(svc *service.ReliabilityService) ServerOption {
	return func(s *Server) {
		s.reliabilitySvc = svc
	}
}

// WithNotifications enables the notification service for agent attention alerts.
func WithNotifications(svc *service.NotificationService) ServerOption {
	return func(s *Server) {
		s.notifSvc = svc
	}
}

// WithHealthPinger enables database health checking in the /health endpoint.
func WithHealthPinger(p HealthPinger) ServerOption {
	return func(s *Server) {
		s.healthPinger = p
	}
}

// WithTracer sets the distributed tracer for webhook trace continuation.
func WithTracer(t port.Tracer) ServerOption {
	return func(s *Server) {
		s.tracer = t
	}
}

// Server handles the Cortex REST API.
type Server struct {
	cfg            *config.Config
	registry       port.ProjectStore
	store          port.AgentStore
	prTracker      port.PRTrackingStore
	spawner        port.AgentSpawner
	prService      *service.PRService
	logger         *log.Logger
	host           string
	port           int
	dashboard      *dashboard.Server
	quotaReader    QuotaReader
	_projects      dashboard.ProjectLister
	traceStore     tracing.TraceReader
	boardSvc       port.BoardService
	tracer         port.Tracer
	interSpawner   InteractiveSpawner
	wsSessions     *wsAdapter.SessionManager
	consent        ConsentChecker
	tourStore      *sqlite.TourStore
	queueSvc       QueueServicer
	analytics      port.AnalyticsTracker
	reliabilitySvc *service.ReliabilityService
	notifSvc       *service.NotificationService
	healthPinger   HealthPinger
}

// NewServer creates a Cortex server with multi-project routing via the registry.
func NewServer(cfg *config.Config, registry port.ProjectStore, store port.AgentStore, prTracker port.PRTrackingStore, host string, port int, opts ...ServerOption) *Server {
	logger := log.New(log.Writer(), "[cortex] ", log.LstdFlags)
	s := &Server{
		cfg:       cfg,
		registry:  registry,
		store:     store,
		prTracker: prTracker,
		spawner:   &defaultSpawner{},
		prService: service.NewPRService(&defaultSpawner{}, logger),
		logger:    logger,
		host:      host,
		port:      port,
		tracer:    defaultTracer,
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.dashboard != nil && cfg != nil {
		s.dashboard.SetBudgetUSD(cfg.BudgetUSD)
	}
	return s
}

// defaultSpawner implements port.AgentSpawner using real claude commands.
type defaultSpawner struct{}

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
		mux.HandleFunc("POST /api/tracks/seed", handleSeedTracks(s._projects, s.analytics))
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
	if s.reliabilitySvc != nil {
		timeoutReaper.SetReliabilityRecorder(s.reliabilitySvc)
	}
	timeoutReaper.Start(ctx)

	// Wire idle-disconnect auto-suspension for interactive agents.
	if s.wsSessions != nil && s.cfg != nil {
		graceSec := s.cfg.GetIdleSuspendSeconds()
		if graceSec > 0 {
			if suspender, ok := s.interSpawner.(agent.AgentSuspender); ok {
				cs := agent.NewConnectionSuspender(suspender, s.store, time.Duration(graceSec)*time.Second)
				if s.dashboard != nil {
					cs.SetEventBus(s.dashboard.EventBus())
				}
				s.wsSessions.SetOnDisconnect(cs.OnAgentDisconnected)
				s.wsSessions.SetOnReconnect(cs.OnAgentReconnected)
				defer cs.Stop()
			}
		}
	}

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
		Agents:         s.store,
		Quota:          s.quotaReader,
		LockMgr:        lockMgr,
		Projects:       s._projects,
		ProjectMgr:     projectSvc,
		GitSync:        gitSync,
		DiffProvider:   gitSync,
		TraceStore:     s.traceStore,
		BoardSvc:       s.boardSvc,
		TrackReader:    service.NewTrackReader(),
		EventBus:       eventBus,
		SSEClients:     sseClients,
		Cfg:            s.cfg,
		InterSpawner:   s.interSpawner,
		WSSessions:     s.wsSessions,
		Consent:        s.consent,
		AgentRemover:   s.store,
		QueueSvc:       s.queueSvc,
		Analytics:      s.analytics,
		ReliabilitySvc: s.reliabilitySvc,
		NotifSvc:       s.notifSvc,
		HealthPinger:   s.healthPinger,
	})

	// Wire notification service event bus (dashboard provides it).
	if s.notifSvc != nil && eventBus != nil {
		s.notifSvc.SetEventBus(eventBus)
	}

	// Wire notification service into relay hook and dashboard watcher.
	if s.notifSvc != nil && s.wsSessions != nil {
		notifSvc := s.notifSvc
		agentStore := s.store
		s.wsSessions.SetOnRelayMessage(func(agentID, msgType string) {
			switch msgType {
			case "turn_end":
				agent, err := agentStore.FindAgent(agentID)
				if err != nil || agent == nil {
					return
				}
				name := agent.Name
				if name == "" {
					name = agentID[:min(8, len(agentID))]
				}
				_ = notifSvc.Create(agentID, name+" needs your attention", "waiting for input")
			case "turn_start":
				_ = notifSvc.DismissForAgent(agentID)
			}
		})
	}
	if s.notifSvc != nil && s.dashboard != nil {
		s.dashboard.SetNotificationChecker(s.notifSvc)
		if s.wsSessions != nil {
			s.dashboard.SetBridgeChecker(s.wsSessions)
		}
	}
	strictHandler := gen.NewStrictHandler(apiHandler, nil)
	gen.HandlerFromMux(strictHandler, mux)

	// WebSocket routes for interactive agents.
	if s.wsSessions != nil {
		wsHandler := wsAdapter.NewHandler(s.wsSessions, s.store, nil)
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

	// Wrap mux with request logging middleware.
	handler := RequestLogger(slog.Default())(mux)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.host, s.port),
		Handler: handler,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	// Graceful agent shutdown on Cortex stop.
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
