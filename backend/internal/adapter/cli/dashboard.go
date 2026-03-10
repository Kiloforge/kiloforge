package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/dashboard"
	gitadapter "kiloforge/internal/adapter/git"
	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/persistence/sqlite"
	"kiloforge/internal/adapter/rest"
	"kiloforge/internal/adapter/rest/gen"
	wsAdapter "kiloforge/internal/adapter/ws"
	"kiloforge/internal/core/service"

	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Start the web dashboard (standalone)",
	Long: `Starts the web dashboard server without starting the orchestrator.
Useful when the orchestrator is already running via 'kf up' and you want
to view the dashboard separately.`,
	RunE: runDashboard,
}

func runDashboard(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("%s", notInitializedError())
	}

	// Open SQLite database for all stores.
	db, err := openDB(cfg)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	store := sqlite.NewAgentStore(db)
	reg := sqlite.NewProjectStore(db)

	tracker := agent.NewQuotaTracker(cfg.DataDir)
	_ = tracker.Load()

	traceStore := sqlite.NewTraceStore(db)
	boardStore := sqlite.NewBoardStore(db)
	boardSvc := service.NewNativeBoardService(boardStore)
	consentStore := sqlite.NewConsentStore(db)

	srv := dashboard.New(cfg.OrchestratorPort, store, tracker, "/", reg, nil)
	srv.SetTraceStore(traceStore)
	srv.SetTrackReader(service.NewTrackReader())
	srv.SetBudgetUSD(cfg.BudgetUSD)

	// Create interactive agent spawner and WebSocket session manager.
	spawner := agent.NewSpawner(cfg, store, tracker)
	wsSessions := wsAdapter.NewSessionManager()

	// Create project service for project management.
	projectSvc := service.NewProjectService(reg, service.ProjectServiceConfig{
		DataDir:          cfg.DataDir,
		OrchestratorPort: cfg.OrchestratorPort,
	})

	// Register OpenAPI generated API handlers on the dashboard mux.
	lockMgr := lock.New(cfg.DataDir)
	lockMgr.StartReaper(ctx)
	apiHandler := rest.NewAPIHandler(rest.APIHandlerOpts{
		Agents:       store,
		Quota:        tracker,
		LockMgr:      lockMgr,
		Projects:     reg,
		ProjectMgr:   projectSvc,
		GitSync:      gitadapter.New(),
		TraceStore:   traceStore,
		BoardSvc:     boardSvc,
		EventBus:     srv.EventBus(),
		SSEClients:   srv.SSEClientCount,
		Cfg:          cfg,
		InterSpawner: spawner,
		WSSessions:   wsSessions,
		Consent:      consentStore,
	})
	strictHandler := gen.NewStrictHandler(apiHandler, nil)
	gen.HandlerFromMux(strictHandler, srv.Mux())

	// Register WebSocket handler for interactive agent sessions.
	wsHandler := wsAdapter.NewHandler(wsSessions, nil)
	wsHandler.RegisterRoutes(srv.Mux())

	fmt.Printf("Dashboard running at http://localhost:%d\n", cfg.OrchestratorPort)
	fmt.Println("Press Ctrl+C to stop.")
	return srv.Run(ctx)
}
