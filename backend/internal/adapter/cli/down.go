package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"crelay/internal/adapter/compose"
	"crelay/internal/adapter/config"
	"crelay/internal/adapter/gitea"

	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop the Gitea server",
	Long:  `Stops the Gitea Docker Compose stack without removing containers or data.`,
	RunE:  runDown,
}

func runDown(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'crelay init' first")
	}

	// Check if already stopped.
	client := gitea.NewClient(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.GiteaAdminPass)
	if _, err := client.CheckVersion(ctx); err != nil {
		fmt.Println("Gitea is not running.")
		return nil
	}

	runner, err := compose.Detect()
	if err != nil {
		return err
	}

	fmt.Println("==> Stopping Gitea...")
	if err := runner.Stop(ctx, cfg.DataDir); err != nil {
		return fmt.Errorf("compose stop: %w", err)
	}
	fmt.Println("    Gitea stopped.")
	fmt.Println()
	fmt.Println("Restart with: crelay up")

	return nil
}
