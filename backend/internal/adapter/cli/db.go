package cli

import (
	"database/sql"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/persistence/sqlite"
)

// openDB opens the SQLite database from the configured data directory.
func openDB(cfg *config.Config) (*sql.DB, error) {
	return sqlite.Open(cfg.DataDir)
}
