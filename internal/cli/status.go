package cli

import (
	"fmt"
	"os/exec"
	"strings"

	"conductor-relay/internal/config"
	"conductor-relay/internal/state"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Gitea, relay, and agent status",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w (have you run 'conductor-relay init'?)", err)
	}

	// Check Gitea container.
	giteaStatus := "stopped"
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Status}}", config.ContainerName).Output()
	if err == nil {
		giteaStatus = strings.TrimSpace(string(out))
	}

	// Check relay by attempting connection.
	relayStatus := "stopped"
	if _, err := exec.Command("curl", "-sf", fmt.Sprintf("http://localhost:%d/health", cfg.RelayPort)).Output(); err == nil {
		relayStatus = "running"
	}

	// Load agent state.
	store, _ := state.Load(cfg.DataDir)
	activeCount := 0
	if store != nil {
		for _, a := range store.Agents {
			if a.Status == "running" || a.Status == "waiting" {
				activeCount++
			}
		}
	}

	fmt.Println("Conductor Relay Status")
	fmt.Println("======================")
	fmt.Printf("Gitea:       %s (http://localhost:%d)\n", giteaStatus, cfg.GiteaPort)
	fmt.Printf("Relay:       %s (http://localhost:%d)\n", relayStatus, cfg.RelayPort)
	fmt.Printf("Project:     %s\n", cfg.ProjectDir)
	fmt.Printf("Repository:  %s/%s\n", config.GiteaAdminUser, cfg.RepoName)
	fmt.Printf("Data:        %s\n", cfg.DataDir)
	fmt.Printf("Agents:      %d active\n", activeCount)

	return nil
}
