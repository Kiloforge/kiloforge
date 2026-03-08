package sqlite

import (
	"database/sql"
	"time"
)

// ConsentStore manages user consent flags in the SQLite config table.
type ConsentStore struct {
	db *sql.DB
}

// NewConsentStore creates a ConsentStore backed by the given database.
func NewConsentStore(db *sql.DB) *ConsentStore {
	return &ConsentStore{db: db}
}

const consentKey = "agent_permissions_consent"

// HasAgentPermissionsConsent returns true if the user has consented to
// agents running with --dangerously-skip-permissions.
func (s *ConsentStore) HasAgentPermissionsConsent() bool {
	var value string
	err := s.db.QueryRow("SELECT value FROM config WHERE key = ?", consentKey).Scan(&value)
	return err == nil && value != ""
}

// ConsentInfo holds consent state details.
type ConsentInfo struct {
	Consented   bool
	ConsentedAt string // RFC3339 timestamp, empty if not consented
}

// GetAgentPermissionsConsent returns the consent state with timestamp.
func (s *ConsentStore) GetAgentPermissionsConsent() ConsentInfo {
	var value string
	err := s.db.QueryRow("SELECT value FROM config WHERE key = ?", consentKey).Scan(&value)
	if err != nil || value == "" {
		return ConsentInfo{Consented: false}
	}
	return ConsentInfo{Consented: true, ConsentedAt: value}
}

// RecordAgentPermissionsConsent stores the user's consent with a timestamp.
func (s *ConsentStore) RecordAgentPermissionsConsent() error {
	ts := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)",
		consentKey, ts,
	)
	return err
}
