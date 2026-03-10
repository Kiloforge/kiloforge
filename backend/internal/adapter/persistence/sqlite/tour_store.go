package sqlite

import (
	"database/sql"
	"encoding/json"
	"time"
)

// TourState holds the guided tour progress for onboarding.
type TourState struct {
	Status      string     `json:"status"`                 // pending|active|dismissed|completed
	CurrentStep int        `json:"current_step"`           // 0-indexed step number
	StartedAt   *time.Time `json:"started_at,omitempty"`   // when tour was accepted
	DismissedAt *time.Time `json:"dismissed_at,omitempty"` // when tour was dismissed
	CompletedAt *time.Time `json:"completed_at,omitempty"` // when tour was completed
}

const tourKey = "tour_state"

// TourStore manages guided tour state in the SQLite config table.
type TourStore struct {
	db *sql.DB
}

// NewTourStore creates a TourStore backed by the given database.
func NewTourStore(db *sql.DB) *TourStore {
	return &TourStore{db: db}
}

// GetTourState returns the current tour state. Defaults to pending if no record exists.
func (s *TourStore) GetTourState() (TourState, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM config WHERE key = ?", tourKey).Scan(&value)
	if err != nil {
		return TourState{Status: "pending"}, nil
	}
	var state TourState
	if err := json.Unmarshal([]byte(value), &state); err != nil {
		return TourState{Status: "pending"}, nil
	}
	return state, nil
}

// UpdateTourState persists the tour state.
func (s *TourStore) UpdateTourState(state TourState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		"INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)",
		tourKey, string(data),
	)
	return err
}
