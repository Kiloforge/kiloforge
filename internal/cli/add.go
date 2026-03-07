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
	"crelay/internal/core/domain"
	"crelay/internal/gitea"
	"crelay/internal/project"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <remote-url>",
	Short: "Clone a remote repo and register it with the Gitea server",
	Long: `Clones a git remote URL into a managed directory and registers it with the
global Gitea instance. Creates a Gitea repo, adds a 'gitea' remote, pushes
the main branch, and sets up a webhook.

The repo name is derived from the remote URL (e.g., git@github.com:user/repo.git → repo).
Use --name to override the derived name.

Examples:
  crelay add git@github.com:user/my-project.git
  crelay add https://github.com/user/my-project.git
  crelay add git@github.com:user/my-project.git --name custom-name`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

var flagAddName string

func init() {
	addCmd.Flags().StringVar(&flagAddName, "name", "", "Project slug (defaults to repo name from URL)")
}

func runAdd(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	remoteURL := args[0]

	// Validate it looks like a remote URL.
	if !isRemoteURL(remoteURL) {
		return fmt.Errorf("not a remote URL: %s\n\nUsage: crelay add <remote-url>\nExample: crelay add git@github.com:user/repo.git", remoteURL)
	}

	// Derive slug from URL (or --name flag).
	slug, err := repoNameFromURL(remoteURL)
	if err != nil {
		return fmt.Errorf("parse remote URL: %w", err)
	}
	if flagAddName != "" {
		slug = flagAddName
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

	// Clone remote into managed directory.
	cloneDir := filepath.Join(cfg.DataDir, "repos", slug)
	if _, err := os.Stat(cloneDir); os.IsNotExist(err) {
		fmt.Printf("==> Cloning %s...\n", remoteURL)
		if err := cloneRepo(ctx, remoteURL, cloneDir); err != nil {
			return fmt.Errorf("clone: %w", err)
		}
	} else {
		fmt.Printf("==> Clone directory already exists: %s\n", cloneDir)
	}

	// Create Gitea repo.
	fmt.Printf("==> Creating Gitea repo '%s'...\n", slug)
	if err := client.CreateRepo(ctx, slug); err != nil {
		if !strings.Contains(err.Error(), "409") {
			return fmt.Errorf("create repo: %w", err)
		}
		fmt.Println("    Repo already exists in Gitea — continuing.")
	}

	// Add gitea remote to cloned repo.
	giteaRemoteURL := fmt.Sprintf("%s/%s/%s.git", cfg.GiteaURL(), cfg.GiteaAdminUser, slug)
	fmt.Println("==> Adding gitea remote...")
	_ = exec.CommandContext(ctx, "git", "-C", cloneDir, "remote", "remove", "gitea").Run()
	if err := exec.CommandContext(ctx, "git", "-C", cloneDir, "remote", "add", "gitea", giteaRemoteURL).Run(); err != nil {
		return fmt.Errorf("add gitea remote: %w", err)
	}
	fmt.Printf("    Remote: %s\n", giteaRemoteURL)

	// Push main branch.
	fmt.Println("==> Pushing to Gitea...")
	pushCmd := exec.CommandContext(ctx, "git", "-C", cloneDir, "push", "-u", "gitea", "main")
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
	p := domain.Project{
		Slug:         slug,
		RepoName:     slug,
		ProjectDir:   cloneDir,
		OriginRemote: remoteURL,
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
	fmt.Printf("  Path:   %s\n", cloneDir)
	fmt.Printf("  Gitea:  %s/%s/%s\n", cfg.GiteaURL(), cfg.GiteaAdminUser, slug)
	fmt.Printf("  Origin: %s\n", remoteURL)
	fmt.Println()
	fmt.Println("View registered projects with 'crelay projects'.")

	return nil
}

// cloneRepo clones a remote git repository into destDir.
func cloneRepo(ctx context.Context, remoteURL, destDir string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", remoteURL, destDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// isRemoteURL returns true if the argument looks like a git remote URL
// (SSH or HTTP(S)), as opposed to a local path.
func isRemoteURL(arg string) bool {
	if strings.HasPrefix(arg, "https://") || strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "ssh://") {
		return true
	}
	// SSH shorthand: git@host:path
	if strings.Contains(arg, "@") && strings.Contains(arg, ":") {
		return true
	}
	return false
}

// repoNameFromURL extracts the repository name from a git remote URL.
// Handles SSH (git@host:user/repo.git) and HTTPS (https://host/user/repo.git) formats.
func repoNameFromURL(rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("empty URL")
	}

	var path string

	// SSH shorthand: git@host:user/repo.git
	if idx := strings.Index(rawURL, ":"); idx != -1 && strings.Contains(rawURL[:idx], "@") && !strings.HasPrefix(rawURL, "ssh://") {
		path = rawURL[idx+1:]
	} else {
		// HTTPS or ssh:// URL — take path after host
		// Strip scheme
		u := rawURL
		for _, prefix := range []string{"https://", "http://", "ssh://"} {
			if strings.HasPrefix(u, prefix) {
				u = u[len(prefix):]
				break
			}
		}
		// Strip user@host
		if idx := strings.Index(u, "/"); idx != -1 {
			path = u[idx+1:]
		}
	}

	if path == "" {
		return "", fmt.Errorf("cannot extract repo name from URL: %s", rawURL)
	}

	// Strip trailing slashes
	path = strings.TrimRight(path, "/")
	// Take the last path component
	name := filepath.Base(path)
	// Strip .git suffix
	name = strings.TrimSuffix(name, ".git")

	if name == "" || name == "." {
		return "", fmt.Errorf("cannot extract repo name from URL: %s", rawURL)
	}

	return name, nil
}
