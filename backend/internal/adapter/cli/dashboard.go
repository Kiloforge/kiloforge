package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/dashboard"
	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/persistence/jsonfile"
	"kiloforge/internal/adapter/rest"
	"kiloforge/internal/adapter/rest/gen"

	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Start the web dashboard (standalone)",
	Long: `Starts the web dashboard server without starting Gitea or the orchestrator.
Useful when the orchestrator is already running via 'kf up' and you want
to view the dashboard separately.`,
	RunE: runDashboard,
}

func runDashboard(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'kf init' first")
	}

	store, err := jsonfile.LoadAgentStore(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load agent store: %w", err)
	}

	tracker := agent.NewQuotaTracker(cfg.DataDir)
	_ = tracker.Load()

	reg, err := jsonfile.LoadProjectStore(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load project store: %w", err)
	}

	srv := dashboard.New(cfg.OrchestratorPort, store, tracker, cfg.GiteaURL(), reg)

	// Register OpenAPI generated API handlers on the dashboard mux.
	lockMgr := lock.New(cfg.DataDir)
	lockMgr.StartReaper(ctx)
	apiHandler := rest.NewAPIHandler(rest.APIHandlerOpts{
		Agents:     store,
		Quota:      tracker,
		LockMgr:    lockMgr,
		Projects:   reg,
		GiteaURL:   cfg.GiteaURL(),
		SSEClients: srv.SSEClientCount,
		Cfg:        cfg,
	})
	strictHandler := gen.NewStrictHandler(apiHandler, nil)
	gen.HandlerFromMux(strictHandler, srv.Mux())

	fmt.Printf("Dashboard running at http://localhost:%d\n", cfg.OrchestratorPort)
	fmt.Println("Press Ctrl+C to stop.")
	return srv.Run(ctx)
}
