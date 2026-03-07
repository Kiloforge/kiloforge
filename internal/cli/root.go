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
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(destroyCmd)
	rootCmd.AddCommand(poolCmd)
	rootCmd.AddCommand(implementCmd)
	rootCmd.AddCommand(agentsCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(attachCmd)
}
