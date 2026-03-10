package port

import "kiloforge/internal/core/domain"

// TrackEntry represents a parsed track from the registry.
type TrackEntry struct {
	ID            string
	Title         string
	Status        string
	DepsCount     int
	DepsMet       int
	ConflictCount int
}

// TrackDependency represents a single dependency with resolved metadata.
type TrackDependency struct {
	ID     string
	Title  string
	Status string
}

// TrackConflict represents a conflict risk pair with resolved metadata.
type TrackConflict struct {
	TrackID    string
	TrackTitle string
	Risk       string
	Note       string
}

// TrackDetail contains the full detail of a track including artifact contents.
type TrackDetail struct {
	ID           string
	Title        string
	Status       string
	Type         string
	Spec         string
	Plan         string
	Phases       ProgressCount
	Tasks        ProgressCount
	CreatedAt    string
	UpdatedAt    string
	Dependencies []TrackDependency
	Conflicts    []TrackConflict
}

// ProgressCount tracks total vs completed counts.
type ProgressCount struct {
	Total     int
	Completed int
}

// TrackReader discovers and reads track information.
type TrackReader interface {
	DiscoverTracks(projectDir string) ([]TrackEntry, error)
	// DiscoverTracksPaginated returns a paginated list of tracks, optionally filtered by statuses.
	DiscoverTracksPaginated(projectDir string, opts domain.PageOpts, statuses ...string) (domain.Page[TrackEntry], error)
	GetTrackDetail(projectDir, trackID string) (*TrackDetail, error)
	RemoveTrack(projectDir, trackID string) error
	IsInitialized(projectDir string) bool
}
