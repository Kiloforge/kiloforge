// Package kf provides a Go SDK for reading and writing Kiloforge track
// management data. It operates on the structured file formats under
// .agent/kf/ — tracks.yaml, deps.yaml, conflicts.yaml, and per-track
// track.yaml files.
//
// All types use stable, canonical field ordering when serialized.
package kf

import "time"

// Status constants for track lifecycle.
const (
	StatusPending    = "pending"
	StatusInProgress = "in-progress"
	StatusCompleted  = "completed"
	StatusArchived   = "archived"
)

// TrackEntry is a single row from tracks.yaml — the track registry.
type TrackEntry struct {
	ID            string `json:"id" yaml:"-"` // Derived from the line key, not stored in JSON
	Title         string `json:"title" yaml:"title"`
	Status        string `json:"status" yaml:"status"`
	Type          string `json:"type" yaml:"type"`
	Created       string `json:"created" yaml:"created"`
	Updated       string `json:"updated" yaml:"updated"`
	ArchivedAt    string `json:"archived_at,omitempty" yaml:"archived_at,omitempty"`
	ArchiveReason string `json:"archive_reason,omitempty" yaml:"archive_reason,omitempty"`
}

// IsActive returns true if the track is pending or in-progress.
func (t TrackEntry) IsActive() bool {
	return t.Status == StatusPending || t.Status == StatusInProgress
}

// Spec holds the specification section of a track.yaml.
type Spec struct {
	Summary            string   `yaml:"summary"`
	Context            string   `yaml:"context,omitempty"`
	CodebaseAnalysis   string   `yaml:"codebase_analysis,omitempty"`
	AcceptanceCriteria []string `yaml:"acceptance_criteria,omitempty"`
	OutOfScope         string   `yaml:"out_of_scope,omitempty"`
	TechnicalNotes     string   `yaml:"technical_notes,omitempty"`
}

// Task is a single item in a plan phase.
type Task struct {
	Text string `yaml:"text"`
	Done bool   `yaml:"done"`
}

// Phase is a group of related tasks.
type Phase struct {
	Name  string `yaml:"phase"`
	Tasks []Task `yaml:"tasks"`
}

// Track is the full structured content of a per-track track.yaml file.
type Track struct {
	ID      string                 `yaml:"id"`
	Title   string                 `yaml:"title"`
	Type    string                 `yaml:"type"`
	Status  string                 `yaml:"status"`
	Created string                 `yaml:"created"`
	Updated string                 `yaml:"updated"`
	Spec    Spec                   `yaml:"spec"`
	Plan    []Phase                `yaml:"plan"`
	Extra   map[string]interface{} `yaml:"extra,omitempty"`
}

// Progress computes completion statistics for a track's plan.
func (t Track) Progress() ProgressStats {
	var stats ProgressStats
	stats.TotalPhases = len(t.Plan)
	for _, phase := range t.Plan {
		phaseTotal := len(phase.Tasks)
		phaseDone := 0
		for _, task := range phase.Tasks {
			if task.Done {
				phaseDone++
			}
		}
		stats.TotalTasks += phaseTotal
		stats.CompletedTasks += phaseDone
		if phaseTotal > 0 && phaseDone == phaseTotal {
			stats.CompletedPhases++
		}
	}
	if stats.TotalTasks > 0 {
		stats.Percent = stats.CompletedTasks * 100 / stats.TotalTasks
	}
	return stats
}

// ProgressStats holds completion counts.
type ProgressStats struct {
	TotalPhases     int
	CompletedPhases int
	TotalTasks      int
	CompletedTasks  int
	Percent         int
}

// ConflictPair represents a conflict risk between two tracks.
// The pair key is always strictly ordered: lower ID first.
type ConflictPair struct {
	TrackA string // Lower alphabetically
	TrackB string // Higher alphabetically
	Risk   string `json:"risk"`
	Note   string `json:"note"`
	Added  string `json:"added"`
}

// PairKey returns the canonical "trackA/trackB" key.
func (c ConflictPair) PairKey() string {
	return c.TrackA + "/" + c.TrackB
}

// Involves returns true if the given track ID is part of this pair.
func (c ConflictPair) Involves(trackID string) bool {
	return c.TrackA == trackID || c.TrackB == trackID
}

// NowISO returns the current UTC time in ISO 8601 format.
func NowISO() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

// TodayISO returns the current UTC date.
func TodayISO() string {
	return time.Now().UTC().Format("2006-01-02")
}
