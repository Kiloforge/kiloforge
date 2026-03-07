package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"crelay/internal/compose"
	"crelay/internal/config"

	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Stop and remove the Gitea server",
	Long: `Tears down the Gitea Docker Compose stack. With --data, also removes
volumes and the data directory.`,
	RunE: runDestroy,
}

var flagDestroyData bool

func init() {
	destroyCmd.Flags().BoolVar(&flagDestroyData, "data", false, "Also delete persistent data (volumes, logs, state)")
}

func runDestroy(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w (have you run 'crelay init'?)", err)
	}

	runner, err := compose.Detect()
	if err != nil {
		return err
	}

	fmt.Println("==> Stopping Gitea...")
	if err := runner.Down(ctx, cfg.DataDir, flagDestroyData); err != nil {
		return fmt.Errorf("compose down: %w", err)
	}
	fmt.Println("    Gitea stopped and removed.")

	if flagDestroyData {
		fmt.Printf("==> Removing data directory %s...\n", cfg.DataDir)
		if err := os.RemoveAll(cfg.DataDir); err != nil {
			return fmt.Errorf("remove data dir: %w", err)
		}
		fmt.Println("    Data removed.")
	}

	fmt.Println("Done.")
	return nil
}
