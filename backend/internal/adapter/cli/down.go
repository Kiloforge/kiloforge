package cli

import (
	"fmt"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/pidfile"

	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop the Cortex",
	Long:  `Stops the Cortex daemon.`,
	RunE:  runDown,
}

func runDown(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("%s", notInitializedError())
	}

	// Stop Cortex daemon.
	pidMgr := pidfile.New(cfg.DataDir)
	if running, pid, _ := pidMgr.IsRunning(); running {
		fmt.Printf("==> Stopping Cortex (PID %d)...\n", pid)
		if err := stopDaemon(cfg.DataDir); err != nil {
			fmt.Printf("    Warning: stop Cortex: %v\n", err)
		} else {
			fmt.Println("    Cortex stopped.")
		}
	} else {
		fmt.Println("Cortex is not running.")
	}

	fmt.Println()
	fmt.Println("Restart with: kf up")

	return nil
}
