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
	Short: "Start the Cortex",
	Long: `Starts the Cortex as a background daemon. Returns immediately
after it is running.

Use 'kf down' to stop the Cortex.`,
	RunE: runUp,
}

func runUp(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("%s", notInitializedError())
	}

	// Start Cortex daemon.
	pidMgr := pidfile.New(cfg.DataDir)
	if running, pid, _ := pidMgr.IsRunning(); running {
		fmt.Printf("Cortex already running (PID %d)\n", pid)
	} else {
		fmt.Println("==> Starting Cortex...")
		pid, err := startDaemon(cfg.DataDir)
		if err != nil {
			return fmt.Errorf("start Cortex: %w", err)
		}
		fmt.Printf("    Cortex started (PID %d)\n", pid)
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
