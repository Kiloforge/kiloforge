package rest

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/adapter/rest/gen"
	wsAdapter "kiloforge/internal/adapter/ws"
)

// ResolveConflict implements gen.StrictServerInterface.
// It spawns a conflict resolver agent to resolve diverged branches.
func (h *APIHandler) ResolveConflict(ctx context.Context, req gen.ResolveConflictRequestObject) (gen.ResolveConflictResponseObject, error) {
	if h.interSpawner == nil || h.wsSessions == nil {
		return gen.ResolveConflict500JSONResponse{Error: "interactive agents not configured"}, nil
	}

	p, ok := h.findProject(req.Slug)
	if !ok {
		return gen.ResolveConflict404JSONResponse{Error: fmt.Sprintf("project %q not found", req.Slug)}, nil
	}

	if req.Body == nil || req.Body.RemoteBranch == "" {
		return gen.ResolveConflict400JSONResponse{Error: "direction and remote_branch are required"}, nil
	}

	// Check Claude CLI authentication.
	if msg := h.checkClaudeAuth(ctx); msg != "" {
		return gen.ResolveConflict500JSONResponse{Error: msg}, nil
	}

	// Check agent permissions consent.
	if msg := h.checkConsent(); msg != "" {
		return gen.ResolveConflict500JSONResponse{Error: msg}, nil
	}

	// Validate required skills for the conflict-resolver role.
	if resp := h.checkSkillsForRole("conflict-resolver", p.ProjectDir); resp != nil {
		return gen.ResolveConflict412JSONResponse(*resp), nil
	}

	// Build prompt for the conflict resolver skill.
	direction := string(req.Body.Direction)
	prompt := fmt.Sprintf("/kf-conflict-resolver %s %s %s", direction, req.Body.RemoteBranch, p.Slug)

	opts := agent.SpawnInteractiveOpts{
		WorkDir: p.ProjectDir,
		Prompt:  prompt,
		Ref:     "conflict-resolver",
	}

	ia, err := h.interSpawner.SpawnInteractive(ctx, opts)
	if err != nil {
		if errors.Is(err, agent.ErrAtCapacity) {
			cap := h.interSpawner.Capacity()
			return gen.ResolveConflict429JSONResponse{
				Error: fmt.Sprintf("at capacity (%d/%d)", cap.Active, cap.Max),
			}, nil
		}
		if strings.Contains(err.Error(), "rate limited") {
			return gen.ResolveConflict429JSONResponse{Error: err.Error()}, nil
		}
		return gen.ResolveConflict500JSONResponse{Error: err.Error()}, nil
	}

	// Register WS bridge for the agent.
	bridge := wsAdapter.NewSDKBridge(ia.Info.ID, ia.Stdin, ia.Done)
	h.wsSessions.RegisterBridge(ia.Info.ID, bridge)

	relayCtx, cancelRelay := context.WithCancel(context.Background())
	ia.SetCancelRelay(cancelRelay)
	go h.wsSessions.StartStructuredRelay(relayCtx, ia.Info.ID, ia.Output)

	return gen.ResolveConflict201JSONResponse(domainAgentToGen(ia.Info, h.quota)), nil
}
