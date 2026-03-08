package badge

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"kiloforge/internal/core/domain"
)

// AgentFinder provides read-only access to agent data for badge rendering.
type AgentFinder interface {
	Agents() []domain.AgentInfo
	FindAgent(idPrefix string) (*domain.AgentInfo, error)
	Load() error
}

// PRTrackingLoader loads PR tracking for a project.
type PRTrackingLoader func(slug string) (*domain.PRTracking, error)

// Handler serves badge endpoints.
type Handler struct {
	agents    AgentFinder
	prLoader  PRTrackingLoader
}

// NewHandler creates a badge handler.
func NewHandler(agents AgentFinder, prLoader PRTrackingLoader) *Handler {
	return &Handler{agents: agents, prLoader: prLoader}
}

// RegisterRoutes adds badge routes to the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/badges/track/{trackId}", h.handleTrackBadge)
	mux.HandleFunc("GET /api/badges/pr/{slug}/{prNumber}", h.handlePRBadge)
	mux.HandleFunc("GET /api/badges/agent/{agentId}", h.handleAgentBadge)
}

func writeSVG(w http.ResponseWriter, svg []byte) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Expires", time.Now().UTC().Format(http.TimeFormat))
	w.Write(svg)
}

func (h *Handler) handleTrackBadge(w http.ResponseWriter, r *http.Request) {
	trackID := r.PathValue("trackId")
	if trackID == "" {
		writeSVG(w, RenderBadge("track", "unknown"))
		return
	}

	_ = h.agents.Load()
	agents := h.agents.Agents()

	// Find agent whose Ref matches the trackID — pick most recent.
	var best *domain.AgentInfo
	for i := range agents {
		if agents[i].Ref == trackID {
			if best == nil || agents[i].StartedAt.After(best.StartedAt) {
				best = &agents[i]
			}
		}
	}

	label := shortID(trackID)
	if best == nil {
		writeSVG(w, RenderBadge(label, "pending"))
		return
	}
	writeSVG(w, RenderBadge(label, best.Status))
}

func (h *Handler) handlePRBadge(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	prNumStr := r.PathValue("prNumber")
	prNum, _ := strconv.Atoi(prNumStr)

	label := fmt.Sprintf("PR #%d", prNum)

	if h.prLoader == nil {
		writeSVG(w, RenderBadge(label, "unknown"))
		return
	}

	tracking, err := h.prLoader(slug)
	if err != nil || tracking == nil {
		writeSVG(w, RenderBadge(label, "unknown"))
		return
	}

	_ = h.agents.Load()
	devStatus := agentStatus(h.agents, tracking.DeveloperAgentID)
	revStatus := agentStatus(h.agents, tracking.ReviewerAgentID)

	writeSVG(w, RenderDualBadge(label, devStatus, revStatus))
}

func (h *Handler) handleAgentBadge(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agentId")

	_ = h.agents.Load()
	a, err := h.agents.FindAgent(agentID)
	if err != nil {
		writeSVG(w, RenderBadge("agent", "unknown"))
		return
	}

	writeSVG(w, RenderBadge(a.Role, a.Status))
}

func agentStatus(agents AgentFinder, id string) string {
	if id == "" {
		return "pending"
	}
	a, err := agents.FindAgent(id)
	if err != nil {
		return "unknown"
	}
	return a.Status
}

// shortID returns the first meaningful portion of a track ID for badge labels.
func shortID(trackID string) string {
	// Remove timestamp suffix like _20260308190000Z
	if idx := strings.LastIndex(trackID, "_"); idx > 0 {
		short := trackID[:idx]
		if len(short) > 20 {
			short = short[:20]
		}
		return short
	}
	if len(trackID) > 20 {
		return trackID[:20]
	}
	return trackID
}
