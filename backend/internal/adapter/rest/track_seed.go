package rest

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"kiloforge/internal/core/port"
	"kiloforge/pkg/kf"
)

// seedTrackRequest is the JSON body for POST /api/tracks/seed.
type seedTrackRequest struct {
	ProjectSlug string          `json:"project"`
	Tracks      []seedTrackInfo `json:"tracks"`
}

type seedTrackInfo struct {
	ID      string         `json:"id"`
	Title   string         `json:"title"`
	Status  string         `json:"status"`
	Type    string         `json:"type"`
	Spec    *seedTrackSpec `json:"spec,omitempty"`
	Plan    []seedPhase    `json:"plan,omitempty"`
	Created string         `json:"created,omitempty"`
	Updated string         `json:"updated,omitempty"`
}

type seedTrackSpec struct {
	Summary            string   `json:"summary,omitempty"`
	Context            string   `json:"context,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	TechnicalNotes     string   `json:"technical_notes,omitempty"`
}

type seedPhase struct {
	Name  string     `json:"phase"`
	Tasks []seedTask `json:"tasks"`
}

type seedTask struct {
	Text string `json:"text"`
	Done bool   `json:"done"`
}

// handleSeedTracks returns an HTTP handler that seeds track data for E2E tests.
func handleSeedTracks(projects ProjectLister, analytics port.AnalyticsTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req seedTrackRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}

		if req.ProjectSlug == "" || len(req.Tracks) == 0 {
			http.Error(w, `{"error":"project and tracks required"}`, http.StatusBadRequest)
			return
		}

		// Find project directory.
		var projectDir string
		for _, p := range projects.List() {
			if p.Slug == req.ProjectSlug {
				projectDir = p.ProjectDir
				break
			}
		}
		if projectDir == "" {
			http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
			return
		}

		// Ensure kf directory structure exists.
		kfDir := filepath.Join(projectDir, ".agent", "kf")
		tracksDir := filepath.Join(kfDir, "tracks")
		if err := os.MkdirAll(tracksDir, 0o755); err != nil {
			http.Error(w, `{"error":"create kf dirs: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		// Create product.md if missing (needed for IsInitialized checks).
		productFile := filepath.Join(kfDir, "product.md")
		if _, err := os.Stat(productFile); os.IsNotExist(err) {
			_ = os.WriteFile(productFile, []byte("# E2E Test Project\n"), 0o644)
		}

		client := kf.NewClientFromProject(projectDir)

		for _, t := range req.Tracks {
			created := t.Created
			if created == "" {
				created = kf.TodayISO()
			}
			updated := t.Updated
			if updated == "" {
				updated = kf.TodayISO()
			}

			entry := kf.TrackEntry{
				ID:      t.ID,
				Title:   t.Title,
				Status:  mapSeedStatus(t.Status),
				Type:    t.Type,
				Created: created,
				Updated: updated,
			}
			if err := client.AddTrack(entry, nil); err != nil {
				http.Error(w, `{"error":"add track: `+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}
			if analytics != nil {
				analytics.Track(r.Context(), "track_created", map[string]any{
					"track_id": t.ID,
					"type":     t.Type,
				})
			}

			// If spec or plan provided, save the full track.yaml.
			if t.Spec != nil || len(t.Plan) > 0 {
				track := &kf.Track{
					ID:      t.ID,
					Title:   t.Title,
					Type:    t.Type,
					Status:  mapSeedStatus(t.Status),
					Created: created,
					Updated: updated,
				}
				if t.Spec != nil {
					track.Spec = kf.Spec{
						Summary:            t.Spec.Summary,
						Context:            t.Spec.Context,
						AcceptanceCriteria: t.Spec.AcceptanceCriteria,
						TechnicalNotes:     t.Spec.TechnicalNotes,
					}
				}
				for _, p := range t.Plan {
					phase := kf.Phase{Name: p.Name}
					for _, task := range p.Tasks {
						phase.Tasks = append(phase.Tasks, kf.Task{Text: task.Text, Done: task.Done})
					}
					track.Plan = append(track.Plan, phase)
				}
				if err := client.SaveTrack(track); err != nil {
					http.Error(w, `{"error":"save track: `+err.Error()+`"}`, http.StatusInternalServerError)
					return
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"project":     req.ProjectSlug,
			"track_count": len(req.Tracks),
		})
	}
}

// mapSeedStatus maps user-provided status strings to kf SDK statuses.
// Accepts both "complete" and "completed", etc.
func mapSeedStatus(status string) string {
	switch status {
	case "complete":
		return kf.StatusCompleted
	default:
		return status
	}
}
