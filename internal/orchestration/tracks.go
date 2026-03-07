package orchestration

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	StatusComplete   = "complete"
	StatusPending    = "pending"
	StatusInProgress = "in-progress"
)

// TrackEntry represents a parsed track from tracks.md.
type TrackEntry struct {
	ID     string
	Title  string
	Status string
}

// DiscoverTracks reads .agent/conductor/tracks.md from projectDir and parses track entries.
func DiscoverTracks(projectDir string) ([]TrackEntry, error) {
	path := filepath.Join(projectDir, ".agent", "conductor", "tracks.md")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open tracks.md: %w", err)
	}
	defer f.Close()

	var tracks []TrackEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		entry, ok := parseTrackLine(line)
		if ok {
			tracks = append(tracks, entry)
		}
	}
	return tracks, scanner.Err()
}

// parseTrackLine extracts a track entry from a markdown table row.
// Expected format: | [x] | track-id | Title | date | date |
func parseTrackLine(line string) (TrackEntry, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") {
		return TrackEntry{}, false
	}

	parts := strings.Split(line, "|")
	// Need at least: empty, status, id, title, ...
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
