package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var escalatedCmd = &cobra.Command{
	Use:   "escalated",
	Short: "Show PRs that hit the review cycle limit",
	Long:  `Lists PRs that have been escalated due to exceeding the maximum review cycle count. These PRs require human intervention.`,
	RunE:  runEscalated,
}

func runEscalated(cmd *cobra.Command, args []string) error {
	rt, err := NewCLIRuntime()
	if err != nil {
		return fmt.Errorf("%s", notInitializedError())
	}
	defer func() { _ = rt.Close() }()

	escalated := rt.Agents.GetEscalated()

	if len(escalated) == 0 {
		fmt.Println(emptyState("escalated PRs", "PRs appear here when they exceed the review cycle limit."))
		return nil
	}

	fmt.Printf("Escalated PRs (%d)\n\n", len(escalated))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROJECT\tPR#\tTRACK\tCYCLES")
	for _, e := range escalated {
		fmt.Fprintf(w, "%s\t#%d\t%s\t%d\n", e.Slug, e.PR, e.TrackID, e.Cycles)
	}
	return w.Flush()
}
