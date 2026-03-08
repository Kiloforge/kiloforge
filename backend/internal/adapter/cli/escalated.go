package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/persistence/jsonfile"

	"github.com/spf13/cobra"
)

var escalatedCmd = &cobra.Command{
	Use:   "escalated",
	Short: "Show PRs that hit the review cycle limit",
	Long:  `Lists PRs that have been escalated due to exceeding the maximum review cycle count. These PRs require human intervention.`,
	RunE:  runEscalated,
}

func runEscalated(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'kf init' first")
	}

	reg, err := jsonfile.LoadProjectStore(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	type escalatedPR struct {
		slug    string
		pr      int
		trackID string
		cycles  int
	}

	var escalated []escalatedPR

	for _, proj := range reg.List() {
		projectDir := filepath.Join(cfg.DataDir, "projects", proj.Slug)
		tracking, err := jsonfile.LoadPRTracking(projectDir)
		if err != nil {
			continue
		}
		if tracking.Status == "escalated" {
			escalated = append(escalated, escalatedPR{
				slug:    proj.Slug,
				pr:      tracking.PRNumber,
				trackID: tracking.TrackID,
				cycles:  tracking.ReviewCycleCount,
			})
		}
	}

	if len(escalated) == 0 {
		fmt.Println("No escalated PRs.")
		return nil
	}

	fmt.Printf("Escalated PRs (%d)\n\n", len(escalated))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROJECT\tPR#\tTRACK\tCYCLES")
	for _, e := range escalated {
		fmt.Fprintf(w, "%s\t#%d\t%s\t%d\n", e.slug, e.pr, e.trackID, e.cycles)
	}
	return w.Flush()
}
