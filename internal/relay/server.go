package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"crelay/internal/config"
	"crelay/internal/core/domain"
	"crelay/internal/core/port"
	"crelay/internal/core/service"
	"crelay/internal/gitea"
	"crelay/internal/orchestration"
	"crelay/internal/pool"
	"crelay/internal/project"
	"crelay/internal/state"
)

// Server handles incoming webhooks from registered projects.
type Server struct {
	cfg       *config.Config
	registry  *project.Registry
	store     *state.Store
	client    *gitea.Client
	spawner   port.AgentSpawner
	prService *service.PRService
	logger    *log.Logger
	port      int
}

// NewServer creates a relay server with multi-project routing via the registry.
func NewServer(cfg *config.Config, registry *project.Registry, port int) *Server {
	store, err := state.Load(cfg.DataDir)
	if err != nil {
		store = &state.Store{}
	}
	client := gitea.NewClient(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.GiteaAdminPass)
	if cfg.APIToken != "" {
		client.SetToken(cfg.APIToken)
	}
	logger := log.New(log.Writer(), "[relay] ", log.LstdFlags)
	return &Server{
		cfg:       cfg,
		registry:  registry,
		store:     store,
		client:    client,
		spawner:   &defaultSpawner{},
		prService: service.NewPRService(client, &defaultSpawner{}, logger),
		logger:    logger,
		port:      port,
	}
}

// newTestableServer creates a server with a custom spawner and client for testing.
func newTestableServer(cfg *config.Config, registry *project.Registry, spawner port.AgentSpawner, client *gitea.Client) *Server {
	store, _ := state.Load(cfg.DataDir)
	if store == nil {
		store = &state.Store{}
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
	mux.HandleFunc("/webhook", s.handleWebhook)
	mux.HandleFunc("/health", s.handleHealth)

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
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":   "ok",
		"projects": len(s.registry.Projects),
	})
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

	switch event {
	case "issues":
		s.handleIssues(slug, payload)
	case "issue_comment":
		s.handleIssueComment(slug, payload)
	case "pull_request":
		s.handlePullRequest(slug, payload)
	case "pull_request_review":
		s.handlePullRequestReview(slug, payload)
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
	tracking := s.prService.CreateTracking(prNumber, branchRef, slug, s.store.Agents, 3)

	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		s.logger.Printf("[%s] Error creating project dir: %v", slug, err)
		return
	}
	if err := orchestration.SavePRTracking(tracking, projectDir); err != nil {
		s.logger.Printf("[%s] Error saving PR tracking: %v", slug, err)
		return
	}

	// Update developer agent status.
	if tracking.DeveloperAgentID != "" {
		s.store.UpdateStatus(tracking.DeveloperAgentID, "waiting-review")
		if err := s.store.Save(s.cfg.DataDir); err != nil {
			s.logger.Printf("[%s] Error saving state: %v", slug, err)
		}
	}

	s.logger.Printf("[%s] PR #%d tracking created (track: %s)", slug, prNumber, branchRef)
}

func (s *Server) spawnReviewerForPR(slug string, prNumber int) {
	projectDir := filepath.Join(s.cfg.DataDir, "projects", slug)
	tracking, err := orchestration.LoadPRTracking(projectDir)
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
	if err := orchestration.SavePRTracking(tracking, projectDir); err != nil {
		s.logger.Printf("[%s] Error saving PR tracking: %v", slug, err)
	}

	s.store.AddAgent(*info)
	if err := s.store.Save(s.cfg.DataDir); err != nil {
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
	tracking, err := orchestration.LoadPRTracking(projectDir)
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
	if err := orchestration.SavePRTracking(tracking, projectDir); err != nil {
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

	opts := orchestration.CleanupOpts{
		Tracking:    tracking,
		Store:       s.store,
		Merger:      s.client,
		PoolReturn:  poolRet,
		DataDir:     s.cfg.DataDir,
		MergeMethod: "merge",
	}

	if err := orchestration.MergeAndCleanup(context.Background(), opts); err != nil {
		s.logger.Printf("[%s] Error in merge/cleanup: %v", slug, err)
		return
	}

	// Save tracking.
	if err := orchestration.SavePRTracking(tracking, projectDir); err != nil {
		s.logger.Printf("[%s] Error saving PR tracking: %v", slug, err)
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
	if err := orchestration.SavePRTracking(tracking, projectDir); err != nil {
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
		_ = s.store.Save(s.cfg.DataDir)
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
	_ = s.store.Save(s.cfg.DataDir)

	tracking.Status = "escalated"
	if err := orchestration.SavePRTracking(tracking, projectDir); err != nil {
		s.logger.Printf("[%s] Error saving PR tracking: %v", slug, err)
	}
}

func (s *Server) handlePRSynchronize(slug string, prNumber int) {
	projectDir := filepath.Join(s.cfg.DataDir, "projects", slug)
	tracking, err := orchestration.LoadPRTracking(projectDir)
	if err != nil {
		s.logger.Printf("[%s] Cannot load PR tracking for synchronize: %v", slug, err)
		return
	}

	// Mark developer as waiting-review and spawn new reviewer.
	if tracking.DeveloperAgentID != "" {
		s.store.UpdateStatus(tracking.DeveloperAgentID, "waiting-review")
		_ = s.store.Save(s.cfg.DataDir)
	}

	tracking.Status = "waiting-review"
	if err := orchestration.SavePRTracking(tracking, projectDir); err != nil {
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
