package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"crelay/internal/adapter/agent"
	"crelay/internal/core/domain"
)

// AgentLister provides read access to agent state.
type AgentLister interface {
	Agents() []domain.AgentInfo
	FindAgent(idPrefix string) (*domain.AgentInfo, error)
	Load() error
}

// QuotaReader provides read access to quota data.
type QuotaReader interface {
	GetAgentUsage(agentID string) *agent.AgentUsage
	GetTotalUsage() agent.TotalUsage
	IsRateLimited() bool
	RetryAfter() time.Duration
}

// Server serves the web dashboard on a dedicated HTTP port.
type Server struct {
	port       int
	agents     AgentLister
	quota      QuotaReader
	giteaURL   string
	projectDir string
	hub        *SSEHub
	mux        *http.ServeMux
}

// New creates a dashboard server.
func New(port int, agents AgentLister, quota QuotaReader, giteaURL, projectDir string) *Server {
	s := &Server{
		port:       port,
		agents:     agents,
		quota:      quota,
		giteaURL:   giteaURL,
		projectDir: projectDir,
		hub:        NewSSEHub(),
		mux:        http.NewServeMux(),
	}
	s.routes()
	return s
}

// Run starts the HTTP server and blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", s.port),
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go s.watchState(ctx)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/agents", s.handleAgents)
	s.mux.HandleFunc("GET /api/agents/{id}", s.handleAgent)
	s.mux.HandleFunc("GET /api/agents/{id}/log", s.handleAgentLog)
	s.mux.HandleFunc("GET /api/quota", s.handleQuota)
	s.mux.HandleFunc("GET /api/tracks", s.handleTracks)
	s.mux.HandleFunc("GET /api/status", s.handleStatus)
	s.mux.HandleFunc("GET /events", s.handleSSE)
	s.mux.Handle("GET /", http.FileServer(http.FS(staticFS)))
}
