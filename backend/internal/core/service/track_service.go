package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	StatusComplete   = "complete"
	StatusPending    = "pending"
	StatusInProgress = "in-progress"
	StatusApproved   = "approved"
	StatusInReview   = "in-review"
)

// TrackEntry represents a parsed track from tracks.md.
type TrackEntry struct {
	ID     string
	Title  string
	Status string
}

// ParseTracks parses track entries from a tracks.md reader.
// This is a pure function — I/O stays in the caller.
func ParseTracks(r io.Reader) ([]TrackEntry, error) {
	var tracks []TrackEntry
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		entry, ok := parseTrackLine(line)
		if ok {
			tracks = append(tracks, entry)
		}
	}
	return tracks, scanner.Err()
}

// FilterByStatus returns tracks matching the given status.
func FilterByStatus(tracks []TrackEntry, status string) []TrackEntry {
	var result []TrackEntry
	for _, t := range tracks {
		if t.Status == status {
			result = append(result, t)
		}
	}
	return result
}

// DiscoverTracks reads .agent/conductor/tracks.md from projectDir and parses track entries.
func DiscoverTracks(projectDir string) ([]TrackEntry, error) {
	path := filepath.Join(projectDir, ".agent", "conductor", "tracks.md")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open tracks.md: %w", err)
	}
	defer f.Close()
	return ParseTracks(f)
}

// TrackDetail contains the full detail of a track including artifact contents.
type TrackDetail struct {
	ID        string
	Title     string
	Status    string
	Type      string
	Spec      string
	Plan      string
	Phases    ProgressCount
	Tasks     ProgressCount
	CreatedAt string
	UpdatedAt string
}

// ProgressCount tracks total vs completed counts.
type ProgressCount struct {
	Total     int
	Completed int
}

// trackMetadata mirrors the JSON structure of metadata.json.
type trackMetadata struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Created string `json:"created"`
	Updated string `json:"updated"`
	Phases  struct {
		Total     int `json:"total"`
		Completed int `json:"completed"`
	} `json:"phases"`
	Tasks struct {
		Total     int `json:"total"`
		Completed int `json:"completed"`
	} `json:"tasks"`
}

// GetTrackDetail reads track artifacts from disk and returns a TrackDetail.
// conductorDir is the path to the .agent/conductor directory.
func GetTrackDetail(conductorDir, trackID string) (*TrackDetail, error) {
	trackDir := filepath.Join(conductorDir, "tracks", trackID)
	if _, err := os.Stat(trackDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("track %q not found", trackID)
	}

	detail := &TrackDetail{ID: trackID}

	// Read metadata.json if present.
	metaPath := filepath.Join(trackDir, "metadata.json")
	if data, err := os.ReadFile(metaPath); err == nil {
		var meta trackMetadata
		if jsonErr := json.Unmarshal(data, &meta); jsonErr == nil {
			detail.Title = meta.Title
			detail.Status = meta.Status
			detail.Type = meta.Type
			detail.CreatedAt = meta.Created
			detail.UpdatedAt = meta.Updated
			detail.Phases = ProgressCount{Total: meta.Phases.Total, Completed: meta.Phases.Completed}
			detail.Tasks = ProgressCount{Total: meta.Tasks.Total, Completed: meta.Tasks.Completed}
		}
	}

	// Read spec.md.
	if data, err := os.ReadFile(filepath.Join(trackDir, "spec.md")); err == nil {
		detail.Spec = string(data)
	}

	// Read plan.md.
	if data, err := os.ReadFile(filepath.Join(trackDir, "plan.md")); err == nil {
		detail.Plan = string(data)
	}

	return detail, nil
}

func parseTrackLine(line string) (TrackEntry, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") {
		return TrackEntry{}, false
	}

	parts := strings.Split(line, "|")
	if len(parts) < 5 {
		return TrackEntry{}, false
	}

	statusCell := strings.TrimSpace(parts[1])
	idCell := strings.TrimSpace(parts[2])
	titleCell := strings.TrimSpace(parts[3])

	var status string
	switch statusCell {
	case "[x]":
		status = StatusComplete
	case "[ ]":
		status = StatusPending
	case "[~]":
		status = StatusInProgress
	case "[!]":
		status = StatusApproved
	case "[r]":
		status = StatusInReview
	default:
		return TrackEntry{}, false
	}

	if idCell == "" || idCell == "Track ID" || idCell == "------" {
		return TrackEntry{}, false
	}

	return TrackEntry{
		ID:     idCell,
		Title:  titleCell,
		Status: status,
	}, true
}
