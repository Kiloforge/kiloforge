package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach <agent-id>",
	Short: "Get the command to interactively resume an agent's Claude session",
	Long: `Looks up the agent's Claude session ID and prints the command to resume it.
The agent will be halted (if running) so you can take over interactively.

Use this when an agent is waiting for input, stuck, or you want to provide
manual guidance.`,
	Args: cobra.ExactArgs(1),
	RunE: runAttach,
}

func runAttach(cmd *cobra.Command, args []string) error {
	rt, err := NewCLIRuntime()
	if err != nil {
		return fmt.Errorf("%s", notInitializedError())
	}
	defer func() { _ = rt.Close() }()

	// Get the agent first to display info before halting.
	agent, err := rt.Agents.GetAgent(args[0])
	if err != nil {
		return err
	}

	wasRunning := agent.Status == "running" && agent.PID > 0

	fmt.Printf("Agent:     %s (%s)\n", agent.ID[:8], agent.Role)
	fmt.Printf("Status:    %s\n", agent.Status)
	fmt.Printf("Session:   %s\n", agent.SessionID)
	fmt.Printf("Worktree:  %s\n", agent.WorktreeDir)
	fmt.Println()

	if wasRunning {
		fmt.Println("This agent is currently running. It will be sent SIGINT to pause it.")
		fmt.Printf("After it stops, resume with:\n\n")
	} else {
		fmt.Printf("Resume this agent's session with:\n\n")
	}

	resumeCmd := fmt.Sprintf("cd %s && claude --resume %s", agent.WorktreeDir, agent.SessionID)
	fmt.Printf("  %s\n\n", resumeCmd)

	if wasRunning {
		if _, err := rt.Agents.AttachAgent(args[0]); err != nil {
			fmt.Printf("Warning: could not halt agent: %v\n", err)
			fmt.Println("You may need to stop it manually before resuming.")
		} else {
			fmt.Println("Agent halted. You can now resume it with the command above.")
		}
	}

	return nil
}
