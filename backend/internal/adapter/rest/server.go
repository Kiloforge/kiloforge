package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/adapter/badge"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/dashboard"
	"kiloforge/internal/adapter/gitea"
	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/persistence/jsonfile"
	"kiloforge/internal/adapter/rest/gen"
	"kiloforge/internal/adapter/pool"
	"kiloforge/internal/adapter/proxy"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/adapter/tracing"
	"kiloforge/internal/core/port"
	"kiloforge/internal/core/service"

	otelattr "go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ShutdownTimeout is how long to wait for agents to exit before force-killing.
const ShutdownTimeout = 10 * time.Second

// ServerOption configures optional features on the relay server.
type ServerOption func(*Server)

// WithDashboard enables dashboard routes on the unified server.
func WithDashboard(agents dashboard.AgentLister, quota dashboard.QuotaReader, giteaURL string, projects dashboard.ProjectLister) ServerOption {
	return func(s *Server) {
		s.dashboard = dashboard.New(0, agents, quota, giteaURL, projects)
		s.quotaReader = quota
		s._projects = projects
	}
}

// WithGiteaProxy enables a reverse proxy to Gitea as the catch-all at /.
func WithGiteaProxy(giteaURL string) ServerOption {
	return func(s *Server) {
		s.giteaProxy = proxy.NewGiteaProxy(giteaURL)
	}
}

// WithTracing enables trace store for the trace API endpoints.
func WithTracing(store *tracing.Store) ServerOption {
	return func(s *Server) {
		s.traceStore = store
	}
}

// WithBoardService enables native board API endpoints.
func WithBoardService(svc *service.NativeBoardService) ServerOption {
	return func(s *Server) {
		s.boardSvc = svc
	}
}

// Server handles incoming webhooks from registered projects.
type Server struct {
	cfg         *config.Config
	registry    *jsonfile.ProjectStore
	store       *jsonfile.AgentStore
	client      *gitea.Client
	spawner     port.AgentSpawner
	prService   *service.PRService
	logger      *log.Logger
	port        int
	dashboard   *dashboard.Server
	giteaProxy  http.Handler
	quotaReader QuotaReader
	_projects   dashboard.ProjectLister
	traceStore  *tracing.Store
	boardSvc    *service.NativeBoardService
}

