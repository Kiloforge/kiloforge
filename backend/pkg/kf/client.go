package kf

import (
	"fmt"
	"os"
	"path/filepath"
)

// Client provides a high-level API for interacting with Kiloforge data.
// All methods operate on the filesystem under the configured kf directory.
type Client struct {
	// KFDir is the path to the .agent/kf directory.
	KFDir string
}

// NewClient creates a new Kiloforge client for the given kf directory.
// Typically: NewClient("/path/to/project/.agent/kf")
func NewClient(kfDir string) *Client {
	return &Client{KFDir: kfDir}
}

// NewClientFromProject creates a client from a project root directory.
func NewClientFromProject(projectDir string) *Client {
	return &Client{KFDir: filepath.Join(projectDir, ".agent", "kf")}
}

func (c *Client) tracksFile() string    { return filepath.Join(c.KFDir, "tracks.yaml") }
func (c *Client) depsFile() string      { return filepath.Join(c.KFDir, "tracks", "deps.yaml") }
func (c *Client) conflictsFile() string { return filepath.Join(c.KFDir, "tracks", "conflicts.yaml") }
func (c *Client) tracksDir() string     { return filepath.Join(c.KFDir, "tracks") }

// --- Registry operations ---

// ListTracks reads all tracks from the registry.
func (c *Client) ListTracks() ([]TrackEntry, error) {
	return ReadRegistryFile(c.tracksFile())
}

// ListActiveTracks returns only pending and in-progress tracks.
func (c *Client) ListActiveTracks() ([]TrackEntry, error) {
	entries, err := c.ListTracks()
	if err != nil {
		return nil, err
	}
	return ActiveEntries(entries), nil
}

// ListReadyTracks returns active tracks with all dependencies satisfied.
func (c *Client) ListReadyTracks() ([]TrackEntry, error) {
	entries, err := c.ListActiveTracks()
	if err != nil {
		return nil, err
	}
	graph, err := c.GetDepsGraph()
	if err != nil {
		return nil, err
	}

	// Build completed set
	allEntries, _ := c.ListTracks()
	completed := make(map[string]bool)
	for _, e := range allEntries {
		if e.Status == StatusCompleted {
			completed[e.ID] = true
		}
	}

	var ready []TrackEntry
	for _, e := range entries {
		if graph.AllDepsSatisfied(e.ID, completed) {
			ready = append(ready, e)
		}
	}
	return ready, nil
}

// GetTrackEntry returns a single track entry from the registry.
func (c *Client) GetTrackEntry(trackID string) (*TrackEntry, error) {
	entries, err := c.ListTracks()
	if err != nil {
		return nil, err
	}
	entry := FindEntry(entries, trackID)
	if entry == nil {
		return nil, fmt.Errorf("track %q not found in registry", trackID)
	}
	return entry, nil
}

// AddTrack adds a new track to the registry and optionally registers dependencies.
func (c *Client) AddTrack(entry TrackEntry, deps []string) error {
	entries, err := c.ListTracks()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if FindEntry(entries, entry.ID) != nil {
		return fmt.Errorf("track %q already exists", entry.ID)
	}
	entries = append(entries, entry)
	if err := WriteRegistryFile(c.tracksFile(), entries); err != nil {
		return err
	}

	// Register deps
	if len(deps) > 0 || true { // Always register in deps.yaml even if no deps
		graph, err := c.GetDepsGraph()
		if err != nil {
			return err
		}
		graph[entry.ID] = deps
		return WriteDepsFile(c.depsFile(), graph)
	}
	return nil
}

// UpdateStatus changes a track's status and handles side effects
// (pruning deps/conflicts on completion or archival).
func (c *Client) UpdateStatus(trackID, status string) error {
	entries, err := c.ListTracks()
	if err != nil {
		return err
	}
	entry := FindEntry(entries, trackID)
	if entry == nil {
		return fmt.Errorf("track %q not found", trackID)
	}
	entry.Status = status
	entry.Updated = TodayISO()

	if status == StatusArchived {
		entry.ArchivedAt = TodayISO()
	}

	if err := WriteRegistryFile(c.tracksFile(), entries); err != nil {
		return err
	}

	// Prune deps and conflicts on completion/archival
	if status == StatusCompleted || status == StatusArchived {
		if graph, err := c.GetDepsGraph(); err == nil {
			graph.RemoveTrack(trackID)
			_ = WriteDepsFile(c.depsFile(), graph)
		}
		if pairs, err := c.GetConflicts(); err == nil {
			pairs = RemoveConflictsForTrack(pairs, trackID)
			_ = WriteConflictsFile(c.conflictsFile(), pairs)
		}
	}
	return nil
}

