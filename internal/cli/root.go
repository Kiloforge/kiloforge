package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "crelay",
	Short: "Gitea + Claude agent relay for conductor workflows",
	Long: `crelay manages a local Gitea instance for conductor-based development
and automated code review with Claude Code agents.

Initialize with 'crelay init' to start the global Gitea server.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(destroyCmd)
	// Project-specific commands (agents, logs, attach, stop) are disabled
	// until project context is restored via 'crelay add' (future track).
}
