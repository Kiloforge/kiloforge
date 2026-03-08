package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	appVersion = "dev"
	appCommit  = "none"
	appDate    = "unknown"
)

// SetVersionInfo is called from main to inject build-time values.
func SetVersionInfo(version, commit, date string) {
	appVersion = version
	appCommit = commit
	appDate = date
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("kf version %s\n", appVersion)
		fmt.Printf("  commit:  %s\n", appCommit)
		fmt.Printf("  built:   %s\n", appDate)
		fmt.Printf("  go:      %s\n", runtime.Version())
		fmt.Printf("  os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}
