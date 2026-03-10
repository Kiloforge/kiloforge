package cli

import (
	"fmt"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/pidfile"

	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop the orchestrator",
	Long:  `Stops the orchestrator daemon.`,
	RunE:  runDown,
}

func runDown(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("%s", notInitializedError())
	}

	// Stop orchestrator daemon.
	pidMgr := pidfile.New(cfg.DataDir)
	if running, pid, _ := pidMgr.IsRunning(); running {
		fmt.Printf("==> Stopping orchestrator (PID %d)...\n", pid)
		if err := stopDaemon(cfg.DataDir); err != nil {
			fmt.Printf("    Warning: stop orchestrator: %v\n", err)
		} else {
			fmt.Println("    Orchestrator stopped.")
		}
	} else {
		fmt.Println("Orchestrator is not running.")
	}

	fmt.Println()
	fmt.Println("Restart with: kf up")

	return nil
}
