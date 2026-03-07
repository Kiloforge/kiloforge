package cli

import (
	"fmt"
	"os/exec"

	"conductor-relay/internal/config"

	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Stop and remove the Gitea container and relay data",
	Long: `Stops the Gitea Docker container, removes it, and optionally
cleans up the data directory. The git remote 'gitea' is also removed.`,
	RunE: runDestroy,
}

var flagDestroyData bool

func init() {
	destroyCmd.Flags().BoolVar(&flagDestroyData, "data", false, "Also delete persistent data (volumes, logs, state)")
}

func runDestroy(cmd *cobra.Command, args []string) error {
	fmt.Println("==> Stopping Gitea container...")
	_ = exec.Command("docker", "stop", config.ContainerName).Run()
	_ = exec.Command("docker", "rm", config.ContainerName).Run()
	fmt.Println("    Container removed.")

	fmt.Println("==> Removing git remote 'gitea'...")
	_ = exec.Command("git", "remote", "remove", "gitea").Run()
	fmt.Println("    Remote removed.")

	if flagDestroyData {
		cfg, err := config.Load()
		if err == nil {
			fmt.Printf("==> Removing data directory %s...\n", cfg.DataDir)
			_ = exec.Command("rm", "-rf", cfg.DataDir).Run()
			fmt.Println("    Data removed.")
		}
	}

	fmt.Println("Done. Gitea and relay have been torn down.")
	return nil
}
