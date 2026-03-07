package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "crelay",
	Short: "Gitea + Claude agent relay for conductor workflows",
	Long: `crelay manages a local Gitea instance and relays webhooks
to spawn, monitor, and control Claude Code agents running conductor skills.

Initialize with 'crelay init' to start Gitea and the relay server.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(agentsCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(destroyCmd)
}
