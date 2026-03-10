package rest

import (
	"context"
	"strings"
	"time"

	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/rest/gen"
)

const trackClaimScopePrefix = "track:"

// ClaimTrack implements gen.StrictServerInterface.
func (h *APIHandler) ClaimTrack(ctx context.Context, req gen.ClaimTrackRequestObject) (gen.ClaimTrackResponseObject, error) {
	if req.Body == nil || req.Body.Holder == "" {
		return gen.ClaimTrack400JSONResponse{Error: "holder required"}, nil
	}

	ttlSec := 120
	if req.Body.TtlSeconds != nil && *req.Body.TtlSeconds > 0 {
		ttlSec = *req.Body.TtlSeconds
	}
	ttl := time.Duration(ttlSec) * time.Second

	scope := trackClaimScopePrefix + req.TrackId

	// Non-blocking acquire — track claims should not wait.
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	cancel()

	l, err := h.lockMgr.Acquire(ctx, scope, req.Body.Holder, ttl)
	if err != nil {
		var currentHolder string
		for _, existing := range h.lockMgr.List() {
			if existing.Scope == scope {
				currentHolder = existing.Holder
				break
			}
		}
		return gen.ClaimTrack409JSONResponse{
			Error:         "track already claimed",
			CurrentHolder: strPtr(currentHolder),
		}, nil
	}

	return gen.ClaimTrack200JSONResponse(claimToGen(req.TrackId, l)), nil
}

// ReleaseTrackClaim implements gen.StrictServerInterface.
func (h *APIHandler) ReleaseTrackClaim(_ context.Context, req gen.ReleaseTrackClaimRequestObject) (gen.ReleaseTrackClaimResponseObject, error) {
	if req.Body == nil || req.Body.Holder == "" {
		return gen.ReleaseTrackClaim400JSONResponse{Error: "holder required"}, nil
	}

	scope := trackClaimScopePrefix + req.TrackId
	if err := h.lockMgr.Release(scope, req.Body.Holder); err != nil {
		return gen.ReleaseTrackClaim404JSONResponse{Error: err.Error()}, nil
	}

	return gen.ReleaseTrackClaim200JSONResponse{Released: true}, nil
}

// HeartbeatTrackClaim implements gen.StrictServerInterface.
func (h *APIHandler) HeartbeatTrackClaim(_ context.Context, req gen.HeartbeatTrackClaimRequestObject) (gen.HeartbeatTrackClaimResponseObject, error) {
	if req.Body == nil || req.Body.Holder == "" {
		return gen.HeartbeatTrackClaim400JSONResponse{Error: "holder required"}, nil
	}

	ttlSec := 120
	if req.Body.TtlSeconds != nil && *req.Body.TtlSeconds > 0 {
		ttlSec = *req.Body.TtlSeconds
	}

	scope := trackClaimScopePrefix + req.TrackId
	l, err := h.lockMgr.Heartbeat(scope, req.Body.Holder, time.Duration(ttlSec)*time.Second)
	if err != nil {
		return gen.HeartbeatTrackClaim404JSONResponse{Error: err.Error()}, nil
	}

	return gen.HeartbeatTrackClaim200JSONResponse(claimToGen(req.TrackId, l)), nil
}

// activeTrackClaims returns a map of track ID → lock for all active track claims.
func (h *APIHandler) activeTrackClaims() map[string]lock.Lock {
	claims := make(map[string]lock.Lock)
	for _, l := range h.lockMgr.List() {
		if strings.HasPrefix(l.Scope, trackClaimScopePrefix) {
			trackID := strings.TrimPrefix(l.Scope, trackClaimScopePrefix)
			claims[trackID] = l
		}
	}
	return claims
}

func claimToGen(trackID string, l *lock.Lock) gen.TrackClaimInfo {
	remaining := time.Until(l.ExpiresAt).Seconds()
	if remaining < 0 {
		remaining = 0
	}
	return gen.TrackClaimInfo{
		TrackId:             trackID,
		Holder:              l.Holder,
		AcquiredAt:          l.AcquiredAt,
		ExpiresAt:           l.ExpiresAt,
		TtlRemainingSeconds: remaining,
	}
}
