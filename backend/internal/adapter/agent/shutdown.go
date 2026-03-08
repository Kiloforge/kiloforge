package agent

import (
	"os"
	"syscall"
	"time"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

// ShutdownResult summarizes the outcome of shutting down agents.
type ShutdownResult struct {
	Suspended   []string
	ForceKilled []string
	AlreadyDead []string
	NoPID       []string
}

// ShutdownManager handles graceful shutdown of running agents.
type ShutdownManager struct {
	store port.AgentStore
}

// NewShutdownManager creates a ShutdownManager.
func NewShutdownManager(store port.AgentStore) *ShutdownManager {
	return &ShutdownManager{store: store}
}

// ShutdownAll sends SIGINT to all running/waiting agents, waits up to timeout,
// then force-kills any remaining.
func (sm *ShutdownManager) ShutdownAll(timeout time.Duration) ShutdownResult {
	var result ShutdownResult
	agents := sm.store.AgentsByStatus("running", "waiting")
	if len(agents) == 0 {
		return result
	}

	// Phase 1: Send SIGINT to all, track which are alive.
	var alive []domain.AgentInfo
	for _, a := range agents {
		if a.PID <= 0 {
			result.NoPID = append(result.NoPID, a.ID)
			sm.store.UpdateStatus(a.ID, string(domain.AgentStatusSuspended))
			continue
		}
		if !ProcessAlive(a.PID) {
			result.AlreadyDead = append(result.AlreadyDead, a.ID)
			sm.store.UpdateStatus(a.ID, string(domain.AgentStatusSuspended))
			continue
		}
		signalProcess(a.PID, syscall.SIGINT)
		sm.store.UpdateStatus(a.ID, string(domain.AgentStatusSuspending))
		alive = append(alive, a)
	}

	// Phase 2: Wait for graceful exit.
	deadline := time.After(timeout)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	waitLoop:
	for len(alive) > 0 {
		select {
		case <-deadline:
			break waitLoop
		case <-ticker.C:
			var stillAlive []domain.AgentInfo
			for _, a := range alive {
				if ProcessAlive(a.PID) {
					stillAlive = append(stillAlive, a)
				} else {
					now := time.Now()
					ag, err := sm.store.FindAgent(a.ID)
					if err == nil {
						ag.SuspendedAt = &now
						ag.ShutdownReason = "relay shutdown"
					}
					sm.store.UpdateStatus(a.ID, string(domain.AgentStatusSuspended))
					result.Suspended = append(result.Suspended, a.ID)
				}
			}
			alive = stillAlive
		}
	}

	// Phase 3: Force-kill stragglers.
	for _, a := range alive {
		killProcess(a.PID)
		sm.store.UpdateStatus(a.ID, string(domain.AgentStatusForceKilled))
		result.ForceKilled = append(result.ForceKilled, a.ID)
	}

	_ = sm.store.Save()
	return result
}

// ProcessAlive checks if a process with the given PID exists.
func ProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

func signalProcess(pid int, sig syscall.Signal) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = proc.Signal(sig)
}

func killProcess(pid int) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = proc.Signal(syscall.SIGKILL)
}
