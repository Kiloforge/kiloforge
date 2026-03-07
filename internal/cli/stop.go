package cli

import (
	"fmt"

	"crelay/internal/config"
	"crelay/internal/state"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop <agent-id>",
	Short: "Stop a running agent",
	Long:  `Sends SIGINT to the agent process, gracefully stopping it. The session is preserved and can be resumed later with 'crelay attach'.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	store, err := state.Load(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	agentID := args[0]
	agent, err := store.FindAgent(agentID)
	if err != nil {
		return err
	}

	if agent.Status != "running" && agent.Status != "waiting" {
		fmt.Printf("Agent %s is not running (status: %s)\n", agentID, agent.Status)
		return nil
	}

	if err := store.HaltAgent(agentID); err != nil {
		return fmt.Errorf("halt agent: %w", err)
	}

	store.UpdateStatus(agentID, "stopped")
	if err := store.Save(cfg.DataDir); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	fmt.Printf("Agent %s stopped.\n", agent.ID[:8])
	fmt.Printf("Resume with: claude --resume %s\n", agent.SessionID)
	return nil
}
