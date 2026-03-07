package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"crelay/internal/adapter/compose"
	"crelay/internal/adapter/config"
	"crelay/internal/adapter/gitea"
	"crelay/internal/adapter/pidfile"

	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop the relay daemon and Gitea server",
	Long:  `Stops the relay daemon and the Gitea Docker Compose stack without removing containers or data.`,
	RunE:  runDown,
}

func runDown(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'crelay init' first")
	}

	// Stop relay daemon.
	pidMgr := pidfile.New(cfg.DataDir)
	if running, pid, _ := pidMgr.IsRunning(); running {
		fmt.Printf("==> Stopping relay daemon (PID %d)...\n", pid)
		if err := stopDaemon(cfg.DataDir); err != nil {
			fmt.Printf("    Warning: stop relay: %v\n", err)
		} else {
			fmt.Println("    Relay stopped.")
		}
	} else {
		fmt.Println("Relay is not running.")
	}

	// Stop Gitea.
	client := gitea.NewClientWithToken(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.APIToken)
	if _, err := client.CheckVersion(ctx); err != nil {
		fmt.Println("Gitea is not running.")
	} else {
		runner, err := compose.Detect()
		if err != nil {
			return err
		}
		fmt.Println("==> Stopping Gitea...")
		if err := runner.Stop(ctx, cfg.DataDir); err != nil {
			return fmt.Errorf("compose stop: %w", err)
		}
		fmt.Println("    Gitea stopped.")
	}

	fmt.Println()
	fmt.Println("Restart with: crelay up")

	return nil
}
