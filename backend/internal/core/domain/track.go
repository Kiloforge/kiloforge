package domain

// TrackEntry represents a parsed track from tracks.md.
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
