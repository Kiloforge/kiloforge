package rest

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"crelay/internal/core/domain"
	"crelay/internal/core/service"
)

// boardSyncer handles webhook-driven board state synchronization.
type boardSyncer struct {
	svc       *service.BoardService
	store     service.BoardStore
	lifecycle *service.LifecycleService
	adminUser string
	logger    *log.Logger

	// commentFn posts a comment on a Gitea issue. Set by the server after init.
	commentFn func(ctx context.Context, slug string, issueNum int, body string)
	// prLoader loads PR tracking for a project slug.
	prLoader func(slug string) (*domain.PRTracking, error)
	// updateIssueFn closes a Gitea issue. Set by the server after init.
	updateIssueFn func(ctx context.Context, slug string, issueNum int, state string)
}

// isSelfTriggered checks if the webhook event was triggered by the admin user (our bot).
// Returns true if the event should be skipped to prevent loops.
func (b *boardSyncer) isSelfTriggered(payload map[string]any) bool {
	sender, _ := payload["sender"].(map[string]any)
	if sender == nil {
		return false
	}
	login, _ := sender["login"].(string)
	return login == b.adminUser
}

// handleLabelUpdated processes issue label changes to move cards to the matching column.
// It also detects backward/forward transitions and triggers agent lifecycle actions.
func (b *boardSyncer) handleLabelUpdated(ctx context.Context, slug string, issue map[string]any) {
	issueNum := int(issue["number"].(float64))

	// Extract labels from the issue.
	labels, _ := issue["labels"].([]any)
	var statusLabel string
	var hasRejected bool
	for _, l := range labels {
		lm, _ := l.(map[string]any)
		if lm == nil {
			continue
		}
		name, _ := lm["name"].(string)
		if name == "rejected" {
			hasRejected = true
		}
		if strings.HasPrefix(name, "status:") {
			statusLabel = strings.TrimPrefix(name, "status:")
		}
	}

	// Handle rejected label.
	if hasRejected {
		b.handleRejectedLabel(ctx, slug, issueNum)
		return
	}

	if statusLabel == "" {
		return
	}

	targetCol := service.StatusToColumn(statusLabel)
	targetKey := service.ColumnKeyFromName(targetCol)

	// Detect column transition direction.
	ti := b.findTrackIssueByNumber(slug, issueNum)
	if ti != nil && b.lifecycle != nil {
		fromCol := ti.Column
		var prTracking *domain.PRTracking
		if b.prLoader != nil {
			prTracking, _ = b.prLoader(slug)
			if prTracking != nil && prTracking.TrackID != ti.TrackID {
				prTracking = nil
			}
		}

		if service.IsBackwardMove(fromCol, targetKey) {
			b.lifecycle.HandleBackwardMove(ctx, ti.TrackID, fromCol, targetKey, prTracking)
			if b.commentFn != nil {
				b.commentFn(ctx, slug, issueNum, fmt.Sprintf("Agent halted — track moved back to %s", targetCol))
			}
			b.logger.Printf("[%s] Board: issue #%d backward %s → %s, agent halted", slug, issueNum, fromCol, targetKey)
		} else if service.IsForwardMove(fromCol, targetKey) {
			resumed, reason := b.lifecycle.HandleRepromotion(ctx, ti.TrackID, targetKey, prTracking)
			if resumed {
				if b.commentFn != nil {
					b.commentFn(ctx, slug, issueNum, "Developer agent resumed — implementation continuing")
				}
				b.logger.Printf("[%s] Board: issue #%d re-promoted to %s, agent resumed", slug, issueNum, targetKey)
			} else if reason != "" && reason != "no agent found for track" && reason != "no action for column "+targetKey {
				if b.commentFn != nil {
					b.commentFn(ctx, slug, issueNum, fmt.Sprintf("Could not resume agent: %s. Use `crelay implement` to restart.", reason))
				}
				b.logger.Printf("[%s] Board: issue #%d re-promoted to %s, resume failed: %s", slug, issueNum, targetKey, reason)
			}
		}
	}

	b.moveCardByIssue(ctx, slug, issueNum, targetCol)
}

// handleIssueClosed moves the card to the Completed column.
// If there's no merged PR for this track, it's treated as a rejection.
func (b *boardSyncer) handleIssueClosed(ctx context.Context, slug string, issue map[string]any) {
	issueNum := int(issue["number"].(float64))

	if b.lifecycle != nil {
		ti := b.findTrackIssueByNumber(slug, issueNum)
		if ti != nil {
			var prTracking *domain.PRTracking
			if b.prLoader != nil {
				prTracking, _ = b.prLoader(slug)
				if prTracking != nil && prTracking.TrackID != ti.TrackID {
					prTracking = nil
				}
			}
			if prTracking == nil || prTracking.Status != "merged" {
				b.lifecycle.HandleRejection(ctx, ti.TrackID, prTracking)
				if b.commentFn != nil {
					b.commentFn(ctx, slug, issueNum, "Track rejected — agent terminated, worktree returned to pool")
				}
				b.logger.Printf("[%s] Board: issue #%d closed without merge, agent terminated", slug, issueNum)
			}
		}
	}

	b.moveCardByIssue(ctx, slug, issueNum, "Completed")
}

