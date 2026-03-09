package service

import "kiloforge/internal/core/port"

// Compile-time check that TrackReaderImpl satisfies port.TrackReader.
var _ port.TrackReader = (*TrackReaderImpl)(nil)

// TrackReaderImpl implements port.TrackReader using filesystem operations.
type TrackReaderImpl struct{}

// NewTrackReader creates a new TrackReaderImpl.
func NewTrackReader() *TrackReaderImpl {
	return &TrackReaderImpl{}
}

func (r *TrackReaderImpl) DiscoverTracks(projectDir string) ([]port.TrackEntry, error) {
	return DiscoverTracks(projectDir)
}

func (r *TrackReaderImpl) GetTrackDetail(conductorDir, trackID string) (*port.TrackDetail, error) {
	return GetTrackDetail(conductorDir, trackID)
}
