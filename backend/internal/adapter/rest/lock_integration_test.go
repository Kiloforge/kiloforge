//go:build integration

package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestIntegration_LockLifecycle(t *testing.T) {
	srv := startTestServer(t)

	// 1. Acquire a lock.
	acquireBody, _ := json.Marshal(map[string]any{
		"holder":      "test-worker",
		"ttl_seconds": 60,
	})
	resp, err := http.Post(srv.URL+"/api/locks/merge/acquire", "application/json", bytes.NewReader(acquireBody))
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("acquire: got %d, want 200", resp.StatusCode)
	}

	// 2. Heartbeat the lock.
	heartbeatBody, _ := json.Marshal(map[string]any{
		"holder":      "test-worker",
		"ttl_seconds": 120,
	})
	resp, err = http.Post(srv.URL+"/api/locks/merge/heartbeat", "application/json", bytes.NewReader(heartbeatBody))
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("heartbeat: got %d, want 200", resp.StatusCode)
	}

	// 3. List locks — should see our lock.
	resp, err = http.Get(srv.URL + "/api/locks")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var locks []map[string]any
	json.NewDecoder(resp.Body).Decode(&locks)
	resp.Body.Close()
	if len(locks) != 1 {
		t.Fatalf("list: got %d locks, want 1", len(locks))
	}
	if locks[0]["holder"] != "test-worker" {
		t.Errorf("holder: got %v, want 'test-worker'", locks[0]["holder"])
	}
	if locks[0]["scope"] != "merge" {
		t.Errorf("scope: got %v, want 'merge'", locks[0]["scope"])
	}

	// 4. Release the lock.
	releaseBody, _ := json.Marshal(map[string]any{
		"holder": "test-worker",
	})
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/locks/merge", bytes.NewReader(releaseBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("release: got %d, want 200", resp.StatusCode)
	}

	// 5. List locks — should be empty.
	resp, err = http.Get(srv.URL + "/api/locks")
	if err != nil {
		t.Fatalf("list after release: %v", err)
	}
	json.NewDecoder(resp.Body).Decode(&locks)
	resp.Body.Close()
	if len(locks) != 0 {
		t.Errorf("list after release: got %d locks, want 0", len(locks))
	}
}

func TestIntegration_LockConflict(t *testing.T) {
	srv := startTestServer(t)

	// Acquire a lock.
	body, _ := json.Marshal(map[string]any{
		"holder":          "worker-1",
		"ttl_seconds":     60,
		"timeout_seconds": 0,
	})
	resp, err := http.Post(srv.URL+"/api/locks/merge/acquire", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first acquire: got %d, want 200", resp.StatusCode)
	}

	// Try to acquire the same lock with a different holder — should conflict.
	body2, _ := json.Marshal(map[string]any{
		"holder":          "worker-2",
		"ttl_seconds":     60,
		"timeout_seconds": 0,
	})
	resp, err = http.Post(srv.URL+"/api/locks/merge/acquire", "application/json", bytes.NewReader(body2))
	if err != nil {
		t.Fatalf("second acquire: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("second acquire: got %d, want 409", resp.StatusCode)
	}

	var conflict map[string]any
	json.NewDecoder(resp.Body).Decode(&conflict)
	if conflict["current_holder"] != "worker-1" {
		t.Errorf("current_holder: got %v, want 'worker-1'", conflict["current_holder"])
	}
}
