package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

const reaperInterval = 60 * time.Second

// TimeoutReaper periodically checks agent durations and force-stops agents
// exceeding the configured max duration.
type TimeoutReaper struct {
	store    port.AgentStore
	cfg      *config.Config
	eventBus port.EventBus
	logger   *log.Logger
}

// NewTimeoutReaper creates a new agent timeout reaper.
func NewTimeoutReaper(store port.AgentStore, cfg *config.Config, eventBus port.EventBus) *TimeoutReaper {
	return &TimeoutReaper{
		store:    store,
		cfg:      cfg,
		eventBus: eventBus,
		logger:   log.New(log.Writer(), "[timeout-reaper] ", log.LstdFlags),
	}
}

// Start runs the reaper in a background goroutine. It stops when ctx is cancelled.
func (r *TimeoutReaper) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(reaperInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.reap()
			}
		}
	}()
}

func (r *TimeoutReaper) reap() {
	maxDuration := r.cfg.GetAgentMaxDuration()
	if maxDuration == 0 {
		return
	}

	now := time.Now()
	for _, a := range r.store.Agents() {
		if !a.IsActive() {
			continue
		}
		if isExcludedFromTimeout(a.Role) {
			continue
		}
		elapsed := now.Sub(a.StartedAt)
		if elapsed <= maxDuration {
			continue
		}

		reason := fmt.Sprintf("exceeded max duration of %s (ran for %s)", maxDuration, elapsed.Truncate(time.Second))
		r.logger.Printf("Halting agent %s (%s): %s", a.ID, a.Role, reason)

		_ = r.store.HaltAgent(a.ID)
		_ = r.store.UpdateStatus(a.ID, string(domain.AgentStatusForceKilled))

		if agent, err := r.store.FindAgent(a.ID); err == nil && agent != nil {
			agent.ShutdownReason = reason
			now := time.Now()
			agent.FinishedAt = &now
		}
		_ = r.store.Save()

		if r.eventBus != nil {
			r.eventBus.Publish(domain.NewAgentUpdateEvent(map[string]any{
				"id":              a.ID,
				"status":          string(domain.AgentStatusForceKilled),
				"shutdown_reason": reason,
			}))
		}
	}
}

func isExcludedFromTimeout(role string) bool {
	return role == "interactive"
}
