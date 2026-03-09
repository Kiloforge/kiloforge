package port

// TrackEntry represents a parsed track from the registry.
type TrackEntry struct {
	ID     string
	Title  string
	Status string
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

// TrackReader discovers and reads track information.
type TrackReader interface {
	DiscoverTracks(projectDir string) ([]TrackEntry, error)
	GetTrackDetail(projectDir, trackID string) (*TrackDetail, error)
	RemoveTrack(projectDir, trackID string) error
	IsInitialized(projectDir string) bool
}
