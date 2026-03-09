package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"kiloforge/internal/core/port"
)

const (
	StatusComplete   = "complete"
	StatusPending    = "pending"
	StatusInProgress = "in-progress"
	StatusApproved   = "approved"
	StatusInReview   = "in-review"
)

// ParseTracks parses track entries from a tracks.md reader.
// This is a pure function — I/O stays in the caller.
func ParseTracks(r io.Reader) ([]port.TrackEntry, error) {
	var tracks []port.TrackEntry
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
func FilterByStatus(tracks []port.TrackEntry, status string) []port.TrackEntry {
	var result []port.TrackEntry
	for _, t := range tracks {
		if t.Status == status {
			result = append(result, t)
		}
	}
	return result
}

// DiscoverTracks reads .agent/conductor/tracks.md from projectDir and parses track entries.
func DiscoverTracks(projectDir string) ([]port.TrackEntry, error) {
	path := filepath.Join(projectDir, ".agent", "conductor", "tracks.md")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open tracks.md: %w", err)
	}
	defer f.Close()
	return ParseTracks(f)
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
func GetTrackDetail(conductorDir, trackID string) (*port.TrackDetail, error) {
	trackDir := filepath.Join(conductorDir, "tracks", trackID)
	if _, err := os.Stat(trackDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("track %q not found", trackID)
	}

	detail := &port.TrackDetail{ID: trackID}

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
			detail.Phases = port.ProgressCount{Total: meta.Phases.Total, Completed: meta.Phases.Completed}
			detail.Tasks = port.ProgressCount{Total: meta.Tasks.Total, Completed: meta.Tasks.Completed}
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

func parseTrackLine(line string) (port.TrackEntry, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") {
		return port.TrackEntry{}, false
	}

	parts := strings.Split(line, "|")
	if len(parts) < 5 {
		return port.TrackEntry{}, false
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
		return port.TrackEntry{}, false
	}

	if idCell == "" || idCell == "Track ID" || idCell == "------" {
		return port.TrackEntry{}, false
	}

	return port.TrackEntry{
		ID:     idCell,
		Title:  titleCell,
		Status: status,
	}, true
}
