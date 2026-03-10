package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/service"

	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push [slug]",
	Short: "Push changes from internal clone to origin remote",
	Long: `Push the main branch (or a specific branch) from the internal clone back to the
project's origin remote (GitHub, GitLab, etc.), using the stored SSH identity if configured.

Examples:
  kf push myapp                       # push main branch to origin
  kf push myapp --branch feature-x    # push specific branch
  kf push --all                       # push all projects' main branches`,
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
		return fmt.Errorf("project slug required (or use --all)\n\nUsage: kf push <slug> [--branch <name>]\n       kf push --all")
	}
	if flagPushAll && len(args) > 0 {
		return fmt.Errorf("cannot use --all with a specific project slug")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	rt, err := NewCLIRuntime()
	if err != nil {
		return fmt.Errorf("%s", notInitializedError())
	}
	defer func() { _ = rt.Close() }()

	syncSvc := service.NewGitSyncService(&execGitRunner{})

	if flagPushAll {
		return pushAll(ctx, rt.Projects.ListProjects(), syncSvc)
	}

	project, err := rt.Projects.GetProject(args[0])
	if err != nil {
		return fmt.Errorf("project %q not found", args[0])
	}

	return pushProject(ctx, *project, flagPushBranch, syncSvc)
}

func pushAll(ctx context.Context, projects []domain.Project, syncSvc *service.GitSyncService) error {
	if len(projects) == 0 {
		fmt.Println(emptyState("projects registered", "Register a project with: kf add <remote-url>"))
		return nil
	}

	var failures []string
	for _, p := range projects {
		if !p.Active {
			continue
		}
		fmt.Printf("==> Pushing %s (main → origin)...\n", p.Slug)
		if err := pushProject(ctx, p, "main", syncSvc); err != nil {
			fmt.Printf("    FAILED: %v\n", err)
			failures = append(failures, p.Slug)
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("push failed for: %s", strings.Join(failures, ", "))
	}
	return nil
}

func pushProject(ctx context.Context, p domain.Project, branch string, syncSvc *service.GitSyncService) error {
	if p.OriginRemote == "" {
		return fmt.Errorf("no origin remote configured for project %q", p.Slug)
	}

	sshEnv := p.GitSSHEnv()

	// Check sync status via service.
	status, err := syncSvc.CheckSyncStatus(ctx, p.ProjectDir, sshEnv, branch)
	if err != nil {
		fmt.Printf("    Warning: %v\n", err)
	} else if status.Behind != 0 || status.Ahead != 0 {
		fmt.Printf("    Status: %d ahead, %d behind origin/%s\n", status.Ahead, status.Behind, branch)
		if status.Behind != 0 {
			fmt.Printf("    Warning: origin has new commits — push may fail if branches diverge\n")
		}
	}

	// Push via service.
	if err := syncSvc.PushBranch(ctx, p.ProjectDir, sshEnv, branch); err != nil {
		return err
	}

	fmt.Printf("    Pushed %s → origin/%s\n", branch, branch)
	return nil
}

// execGitRunner implements service.GitCommandRunner using exec.Command.
type execGitRunner struct{}

func (r *execGitRunner) RunGitCommand(ctx context.Context, dir string, sshEnv []string, args ...string) (string, error) {
	fullArgs := append([]string{"-C", dir}, args...)
	cmd := exec.CommandContext(ctx, "git", fullArgs...)
	if len(sshEnv) > 0 {
		cmd.Env = append(os.Environ(), sshEnv...)
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}
