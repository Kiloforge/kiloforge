package cli

import (
	"context"
	"os"

	"kiloforge/internal/adapter/analytics"
	"kiloforge/internal/adapter/config"

	"github.com/spf13/cobra"
)

var (
	flagNoBrowser bool
	flagDataDir   string
	flagHost      string
	flagPort      int
)

var rootCmd = &cobra.Command{
	Use:   "kf",
	Short: "Kiloforge Cortex — local control plane for Claude Code agent swarms",
	Long: `Kiloforge — the development forge for the Kiloforger.

Runs the Cortex control plane, Command Deck, and Claude Code swarms
directly on your machine. Transforming pure intent into meaningful action.

Run 'kf up' to start the Cortex.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Fire-and-forget CLI command tracking.
		// Only tracks if kf is initialized and analytics is enabled.
		cfg, err := config.Resolve()
		if err != nil || !cfg.IsAnalyticsEnabled() {
			return
		}
		apiKey := cfg.PostHogAPIKey
		if apiKey == "" {
			apiKey = analytics.DefaultPostHogAPIKey
		}
		tracker := analytics.NewPostHog(apiKey, analytics.AnonymousID(cfg.DataDir))
		tracker.Track(cmd.Context(), "cli_command", map[string]any{
			"command": cmd.Name(),
		})
		// Best-effort: drain in background. The process may exit before send
		// completes for very short commands — that's acceptable for CLI telemetry.
		go func() { _ = tracker.Shutdown(context.Background()) }()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagNoBrowser, "no-browser", os.Getenv("KF_NO_BROWSER") == "1", "Do not open the Command Deck in a browser")
	rootCmd.PersistentFlags().StringVar(&flagDataDir, "data-dir", "", "Persistent data directory (defaults to ~/.kiloforge)")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(destroyCmd)
	rootCmd.AddCommand(poolCmd)
	rootCmd.AddCommand(implementCmd)
	rootCmd.AddCommand(agentsCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(escalatedCmd)
	rootCmd.AddCommand(costCmd)
	rootCmd.AddCommand(dashboardCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(skillsCmd)
	rootCmd.AddCommand(versionCmd)
}
