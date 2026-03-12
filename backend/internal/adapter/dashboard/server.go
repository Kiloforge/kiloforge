package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/adapter/tracing"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
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
	TokensPerMin(window time.Duration) float64
	CostPerHour(window time.Duration) float64
}

// ProjectLister provides read access to registered projects.
type ProjectLister interface {
	List() []domain.Project
}

// NotificationChecker creates and dismisses agent attention notifications.
type NotificationChecker interface {
	Create(agentID, title, body string) error
	DismissForAgent(agentID string) error
	CleanForAgent(agentID string) error
}

// BridgeChecker checks whether an agent has a WebSocket bridge.
type BridgeChecker interface {
	HasBridge(agentID string) bool
}

// Server serves the web dashboard on a dedicated HTTP port.
type Server struct {
	port          int
	agents        AgentLister
	quota         QuotaReader
	projects      ProjectLister
	hub           *SSEHub
	eventBus      port.EventBus
	traceStore    tracing.TraceReader
	trackReader   port.TrackReader
	budgetUSD     float64
	mux           *http.ServeMux
	notifChecker  NotificationChecker
	bridgeChecker BridgeChecker
}

// New creates a dashboard server. If eventBus is nil, a new SSEHub is created
// and used as both the SSE transport and the event bus. If eventBus is provided
// (and is an *SSEHub), it is used directly so the bus can be shared with other
// components like the REST API handler.
func New(port int, agents AgentLister, quota QuotaReader, projects ProjectLister, eventBus port.EventBus) *Server {
	var hub *SSEHub
	if eventBus == nil {
		hub = NewSSEHub()
		eventBus = hub
	} else if h, ok := eventBus.(*SSEHub); ok {
		hub = h
	} else {
		// Non-SSEHub event bus: create an SSEHub that bridges events to SSE clients.
		hub = NewSSEHub()
	}

	s := &Server{
		port:     port,
		agents:   agents,
		quota:    quota,
		projects: projects,
		hub:      hub,
		eventBus: eventBus,
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
	mux.HandleFunc("GET /events", s.handleSSE)
	mux.HandleFunc("GET /tracks/{trackId}", s.handleTrackDetail)
	mux.HandleFunc("GET /pr/{slug}/{prNumber}", s.handlePRDetail)
	// SPA catch-all: method-agnostic so it doesn't conflict with more specific
	// method-scoped patterns above. More specific patterns take priority.
	mux.Handle("/", spaFileServer(http.FS(staticFS)))
}

// Mux returns the server's internal mux for registering additional routes.
func (s *Server) Mux() *http.ServeMux {
	return s.mux
}

// SSEClientCount returns the number of connected SSE clients.
func (s *Server) SSEClientCount() int {
	return s.hub.ClientCount()
}

// EventBus returns the event bus used by this server.
func (s *Server) EventBus() port.EventBus {
	return s.eventBus
}

// SetTraceStore sets the trace store for watcher-driven trace events.
func (s *Server) SetTraceStore(store tracing.TraceReader) {
	s.traceStore = store
}

// SetTrackReader sets the track reader for track discovery.
func (s *Server) SetTrackReader(reader port.TrackReader) {
	s.trackReader = reader
}

// SetBudgetUSD sets the budget limit for quota gauge display.
func (s *Server) SetBudgetUSD(budget float64) {
	s.budgetUSD = budget
}

// SetNotificationChecker sets the notification checker for the watcher.
func (s *Server) SetNotificationChecker(nc NotificationChecker) {
	s.notifChecker = nc
}

// SetBridgeChecker sets the bridge checker for detecting bridgeless agents.
func (s *Server) SetBridgeChecker(bc BridgeChecker) {
	s.bridgeChecker = bc
}

func (s *Server) routes() {
	s.RegisterNonAPIRoutes(s.mux)
}
