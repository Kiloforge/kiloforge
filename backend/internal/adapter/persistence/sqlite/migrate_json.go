package sqlite

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"kiloforge/internal/core/domain"
)

// MigrateFromJSON imports data from existing JSON files into the SQLite database.
// Called once when the database is freshly created and JSON files exist.
// Errors are logged but do not halt migration.
func MigrateFromJSON(db *sql.DB, dataDir string) {
	migrateProjects(db, dataDir)
	migrateAgents(db, dataDir)
	migrateConfig(db, dataDir)
}

func migrateProjects(db *sql.DB, dataDir string) {
	data, err := os.ReadFile(filepath.Join(dataDir, "projects.json"))
	if err != nil {
		return
	}
	var store struct {
		Projects map[string]domain.Project `json:"projects"`
	}
	if err := json.Unmarshal(data, &store); err != nil {
		log.Printf("[sqlite-migrate] projects.json parse error: %v", err)
		return
	}
	ps := NewProjectStore(db)
	for _, p := range store.Projects {
		if err := ps.Add(p); err != nil {
			log.Printf("[sqlite-migrate] project %q: %v", p.Slug, err)
		}
	}
	log.Printf("[sqlite-migrate] imported %d projects", len(store.Projects))
}

func migrateAgents(db *sql.DB, dataDir string) {
	data, err := os.ReadFile(filepath.Join(dataDir, "state.json"))
	if err != nil {
		return
	}
	var store struct {
		Agents []domain.AgentInfo `json:"agents"`
	}
	if err := json.Unmarshal(data, &store); err != nil {
		log.Printf("[sqlite-migrate] state.json parse error: %v", err)
		return
	}
	as := NewAgentStore(db)
	for _, a := range store.Agents {
		as.AddAgent(a)
	}
	log.Printf("[sqlite-migrate] imported %d agents", len(store.Agents))
}

func migrateConfig(db *sql.DB, dataDir string) {
	data, err := os.ReadFile(filepath.Join(dataDir, "config.json"))
	if err != nil {
		return
	}
	// Store the raw JSON blob directly.
	db.Exec("INSERT OR REPLACE INTO config (key, value) VALUES ('app', ?)", string(data))
	log.Printf("[sqlite-migrate] imported config")
}
