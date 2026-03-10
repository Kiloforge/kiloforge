package rest

import (
	"context"
	"testing"

	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/rest/gen"
)

func newTestClaimHandler(t *testing.T) *APIHandler {
	t.Helper()
	dir := t.TempDir()
	lockMgr := lock.New(dir)
	lockMgr.StartReaper(t.Context())
	return &APIHandler{
		lockMgr: lockMgr,
	}
}

func TestClaimTrack(t *testing.T) {
	t.Parallel()
	h := newTestClaimHandler(t)

	resp, err := h.ClaimTrack(context.Background(), gen.ClaimTrackRequestObject{
		TrackId: "test-track_20260310Z",
		Body: &gen.ClaimTrackJSONRequestBody{
			Holder:     "worker-1",
			TtlSeconds: intPtr(120),
		},
	})
	if err != nil {
		t.Fatalf("ClaimTrack: %v", err)
	}

	ok, isOK := resp.(gen.ClaimTrack200JSONResponse)
	if !isOK {
		t.Fatalf("expected 200, got %T", resp)
	}
	if ok.TrackId != "test-track_20260310Z" {
		t.Errorf("track_id: got %q, want %q", ok.TrackId, "test-track_20260310Z")
	}
	if ok.Holder != "worker-1" {
		t.Errorf("holder: got %q, want %q", ok.Holder, "worker-1")
	}
}

func TestClaimTrack_EmptyHolder(t *testing.T) {
	t.Parallel()
	h := newTestClaimHandler(t)

	resp, err := h.ClaimTrack(context.Background(), gen.ClaimTrackRequestObject{
		TrackId: "any-track",
		Body: &gen.ClaimTrackJSONRequestBody{
			Holder: "",
		},
	})
	if err != nil {
		t.Fatalf("ClaimTrack: %v", err)
	}
	if _, is400 := resp.(gen.ClaimTrack400JSONResponse); !is400 {
		t.Fatalf("expected 400, got %T", resp)
	}
}

func TestClaimTrack_NilBody(t *testing.T) {
	t.Parallel()
	h := newTestClaimHandler(t)

	resp, err := h.ClaimTrack(context.Background(), gen.ClaimTrackRequestObject{
		TrackId: "any-track",
		Body:    nil,
	})
	if err != nil {
		t.Fatalf("ClaimTrack: %v", err)
	}
	if _, is400 := resp.(gen.ClaimTrack400JSONResponse); !is400 {
		t.Fatalf("expected 400, got %T", resp)
	}
}

func TestClaimTrack_Conflict(t *testing.T) {
	t.Parallel()
	h := newTestClaimHandler(t)

	// First claim.
	_, err := h.ClaimTrack(context.Background(), gen.ClaimTrackRequestObject{
		TrackId: "conflict-track",
		Body: &gen.ClaimTrackJSONRequestBody{
			Holder:     "worker-1",
			TtlSeconds: intPtr(60),
		},
	})
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}

	// Second claim by different holder.
	resp, err := h.ClaimTrack(context.Background(), gen.ClaimTrackRequestObject{
		TrackId: "conflict-track",
		Body: &gen.ClaimTrackJSONRequestBody{
			Holder:     "worker-2",
			TtlSeconds: intPtr(60),
		},
	})
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	conflict, is409 := resp.(gen.ClaimTrack409JSONResponse)
	if !is409 {
		t.Fatalf("expected 409, got %T", resp)
	}
	if conflict.CurrentHolder == nil || *conflict.CurrentHolder != "worker-1" {
		t.Errorf("current_holder: got %v, want 'worker-1'", conflict.CurrentHolder)
	}
}

func TestClaimTrack_Reentrant(t *testing.T) {
	t.Parallel()
	h := newTestClaimHandler(t)

	req := gen.ClaimTrackRequestObject{
		TrackId: "reentrant-track",
		Body: &gen.ClaimTrackJSONRequestBody{
			Holder:     "worker-1",
			TtlSeconds: intPtr(60),
		},
	}

	// Claim twice — should succeed both times (re-entrant).
	resp1, err := h.ClaimTrack(context.Background(), req)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if _, isOK := resp1.(gen.ClaimTrack200JSONResponse); !isOK {
		t.Fatalf("first claim: expected 200, got %T", resp1)
	}

	resp2, err := h.ClaimTrack(context.Background(), req)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if _, isOK := resp2.(gen.ClaimTrack200JSONResponse); !isOK {
		t.Fatalf("second claim: expected 200, got %T", resp2)
	}
}

func TestReleaseTrackClaim(t *testing.T) {
	t.Parallel()
	h := newTestClaimHandler(t)

	// Claim first.
	_, _ = h.ClaimTrack(context.Background(), gen.ClaimTrackRequestObject{
		TrackId: "release-track",
		Body: &gen.ClaimTrackJSONRequestBody{
			Holder:     "worker-1",
			TtlSeconds: intPtr(60),
		},
	})

	// Release.
	resp, err := h.ReleaseTrackClaim(context.Background(), gen.ReleaseTrackClaimRequestObject{
		TrackId: "release-track",
		Body:    &gen.ReleaseTrackClaimJSONRequestBody{Holder: "worker-1"},
	})
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	released, isOK := resp.(gen.ReleaseTrackClaim200JSONResponse)
	if !isOK {
		t.Fatalf("expected 200, got %T", resp)
	}
	if !released.Released {
		t.Error("expected released=true")
	}
}

