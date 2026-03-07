package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"crelay/internal/adapter/persistence/jsonfile"
	"crelay/internal/adapter/config"

	"github.com/spf13/cobra"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "List active and recent agents",
	RunE:  runAgents,
}

var flagAgentsJSON bool

func init() {
	agentsCmd.Flags().BoolVar(&flagAgentsJSON, "json", false, "Output as JSON")
}

func runAgents(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("load config: %w (have you run 'crelay init'?)", err)
	}

	store, err := jsonfile.LoadAgentStore(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	agents := store.AgentList

	if flagAgentsJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(agents)
	}

	if len(agents) == 0 {
		fmt.Println("No agents tracked.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tROLE\tTRACK/PR\tSTATUS\tSESSION\tSTARTED\tINFO")
	for _, a := range agents {
		info := agentStatusInfo(a.Status, a.ResumeError)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			a.ID[:8], a.Role, a.Ref, a.Status, a.SessionID[:8], a.StartedAt.Format("15:04:05"), info)
	}
	return w.Flush()
}

func agentStatusInfo(status, resumeErr string) string {
	switch status {
	case "resume-failed":
		if resumeErr != "" {
			return resumeErr
		}
		return "resume failed"
	case "suspended":
		return "will auto-resume on startup"
	case "force-killed":
		return "session may be corrupt"
	default:
		return ""
	}
}