// handleRejectedLabel terminates the agent and closes the issue when the rejected label is applied.
func (b *boardSyncer) handleRejectedLabel(ctx context.Context, slug string, issueNum int) {
	ti := b.findTrackIssueByNumber(slug, issueNum)
	if ti == nil {
		return
	}

	if b.lifecycle != nil {
		var prTracking *domain.PRTracking
		if b.prLoader != nil {
			prTracking, _ = b.prLoader(slug)
			if prTracking != nil && prTracking.TrackID != ti.TrackID {
				prTracking = nil
			}
		}
		b.lifecycle.HandleRejection(ctx, ti.TrackID, prTracking)
	}

	if b.updateIssueFn != nil {
		b.updateIssueFn(ctx, slug, issueNum, "closed")
	}

	if b.commentFn != nil {
		b.commentFn(ctx, slug, issueNum, "Track rejected — agent terminated, worktree returned to pool")
	}
	b.logger.Printf("[%s] Board: issue #%d rejected label applied, agent terminated", slug, issueNum)
}

// handleIssueAssigned moves the card to In Progress if currently in Suggested/Approved.
func (b *boardSyncer) handleIssueAssigned(ctx context.Context, slug string, issue map[string]any) {
	issueNum := int(issue["number"].(float64))

	ti := b.findTrackIssueByNumber(slug, issueNum)
	if ti == nil {
		return
	}

	// Only move if in early columns.
	if ti.Column != "suggested" && ti.Column != "approved" {
		return
	}

	b.moveCard(ctx, slug, ti, "In Progress")
}

// handlePROpened moves the track's card to In Review when a PR is opened.
func (b *boardSyncer) handlePROpened(ctx context.Context, slug, trackID string, prNumber int) {
	ti := b.findTrackIssue(slug, trackID)
	if ti == nil {
		return
	}

	b.moveCard(ctx, slug, ti, "In Review")
	b.logger.Printf("[%s] Board: issue #%d → In Review (PR #%d opened)", slug, ti.IssueNumber, prNumber)
}

// handlePRMerged moves the track's card to Completed and closes the issue.
func (b *boardSyncer) handlePRMerged(ctx context.Context, slug, trackID string, prNumber int) {
	ti := b.findTrackIssue(slug, trackID)
	if ti == nil {
		return
	}

	b.moveCard(ctx, slug, ti, "Completed")

	proj := domain.Project{Slug: slug, RepoName: slug}
	_ = b.svc.CloseTrackIssue(ctx, proj, ti.IssueNumber)
	b.logger.Printf("[%s] Board: issue #%d → Completed (PR #%d merged)", slug, ti.IssueNumber, prNumber)
}

// handleImplementStarted moves the track's card to In Progress.
func (b *boardSyncer) handleImplementStarted(ctx context.Context, slug, trackID string) {
	ti := b.findTrackIssue(slug, trackID)
	if ti == nil {
		return
	}

	b.moveCard(ctx, slug, ti, "In Progress")
	b.logger.Printf("[%s] Board: issue #%d → In Progress (implement started)", slug, ti.IssueNumber)
}

// moveCardByIssue finds a track issue by issue number and moves its card.
func (b *boardSyncer) moveCardByIssue(ctx context.Context, slug string, issueNum int, colName string) {
	ti := b.findTrackIssueByNumber(slug, issueNum)
	if ti == nil {
		return
	}
	b.moveCard(ctx, slug, ti, colName)
}

// moveCard moves a card to the named column and updates the mapping.
func (b *boardSyncer) moveCard(ctx context.Context, slug string, ti *domain.TrackIssue, colName string) {
	cfg, err := b.store.GetBoardConfig(slug)
	if err != nil || cfg == nil {
		return
	}

	targetKey := service.ColumnKeyFromName(colName)
	if ti.Column == targetKey {
		return // Already there.
	}

	colID, ok := cfg.Columns[targetKey]
	if !ok {
		return
	}

	if err := b.svc.MoveCard(ctx, slug, ti.CardID, colID); err != nil {
		b.logger.Printf("[%s] Board sync error moving card: %v", slug, err)
		return
	}

	ti.Column = targetKey
	ti.LastSynced = time.Now().Truncate(time.Second)
	_ = b.store.SaveTrackIssue(slug, *ti)
}

// findTrackIssue looks up a track issue by track ID.
func (b *boardSyncer) findTrackIssue(slug, trackID string) *domain.TrackIssue {
	ti, err := b.store.GetTrackIssue(slug, trackID)
	if err != nil || ti == nil {
		return nil
	}
	return ti
}

// findTrackIssueByNumber scans track issues to find one matching the issue number.
func (b *boardSyncer) findTrackIssueByNumber(slug string, issueNum int) *domain.TrackIssue {
	issues, err := b.store.ListTrackIssues(slug)
	if err != nil {
		return nil
	}
	for i := range issues {
		if issues[i].IssueNumber == issueNum {
			return &issues[i]
		}
	}
	return nil
}
