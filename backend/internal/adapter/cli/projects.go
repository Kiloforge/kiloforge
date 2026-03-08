package cli

import (
	"fmt"
	"text/tabwriter"
	"os"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/persistence/jsonfile"

	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List registered projects",
	RunE:  runProjects,
}

func runProjects(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'crelay init' first")
	}

	reg, err := jsonfile.LoadProjectStore(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load project registry: %w", err)
	}

	projects := reg.List()
	if len(projects) == 0 {
		fmt.Println("No projects registered.")
		fmt.Println()
		fmt.Println("Register a project with: crelay add <repo-path>")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SLUG\tPATH\tORIGIN\tSSH KEY\tREGISTERED\tACTIVE")
	for _, p := range projects {
		active := "yes"
		if !p.Active {
			active = "no"
		}
		origin := p.OriginRemote
		if origin == "" {
			origin = "(none)"
		}
		sshKey := "(default)"
		if p.SSHKeyPath != "" {
			sshKey = p.SSHKeyPath
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			p.Slug, p.ProjectDir, origin, sshKey,
			p.RegisteredAt.Format("2006-01-02"), active)
	}
	w.Flush()

	return nil
}
