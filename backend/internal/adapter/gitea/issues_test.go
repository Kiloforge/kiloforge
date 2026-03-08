package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateIssue_Success(t *testing.T) {
	t.Parallel()

	var gotPayload map[string]any
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotPayload)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"number": 42, "title": "test issue"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "kiloforger", "pass")
	num, err := client.CreateIssue(context.Background(), "myapp", "test issue", "issue body", []string{"bug", "urgent"})
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if num != 42 {
		t.Errorf("issue number: want 42, got %d", num)
	}
	if gotPath != "/api/v1/repos/conductor/myapp/issues" {
		t.Errorf("path: want /api/v1/repos/conductor/myapp/issues, got %s", gotPath)
	}
	if gotPayload["title"] != "test issue" {
		t.Errorf("title: want %q, got %v", "test issue", gotPayload["title"])
	}
	if gotPayload["body"] != "issue body" {
		t.Errorf("body: want %q, got %v", "issue body", gotPayload["body"])
	}
}

func TestCreateIssue_Error(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "server error"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "kiloforger", "pass")
	_, err := client.CreateIssue(context.Background(), "myapp", "test", "", nil)
	if err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestUpdateIssue_Success(t *testing.T) {
	t.Parallel()

	var gotPayload map[string]any
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		json.NewDecoder(r.Body).Decode(&gotPayload)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"number": 5}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "kiloforger", "pass")
	err := client.UpdateIssue(context.Background(), "myapp", 5, "new title", "", "closed")
	if err != nil {
		t.Fatalf("UpdateIssue: %v", err)
	}
	if gotMethod != "PATCH" {
		t.Errorf("method: want PATCH, got %s", gotMethod)
	}
	if gotPath != "/api/v1/repos/conductor/myapp/issues/5" {
		t.Errorf("path: want /api/v1/repos/conductor/myapp/issues/5, got %s", gotPath)
	}
	if gotPayload["title"] != "new title" {
		t.Errorf("title: want %q, got %v", "new title", gotPayload["title"])
	}
	if gotPayload["state"] != "closed" {
		t.Errorf("state: want %q, got %v", "closed", gotPayload["state"])
	}
	if _, ok := gotPayload["body"]; ok {
		t.Error("body should not be sent when empty")
	}
}

func TestGetIssues_Success(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"number":1,"title":"First","state":"open"},{"number":2,"title":"Second","state":"closed"}]`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "kiloforger", "pass")
	issues, err := client.GetIssues(context.Background(), "myapp", "open", []string{"bug"})
	if err != nil {
		t.Fatalf("GetIssues: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Number != 1 {
		t.Errorf("first issue number: want 1, got %d", issues[0].Number)
	}
	if gotPath != "/api/v1/repos/conductor/myapp/issues?state=open&labels=bug" {
		t.Errorf("path: want ?state=open&labels=bug, got %s", gotPath)
	}
}

func TestGetIssues_Empty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "kiloforger", "pass")
	issues, err := client.GetIssues(context.Background(), "myapp", "", nil)
	if err != nil {
		t.Fatalf("GetIssues: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestEnsureLabels_AllNew(t *testing.T) {
	t.Parallel()

	var createCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
		case r.Method == "POST":
			createCalls++
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id": 1}`))
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "kiloforger", "pass")
	err := client.EnsureLabels(context.Background(), "myapp", []LabelDef{
		{Name: "track", Color: "#0075ca"},
		{Name: "phase", Color: "#e4e669"},
	})
	if err != nil {
		t.Fatalf("EnsureLabels: %v", err)
	}
	if createCalls != 2 {
		t.Errorf("expected 2 create calls, got %d", createCalls)
	}
}

func TestEnsureLabels_AllExisting(t *testing.T) {
	t.Parallel()

	var createCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"name":"track"},{"name":"phase"}]`))
		case r.Method == "POST":
			createCalls++
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id": 1}`))
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "kiloforger", "pass")
	err := client.EnsureLabels(context.Background(), "myapp", []LabelDef{
		{Name: "track", Color: "#0075ca"},
		{Name: "phase", Color: "#e4e669"},
	})
	if err != nil {
		t.Fatalf("EnsureLabels: %v", err)
	}
	if createCalls != 0 {
		t.Errorf("expected 0 create calls (all exist), got %d", createCalls)
	}
}

func TestEnsureLabels_SomeExisting(t *testing.T) {
	t.Parallel()

	var createCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"name":"track"}]`))
		case r.Method == "POST":
			createCalls++
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id": 2}`))
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "kiloforger", "pass")
	err := client.EnsureLabels(context.Background(), "myapp", []LabelDef{
		{Name: "track", Color: "#0075ca"},
		{Name: "phase", Color: "#e4e669"},
	})
	if err != nil {
		t.Fatalf("EnsureLabels: %v", err)
	}
	if createCalls != 1 {
		t.Errorf("expected 1 create call, got %d", createCalls)
	}
}
