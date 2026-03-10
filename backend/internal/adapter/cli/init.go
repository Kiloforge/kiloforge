package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:    "init",
	Short:  "Deprecated: use 'kf up' instead",
	Long:   `Deprecated: 'kf init' has been merged into 'kf up'. Use 'kf up' for both first-time setup and starting the Cortex.`,
	Hidden: true,
	RunE:   runInitDeprecated,
}

func runInitDeprecated(cmd *cobra.Command, args []string) error {
	fmt.Println("Note: 'kf init' has been merged into 'kf up'.")
	fmt.Println("      'kf up' now auto-initializes on first run.")
	fmt.Println()
	return runUp(cmd, args)
}