// ArchiveTrack archives a track with a reason.
func (c *Client) ArchiveTrack(trackID, reason string) error {
	entries, err := c.ListTracks()
	if err != nil {
		return err
	}
	entry := FindEntry(entries, trackID)
	if entry == nil {
		return fmt.Errorf("track %q not found", trackID)
	}
	entry.Status = StatusArchived
	entry.Updated = TodayISO()
	entry.ArchivedAt = TodayISO()
	entry.ArchiveReason = reason

	if err := WriteRegistryFile(c.tracksFile(), entries); err != nil {
		return err
	}

	// Prune deps and conflicts
	if graph, err := c.GetDepsGraph(); err == nil {
		graph.RemoveTrack(trackID)
		_ = WriteDepsFile(c.depsFile(), graph)
	}
	if pairs, err := c.GetConflicts(); err == nil {
		pairs = RemoveConflictsForTrack(pairs, trackID)
		_ = WriteConflictsFile(c.conflictsFile(), pairs)
	}
	return nil
}

// --- Deps operations ---

// GetDepsGraph reads the full dependency graph.
func (c *Client) GetDepsGraph() (DepsGraph, error) {
	return ReadDepsFile(c.depsFile())
}

// CheckDeps checks if all dependencies of a track are satisfied.
func (c *Client) CheckDeps(trackID string) (satisfied bool, unmet []string, err error) {
	graph, err := c.GetDepsGraph()
	if err != nil {
		return false, nil, err
	}
	entries, err := c.ListTracks()
	if err != nil {
		return false, nil, err
	}
	completed := make(map[string]bool)
	for _, e := range entries {
		if e.Status == StatusCompleted {
			completed[e.ID] = true
		}
	}
	deps := graph.GetDeps(trackID)
	for _, d := range deps {
		if !completed[d] {
			unmet = append(unmet, d)
		}
	}
	return len(unmet) == 0, unmet, nil
}

// --- Conflicts operations ---

// GetConflicts reads all conflict pairs.
func (c *Client) GetConflicts() ([]ConflictPair, error) {
	return ReadConflictsFile(c.conflictsFile())
}

// GetConflictsForTrack returns conflict pairs involving the given track.
func (c *Client) GetConflictsForTrack(trackID string) ([]ConflictPair, error) {
	pairs, err := c.GetConflicts()
	if err != nil {
		return nil, err
	}
	return FindConflicts(pairs, trackID), nil
}

// AddConflict adds or updates a conflict pair.
func (c *Client) AddConflict(idA, idB, risk, note string) error {
	pairs, err := c.GetConflicts()
	if err != nil {
		return err
	}
	pair := NewConflictPair(idA, idB, risk, note)
	pairs = AddOrUpdateConflict(pairs, pair)
	return WriteConflictsFile(c.conflictsFile(), pairs)
}

// RemoveConflict removes a conflict pair.
func (c *Client) RemoveConflict(idA, idB string) error {
	pairs, err := c.GetConflicts()
	if err != nil {
		return err
	}
	key := NewConflictPair(idA, idB, "", "").PairKey()
	var filtered []ConflictPair
	for _, p := range pairs {
		if p.PairKey() != key {
			filtered = append(filtered, p)
		}
	}
	return WriteConflictsFile(c.conflictsFile(), filtered)
}

// --- Track content operations ---

// GetTrack reads the full track.yaml for a track.
func (c *Client) GetTrack(trackID string) (*Track, error) {
	return ReadTrackByID(c.tracksDir(), trackID)
}

// SaveTrack writes a track's track.yaml.
func (c *Client) SaveTrack(t *Track) error {
	return WriteTrackByID(c.tracksDir(), t)
}

// GetTrackProgress returns completion statistics for a track.
func (c *Client) GetTrackProgress(trackID string) (*ProgressStats, error) {
	t, err := c.GetTrack(trackID)
	if err != nil {
		return nil, err
	}
	stats := t.Progress()
	return &stats, nil
}
