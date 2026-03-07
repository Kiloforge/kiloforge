package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"crelay/internal/adapter/config"
	"crelay/internal/adapter/persistence/jsonfile"
	"crelay/internal/core/domain"

	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push [slug]",
	Short: "Push changes from internal clone to origin remote",
	Long: `Push the main branch (or a specific branch) from the internal clone back to the
project's origin remote (GitHub, GitLab, etc.), using the stored SSH identity if configured.

Examples:
  crelay push myapp                       # push main branch to origin
  crelay push myapp --branch feature-x    # push specific branch
  crelay push --all                       # push all projects' main branches`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPush,
}

var (
	flagPushBranch string
	flagPushAll    bool
)

func init() {
	pushCmd.Flags().StringVar(&flagPushBranch, "branch", "main", "Branch to push")
	pushCmd.Flags().BoolVar(&flagPushAll, "all", false, "Push all projects' main branches")
}

func runPush(cmd *cobra.Command, args []string) error {
	if !flagPushAll && len(args) == 0 {
		return fmt.Errorf("project slug required (or use --all)\n\nUsage: crelay push <slug> [--branch <name>]\n       crelay push --all")
	}
	if flagPushAll && len(args) > 0 {
		return fmt.Errorf("cannot use --all with a specific project slug")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'crelay init' first")
	}

	reg, err := jsonfile.LoadProjectStore(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load project registry: %w", err)
	}

	if flagPushAll {
		return pushAll(ctx, reg.List())
	}

	project, ok := reg.Get(args[0])
	if !ok {
		return fmt.Errorf("project %q not found", args[0])
	}

	return pushProject(ctx, project, flagPushBranch)
}

func pushAll(ctx context.Context, projects []domain.Project) error {
	if len(projects) == 0 {
		fmt.Println("No projects registered.")
		return nil
	}

	var failures []string
	for _, p := range projects {
		if !p.Active {
			continue
		}
		fmt.Printf("==> Pushing %s (main → origin)...\n", p.Slug)
		if err := pushProject(ctx, p, "main"); err != nil {
			fmt.Printf("    FAILED: %v\n", err)
			failures = append(failures, p.Slug)
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("push failed for: %s", strings.Join(failures, ", "))
	}
	return nil
}

func pushProject(ctx context.Context, p domain.Project, branch string) error {
	if p.OriginRemote == "" {
		return fmt.Errorf("no origin remote configured for project %q", p.Slug)
	}

	// Fetch origin to check ahead/behind status.
	fetchCmd := gitCmd(ctx, p, "fetch", "origin", branch)
	if out, err := fetchCmd.CombinedOutput(); err != nil {
		fmt.Printf("    Warning: fetch failed: %s\n", strings.TrimSpace(string(out)))
	} else {
		// Check ahead/behind.
		revList := gitCmd(ctx, p, "rev-list", "--left-right", "--count",
			fmt.Sprintf("origin/%s...%s", branch, branch))
		if out, err := revList.Output(); err == nil {
			parts := strings.Fields(strings.TrimSpace(string(out)))
			if len(parts) == 2 {
				behind, ahead := parts[0], parts[1]
				if behind != "0" || ahead != "0" {
					fmt.Printf("    Status: %s ahead, %s behind origin/%s\n", ahead, behind, branch)
				}
				if behind != "0" {
					fmt.Printf("    Warning: origin has new commits — push may fail if branches diverge\n")
				}
			}
		}
	}

	// Push.
	pushCmd := gitCmd(ctx, p, "push", "origin", branch)
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr
	if err := pushCmd.Run(); err != nil {
		if strings.Contains(err.Error(), "non-fast-forward") {
			return fmt.Errorf("origin has diverged — pull and resolve conflicts first")
		}
		return fmt.Errorf("push failed: %w", err)
	}

	fmt.Printf("    Pushed %s → origin/%s\n", branch, branch)
	return nil
}

// gitCmd creates a git command for the project's clone directory with SSH env if configured.
func gitCmd(ctx context.Context, p domain.Project, args ...string) *exec.Cmd {
	fullArgs := append([]string{"-C", p.ProjectDir}, args...)
	cmd := exec.CommandContext(ctx, "git", fullArgs...)
	if sshEnv := p.GitSSHEnv(); len(sshEnv) > 0 {
		cmd.Env = append(os.Environ(), sshEnv...)
	}
	return cmd
}
