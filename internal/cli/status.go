package cli

import (
	"context"
	"fmt"
	"strings"

	"crelay/internal/compose"
	"crelay/internal/config"
	"crelay/internal/gitea"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Gitea server status",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("load config: %w (have you run 'crelay init'?)", err)
	}

	// Check Gitea via API.
	giteaStatus := "stopped"
	giteaVersion := ""
	client := gitea.NewClient(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.GiteaAdminPass)
	if v, err := client.CheckVersion(ctx); err == nil {
		giteaStatus = "running"
		giteaVersion = v
	}

	// Try compose ps for container details.
	composeInfo := ""
	if runner, err := compose.Detect(); err == nil {
		if ps, err := runner.Ps(ctx, cfg.DataDir); err == nil {
			composeInfo = strings.TrimSpace(ps)
		}
	}

	fmt.Println("Conductor Relay Status")
	fmt.Println("======================")
	if giteaVersion != "" {
		fmt.Printf("Gitea:       %s (v%s) — %s\n", giteaStatus, giteaVersion, cfg.GiteaURL())
	} else {
		fmt.Printf("Gitea:       %s\n", giteaStatus)
	}
	fmt.Printf("Data:        %s\n", cfg.DataDir)
	fmt.Printf("Compose:     %s\n", cfg.ComposeFile)

	if composeInfo != "" {
		fmt.Println()
		fmt.Println(composeInfo)
	}

	return nil
}
