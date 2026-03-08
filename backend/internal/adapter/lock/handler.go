package lock

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Handler provides HTTP endpoints for the lock service.
type Handler struct {
	mgr *Manager
}

// NewHandler creates a lock HTTP handler.
func NewHandler(mgr *Manager) *Handler {
	return &Handler{mgr: mgr}
}

// RegisterRoutes adds lock endpoints to the given mux.
// Routes use explicit method prefixes to avoid conflicts with other catch-all
// patterns (e.g. "GET /-/") on the same ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/locks", h.handleList)
	mux.HandleFunc("POST /api/locks/{scope}/acquire", h.handleAcquire)
	mux.HandleFunc("POST /api/locks/{scope}/heartbeat", h.handleHeartbeat)
	mux.HandleFunc("DELETE /api/locks/{scope}", h.handleRelease)
}

type acquireRequest struct {
	Holder         string `json:"holder"`
	TTLSeconds     int    `json:"ttl_seconds"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

type heartbeatRequest struct {
	Holder     string `json:"holder"`
	TTLSeconds int    `json:"ttl_seconds"`
}

type releaseRequest struct {
	Holder string `json:"holder"`
}

type lockResponse struct {
	Scope              string    `json:"scope"`
	Holder             string    `json:"holder"`
	AcquiredAt         time.Time `json:"acquired_at"`
	ExpiresAt          time.Time `json:"expires_at"`
	TTLRemainingSeconds float64  `json:"ttl_remaining_seconds"`
}

type errorResponse struct {
	Error         string `json:"error"`
	CurrentHolder string `json:"current_holder,omitempty"`
}

func lockToResponse(l *Lock) lockResponse {
	remaining := time.Until(l.ExpiresAt).Seconds()
	if remaining < 0 {
		remaining = 0
	}
	return lockResponse{
		Scope:               l.Scope,
		Holder:              l.Holder,
		AcquiredAt:          l.AcquiredAt,
		ExpiresAt:           l.ExpiresAt,
		TTLRemainingSeconds: remaining,
	}
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	locks := h.mgr.List()
	resp := make([]lockResponse, 0, len(locks))
	for i := range locks {
		resp = append(resp, lockToResponse(&locks[i]))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleAcquire(w http.ResponseWriter, r *http.Request) {
	scope := r.PathValue("scope")
	var req acquireRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	if req.Holder == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "holder required"})
		return
	}
	if req.TTLSeconds <= 0 {
		req.TTLSeconds = 60
	}
	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 0 // non-blocking
	}

	ttl := time.Duration(req.TTLSeconds) * time.Second

	ctx := r.Context()
	if req.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
		defer cancel()
	} else {
		// Non-blocking: already-cancelled context so Acquire returns immediately.
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		cancel()
	}

	l, err := h.mgr.Acquire(ctx, scope, req.Holder, ttl)
	if err != nil {
		// Get current holder for error response.
		var currentHolder string
		locks := h.mgr.List()
		for _, existing := range locks {
			if existing.Scope == scope {
				currentHolder = existing.Holder
				break
			}
		}
		writeJSON(w, http.StatusConflict, errorResponse{
			Error:         "timeout waiting for lock",
			CurrentHolder: currentHolder,
		})
		return
	}

	writeJSON(w, http.StatusOK, lockToResponse(l))
}

func (h *Handler) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	scope := r.PathValue("scope")
	var req heartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	if req.Holder == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "holder required"})
		return
	}
	if req.TTLSeconds <= 0 {
		req.TTLSeconds = 60
	}

	l, err := h.mgr.Heartbeat(scope, req.Holder, time.Duration(req.TTLSeconds)*time.Second)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, lockToResponse(l))
}

func (h *Handler) handleRelease(w http.ResponseWriter, r *http.Request) {
	scope := r.PathValue("scope")
	var req releaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	if req.Holder == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "holder required"})
		return
	}

	if err := h.mgr.Release(scope, req.Holder); err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"released": true})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
