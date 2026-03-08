package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"kiloforge/internal/adapter/persistence/sqlite"
	"kiloforge/internal/core/domain"
)

// TourHandler serves guided tour REST endpoints.
type TourHandler struct {
	store *sqlite.TourStore
}

// NewTourHandler creates a tour handler backed by the given store.
func NewTourHandler(store *sqlite.TourStore) *TourHandler {
	return &TourHandler{store: store}
}

// RegisterRoutes adds tour routes to the given mux.
func (h *TourHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tour", h.handleGetTour)
	mux.HandleFunc("PUT /api/tour", h.handleUpdateTour)
	mux.HandleFunc("GET /api/tour/demo-board", h.handleDemoBoard)
}

func (h *TourHandler) handleGetTour(w http.ResponseWriter, _ *http.Request) {
	state, err := h.store.GetTourState()
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

type updateTourRequest struct {
	Action string `json:"action"` // accept, advance, dismiss, complete
	Step   *int   `json:"step,omitempty"`
}

func (h *TourHandler) handleUpdateTour(w http.ResponseWriter, r *http.Request) {
	var req updateTourRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	state, err := h.store.GetTourState()
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()

	switch req.Action {
	case "accept":
		state.Status = "active"
		state.CurrentStep = 0
		state.StartedAt = &now
	case "advance":
		if req.Step != nil {
			state.CurrentStep = *req.Step
		} else {
			state.CurrentStep++
		}
	case "dismiss":
		state.Status = "dismissed"
		state.DismissedAt = &now
	case "complete":
		state.Status = "completed"
		state.CompletedAt = &now
	default:
		http.Error(w, `{"error":"invalid action, must be accept|advance|dismiss|complete"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateTourState(state); err != nil {
		http.Error(w, `{"error":"failed to save tour state"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

func (h *TourHandler) handleDemoBoard(w http.ResponseWriter, _ *http.Request) {
	now := time.Now().UTC()

	board := domain.BoardState{
		Columns: domain.BoardColumns,
		Cards: map[string]domain.BoardCard{
			"demo-auth-flow": {
				TrackID:   "demo-auth-flow",
				Title:     "Add User Authentication",
				Type:      "feature",
				Column:    domain.ColumnBacklog,
				Position:  0,
				MovedAt:   now,
				CreatedAt: now,
			},
			"demo-fix-login": {
				TrackID:   "demo-fix-login",
				Title:     "Fix Login Redirect Loop",
				Type:      "bug",
				Column:    domain.ColumnApproved,
				Position:  0,
				MovedAt:   now,
				CreatedAt: now,
			},
			"demo-refactor-api": {
				TrackID:   "demo-refactor-api",
				Title:     "Refactor API Error Handling",
				Type:      "refactor",
				Column:    domain.ColumnInProgress,
				Position:  0,
				MovedAt:   now,
				CreatedAt: now,
			},
			"demo-add-tests": {
				TrackID:   "demo-add-tests",
				Title:     "Add Integration Tests",
				Type:      "feature",
				Column:    domain.ColumnDone,
				Position:  0,
				MovedAt:   now,
				CreatedAt: now,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(board)
}
