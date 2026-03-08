package cli

import (
	"database/sql"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/persistence/sqlite"
	"kiloforge/internal/core/service"
)

// CLIRuntime holds the shared service graph for CLI commands.
// Commands call NewCLIRuntime to open the database and construct stores/services,
// then defer rt.Close() to release the database connection.
type CLIRuntime struct {
	Cfg      *config.Config
	DB       *sql.DB
	Agents   *service.AgentService
	Projects *service.ProjectService
	Board    *service.NativeBoardService
	Quota    *agent.QuotaTracker
}

// NewCLIRuntime resolves config, opens the SQLite database, and wires up all
// stores and services used by CLI commands.
func NewCLIRuntime() (*CLIRuntime, error) {
	cfg, err := config.Resolve()
	if err != nil {
		return nil, err
	}
	return NewCLIRuntimeFromConfig(cfg)
}

// NewCLIRuntimeFromConfig builds the runtime from an already-resolved config.
func NewCLIRuntimeFromConfig(cfg *config.Config) (*CLIRuntime, error) {
	db, err := sqlite.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}

	agentStore := sqlite.NewAgentStore(db)
	projectStore := sqlite.NewProjectStore(db)
	prTrackingStore := sqlite.NewPRTrackingStore(db)
	boardStore := sqlite.NewBoardStore(db)

	tracker := agent.NewQuotaTracker(cfg.DataDir)
	_ = tracker.Load()

	return &CLIRuntime{
		Cfg:      cfg,
		DB:       db,
		Agents:   service.NewAgentService(agentStore, projectStore, prTrackingStore),
		Projects: service.NewProjectService(projectStore, nil, service.ProjectServiceConfig{DataDir: cfg.DataDir}),
		Board:    service.NewNativeBoardService(boardStore),
		Quota:    tracker,
	}, nil
}

// Close releases the database connection.
func (rt *CLIRuntime) Close() error {
	if rt.DB != nil {
		return rt.DB.Close()
	}
	return nil
}
