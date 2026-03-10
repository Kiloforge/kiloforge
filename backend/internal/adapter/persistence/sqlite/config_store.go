package sqlite

import (
	"database/sql"
	"encoding/json"

	"kiloforge/internal/adapter/config"
)

// ConfigStore reads and writes config to the SQLite config table.
// It stores a single JSON blob under the key "app".
type ConfigStore struct {
	db *sql.DB
}

// NewConfigStore creates a ConfigStore backed by the given database.
func NewConfigStore(db *sql.DB) *ConfigStore {
	return &ConfigStore{db: db}
}

// Load reads config from SQLite. Returns a zero Config if not found.
func (s *ConfigStore) Load() (*config.Config, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM config WHERE key = 'app'").Scan(&value)
	if err != nil {
		return &config.Config{}, nil
	}
	var cfg config.Config
	if err := json.Unmarshal([]byte(value), &cfg); err != nil {
		return &config.Config{}, nil
	}
	return &cfg, nil
}

// Save writes config to SQLite.
func (s *ConfigStore) Save(cfg *config.Config) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		"INSERT OR REPLACE INTO config (key, value) VALUES ('app', ?)",
		string(data),
	)
	return err
}
