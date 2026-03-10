package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/pool"

	"github.com/spf13/cobra"
)

var poolCmd = &cobra.Command{
	Use:   "pool",
	Short: "Show worktree pool status",
	Long:  `Displays the status of all worktrees in the pool, including which are idle and which are in use by developer agents.`,
	RunE:  runPool,
}

func runPool(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'kf init' first")
	}

	p, err := pool.Load(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load pool: %w", err)
	}

	statuses := p.Status()
	if len(statuses) == 0 {
		fmt.Println("No worktrees in pool.")
		fmt.Printf("Pool max size: %d\n", p.MaxSize)
		fmt.Println("\nWorktrees are created automatically when needed by 'kf implement'.")
		return nil
	}

	fmt.Printf("Worktree Pool (%d/%d)\n", len(statuses), p.MaxSize)
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tTRACK\tAGENT\tACQUIRED")
	for _, wt := range statuses {
		track := "-"
		if wt.TrackID != "" {
			track = wt.TrackID
		}
		agent := "-"
		if wt.AgentID != "" {
			agent = wt.AgentID
		}
		acquired := "-"
		if wt.AcquiredAt != nil {
			acquired = wt.AcquiredAt.Format("2006-01-02 15:04:05")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", wt.Name, wt.Status, track, agent, acquired)
	}
	w.Flush()

	return nil
}
