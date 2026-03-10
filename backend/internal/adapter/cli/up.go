package cli

import (
	"fmt"
	"os"

	"kiloforge/internal/adapter/browser"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/pidfile"

	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the orchestrator",
	Long: `Starts the orchestrator as a background daemon. Returns immediately
after it is running.

Use 'kf down' to stop the orchestrator.`,
	RunE: runUp,
}

func runUp(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'kf init' first")
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
	fmt.Printf("Dashboard:   http://localhost:%d/\n", cfg.OrchestratorPort)
	fmt.Println()
	fmt.Println("Use 'kf down' to stop.")

	if !flagNoBrowser {
		dashURL := fmt.Sprintf("http://localhost:%d/", cfg.OrchestratorPort)
		if err := browser.Open(dashURL); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not open browser: %v\n", err)
		}
	}

	return nil
}
