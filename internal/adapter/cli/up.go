package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"crelay/internal/adapter/agent"
	"crelay/internal/adapter/compose"
	"crelay/internal/adapter/config"
	"crelay/internal/adapter/gitea"
	"crelay/internal/adapter/persistence/jsonfile"
	"crelay/internal/adapter/rest"

	"github.com/spf13/cobra"
)

var flagNoDashboard bool

func init() {
	upCmd.Flags().BoolVar(&flagNoDashboard, "no-dashboard", false, "Disable the web dashboard")
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the Gitea server and relay",
	Long: `Starts the Gitea Docker Compose stack and runs the webhook relay server
in the foreground. Requires 'crelay init' to have been run first.

Press Ctrl+C to stop the relay. Gitea stays running via Docker Compose.
Use 'crelay down' to stop Gitea.`,
	RunE: runUp,
}

func runUp(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'crelay init' first")
	}

	client := gitea.NewClient(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.GiteaAdminPass)

	// Start Gitea if not running.
	if _, err := client.CheckVersion(ctx); err != nil {
		runner, err := compose.Detect()
		if err != nil {
			return err
		}
		fmt.Println("==> Starting Gitea...")
		manager := gitea.NewManager(cfg, runner)
		if err := manager.Start(ctx); err != nil {
			return fmt.Errorf("start gitea: %w", err)
		}
		fmt.Printf("    Gitea running at %s\n", cfg.GiteaURL())
	} else {
		fmt.Printf("Gitea already running at %s\n", cfg.GiteaURL())
	}

	// Load project registry.
	reg, err := jsonfile.LoadProjectStore(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load project registry: %w", err)
	}

	projects := reg.List()
	if len(projects) == 0 {
		fmt.Println()
		fmt.Println("No projects registered. Use 'crelay add <path>' to register a project.")
		fmt.Println()
	}

	// Build server options.
	opts := []rest.ServerOption{
		rest.WithGiteaProxy(cfg.GiteaURL()),
	}
	if cfg.IsDashboardEnabled() && !flagNoDashboard {
		store, err := jsonfile.LoadAgentStore(cfg.DataDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: load agent store for dashboard: %v\n", err)
		} else {
			tracker := agent.NewQuotaTracker(cfg.DataDir)
			_ = tracker.Load()
			projectDir, _ := os.Getwd()
			opts = append(opts, rest.WithDashboard(store, tracker, "/gitea", projectDir))
			fmt.Printf("==> Dashboard at http://localhost:%d\n", cfg.RelayPort)
		}
	}

	// Start unified server (blocking).
	fmt.Printf("==> Gitea proxy at http://localhost:%d/gitea/\n", cfg.RelayPort)
	fmt.Printf("==> Starting server on :%d (%d project(s))...\n", cfg.RelayPort, len(projects))
	fmt.Println("    Press Ctrl+C to stop.")
	fmt.Println()

	srv := rest.NewServer(cfg, reg, cfg.RelayPort, opts...)
	return srv.Run(ctx)
}
