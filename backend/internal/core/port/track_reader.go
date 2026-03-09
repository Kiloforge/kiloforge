package port

import "kiloforge/internal/core/domain"

// Re-export domain types for backward compatibility.
type TrackEntry = domain.TrackEntry
type TrackDetail = domain.TrackDetail
type ProgressCount = domain.ProgressCount

// TrackReader discovers and reads track information from the filesystem.
type TrackReader interface {
	DiscoverTracks(projectDir string) ([]TrackEntry, error)
	GetTrackDetail(conductorDir, trackID string) (*TrackDetail, error)
}