func TestReleaseTrackClaim_NotOwned(t *testing.T) {
	t.Parallel()
	h := newTestClaimHandler(t)

	// Claim as worker-1.
	_, _ = h.ClaimTrack(context.Background(), gen.ClaimTrackRequestObject{
		TrackId: "owned-track",
		Body: &gen.ClaimTrackJSONRequestBody{
			Holder:     "worker-1",
			TtlSeconds: intPtr(60),
		},
	})

	// Release as worker-2 — should 404.
	resp, err := h.ReleaseTrackClaim(context.Background(), gen.ReleaseTrackClaimRequestObject{
		TrackId: "owned-track",
		Body:    &gen.ReleaseTrackClaimJSONRequestBody{Holder: "worker-2"},
	})
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	if _, is404 := resp.(gen.ReleaseTrackClaim404JSONResponse); !is404 {
		t.Fatalf("expected 404, got %T", resp)
	}
}

func TestReleaseTrackClaim_EmptyHolder(t *testing.T) {
	t.Parallel()
	h := newTestClaimHandler(t)

	resp, err := h.ReleaseTrackClaim(context.Background(), gen.ReleaseTrackClaimRequestObject{
		TrackId: "any-track",
		Body:    &gen.ReleaseTrackClaimJSONRequestBody{Holder: ""},
	})
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	if _, is400 := resp.(gen.ReleaseTrackClaim400JSONResponse); !is400 {
		t.Fatalf("expected 400, got %T", resp)
	}
}

func TestHeartbeatTrackClaim(t *testing.T) {
	t.Parallel()
	h := newTestClaimHandler(t)

	// Claim first.
	_, _ = h.ClaimTrack(context.Background(), gen.ClaimTrackRequestObject{
		TrackId: "hb-track",
		Body: &gen.ClaimTrackJSONRequestBody{
			Holder:     "worker-1",
			TtlSeconds: intPtr(60),
		},
	})

	// Heartbeat.
	resp, err := h.HeartbeatTrackClaim(context.Background(), gen.HeartbeatTrackClaimRequestObject{
		TrackId: "hb-track",
		Body: &gen.TrackClaimHeartbeatRequest{
			Holder:     "worker-1",
			TtlSeconds: intPtr(120),
		},
	})
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	info, isOK := resp.(gen.HeartbeatTrackClaim200JSONResponse)
	if !isOK {
		t.Fatalf("expected 200, got %T", resp)
	}
	if info.Holder != "worker-1" {
		t.Errorf("holder: got %q, want 'worker-1'", info.Holder)
	}
}

func TestHeartbeatTrackClaim_NoClaim(t *testing.T) {
	t.Parallel()
	h := newTestClaimHandler(t)

	resp, err := h.HeartbeatTrackClaim(context.Background(), gen.HeartbeatTrackClaimRequestObject{
		TrackId: "unclaimed-track",
		Body: &gen.TrackClaimHeartbeatRequest{
			Holder:     "worker-1",
			TtlSeconds: intPtr(60),
		},
	})
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if _, is404 := resp.(gen.HeartbeatTrackClaim404JSONResponse); !is404 {
		t.Fatalf("expected 404, got %T", resp)
	}
}

func TestHeartbeatTrackClaim_EmptyHolder(t *testing.T) {
	t.Parallel()
	h := newTestClaimHandler(t)

	resp, err := h.HeartbeatTrackClaim(context.Background(), gen.HeartbeatTrackClaimRequestObject{
		TrackId: "any-track",
		Body: &gen.TrackClaimHeartbeatRequest{
			Holder: "",
		},
	})
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if _, is400 := resp.(gen.HeartbeatTrackClaim400JSONResponse); !is400 {
		t.Fatalf("expected 400, got %T", resp)
	}
}

func TestActiveTrackClaims(t *testing.T) {
	t.Parallel()
	h := newTestClaimHandler(t)

	// Claim two tracks.
	_, _ = h.ClaimTrack(context.Background(), gen.ClaimTrackRequestObject{
		TrackId: "track-a",
		Body:    &gen.ClaimTrackJSONRequestBody{Holder: "worker-1", TtlSeconds: intPtr(60)},
	})
	_, _ = h.ClaimTrack(context.Background(), gen.ClaimTrackRequestObject{
		TrackId: "track-b",
		Body:    &gen.ClaimTrackJSONRequestBody{Holder: "worker-2", TtlSeconds: intPtr(60)},
	})

	claims := h.activeTrackClaims()
	if len(claims) != 2 {
		t.Fatalf("expected 2 claims, got %d", len(claims))
	}
	if claims["track-a"].Holder != "worker-1" {
		t.Errorf("track-a holder: got %q, want 'worker-1'", claims["track-a"].Holder)
	}
	if claims["track-b"].Holder != "worker-2" {
		t.Errorf("track-b holder: got %q, want 'worker-2'", claims["track-b"].Holder)
	}
}
