package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop <agent-id>",
	Short: "Stop a running agent",
	Long:  `Sends SIGINT to the agent process, gracefully stopping it. The session is preserved and can be resumed later with 'kf attach'.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	rt, err := NewCLIRuntime()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	defer rt.Close()

	agent, err := rt.Agents.StopAgent(args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Agent %s stopped.\n", agent.ID[:8])
	fmt.Printf("Resume with: claude --resume %s\n", agent.SessionID)
	return nil
}
