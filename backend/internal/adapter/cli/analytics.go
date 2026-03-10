package cli

import (
	"fmt"
	"os"

	"kiloforge/internal/adapter/config"

	"github.com/spf13/cobra"
)

var analyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "Manage anonymous usage analytics",
	Long: `View and manage anonymous usage analytics settings.

Running 'kf analytics' with no subcommand shows the current analytics state.`,
	RunE: runAnalyticsStatus,
}

var analyticsEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable anonymous usage analytics",
	RunE:  runAnalyticsEnable,
}

var analyticsDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable anonymous usage analytics",
	RunE:  runAnalyticsDisable,
}

var analyticsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current analytics state and source",
	RunE:  runAnalyticsStatus,
}

func init() {
	analyticsCmd.AddCommand(analyticsEnableCmd)
	analyticsCmd.AddCommand(analyticsDisableCmd)
	analyticsCmd.AddCommand(analyticsStatusCmd)
}

func runAnalyticsEnable(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("%s", notInitializedError())
	}

	enabled := true
	cfg.AnalyticsEnabled = &enabled
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	cmd.Println("Analytics enabled.")
	return nil
}

func runAnalyticsDisable(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("%s", notInitializedError())
	}

	disabled := false
	cfg.AnalyticsEnabled = &disabled
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	cmd.Println("Analytics disabled.")
	return nil
}

func runAnalyticsStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("%s", notInitializedError())
	}

	// Determine effective state and source.
	envVal := os.Getenv("KF_ANALYTICS_ENABLED")
	var state, source string

	switch {
	case envVal != "":
		// Env var overrides everything.
		source = "env override"
		if cfg.IsAnalyticsEnabled() {
			state = "enabled"
		} else {
			state = "disabled"
		}
	case cfg.AnalyticsEnabled != nil:
		source = "config"
		if *cfg.AnalyticsEnabled {
			state = "enabled"
		} else {
			state = "disabled"
		}
	default:
		source = "default"
		state = "enabled"
	}

	cmd.Printf("Analytics: %s (%s)\n", state, source)
	return nil
}