// NewServer creates a relay server with multi-project routing via the registry.
func NewServer(cfg *config.Config, registry *jsonfile.ProjectStore, port int, opts ...ServerOption) *Server {
	store, err := jsonfile.LoadAgentStore(cfg.DataDir)
	if err != nil {
		store = &jsonfile.AgentStore{}
	}
	client := gitea.NewClientWithToken(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.APIToken)
	logger := log.New(log.Writer(), "[relay] ", log.LstdFlags)
	s := &Server{
		cfg:       cfg,
		registry:  registry,
		store:     store,
		client:    client,
		spawner:   &defaultSpawner{},
		prService: service.NewPRService(client, &defaultSpawner{}, logger),
		logger:    logger,
		port:      port,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// newTestableServer creates a server with a custom spawner and client for testing.
func newTestableServer(cfg *config.Config, registry *jsonfile.ProjectStore, spawner port.AgentSpawner, client *gitea.Client) *Server {
	store, _ := jsonfile.LoadAgentStore(cfg.DataDir)
	if store == nil {
		store = &jsonfile.AgentStore{}
	}
	logger := log.New(log.Writer(), "[relay] ", log.LstdFlags)
	return &Server{
		cfg:       cfg,
		registry:  registry,
		store:     store,
		client:    client,
		spawner:   spawner,
		prService: service.NewPRService(client, spawner, logger),
		logger:    logger,
		port:      3001,
	}
}

// defaultSpawner implements port.AgentSpawner using real claude commands.
type defaultSpawner struct{}

func (d *defaultSpawner) SpawnReviewer(ctx context.Context, opts port.ReviewerOpts) (*domain.AgentInfo, error) {
	// In production, use agent.Spawner. For now, use exec directly.
	cmd := exec.CommandContext(ctx, "claude",
		"-p", fmt.Sprintf("/conductor-reviewer %s", opts.PRURL),
		"--output-format", "stream-json",
	)
	cmd.Dir = opts.WorkDir
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start reviewer: %w", err)
	}
	return &domain.AgentInfo{
		ID:     fmt.Sprintf("reviewer-%d", cmd.Process.Pid),
		Role:   "reviewer",
		Ref:    fmt.Sprintf("PR #%d", opts.PRNumber),
		Status: "running",
		PID:    cmd.Process.Pid,
	}, nil
}

func (d *defaultSpawner) ResumeDeveloper(ctx context.Context, sessionID, workDir string) error {
	cmd := exec.CommandContext(ctx, "claude", "--resume", sessionID)
	cmd.Dir = workDir
	return cmd.Start()
}

// Run starts the HTTP server and blocks until the context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()

	// Manual routes: webhook is Gitea-defined, not our API spec.
	mux.HandleFunc("/webhook", s.handleWebhook)

	// Lock service (shared with generated API handler).
	lockMgr := lock.New(s.cfg.DataDir)
	lockMgr.StartReaper(ctx)

	// Wire generated OpenAPI routes (health, agents, quota, tracks, status, locks).
	var sseClients func() int
	if s.dashboard != nil {
		sseClients = s.dashboard.SSEClientCount
	}
	apiHandler := NewAPIHandler(APIHandlerOpts{
		Agents:     s.store,
		Quota:      s.quotaReader,
		LockMgr:    lockMgr,
		Projects:   s._projects,
		TraceStore: s.traceStore,
		BoardSvc:   s.boardSvc,
		GiteaURL:   s.cfg.GiteaURL(),
		SSEClients: sseClients,
		Cfg:        s.cfg,
	})
	strictHandler := gen.NewStrictHandler(apiHandler, nil)
	gen.HandlerFromMux(strictHandler, mux)

	// Badge endpoints (SVG, not JSON — stays manual).
	prLoader := func(slug string) (*domain.PRTracking, error) {
		projectDir := filepath.Join(s.cfg.DataDir, "projects", slug)
		return jsonfile.LoadPRTracking(projectDir)
	}
	badgeHandler := badge.NewHandler(s.store, prLoader)
	badgeHandler.RegisterRoutes(mux)

	// Mount dashboard non-API routes (SSE, HTML pages, SPA static).
	if s.dashboard != nil {
		s.dashboard.RegisterNonAPIRoutes(mux)
		s.dashboard.StartWatcher(ctx)
	}

	// Mount Gitea reverse proxy as catch-all. All non-crelay routes
	// are forwarded to Gitea so its UI and assets load naturally.
	if s.giteaProxy != nil {
		mux.Handle("/", s.giteaProxy)
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

	// Graceful agent shutdown on relay stop.
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

	// Record a trace span for the webhook event.
	_, span := trace.SpanFromContext(r.Context()).TracerProvider().
		Tracer("kiloforge/webhook").
		Start(r.Context(), "webhook/"+event,
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
		}
	case "pull_request_review":
		s.handlePullRequestReview(slug, payload)
		if review, _ := payload["review"].(map[string]any); review != nil {
			state, _ := review["state"].(string)
			span.AddEvent("review."+state)
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

	projectDir := filepath.Join(s.cfg.DataDir, "projects", slug)
	tracking := s.prService.CreateTracking(prNumber, branchRef, slug, s.store.AgentList, 3)

	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		s.logger.Printf("[%s] Error creating project dir: %v", slug, err)
		return
	}
	if err := jsonfile.SavePRTracking(tracking, projectDir); err != nil {
		s.logger.Printf("[%s] Error saving PR tracking: %v", slug, err)
		return
	}

	// Update developer agent status.
	if tracking.DeveloperAgentID != "" {
		s.store.UpdateStatus(tracking.DeveloperAgentID, "waiting-review")
		if err := s.store.Save(); err != nil {
			s.logger.Printf("[%s] Error saving state: %v", slug, err)
		}
	}

	s.logger.Printf("[%s] PR #%d tracking created (track: %s)", slug, prNumber, branchRef)
}

func (s *Server) spawnReviewerForPR(slug string, prNumber int) {
	projectDir := filepath.Join(s.cfg.DataDir, "projects", slug)
	tracking, err := jsonfile.LoadPRTracking(projectDir)
	if err != nil {
		s.logger.Printf("[%s] Cannot load PR tracking for reviewer spawn: %v", slug, err)
		return
	}

	prURL := fmt.Sprintf("%s/%s/%s/pulls/%d", s.cfg.GiteaURL(), s.cfg.GiteaAdminUser, slug, prNumber)
	logDir := filepath.Join(s.cfg.DataDir, "projects", slug, "logs")

	workDir := tracking.DeveloperWorkDir
	if workDir == "" {
		workDir = projectDir
	}

	info, err := s.spawner.SpawnReviewer(context.Background(), port.ReviewerOpts{
		PRNumber: prNumber,
		PRURL:    prURL,
		WorkDir:  workDir,
		LogDir:   logDir,
	})
	if err != nil {
		s.logger.Printf("[%s] Error spawning reviewer for PR #%d: %v", slug, prNumber, err)
		return
	}

	// Update tracking with reviewer info.
	tracking.ReviewerAgentID = info.ID
	tracking.ReviewerSession = info.SessionID
	tracking.Status = "in-review"
	if err := jsonfile.SavePRTracking(tracking, projectDir); err != nil {
		s.logger.Printf("[%s] Error saving PR tracking: %v", slug, err)
	}

	s.store.AddAgent(*info)
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

	projectDir := filepath.Join(s.cfg.DataDir, "projects", slug)
	tracking, err := jsonfile.LoadPRTracking(projectDir)
	if err != nil {
		s.logger.Printf("[%s] Cannot load PR tracking for review handling: %v", slug, err)
		return
	}

	switch reviewState {
	case "approved":
		s.handleReviewApproved(slug, tracking, projectDir)
	case "changes_requested", "request_changes":
		s.handleReviewChangesRequested(slug, tracking, projectDir)
	}
}

func (s *Server) handleReviewApproved(slug string, tracking *domain.PRTracking, projectDir string) {
	s.logger.Printf("[%s] PR #%d approved — merging and cleaning up", slug, tracking.PRNumber)

	tracking.Status = "approved"
	if err := jsonfile.SavePRTracking(tracking, projectDir); err != nil {
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

	if err := service.MergeAndCleanup(context.Background(), opts); err != nil {
		s.logger.Printf("[%s] Error in merge/cleanup: %v", slug, err)
		return
	}

	// Save tracking.
	if err := jsonfile.SavePRTracking(tracking, projectDir); err != nil {
		s.logger.Printf("[%s] Error saving PR tracking: %v", slug, err)
	}

	// Move track card to Done on native board.
	if s.boardSvc != nil && tracking.TrackID != "" {
		if _, err := s.boardSvc.MoveCard(slug, tracking.TrackID, domain.ColumnDone); err != nil {
			s.logger.Printf("[%s] Board move to done: %v", slug, err)
		}
	}

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

func (s *Server) handleReviewChangesRequested(slug string, tracking *domain.PRTracking, projectDir string) {
	resumeDev := s.prService.HandleChangesRequested(tracking)

	if !resumeDev {
		s.logger.Printf("[%s] PR #%d review cycle limit reached (%d/%d) — escalating",
			slug, tracking.PRNumber, tracking.ReviewCycleCount, tracking.MaxReviewCycles)
		s.escalatePR(slug, tracking, projectDir)
		return
	}

	s.logger.Printf("[%s] PR #%d changes requested (cycle %d/%d) — resuming developer",
		slug, tracking.PRNumber, tracking.ReviewCycleCount, tracking.MaxReviewCycles)
	if err := jsonfile.SavePRTracking(tracking, projectDir); err != nil {
		s.logger.Printf("[%s] Error saving PR tracking: %v", slug, err)
	}

	// Resume developer for revisions.
	if tracking.DeveloperSession != "" {
		workDir := tracking.DeveloperWorkDir
		if workDir == "" {
			workDir = projectDir
		}
		if err := s.spawner.ResumeDeveloper(context.Background(), tracking.DeveloperSession, workDir); err != nil {
			s.logger.Printf("[%s] Error resuming developer: %v", slug, err)
			return
		}
		s.store.UpdateStatus(tracking.DeveloperAgentID, "running")
		_ = s.store.Save()
	}
}

func (s *Server) escalatePR(slug string, tracking *domain.PRTracking, projectDir string) {
	s.prService.Escalate(context.Background(), tracking, s.client)

	// Stop agents.
	if tracking.DeveloperAgentID != "" {
		_ = s.store.HaltAgent(tracking.DeveloperAgentID)
		s.store.UpdateStatus(tracking.DeveloperAgentID, "stopped")
	}
	if tracking.ReviewerAgentID != "" {
		_ = s.store.HaltAgent(tracking.ReviewerAgentID)
		s.store.UpdateStatus(tracking.ReviewerAgentID, "stopped")
	}
	_ = s.store.Save()

	tracking.Status = "escalated"
	if err := jsonfile.SavePRTracking(tracking, projectDir); err != nil {
		s.logger.Printf("[%s] Error saving PR tracking: %v", slug, err)
	}
}

func (s *Server) handlePRSynchronize(slug string, prNumber int) {
	projectDir := filepath.Join(s.cfg.DataDir, "projects", slug)
	tracking, err := jsonfile.LoadPRTracking(projectDir)
	if err != nil {
		s.logger.Printf("[%s] Cannot load PR tracking for synchronize: %v", slug, err)
		return
	}

	// Mark developer as waiting-review and spawn new reviewer.
	if tracking.DeveloperAgentID != "" {
		s.store.UpdateStatus(tracking.DeveloperAgentID, "waiting-review")
		_ = s.store.Save()
	}

	tracking.Status = "waiting-review"
	if err := jsonfile.SavePRTracking(tracking, projectDir); err != nil {
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

func (s *Server) handlePush(slug string, payload map[string]any) {
	ref, _ := payload["ref"].(string)
	commits, _ := payload["commits"].([]any)
	s.logger.Printf("[%s] Push to %s — %d commit(s)", slug, ref, len(commits))
}
