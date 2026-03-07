package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"conductor-relay/internal/agent"
	"conductor-relay/internal/config"
	"conductor-relay/internal/gitea"
	"conductor-relay/internal/state"
)

// Server handles incoming webhooks and manages agents.
type Server struct {
	cfg     *config.Config
	client  *gitea.Client
	spawner *agent.Spawner
	store   *state.Store
	logger  *log.Logger
}

func NewServer(cfg *config.Config, client *gitea.Client) *Server {
	store, err := state.Load(cfg.DataDir)
	if err != nil {
		store = &state.Store{}
	}
	return &Server{
		cfg:     cfg,
		client:  client,
		spawner: agent.NewSpawner(cfg, store),
		store:   store,
		logger:  log.New(log.Writer(), "[relay] ", log.LstdFlags),
	}
}

func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", s.handleWebhook)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/agents", s.handleAgents)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.RelayPort),
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
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.store.Agents)
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	event := r.Header.Get("X-Gitea-Event")
	s.logger.Printf("Received webhook: %s", event)

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.logger.Printf("Error decoding webhook: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	switch event {
	case "pull_request":
		s.handlePullRequest(r.Context(), payload)
	case "pull_request_review":
		s.handlePullRequestReview(r.Context(), payload)
	case "pull_request_comment":
		s.handlePullRequestComment(r.Context(), payload)
	default:
		s.logger.Printf("Unhandled event type: %s", event)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}

func (s *Server) handlePullRequest(ctx context.Context, payload map[string]any) {
	action, _ := payload["action"].(string)
	pr, _ := payload["pull_request"].(map[string]any)
	if pr == nil {
		return
	}

	prNumber := int(pr["number"].(float64))
	prTitle, _ := pr["title"].(string)

	switch action {
	case "opened", "reopened":
		// A new PR was created — spawn a reviewer.
		s.logger.Printf("PR #%d opened: %s — spawning reviewer", prNumber, prTitle)
		prURL := fmt.Sprintf("%s/%s/%s/pulls/%d",
			s.client.BaseURL(), config.GiteaAdminUser, s.cfg.RepoName, prNumber)

		info, err := s.spawner.SpawnReviewer(ctx, prNumber, prURL)
		if err != nil {
			s.logger.Printf("Error spawning reviewer: %v", err)
			return
		}
		s.logger.Printf("Reviewer spawned: %s (session: %s)", info.ID[:8], info.SessionID[:8])

	case "synchronize":
		// PR was updated (new commits pushed) — could re-trigger reviewer.
		s.logger.Printf("PR #%d updated — new commits pushed", prNumber)

	default:
		s.logger.Printf("PR action: %s (no handler)", action)
	}
}

func (s *Server) handlePullRequestReview(ctx context.Context, payload map[string]any) {
	action, _ := payload["action"].(string)
	review, _ := payload["review"].(map[string]any)
	pr, _ := payload["pull_request"].(map[string]any)
	if review == nil || pr == nil {
		return
	}

	prNumber := int(pr["number"].(float64))
	reviewState, _ := review["state"].(string)

	s.logger.Printf("PR #%d review %s: %s", prNumber, action, reviewState)

	// Find the developer agent for this PR and notify it.
	prRef := "PR #" + strconv.Itoa(prNumber)
	for _, a := range s.store.Agents {
		if a.Role == "developer" && a.Ref == prRef && a.Status == "waiting" {
			s.logger.Printf("Developer agent %s is waiting for review — would send feedback", a.ID[:8])
			// In a PTY-based implementation, we would write to the agent's stdin here.
			// With stream-json, the developer needs to be resumed manually.
			break
		}
	}
}

func (s *Server) handlePullRequestComment(ctx context.Context, payload map[string]any) {
	action, _ := payload["action"].(string)
	comment, _ := payload["comment"].(map[string]any)
	if comment == nil {
		return
	}

	body, _ := comment["body"].(string)
	s.logger.Printf("PR comment %s: %.60s", action, body)
}
