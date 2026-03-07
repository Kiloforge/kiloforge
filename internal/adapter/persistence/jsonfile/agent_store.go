package jsonfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"crelay/internal/core/domain"
	"crelay/internal/core/port"
)

var _ port.AgentStore = (*AgentStore)(nil)

const stateFile = "state.json"

// AgentStore persists agent state to a JSON file.
type AgentStore struct {
	AgentList []domain.AgentInfo `json:"agents"`
	dataDir   string
}

// LoadAgentStore reads agent state from the data directory.
func LoadAgentStore(dataDir string) (*AgentStore, error) {
	path := filepath.Join(dataDir, stateFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &AgentStore{dataDir: dataDir}, nil
	}
	if err != nil {
		return nil, err
	}
	var store AgentStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	store.dataDir = dataDir
	return &store, nil
}

// Load re-reads the agent store from disk.
func (s *AgentStore) Load() error {
	path := filepath.Join(s.dataDir, stateFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		s.AgentList = nil
		return nil
	}
	if err != nil {
		return err
	}
	var loaded AgentStore
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	s.AgentList = loaded.AgentList
	return nil
}

// Save writes the agent store to disk.
func (s *AgentStore) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dataDir, stateFile), data, 0o644)
}

// Agents returns all tracked agents.
func (s *AgentStore) Agents() []domain.AgentInfo {
	return s.AgentList
}

// AddAgent adds an agent to the store.
func (s *AgentStore) AddAgent(agent domain.AgentInfo) {
	s.AgentList = append(s.AgentList, agent)
}

// FindAgent looks up an agent by ID prefix.
func (s *AgentStore) FindAgent(idPrefix string) (*domain.AgentInfo, error) {
	for i := range s.AgentList {
		if s.AgentList[i].ID == idPrefix || strings.HasPrefix(s.AgentList[i].ID, idPrefix) {
			return &s.AgentList[i], nil
		}
	}
	return nil, fmt.Errorf("agent not found: %s", idPrefix)
}

// UpdateStatus updates the status of an agent by ID prefix.
func (s *AgentStore) UpdateStatus(idPrefix, status string) {
	for i := range s.AgentList {
		if s.AgentList[i].ID == idPrefix || strings.HasPrefix(s.AgentList[i].ID, idPrefix) {
			s.AgentList[i].Status = status
			s.AgentList[i].UpdatedAt = time.Now()
			return
		}
	}
}

// AgentsByStatus returns agents matching any of the given statuses.
func (s *AgentStore) AgentsByStatus(statuses ...string) []domain.AgentInfo {
	var result []domain.AgentInfo
	for _, a := range s.AgentList {
		for _, st := range statuses {
			if a.Status == st {
				result = append(result, a)
				break
			}
		}
	}
	return result
}

// HaltAgent sends SIGINT to the agent process.
func (s *AgentStore) HaltAgent(idPrefix string) error {
	agent, err := s.FindAgent(idPrefix)
	if err != nil {
		return err
	}
	if agent.PID <= 0 {
		return fmt.Errorf("no PID recorded for agent %s", idPrefix)
	}

	proc, err := os.FindProcess(agent.PID)
	if err != nil {
		return fmt.Errorf("find process: %w", err)
	}
	return proc.Signal(syscall.SIGINT)
}
