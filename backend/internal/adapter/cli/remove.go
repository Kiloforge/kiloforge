package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <slug>",
	Short: "Remove a registered project",
	Long: `Deregisters a project from kiloforge.

By default only removes the project registration. Use --cleanup to also
delete local filesystem data (repos/, projects/, and internal mirror directories).

You will be prompted for confirmation unless --force is used.`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

var (
	flagRemoveCleanup bool
	flagRemoveForce   bool
)

func init() {
	removeCmd.Flags().BoolVar(&flagRemoveCleanup, "cleanup", false, "Also delete local filesystem data (repos, projects, mirrors)")
	removeCmd.Flags().BoolVar(&flagRemoveForce, "force", false, "Skip confirmation prompt")
}

func runRemove(cmd *cobra.Command, args []string) error {
	slug := args[0]

	rt, err := NewCLIRuntime()
	if err != nil {
		return fmt.Errorf("%s", notInitializedError())
	}
	defer func() { _ = rt.Close() }()

	// Verify the project exists before prompting.
	projects := rt.Projects.ListProjects()
	var found bool
	for _, p := range projects {
		if p.Slug == slug {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("project %q not found", slug)
	}

	if !flagRemoveForce {
		fmt.Println()
		fmt.Printf("  WARNING: This will remove project %q.\n", slug)
		if flagRemoveCleanup {
			fmt.Println("  The --cleanup flag is set — local data will also be deleted:")
			fmt.Println("    - repos/<slug>/")
			fmt.Println("    - projects/<slug>/")
			fmt.Println("    - Internal mirror directories")
		}
		fmt.Println()
		fmt.Print("  Type \"yes\" to confirm: ")

		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := rt.Projects.RemoveProject(cmd.Context(), slug, flagRemoveCleanup); err != nil {
		return fmt.Errorf("remove project: %w", err)
	}

	fmt.Printf("Project %q removed.\n", slug)
	return nil
}
