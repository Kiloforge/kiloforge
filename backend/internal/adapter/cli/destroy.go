package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"kiloforge/internal/adapter/config"

	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Permanently destroy all kiloforge data",
	Long: `Removes the Cortex, all project registrations, and the entire data directory.

This action cannot be undone. You will be prompted to confirm unless --force is used.`,
	RunE: runDestroy,
}

var flagDestroyForce bool

func init() {
	destroyCmd.Flags().BoolVar(&flagDestroyForce, "force", false, "Skip confirmation prompt")
}

func runDestroy(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		fmt.Println("Nothing to destroy — kiloforge is not initialized.")
		return nil
	}

	if !flagDestroyForce {
		fmt.Println()
		fmt.Println("  WARNING: This will permanently delete:")
		fmt.Println("    - All project registrations")
		fmt.Println("    - All agent state and logs")
		fmt.Printf("    - Data directory: %s\n", cfg.DataDir)
		fmt.Println()
		fmt.Println("  This action cannot be undone.")
		fmt.Println()
		fmt.Print("  Type \"yes\" to confirm: ")

		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Stop Cortex daemon first.
	fmt.Println("==> Stopping Cortex...")
	if err := stopDaemon(cfg.DataDir); err != nil {
		fmt.Printf("    Warning: stop Cortex: %v\n", err)
	}

	fmt.Printf("==> Removing data directory %s...\n", cfg.DataDir)
	if err := os.RemoveAll(cfg.DataDir); err != nil {
		return fmt.Errorf("remove data dir: %w", err)
	}

	fmt.Println("Done. All kiloforge data has been destroyed.")
	return nil
}
