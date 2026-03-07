package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateProject_Success(t *testing.T) {
	t.Parallel()

	var gotPayload map[string]any
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotPayload)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 10, "title": "Kanban"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "conductor", "pass")
	id, err := client.CreateProject(context.Background(), "myapp", "Kanban", "Track board")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if id != 10 {
		t.Errorf("project id: want 10, got %d", id)
	}
	if gotPath != "/api/v1/repos/conductor/myapp/projects" {
		t.Errorf("path: got %s", gotPath)
	}
	if gotPayload["title"] != "Kanban" {
		t.Errorf("title: want %q, got %v", "Kanban", gotPayload["title"])
	}
}

func TestCreateProject_Error(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "error"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "conductor", "pass")
	_, err := client.CreateProject(context.Background(), "myapp", "Test", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetProjects_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":1,"title":"Board 1"},{"id":2,"title":"Board 2"}]`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "conductor", "pass")
	projects, err := client.GetProjects(context.Background(), "myapp")
	if err != nil {
		t.Fatalf("GetProjects: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
	if projects[0].Title != "Board 1" {
		t.Errorf("first project title: want %q, got %q", "Board 1", projects[0].Title)
	}
}

func TestGetProjects_Empty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "conductor", "pass")
	projects, err := client.GetProjects(context.Background(), "myapp")
	if err != nil {
		t.Fatalf("GetProjects: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestCreateColumn_Success(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 20, "title": "To Do"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "conductor", "pass")
	id, err := client.CreateColumn(context.Background(), 10, "To Do")
	if err != nil {
		t.Fatalf("CreateColumn: %v", err)
	}
	if id != 20 {
		t.Errorf("column id: want 20, got %d", id)
	}
	if gotPath != "/api/v1/projects/10/columns" {
		t.Errorf("path: got %s", gotPath)
	}
}

func TestGetColumns_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":20,"title":"To Do"},{"id":21,"title":"In Progress"},{"id":22,"title":"Done"}]`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "conductor", "pass")
	cols, err := client.GetColumns(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetColumns: %v", err)
	}
	if len(cols) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(cols))
	}
	if cols[2].Title != "Done" {
		t.Errorf("third column: want %q, got %q", "Done", cols[2].Title)
	}
}

func TestCreateCard_Success(t *testing.T) {
	t.Parallel()

	var gotPayload map[string]any
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotPayload)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 100}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "conductor", "pass")
	id, err := client.CreateCard(context.Background(), 20, 42)
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if id != 100 {
		t.Errorf("card id: want 100, got %d", id)
	}
	if gotPath != "/api/v1/projects/columns/20/cards" {
		t.Errorf("path: got %s", gotPath)
	}
	if int(gotPayload["content_id"].(float64)) != 42 {
		t.Errorf("content_id: want 42, got %v", gotPayload["content_id"])
	}
}

func TestMoveCard_Success(t *testing.T) {
	t.Parallel()

	var gotPayload map[string]any
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		json.NewDecoder(r.Body).Decode(&gotPayload)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 100}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "conductor", "pass")
	err := client.MoveCard(context.Background(), 100, 22)
	if err != nil {
		t.Fatalf("MoveCard: %v", err)
	}
	if gotMethod != "PATCH" {
		t.Errorf("method: want PATCH, got %s", gotMethod)
	}
	if gotPath != "/api/v1/projects/columns/cards/100" {
		t.Errorf("path: got %s", gotPath)
	}
	if int(gotPayload["column_id"].(float64)) != 22 {
		t.Errorf("column_id: want 22, got %v", gotPayload["column_id"])
	}
}

func TestMoveCard_Error(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "card not found"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "conductor", "pass")
	err := client.MoveCard(context.Background(), 999, 22)
	if err == nil {
		t.Fatal("expected error on 404")
	}
}
