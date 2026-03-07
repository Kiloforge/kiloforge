package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"crelay/internal/config"
	"crelay/internal/gitea"
	"crelay/internal/project"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [repo-path]",
	Short: "Register a project with the Gitea server",
	Long: `Registers a git repository with the global Gitea instance. Creates a Gitea
repo, adds a 'gitea' remote, pushes the main branch, and sets up a webhook.

If no path is given, uses the current working directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAdd,
}

var (
	flagAddName   string
	flagAddOrigin string
)

func init() {
	addCmd.Flags().StringVar(&flagAddName, "name", "", "Project slug (defaults to directory basename)")
	addCmd.Flags().StringVar(&flagAddOrigin, "origin", "", "Origin remote URL override")
}

func runAdd(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Resolve repo path.
	repoPath := "."
	if len(args) > 0 {
		repoPath = args[0]
	}
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Verify it's a git repo.
	gitDir := filepath.Join(absPath, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return fmt.Errorf("not a git repository: %s", absPath)
	}

	// Load global config, verify Gitea is initialized and running.
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'crelay init' first")
	}

	client := gitea.NewClient(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.GiteaAdminPass)
	if cfg.APIToken != "" {
		client.SetToken(cfg.APIToken)
	}
	if _, err := client.CheckVersion(ctx); err != nil {
		return fmt.Errorf("Gitea is not running — run 'crelay init' or 'crelay up' first")
	}

	// Derive slug.
	slug := filepath.Base(absPath)
	if flagAddName != "" {
		slug = flagAddName
	}

	// Load registry and check for duplicate.
	reg, err := project.LoadRegistry(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load project registry: %w", err)
	}
	if p, ok := reg.Get(slug); ok {
		fmt.Printf("Project %q is already registered.\n", slug)
		fmt.Printf("  Path:   %s\n", p.ProjectDir)
		fmt.Printf("  Gitea:  %s/%s/%s\n", cfg.GiteaURL(), cfg.GiteaAdminUser, p.RepoName)
		return nil
	}

	// Detect origin remote.
	originRemote := flagAddOrigin
	if originRemote == "" {
		originRemote = detectOriginRemote(ctx, absPath)
	}

	// Create Gitea repo.
	fmt.Printf("==> Creating Gitea repo '%s'...\n", slug)
	if err := client.CreateRepo(ctx, slug); err != nil {
		if !strings.Contains(err.Error(), "409") {
			return fmt.Errorf("create repo: %w", err)
		}
		fmt.Println("    Repo already exists in Gitea — continuing.")
	}

	// Add gitea remote.
	giteaRemoteURL := fmt.Sprintf("%s/%s/%s.git", cfg.GiteaURL(), cfg.GiteaAdminUser, slug)
	fmt.Println("==> Adding gitea remote...")
	_ = exec.CommandContext(ctx, "git", "-C", absPath, "remote", "remove", "gitea").Run()
	if err := exec.CommandContext(ctx, "git", "-C", absPath, "remote", "add", "gitea", giteaRemoteURL).Run(); err != nil {
		return fmt.Errorf("add gitea remote: %w", err)
	}
	fmt.Printf("    Remote: %s\n", giteaRemoteURL)

	// Push main branch.
	fmt.Println("==> Pushing to Gitea...")
	pushCmd := exec.CommandContext(ctx, "git", "-C", absPath, "push", "-u", "gitea", "main")
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("push to gitea: %w", err)
	}

	// Create webhook.
	fmt.Println("==> Registering webhook...")
	if err := client.CreateWebhook(ctx, slug, cfg.RelayPort); err != nil {
		fmt.Printf("    Warning: webhook creation failed: %v\n", err)
		fmt.Println("    (Webhook can be added later when the relay server is configured)")
	}

	// Create project data directory.
	if err := project.EnsureProjectDir(cfg.DataDir, slug); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}

	// Register in projects.json.
	p := project.Project{
		Slug:         slug,
		RepoName:     slug,
		ProjectDir:   absPath,
		OriginRemote: originRemote,
		RegisteredAt: time.Now().Truncate(time.Second),
		Active:       true,
	}
	if err := reg.Add(p); err != nil {
		return fmt.Errorf("register project: %w", err)
	}
	if err := reg.Save(cfg.DataDir); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	fmt.Println()
	fmt.Printf("Project '%s' registered!\n", slug)
	fmt.Printf("  Path:   %s\n", absPath)
	fmt.Printf("  Gitea:  %s/%s/%s\n", cfg.GiteaURL(), cfg.GiteaAdminUser, slug)
	if originRemote != "" {
		fmt.Printf("  Origin: %s\n", originRemote)
	}
	fmt.Println()
	fmt.Println("View registered projects with 'crelay projects'.")

	return nil
}

func detectOriginRemote(ctx context.Context, repoPath string) string {
	out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "remote", "get-url", "origin").Output()
	if err != nil {
		fmt.Println("    Warning: no 'origin' remote found — origin bridging won't be available")
		return ""
	}
	return strings.TrimSpace(string(out))
}
