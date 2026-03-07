package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"crelay/internal/adapter/agent"
	"crelay/internal/adapter/config"
	"crelay/internal/adapter/dashboard"
	"crelay/internal/adapter/persistence/jsonfile"

	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Start the web dashboard (standalone)",
	Long: `Starts the web dashboard server without starting Gitea or the relay.
Useful when the relay is already running via 'crelay up' and you want
to view the dashboard separately.`,
	RunE: runDashboard,
}

func runDashboard(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'crelay init' first")
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

	srv := dashboard.New(cfg.RelayPort, store, tracker, cfg.GiteaURL(), reg)
	fmt.Printf("Dashboard running at http://localhost:%d\n", cfg.RelayPort)
	fmt.Println("Press Ctrl+C to stop.")
	return srv.Run(ctx)
}
