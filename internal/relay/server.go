package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"crelay/internal/config"
	"crelay/internal/orchestration"
	"crelay/internal/project"
	"crelay/internal/state"
)

// Server handles incoming webhooks from registered projects.
type Server struct {
	cfg      *config.Config
	registry *project.Registry
	store    *state.Store
	logger   *log.Logger
	port     int
}

// NewServer creates a relay server with multi-project routing via the registry.
func NewServer(cfg *config.Config, registry *project.Registry, port int) *Server {
	store, err := state.Load(cfg.DataDir)
	if err != nil {
		store = &state.Store{}
	}
	return &Server{
		cfg:      cfg,
		registry: registry,
		store:    store,
		logger:   log.New(log.Writer(), "[relay] ", log.LstdFlags),
		port:     port,
	}
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

func (s *Server) resolveProject(payload map[string]any) (project.Project, bool) {
	repo, ok := payload["repository"].(map[string]any)
	if !ok {
		return project.Project{}, false
	}
	repoName, _ := repo["name"].(string)
	if repoName == "" {
		return project.Project{}, false
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
	case "closed":
		merged, _ := pr["merged"].(bool)
		if merged {
			s.logger.Printf("[%s] PR #%d merged: %q", slug, prNumber, prTitle)
		} else {
			s.logger.Printf("[%s] PR #%d closed: %q", slug, prNumber, prTitle)
		}
	case "synchronize":
		s.logger.Printf("[%s] PR #%d updated — new commits pushed", slug, prNumber)
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

	// Find developer agent by track ID (branch ref).
	var devAgentID, devSession string
	for _, a := range s.store.Agents {
		if a.Role == "developer" && a.Ref == branchRef {
			devAgentID = a.ID
			devSession = a.SessionID
			break
		}
	}

	projectDir := filepath.Join(s.cfg.DataDir, "projects", slug)
	tracking := &orchestration.PRTracking{
		PRNumber:         prNumber,
		TrackID:          branchRef,
		ProjectSlug:      slug,
		DeveloperAgentID: devAgentID,
		DeveloperSession: devSession,
		MaxReviewCycles:  5,
		Status:           "waiting-review",
	}

	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		s.logger.Printf("[%s] Error creating project dir: %v", slug, err)
		return
	}
	if err := tracking.Save(projectDir); err != nil {
		s.logger.Printf("[%s] Error saving PR tracking: %v", slug, err)
		return
	}

	// Update developer agent status.
	if devAgentID != "" {
		s.store.UpdateStatus(devAgentID, "waiting-review")
		if err := s.store.Save(s.cfg.DataDir); err != nil {
			s.logger.Printf("[%s] Error saving state: %v", slug, err)
		}
	}

	s.logger.Printf("[%s] PR #%d tracking created (track: %s)", slug, prNumber, branchRef)
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
