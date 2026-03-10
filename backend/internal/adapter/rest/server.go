package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"time"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/adapter/badge"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/dashboard"
	gitadapter "kiloforge/internal/adapter/git"
	"kiloforge/internal/adapter/gitea"
	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/persistence/sqlite"
	"kiloforge/internal/adapter/pool"
	"kiloforge/internal/adapter/proxy"
	"kiloforge/internal/adapter/rest/gen"
	"kiloforge/internal/adapter/skills"
	"kiloforge/internal/adapter/tracing"
	wsAdapter "kiloforge/internal/adapter/ws"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
	"kiloforge/internal/core/service"

	otelattr "go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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

// WithGiteaProxy enables a reverse proxy to Gitea at /gitea/.
// authUser is injected as X-WEBAUTH-USER for reverse proxy authentication.
func WithGiteaProxy(giteaURL, authUser string) ServerOption {
	return func(s *Server) {
		s.giteaProxy = proxy.NewGiteaProxy(giteaURL, authUser)
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

// Server handles incoming webhooks from registered projects.
type Server struct {
	cfg          *config.Config
	registry     port.ProjectStore
	store        port.AgentStore
	prTracker    port.PRTrackingStore
	client       *gitea.Client
	spawner      port.AgentSpawner
	prService    *service.PRService
	logger       *log.Logger
	port         int
	dashboard    *dashboard.Server
	giteaProxy   http.Handler
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
	client := gitea.NewClientWithToken(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.APIToken)
	logger := log.New(log.Writer(), "[orchestrator] ", log.LstdFlags)
	s := &Server{
		cfg:       cfg,
		registry:  registry,
		store:     store,
		prTracker: prTracker,
		client:    client,
		spawner:   &defaultSpawner{},
		prService: service.NewPRService(client, &defaultSpawner{}, logger),
		logger:    logger,
		port:      port,
		tracer:    defaultTracer,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// newTestableServer creates a server with a custom spawner and client for testing.
func newTestableServer(cfg *config.Config, registry port.ProjectStore, store port.AgentStore, prTracker port.PRTrackingStore, spawner port.AgentSpawner, client *gitea.Client) *Server {
	logger := log.New(log.Writer(), "[orchestrator] ", log.LstdFlags)
	return &Server{
		cfg:       cfg,
		registry:  registry,
		store:     store,
		prTracker: prTracker,
		client:    client,
		spawner:   spawner,
		prService: service.NewPRService(client, spawner, logger),
		logger:    logger,
		port:      3001,
		tracer:    defaultTracer,
	}
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

	// Manual routes: webhook is Gitea-defined, not our API spec.
	mux.HandleFunc("POST /webhook", s.handleWebhook)

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

	// Wire generated OpenAPI routes (health, agents, quota, tracks, status, locks).
	var sseClients func() int
	if s.dashboard != nil {
		sseClients = s.dashboard.SSEClientCount
	}
	var eventBus port.EventBus
	if s.dashboard != nil {
		eventBus = s.dashboard.EventBus()
	}
	projectSvc := service.NewProjectService(s.registry, s.client, service.ProjectServiceConfig{
		DataDir:          s.cfg.DataDir,
		OrchestratorPort: s.cfg.OrchestratorPort,
		GiteaAdminUser:   s.cfg.GiteaAdminUser,
		APIToken:         s.cfg.APIToken,
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
		GiteaURL:     s.cfg.GiteaURL(),
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

	// Mount Gitea reverse proxy at /gitea/. Path stripping is handled by
	// the proxy handler so Gitea receives requests at its expected paths.
	if s.giteaProxy != nil {
		mux.Handle("/gitea/", s.giteaProxy)
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

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	event := r.Header.Get("X-Gitea-Event")

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.logger.Printf("Error decoding webhook: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	proj, ok := s.resolveProject(payload)
	if !ok {
		s.logger.Printf("Ignoring event from unknown repo")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ignored"})
		return
	}

	slug := proj.Slug

	// For PR events, try to join the track's trace for continuity.
	ctx := r.Context()
	if trackID := extractTrackIDFromPayload(payload); trackID != "" && s.boardSvc != nil {
		if storedTraceID, ok := s.boardSvc.GetTraceID(slug, trackID); ok {
			ctx, _ = s.tracer.StartSpanWithTraceID(ctx, storedTraceID, "webhook/"+event,
				port.StringAttr("webhook.event", event),
				port.StringAttr("project.slug", slug),
				port.StringAttr("track.id", trackID),
			)
		}
	}

	// Record a trace span for the webhook event.
	_, span := trace.SpanFromContext(ctx).TracerProvider().
		Tracer("kiloforge/webhook").
		Start(ctx, "webhook/"+event,
			trace.WithAttributes(
				otelattr.String("webhook.event", event),
				otelattr.String("project.slug", slug),
			))
	defer span.End()

	switch event {
	case "issues":
		s.handleIssues(slug, payload)
	case "issue_comment":
		s.handleIssueComment(slug, payload)
	case "pull_request":
		s.handlePullRequest(slug, payload)
		if action, _ := payload["action"].(string); action == "opened" || action == "reopened" {
			span.AddEvent("pr.opened", trace.WithAttributes(otelattr.String("action", action)))
			if prData, _ := payload["pull_request"].(map[string]any); prData != nil {
				prNum := int(prData["number"].(float64))
				span.SetAttributes(otelattr.Int("pr.number", prNum))
			}
		}
	case "pull_request_review":
		s.handlePullRequestReview(slug, payload)
		if review, _ := payload["review"].(map[string]any); review != nil {
			state, _ := review["state"].(string)
			span.AddEvent("review." + state)
		}
	case "pull_request_comment":
		s.handlePullRequestComment(slug, payload)
	case "push":
		s.handlePush(slug, payload)
	default:
		s.logger.Printf("[%s] Unhandled event: %s", slug, event)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}

func (s *Server) resolveProject(payload map[string]any) (domain.Project, bool) {
	repo, ok := payload["repository"].(map[string]any)
	if !ok {
		return domain.Project{}, false
	}
	repoName, _ := repo["name"].(string)
	if repoName == "" {
		return domain.Project{}, false
	}
	return s.registry.FindByRepoName(repoName)
}

func (s *Server) handleIssues(slug string, payload map[string]any) {
	action, _ := payload["action"].(string)
	issue, _ := payload["issue"].(map[string]any)
	if issue == nil {
		return
	}

	number := int(issue["number"].(float64))
	title, _ := issue["title"].(string)

	switch action {
	case "opened":
		s.logger.Printf("[%s] Issue #%d created: %q", slug, number, title)
	case "edited":
		s.logger.Printf("[%s] Issue #%d edited: %q", slug, number, title)
	case "closed":
		s.logger.Printf("[%s] Issue #%d closed: %q", slug, number, title)
	case "label_updated":
		s.logger.Printf("[%s] Issue #%d labels updated: %q", slug, number, title)
	case "assigned":
		s.logger.Printf("[%s] Issue #%d assigned: %q", slug, number, title)
	default:
		s.logger.Printf("[%s] Issue #%d %s: %q", slug, number, action, title)
	}
}

func (s *Server) handleIssueComment(slug string, payload map[string]any) {
	action, _ := payload["action"].(string)
	comment, _ := payload["comment"].(map[string]any)
	issue, _ := payload["issue"].(map[string]any)
	if comment == nil || issue == nil {
		return
	}

	number := int(issue["number"].(float64))
	body, _ := comment["body"].(string)
	if len(body) > 60 {
		body = body[:60] + "..."
	}

	if action == "created" {
		s.logger.Printf("[%s] Issue #%d comment: %s", slug, number, body)
	}
}

func (s *Server) handlePullRequest(slug string, payload map[string]any) {
	action, _ := payload["action"].(string)
	pr, _ := payload["pull_request"].(map[string]any)
	if pr == nil {
		return
	}

	prNumber := int(pr["number"].(float64))
	prTitle, _ := pr["title"].(string)

	switch action {
	case "opened", "reopened":
		s.logger.Printf("[%s] PR #%d opened: %q", slug, prNumber, prTitle)
		s.createPRTracking(slug, prNumber, pr)
		s.spawnReviewerForPR(slug, prNumber)
		// Move track card to In Review on native board.
		if s.boardSvc != nil {
			if head, _ := pr["head"].(map[string]any); head != nil {
				if trackID, _ := head["ref"].(string); trackID != "" {
					if _, err := s.boardSvc.MoveCard(slug, trackID, domain.ColumnInReview); err != nil {
						s.logger.Printf("[%s] Board move to in_review: %v", slug, err)
					}
				}
			}
		}
	case "closed":
		merged, _ := pr["merged"].(bool)
		if merged {
			s.logger.Printf("[%s] PR #%d merged: %q", slug, prNumber, prTitle)
		} else {
			s.logger.Printf("[%s] PR #%d closed: %q", slug, prNumber, prTitle)
		}
	case "synchronize":
		s.logger.Printf("[%s] PR #%d updated — new commits pushed", slug, prNumber)
		s.handlePRSynchronize(slug, prNumber)
	default:
		s.logger.Printf("[%s] PR #%d %s: %q", slug, prNumber, action, prTitle)
	}
}

func (s *Server) createPRTracking(slug string, prNumber int, pr map[string]any) {
	// Extract track ID from branch name (head ref).
	head, _ := pr["head"].(map[string]any)
	if head == nil {
		return
	}
	branchRef, _ := head["ref"].(string)
	if branchRef == "" {
		return
	}

	tracking := s.prService.CreateTracking(prNumber, branchRef, slug, s.store.Agents(), 3)

	if err := s.prTracker.SavePRTracking(slug, tracking); err != nil {
		s.logger.Printf("[%s] Error saving PR tracking: %v", slug, err)
		return
	}

	// Update developer agent status.
	if tracking.DeveloperAgentID != "" {
		if err := s.store.UpdateStatus(tracking.DeveloperAgentID, "waiting-review"); err != nil {
			s.logger.Printf("[%s] Error updating agent status: %v", slug, err)
		}
		if err := s.store.Save(); err != nil {
			s.logger.Printf("[%s] Error saving state: %v", slug, err)
		}
	}

	s.logger.Printf("[%s] PR #%d tracking created (track: %s)", slug, prNumber, branchRef)
}

func (s *Server) spawnReviewerForPR(slug string, prNumber int) {
	tracking, err := s.prTracker.LoadPRTracking(slug)
	if err != nil {
		s.logger.Printf("[%s] Cannot load PR tracking for reviewer spawn: %v", slug, err)
		return
	}

	prURL := fmt.Sprintf("%s/%s/%s/pulls/%d", s.cfg.GiteaURL(), s.cfg.GiteaAdminUser, slug, prNumber)
	logDir := filepath.Join(s.cfg.DataDir, "projects", slug, "logs")

	workDir := tracking.DeveloperWorkDir
	if workDir == "" {
		workDir = filepath.Join(s.cfg.DataDir, "projects", slug)
	}

	// Validate reviewer skill is installed before spawning.
	required := skills.RequiredSkillsForRole("reviewer")
	globalDir := s.cfg.GetSkillsDir()
	localDir := filepath.Join(workDir, ".claude", "skills")
	if missing := skills.CheckRequired(required, globalDir, localDir); len(missing) > 0 {
		s.logger.Printf("[%s] Cannot spawn reviewer for PR #%d: required skill %q not installed", slug, prNumber, missing[0].Name)
		return
	}

	info, err := s.spawner.SpawnReviewer(context.Background(), port.ReviewerOpts{
		PRNumber: prNumber,
		PRURL:    prURL,
		WorkDir:  workDir,
		LogDir:   logDir,
		Model:    s.cfg.Model,
	})
	if err != nil {
		s.logger.Printf("[%s] Error spawning reviewer for PR #%d: %v", slug, prNumber, err)
		return
	}

	// Update tracking with reviewer info.
	tracking.ReviewerAgentID = info.ID
	tracking.ReviewerSession = info.SessionID
	tracking.Status = "in-review"
	if err := s.prTracker.SavePRTracking(slug, tracking); err != nil {
		s.logger.Printf("[%s] Error saving PR tracking: %v", slug, err)
	}

	if err := s.store.AddAgent(*info); err != nil {
		s.logger.Printf("[%s] Error adding agent: %v", slug, err)
	}
	if err := s.store.Save(); err != nil {
		s.logger.Printf("[%s] Error saving state: %v", slug, err)
	}

	s.logger.Printf("[%s] Reviewer spawned for PR #%d (agent: %s)", slug, prNumber, info.ID)
}

func (s *Server) handlePullRequestReview(slug string, payload map[string]any) {
	review, _ := payload["review"].(map[string]any)
	pr, _ := payload["pull_request"].(map[string]any)
	if review == nil || pr == nil {
		return
	}

	prNumber := int(pr["number"].(float64))
	reviewState, _ := review["state"].(string)

	s.logger.Printf("[%s] PR #%d review %s", slug, prNumber, reviewState)

	tracking, err := s.prTracker.LoadPRTracking(slug)
	if err != nil {
		s.logger.Printf("[%s] Cannot load PR tracking for review handling: %v", slug, err)
		return
	}

	switch reviewState {
	case "approved":
		s.handleReviewApproved(slug, tracking)
	case "changes_requested", "request_changes":
		s.handleReviewChangesRequested(slug, tracking)
	}
}

func (s *Server) handleReviewApproved(slug string, tracking *domain.PRTracking) {
	s.logger.Printf("[%s] PR #%d approved — merging and cleaning up", slug, tracking.PRNumber)

	// Reconstruct trace context for merge spans.
	ctx := context.Background()
	var mergeSpan port.SpanEnder
	if tracking.TrackID != "" && s.boardSvc != nil {
		if storedTraceID, ok := s.boardSvc.GetTraceID(slug, tracking.TrackID); ok {
			ctx, mergeSpan = s.tracer.StartSpanWithTraceID(ctx, storedTraceID, "track.merge",
				port.StringAttr("track.id", tracking.TrackID),
				port.IntAttr("pr.number", tracking.PRNumber),
			)
		}
	}
	if mergeSpan == nil {
		_, mergeSpan = s.tracer.StartSpan(ctx, "track.merge",
			port.StringAttr("track.id", tracking.TrackID),
			port.IntAttr("pr.number", tracking.PRNumber),
		)
	}
	defer mergeSpan.End()

	tracking.Status = "approved"
	if err := s.prTracker.SavePRTracking(slug, tracking); err != nil {
		s.logger.Printf("[%s] Error saving PR tracking: %v", slug, err)
	}

	// Load pool for worktree return.
	p, err := pool.Load(s.cfg.DataDir)
	if err != nil {
		s.logger.Printf("[%s] Error loading pool: %v", slug, err)
		p = nil
	}

	var poolRet port.PoolReturner
	if p != nil {
		poolRet = &poolReturnerAdapter{pool: p, dataDir: s.cfg.DataDir}
	}

	opts := service.CleanupOpts{
		Tracking:    tracking,
		AgentStore:  s.store,
		Merger:      s.client,
		PoolReturn:  poolRet,
		MergeMethod: "merge",
	}

	mergeSpan.AddEvent("pr.merge")
	if err := service.MergeAndCleanup(context.Background(), opts); err != nil {
		s.logger.Printf("[%s] Error in merge/cleanup: %v", slug, err)
		mergeSpan.SetError(err)
		return
	}
	mergeSpan.AddEvent("agents.cleanup")

	// Save tracking.
	if err := s.prTracker.SavePRTracking(slug, tracking); err != nil {
		s.logger.Printf("[%s] Error saving PR tracking: %v", slug, err)
	}

	// Move track card to Done on native board.
	if s.boardSvc != nil && tracking.TrackID != "" {
		if _, err := s.boardSvc.MoveCard(slug, tracking.TrackID, domain.ColumnDone); err != nil {
			s.logger.Printf("[%s] Board move to done: %v", slug, err)
		}
	}

	mergeSpan.AddEvent("track.completed")
	s.logger.Printf("[%s] PR #%d merged and cleaned up (track: %s)", slug, tracking.PRNumber, tracking.TrackID)
}

// poolReturnerAdapter wraps pool.Pool to implement port.PoolReturner.
type poolReturnerAdapter struct {
	pool    *pool.Pool
	dataDir string
}

func (a *poolReturnerAdapter) ReturnByTrackID(trackID string) error {
	if err := a.pool.ReturnByTrackID(trackID); err != nil {
		return err
	}
	return a.pool.Save(a.dataDir)
}

func (s *Server) handleReviewChangesRequested(slug string, tracking *domain.PRTracking) {
	resumeDev := s.prService.HandleChangesRequested(tracking)

	if !resumeDev {
		s.logger.Printf("[%s] PR #%d review cycle limit reached (%d/%d) — escalating",
			slug, tracking.PRNumber, tracking.ReviewCycleCount, tracking.MaxReviewCycles)
		s.escalatePR(slug, tracking)
		return
	}

	s.logger.Printf("[%s] PR #%d changes requested (cycle %d/%d) — resuming developer",
		slug, tracking.PRNumber, tracking.ReviewCycleCount, tracking.MaxReviewCycles)
	if err := s.prTracker.SavePRTracking(slug, tracking); err != nil {
		s.logger.Printf("[%s] Error saving PR tracking: %v", slug, err)
	}

	// Resume developer for revisions.
	if tracking.DeveloperSession != "" {
		workDir := tracking.DeveloperWorkDir
		if workDir == "" {
			workDir = filepath.Join(s.cfg.DataDir, "projects", slug)
		}
		if err := s.spawner.ResumeDeveloper(context.Background(), tracking.DeveloperSession, workDir); err != nil {
			s.logger.Printf("[%s] Error resuming developer: %v", slug, err)
			return
		}
		if err := s.store.UpdateStatus(tracking.DeveloperAgentID, "running"); err != nil {
			s.logger.Printf("[%s] Error updating agent status: %v", slug, err)
		}
		_ = s.store.Save()
	}
}

func (s *Server) escalatePR(slug string, tracking *domain.PRTracking) {
	s.prService.Escalate(context.Background(), tracking, s.client)

	// Stop agents.
	if tracking.DeveloperAgentID != "" {
		_ = s.store.HaltAgent(tracking.DeveloperAgentID)
		if err := s.store.UpdateStatus(tracking.DeveloperAgentID, "stopped"); err != nil {
			s.logger.Printf("[%s] Error updating agent status: %v", slug, err)
		}
	}
	if tracking.ReviewerAgentID != "" {
		_ = s.store.HaltAgent(tracking.ReviewerAgentID)
		if err := s.store.UpdateStatus(tracking.ReviewerAgentID, "stopped"); err != nil {
			s.logger.Printf("[%s] Error updating agent status: %v", slug, err)
		}
	}
	_ = s.store.Save()

	tracking.Status = "escalated"
	if err := s.prTracker.SavePRTracking(slug, tracking); err != nil {
		s.logger.Printf("[%s] Error saving PR tracking: %v", slug, err)
	}
}

func (s *Server) handlePRSynchronize(slug string, prNumber int) {
	tracking, err := s.prTracker.LoadPRTracking(slug)
	if err != nil {
		s.logger.Printf("[%s] Cannot load PR tracking for synchronize: %v", slug, err)
		return
	}

	// Mark developer as waiting-review and spawn new reviewer.
	if tracking.DeveloperAgentID != "" {
		if err := s.store.UpdateStatus(tracking.DeveloperAgentID, "waiting-review"); err != nil {
			s.logger.Printf("[%s] Error updating agent status: %v", slug, err)
		}
		_ = s.store.Save()
	}

	tracking.Status = "waiting-review"
	if err := s.prTracker.SavePRTracking(slug, tracking); err != nil {
		s.logger.Printf("[%s] Error saving PR tracking: %v", slug, err)
	}

	s.spawnReviewerForPR(slug, prNumber)
}

func (s *Server) handlePullRequestComment(slug string, payload map[string]any) {
	action, _ := payload["action"].(string)
	comment, _ := payload["comment"].(map[string]any)
	pr, _ := payload["pull_request"].(map[string]any)
	if comment == nil {
		return
	}

	body, _ := comment["body"].(string)
	if len(body) > 60 {
		body = body[:60] + "..."
	}

	prNumber := 0
	if pr != nil {
		prNumber = int(pr["number"].(float64))
	}

	if action == "created" {
		s.logger.Printf("[%s] PR #%d comment: %s", slug, prNumber, body)
	}
}

// extractTrackIDFromPayload extracts the track ID from a webhook payload
// by looking at the PR head branch ref (e.g., "feature/my-track_123Z" → "my-track_123Z").
func extractTrackIDFromPayload(payload map[string]any) string {
	pr, _ := payload["pull_request"].(map[string]any)
	if pr == nil {
		return ""
	}
	head, _ := pr["head"].(map[string]any)
	if head == nil {
		return ""
	}
	ref, _ := head["ref"].(string)
	if ref == "" {
		return ""
	}
	// Branch format: {type}/{trackId} — extract the track ID after the slash.
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '/' {
			return ref[i+1:]
		}
	}
	return ref
}

func (s *Server) handlePush(slug string, payload map[string]any) {
	ref, _ := payload["ref"].(string)
	commits, _ := payload["commits"].([]any)
	s.logger.Printf("[%s] Push to %s — %d commit(s)", slug, ref, len(commits))
}
