package service

import (
	"context"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

// PRService handles PR lifecycle operations.
type PRService struct {
	spawner port.AgentSpawner
	logger  port.Logger
}

// NewPRService creates a PRService with the given dependencies.
func NewPRService(spawner port.AgentSpawner, logger port.Logger) *PRService {
	return &PRService{spawner: spawner, logger: logger}
}

// CreateTracking builds a PRTracking record for a newly opened PR.
func (s *PRService) CreateTracking(prNumber int, branchRef, slug string, agents []domain.AgentInfo, maxCycles int) *domain.PRTracking {
	var devAgentID, devSession, devWorkDir string
	for _, a := range agents {
		if a.Role == "developer" && a.Ref == branchRef {
			devAgentID = a.ID
			devSession = a.SessionID
			devWorkDir = a.WorktreeDir
			break
		}
	}

	return &domain.PRTracking{
		PRNumber:         prNumber,
		TrackID:          branchRef,
		ProjectSlug:      slug,
		DeveloperAgentID: devAgentID,
		DeveloperSession: devSession,
		DeveloperWorkDir: devWorkDir,
		MaxReviewCycles:  maxCycles,
		Status:           "waiting-review",
	}
}

// HandleApproval processes an approved review: merges PR and returns cleanup opts.
func (s *PRService) HandleApproval(ctx context.Context, tracking *domain.PRTracking) error {
	tracking.Status = "approved"
	return nil
}

// HandleChangesRequested processes a changes-requested review.
// Returns true if the developer should be resumed, false if escalation is needed.
func (s *PRService) HandleChangesRequested(tracking *domain.PRTracking) (resumeDev bool) {
	tracking.ReviewCycleCount++
	if tracking.ReviewCycleCount >= tracking.MaxReviewCycles {
		return false
	}
	tracking.Status = "changes-requested"
	return true
}
