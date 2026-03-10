package kf

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ReadTrack reads a per-track track.yaml file.
func ReadTrack(path string) (*Track, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Track
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("unmarshal track.yaml: %w", err)
	}
	return &t, nil
}

// ReadTrackByID reads track.yaml from the standard location: tracksDir/<id>/track.yaml.
func ReadTrackByID(tracksDir, trackID string) (*Track, error) {
	path := filepath.Join(tracksDir, trackID, "track.yaml")
	return ReadTrack(path)
}

// WriteTrack writes a track to its track.yaml file with canonical field ordering.
func WriteTrack(path string, t *Track) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	// Use a custom node to enforce field order
	data, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal track.yaml: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// WriteTrackByID writes track.yaml to the standard location: tracksDir/<id>/track.yaml.
func WriteTrackByID(tracksDir string, t *Track) error {
	path := filepath.Join(tracksDir, t.ID, "track.yaml")
	return WriteTrack(path, t)
}

// SetTaskDone sets the done status of a specific task by phase and task index (1-based).
func (t *Track) SetTaskDone(phaseIdx, taskIdx int, done bool) error {
	if phaseIdx < 1 || phaseIdx > len(t.Plan) {
		return fmt.Errorf("phase %d out of range (have %d phases)", phaseIdx, len(t.Plan))
	}
	phase := &t.Plan[phaseIdx-1]
	if taskIdx < 1 || taskIdx > len(phase.Tasks) {
		return fmt.Errorf("task %d out of range in phase %d (have %d tasks)", taskIdx, phaseIdx, len(phase.Tasks))
	}
	phase.Tasks[taskIdx-1].Done = done
	t.Updated = TodayISO()
	return nil
}

// NewTrack creates a new Track with default fields populated.
func NewTrack(id, title, trackType, summary string) *Track {
	today := TodayISO()
	return &Track{
		ID:      id,
		Title:   title,
		Type:    trackType,
		Status:  StatusPending,
		Created: today,
		Updated: today,
		Spec: Spec{
			Summary: summary,
		},
		Extra: make(map[string]interface{}),
	}
}
