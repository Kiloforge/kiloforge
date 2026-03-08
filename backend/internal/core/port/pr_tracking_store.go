package port

import "kiloforge/internal/core/domain"

// PRTrackingStore persists PR tracking records per project.
type PRTrackingStore interface {
	LoadPRTracking(slug string) (*domain.PRTracking, error)
	SavePRTracking(slug string, t *domain.PRTracking) error
}
