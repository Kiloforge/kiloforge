//go:build e2e

package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestE2E_AddProject_HTTPS(t *testing.T) {
	srv := startE2EServer(t)

	body := map[string]string{"remote_url": "https://github.com/user/my-repo.git"}
	b, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+"/api/projects", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST /api/projects: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("expected 201, got %d: %v", resp.StatusCode, errBody)
	}

	var project map[string]any
	json.NewDecoder(resp.Body).Decode(&project)
	if project["slug"] != "my-repo" {
		t.Errorf("expected slug=my-repo, got %v", project["slug"])
	}
	if project["active"] != true {
		t.Errorf("expected active=true, got %v", project["active"])
	}

	// Verify project appears in list.
	listResp, err := http.Get(srv.URL + "/api/projects")
	if err != nil {
		t.Fatalf("GET /api/projects: %v", err)
	}
	defer listResp.Body.Close()

	var projects []map[string]any
	json.NewDecoder(listResp.Body).Decode(&projects)
	found := false
	for _, p := range projects {
		if p["slug"] == "my-repo" {
			found = true
			break
		}
	}
	if !found {
		t.Error("project 'my-repo' not found in project list")
	}
}

func TestE2E_AddProject_SSH(t *testing.T) {
	srv := startE2EServer(t)

	body := map[string]string{"remote_url": "git@github.com:user/ssh-repo.git"}
	b, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+"/api/projects", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST /api/projects: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("expected 201, got %d: %v", resp.StatusCode, errBody)
	}

	var project map[string]any
	json.NewDecoder(resp.Body).Decode(&project)
	if project["slug"] != "ssh-repo" {
		t.Errorf("expected slug=ssh-repo, got %v", project["slug"])
	}
}

func TestE2E_AddProject_EmptyURL(t *testing.T) {
	srv := startE2EServer(t)

	body := map[string]string{"remote_url": ""}
	b, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+"/api/projects", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST /api/projects: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestE2E_AddProject_Duplicate(t *testing.T) {
	srv := startE2EServer(t)

	body := map[string]string{"remote_url": "https://github.com/user/dup-repo.git"}
	b, _ := json.Marshal(body)

	// First add should succeed.
	resp, err := http.Post(srv.URL+"/api/projects", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("first POST: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first add: expected 201, got %d", resp.StatusCode)
	}

	// Second add with same URL should fail with 409.
	resp2, err := http.Post(srv.URL+"/api/projects", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("second POST: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate add: expected 409, got %d", resp2.StatusCode)
	}
}

func TestE2E_AddProject_WithName(t *testing.T) {
	srv := startE2EServer(t)

	body := map[string]string{
		"remote_url": "https://github.com/user/original-name.git",
		"name":       "custom-slug",
	}
	b, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+"/api/projects", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST /api/projects: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("expected 201, got %d: %v", resp.StatusCode, errBody)
	}

	var project map[string]any
	json.NewDecoder(resp.Body).Decode(&project)
	if project["slug"] != "custom-slug" {
		t.Errorf("expected slug=custom-slug, got %v", project["slug"])
	}
}

func TestE2E_RemoveProject(t *testing.T) {
	srv := startE2EServer(t)

	// Add a project first.
	body := map[string]string{"remote_url": "https://github.com/user/to-remove.git"}
	b, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+"/api/projects", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()

	// Remove it.
	req, _ := http.NewRequest("DELETE", srv.URL+"/api/projects/to-remove", nil)
	delResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer delResp.Body.Close()

	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", delResp.StatusCode)
	}

	// Verify project no longer in list.
	listResp, err := http.Get(srv.URL + "/api/projects")
	if err != nil {
		t.Fatalf("GET /api/projects: %v", err)
	}
	defer listResp.Body.Close()
	var projects []map[string]any
	json.NewDecoder(listResp.Body).Decode(&projects)
	for _, p := range projects {
		if p["slug"] == "to-remove" {
			t.Error("project 'to-remove' still in list after deletion")
		}
	}
}

func TestE2E_RemoveProject_WithCleanup(t *testing.T) {
	srv := startE2EServer(t)

	body := map[string]string{"remote_url": "https://github.com/user/cleanup-me.git"}
	b, _ := json.Marshal(body)
	resp, _ := http.Post(srv.URL+"/api/projects", "application/json", bytes.NewReader(b))
	resp.Body.Close()

	req, _ := http.NewRequest("DELETE", srv.URL+"/api/projects/cleanup-me?cleanup=true", nil)
	delResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE with cleanup: %v", err)
	}
	defer delResp.Body.Close()

	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", delResp.StatusCode)
	}
}

func TestE2E_RemoveProject_NotFound(t *testing.T) {
	srv := startE2EServer(t)

	req, _ := http.NewRequest("DELETE", srv.URL+"/api/projects/nonexistent", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestE2E_ListProjects_Empty(t *testing.T) {
	srv := startE2EServer(t)

	resp, err := http.Get(srv.URL + "/api/projects")
	if err != nil {
		t.Fatalf("GET /api/projects: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var projects []any
	json.NewDecoder(resp.Body).Decode(&projects)
	if len(projects) != 0 {
		t.Errorf("expected empty list, got %d projects", len(projects))
	}
}
