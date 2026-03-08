package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/core/domain"
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

// ProjectLister provides read access to registered projects.
type ProjectLister interface {
	List() []domain.Project
}

// Server serves the web dashboard on a dedicated HTTP port.
type Server struct {
	port     int
	agents   AgentLister
	quota    QuotaReader
	giteaURL string
	projects ProjectLister
	hub      *SSEHub
	mux      *http.ServeMux
}

// New creates a dashboard server.
func New(port int, agents AgentLister, quota QuotaReader, giteaURL string, projects ProjectLister) *Server {
	s := &Server{
		port:     port,
		agents:   agents,
		quota:    quota,
		giteaURL: giteaURL,
		projects: projects,
		hub:      NewSSEHub(),
		mux:      http.NewServeMux(),
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

// RegisterNonAPIRoutes mounts only the non-API routes (SSE, HTML pages, SPA static).
// All JSON API routes are served by the generated OpenAPI handler.
func (s *Server) RegisterNonAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /-/events", s.handleSSE)
	mux.HandleFunc("GET /-/tracks/{trackId}", s.handleTrackDetail)
	mux.HandleFunc("GET /-/pr/{slug}/{prNumber}", s.handlePRDetail)
	mux.Handle("GET /-/", http.StripPrefix("/-", spaFileServer(http.FS(staticFS))))
}

// Mux returns the server's internal mux for registering additional routes.
func (s *Server) Mux() *http.ServeMux {
	return s.mux
}

// SSEClientCount returns the number of connected SSE clients.
func (s *Server) SSEClientCount() int {
	return s.hub.ClientCount()
}

func (s *Server) routes() {
	s.RegisterNonAPIRoutes(s.mux)
}
