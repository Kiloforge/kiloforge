//go:build integration

package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestIntegration_TrackClaimLifecycle(t *testing.T) {
	srv := startTestServer(t)

	trackID := "test-track_20260310120000Z"

	// 1. Claim the track.
	claimBody, _ := json.Marshal(map[string]any{
		"holder":      "worker-1",
		"ttl_seconds": 120,
	})
	resp, err := http.Post(srv.URL+"/api/tracks/"+trackID+"/claim", "application/json", bytes.NewReader(claimBody))
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	var claimInfo map[string]any
	json.NewDecoder(resp.Body).Decode(&claimInfo)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("claim: got %d, want 200", resp.StatusCode)
	}
	if claimInfo["track_id"] != trackID {
		t.Errorf("track_id: got %v, want %q", claimInfo["track_id"], trackID)
	}
	if claimInfo["holder"] != "worker-1" {
		t.Errorf("holder: got %v, want 'worker-1'", claimInfo["holder"])
	}

	// 2. Heartbeat the claim.
	hbBody, _ := json.Marshal(map[string]any{
		"holder":      "worker-1",
		"ttl_seconds": 120,
	})
	resp, err = http.Post(srv.URL+"/api/tracks/"+trackID+"/claim/heartbeat", "application/json", bytes.NewReader(hbBody))
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("heartbeat: got %d, want 200", resp.StatusCode)
	}

	// 3. Verify claim shows in locks list with track: prefix.
	resp, err = http.Get(srv.URL + "/api/locks")
	if err != nil {
		t.Fatalf("list locks: %v", err)
	}
	var locks []map[string]any
	json.NewDecoder(resp.Body).Decode(&locks)
	resp.Body.Close()
	found := false
	for _, l := range locks {
		if l["scope"] == "track:"+trackID {
			found = true
			if l["holder"] != "worker-1" {
				t.Errorf("lock holder: got %v, want 'worker-1'", l["holder"])
			}
		}
	}
	if !found {
		t.Error("expected to find track claim in locks list")
	}

	// 4. Release the claim.
	releaseBody, _ := json.Marshal(map[string]any{
		"holder": "worker-1",
	})
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/tracks/"+trackID+"/claim", bytes.NewReader(releaseBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("release: got %d, want 200", resp.StatusCode)
	}
}

func TestIntegration_TrackClaimConflict(t *testing.T) {
	srv := startTestServer(t)

	trackID := "conflict-track_20260310120000Z"

	// First claim succeeds.
	body, _ := json.Marshal(map[string]any{
		"holder":      "worker-1",
		"ttl_seconds": 60,
	})
	resp, err := http.Post(srv.URL+"/api/tracks/"+trackID+"/claim", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first claim: got %d, want 200", resp.StatusCode)
	}

	// Second claim from different holder should conflict.
	body2, _ := json.Marshal(map[string]any{
		"holder":      "worker-2",
		"ttl_seconds": 60,
	})
	resp, err = http.Post(srv.URL+"/api/tracks/"+trackID+"/claim", "application/json", bytes.NewReader(body2))
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("second claim: got %d, want 409", resp.StatusCode)
	}

	var conflict map[string]any
	json.NewDecoder(resp.Body).Decode(&conflict)
	if conflict["current_holder"] != "worker-1" {
		t.Errorf("current_holder: got %v, want 'worker-1'", conflict["current_holder"])
	}
}

func TestIntegration_TrackClaimRelease_NotOwned(t *testing.T) {
	srv := startTestServer(t)

	trackID := "owned-track_20260310120000Z"

	// Claim as worker-1.
	body, _ := json.Marshal(map[string]any{
		"holder":      "worker-1",
		"ttl_seconds": 60,
	})
	resp, err := http.Post(srv.URL+"/api/tracks/"+trackID+"/claim", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	resp.Body.Close()

	// Try to release as worker-2 — should fail.
	releaseBody, _ := json.Marshal(map[string]any{
		"holder": "worker-2",
	})
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/tracks/"+trackID+"/claim", bytes.NewReader(releaseBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("release by wrong holder: got %d, want 404", resp.StatusCode)
	}
}

func TestIntegration_TrackClaimHeartbeat_NoExistingClaim(t *testing.T) {
	srv := startTestServer(t)

	// Heartbeat without claim — should 404.
	body, _ := json.Marshal(map[string]any{
		"holder":      "worker-1",
		"ttl_seconds": 60,
	})
	resp, err := http.Post(srv.URL+"/api/tracks/unclaimed-track/claim/heartbeat", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("heartbeat without claim: got %d, want 404", resp.StatusCode)
	}
}

func TestIntegration_TrackClaimReentrant(t *testing.T) {
	srv := startTestServer(t)

	trackID := "reentrant-track_20260310120000Z"

	// Claim twice by same holder — should succeed (re-entrant).
	body, _ := json.Marshal(map[string]any{
		"holder":      "worker-1",
		"ttl_seconds": 60,
	})

	resp, err := http.Post(srv.URL+"/api/tracks/"+trackID+"/claim", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first claim: got %d, want 200", resp.StatusCode)
	}

	resp, err = http.Post(srv.URL+"/api/tracks/"+trackID+"/claim", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("second claim (re-entrant): %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("second claim (re-entrant): got %d, want 200", resp.StatusCode)
	}
}

func TestIntegration_TrackClaim_EmptyHolder(t *testing.T) {
	srv := startTestServer(t)

	// Empty holder should fail.
	body, _ := json.Marshal(map[string]any{
		"holder": "",
	})
	resp, err := http.Post(srv.URL+"/api/tracks/any-track/claim", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty holder: got %d, want 400", resp.StatusCode)
	}
}
