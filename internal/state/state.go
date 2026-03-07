package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"crelay/internal/core/domain"
)

const stateFile = "state.json"

// Store holds all tracked agents.
type Store struct {
	Agents []domain.AgentInfo `json:"agents"`
}

func Load(dataDir string) (*Store, error) {
	path := filepath.Join(dataDir, stateFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Store{}, nil
	}
	if err != nil {
		return nil, err
	}
	var store Store
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	return &store, nil
}

func (s *Store) Save(dataDir string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dataDir, stateFile), data, 0o644)
}

func (s *Store) AddAgent(agent domain.AgentInfo) {
	s.Agents = append(s.Agents, agent)
}

func (s *Store) FindAgent(idPrefix string) (*domain.AgentInfo, error) {
	for i := range s.Agents {
		if s.Agents[i].ID == idPrefix || strings.HasPrefix(s.Agents[i].ID, idPrefix) {
			return &s.Agents[i], nil
		}
	}
	return nil, fmt.Errorf("agent not found: %s", idPrefix)
}

func (s *Store) UpdateStatus(idPrefix, status string) {
	for i := range s.Agents {
		if s.Agents[i].ID == idPrefix || strings.HasPrefix(s.Agents[i].ID, idPrefix) {
			s.Agents[i].Status = status
			s.Agents[i].UpdatedAt = time.Now()
			return
		}
	}
}

// HaltAgent sends SIGINT to the agent process.
func (s *Store) HaltAgent(idPrefix string) error {
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
