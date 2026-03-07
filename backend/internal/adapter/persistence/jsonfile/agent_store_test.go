package jsonfile

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"crelay/internal/core/domain"
)

func TestAgentStore_LoadEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := LoadAgentStore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.Agents()) != 0 {
		t.Errorf("expected empty agents, got %d", len(store.Agents()))
	}
}

func TestAgentStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, _ := LoadAgentStore(dir)

	store.AddAgent(domain.AgentInfo{ID: "agent-1", Role: "developer", Status: "running"})
	store.AddAgent(domain.AgentInfo{ID: "agent-2", Role: "reviewer", Status: "completed"})

	if err := store.Save(); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := LoadAgentStore(dir)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	agents := loaded.Agents()
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	if agents[0].ID != "agent-1" {
		t.Errorf("expected agent-1, got %q", agents[0].ID)
	}
}

func TestAgentStore_FindAgent(t *testing.T) {
	t.Parallel()

	store, _ := LoadAgentStore(t.TempDir())
	store.AddAgent(domain.AgentInfo{ID: "abc-123", Role: "developer"})
	store.AddAgent(domain.AgentInfo{ID: "def-456", Role: "reviewer"})

	tests := []struct {
		name    string
		prefix  string
		wantID  string
		wantErr bool
	}{
		{"exact match", "abc-123", "abc-123", false},
		{"prefix match", "abc", "abc-123", false},
		{"not found", "xyz", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			agent, err := store.FindAgent(tt.prefix)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if agent.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", agent.ID, tt.wantID)
			}
		})
	}
}

func TestAgentStore_UpdateStatus(t *testing.T) {
	t.Parallel()

	store, _ := LoadAgentStore(t.TempDir())
	store.AddAgent(domain.AgentInfo{ID: "agent-1", Status: "running"})

	store.UpdateStatus("agent-1", "completed")

	agent, _ := store.FindAgent("agent-1")
	if agent.Status != "completed" {
		t.Errorf("Status = %q, want %q", agent.Status, "completed")
	}
	if agent.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestAgentStore_UpdateStatus_NotFound(t *testing.T) {
	t.Parallel()

	store, _ := LoadAgentStore(t.TempDir())
	// Should not panic when updating non-existent agent.
	store.UpdateStatus("nonexistent", "completed")
}

func TestAgentStore_Load_Reload(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, _ := LoadAgentStore(dir)
	store.AddAgent(domain.AgentInfo{ID: "a1"})
	store.Save()

	// Create a second store and modify.
	store2, _ := LoadAgentStore(dir)
	store2.AddAgent(domain.AgentInfo{ID: "a2"})
	store2.Save()

	// Reload first store.
	store.Load()
	if len(store.Agents()) != 2 {
		t.Errorf("expected 2 agents after reload, got %d", len(store.Agents()))
	}
}

func TestAgentStore_CorruptJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, stateFile), []byte("{invalid json}"), 0o644)

	_, err := LoadAgentStore(dir)
	if err == nil {
		t.Error("expected error for corrupt JSON")
	}
}

func TestAgentStore_AgentsByStatus(t *testing.T) {
	t.Parallel()

	store, _ := LoadAgentStore(t.TempDir())
	store.AddAgent(domain.AgentInfo{ID: "a1", Status: "running"})
	store.AddAgent(domain.AgentInfo{ID: "a2", Status: "completed"})
	store.AddAgent(domain.AgentInfo{ID: "a3", Status: "running"})
	store.AddAgent(domain.AgentInfo{ID: "a4", Status: "failed"})

	running := store.AgentsByStatus("running")
	if len(running) != 2 {
		t.Errorf("expected 2 running, got %d", len(running))
	}

	multi := store.AgentsByStatus("running", "failed")
	if len(multi) != 3 {
		t.Errorf("expected 3 running+failed, got %d", len(multi))
	}

	none := store.AgentsByStatus("nonexistent")
	if len(none) != 0 {
		t.Errorf("expected 0, got %d", len(none))
	}
}

func TestAgentStore_FindByRef(t *testing.T) {
	t.Parallel()

	store, _ := LoadAgentStore(t.TempDir())
	now := time.Now()

	store.AddAgent(domain.AgentInfo{ID: "a1", Ref: "track-1", StartedAt: now.Add(-2 * time.Hour)})
	store.AddAgent(domain.AgentInfo{ID: "a2", Ref: "track-1", StartedAt: now.Add(-1 * time.Hour)})
	store.AddAgent(domain.AgentInfo{ID: "a3", Ref: "track-2", StartedAt: now})

	tests := []struct {
		name   string
		ref    string
		wantID string
	}{
		{"most recent match", "track-1", "a2"},
		{"single match", "track-2", "a3"},
		{"no match", "track-999", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := store.FindByRef(tt.ref)
			if tt.wantID == "" {
				if got != nil {
					t.Errorf("expected nil, got %q", got.ID)
				}
				return
			}
			if got == nil {
				t.Fatal("expected agent, got nil")
			}
			if got.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", got.ID, tt.wantID)
			}
		})
	}
}
