package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"kiloforge/internal/adapter/compose"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/gitea"
	"kiloforge/internal/adapter/pidfile"

	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the Gitea server and orchestrator",
	Long: `Starts the Gitea Docker Compose stack and the orchestrator as a background
daemon. Returns immediately after both are running.

Use 'kf down' to stop both Gitea and the orchestrator.`,
	RunE: runUp,
}

func runUp(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'kf init' first")
	}

	client := gitea.NewClientWithToken(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.APIToken)

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

	// Start orchestrator daemon.
	pidMgr := pidfile.New(cfg.DataDir)
	if running, pid, _ := pidMgr.IsRunning(); running {
		fmt.Printf("Orchestrator already running (PID %d)\n", pid)
	} else {
		fmt.Println("==> Starting orchestrator...")
		pid, err := startDaemon(cfg.DataDir)
		if err != nil {
			return fmt.Errorf("start orchestrator: %w", err)
		}
		fmt.Printf("    Orchestrator started (PID %d)\n", pid)
	}

	fmt.Println()
	fmt.Printf("Server:      http://localhost:%d\n", cfg.OrchestratorPort)
	fmt.Printf("Dashboard:   http://localhost:%d/-/\n", cfg.OrchestratorPort)
	fmt.Printf("Gitea:       http://localhost:%d/\n", cfg.OrchestratorPort)
	fmt.Println()
	fmt.Println("Use 'kf down' to stop.")

	return nil
}
