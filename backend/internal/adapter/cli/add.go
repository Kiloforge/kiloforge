package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"kiloforge/internal/adapter/auth"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/gitea"
	"kiloforge/internal/core/service"

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
  kf add git@github.com:user/my-project.git
  kf add https://github.com/user/my-project.git
  kf add git@github.com:user/my-project.git --name custom-name`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

var (
	flagAddName   string
	flagAddSSHKey string
)

func init() {
	addCmd.Flags().StringVar(&flagAddName, "name", "", "Project slug (defaults to repo name from URL)")
	addCmd.Flags().StringVar(&flagAddSSHKey, "ssh-key", "", "Path to SSH private key for this project's git operations")
}

func runAdd(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	remoteURL := args[0]

	// Validate it looks like a remote URL.
	if !isRemoteURL(remoteURL) {
		return fmt.Errorf("not a remote URL: %s\n\nUsage: kf add <remote-url>\nExample: kf add git@github.com:user/repo.git", remoteURL)
	}

	// Derive repo name from URL.
	repoName, err := repoNameFromURL(remoteURL)
	if err != nil {
		return fmt.Errorf("parse remote URL: %w", err)
	}
	// Slug defaults to repo name; --name overrides the slug only.
	slug := repoName
	if flagAddName != "" {
		slug = flagAddName
	}

	// Resolve SSH key path.
	var sshKeyPath string
	if flagAddSSHKey != "" {
		// Explicit --ssh-key flag: use as-is.
		sshKeyPath, err = expandPath(flagAddSSHKey)
		if err != nil {
			return fmt.Errorf("resolve SSH key path: %w", err)
		}
		if _, err := os.Stat(sshKeyPath); err != nil {
			return fmt.Errorf("SSH key not found: %s", sshKeyPath)
		}
	} else if isSSHRemote(remoteURL) {
		// SSH remote without --ssh-key: discover and prompt.
		sshKeyPath = discoverAndSelectSSHKey()
	}
	// Load global config, verify Gitea is initialized and running.
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'kf init' first")
	}

	client := gitea.NewClientWithToken(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.APIToken)
	if _, err := client.CheckVersion(ctx); err != nil {
		return fmt.Errorf("Gitea is not running — run 'kf init' or 'kf up' first")
	}

	// Open database and wire up project service.
	rt, err := NewCLIRuntimeFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer rt.Close()

	// Build a project service with the real Gitea client for add operations.
	projectSvc := service.NewProjectService(
		rt.Projects.Store(),
		client,
		service.ProjectServiceConfig{
			DataDir:          cfg.DataDir,
			OrchestratorPort: cfg.OrchestratorPort,
			GiteaAdminUser:   cfg.GiteaAdminUser,
			APIToken:         cfg.APIToken,
		},
	)

	if p, err := rt.Projects.GetProject(slug); err == nil {
		fmt.Printf("Project %q is already registered.\n", slug)
		fmt.Printf("  Path:   %s\n", p.ProjectDir)
		fmt.Printf("  Gitea:  %s/%s/%s\n", cfg.GiteaURL(), cfg.GiteaAdminUser, p.RepoName)
		return nil
	}

	fmt.Printf("==> Adding project %q from %s...\n", slug, remoteURL)
	result, err := projectSvc.AddProject(ctx, remoteURL, flagAddName, service.AddProjectOpts{
		SSHKeyPath: sshKeyPath,
	})
	if err != nil {
		return fmt.Errorf("add project: %w", err)
	}

	if result.EmptyRepo {
		fmt.Println("==> Repository has no commits — skipping push to Gitea.")
		fmt.Println("    Push commits to the repository and run 'kf sync' to update Gitea.")
	}

	// Register SSH public key with Gitea if --ssh-key was given.
	if sshKeyPath != "" {
		pubKeyPath := sshKeyPath + ".pub"
		if pubData, err := os.ReadFile(pubKeyPath); err == nil {
			fmt.Println("==> Registering SSH public key with Gitea...")
			keyTitle := fmt.Sprintf("kf-%s", slug)
			if err := client.AddSSHKey(ctx, keyTitle, strings.TrimSpace(string(pubData))); err != nil {
				fmt.Printf("    Warning: SSH key registration failed: %v\n", err)
			}
		} else {
			fmt.Printf("    Warning: public key not found at %s — skipping Gitea registration\n", pubKeyPath)
		}
	}

	p := result.Project
	fmt.Println()
	fmt.Printf("Project '%s' registered!\n", p.Slug)
	fmt.Printf("  Path:   %s\n", p.ProjectDir)
	fmt.Printf("  Gitea:  %s/%s/%s\n", cfg.GiteaURL(), cfg.GiteaAdminUser, p.RepoName)
	fmt.Printf("  Origin: %s\n", p.OriginRemote)
	fmt.Println()
	fmt.Println("View registered projects with 'kf projects'.")

	return nil
}

// expandPath expands a leading ~/ to the user's home directory.
func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
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

// isSSHRemote returns true if the URL uses SSH protocol.
func isSSHRemote(url string) bool {
	if strings.HasPrefix(url, "ssh://") {
		return true
	}
	// SSH shorthand: git@host:path
	if strings.Contains(url, "@") && strings.Contains(url, ":") {
		return true
	}
	return false
}

// discoverAndSelectSSHKey discovers SSH keys in ~/.ssh/ and prompts the user
// to select one. Returns the selected private key path, or "" if none selected.
func discoverAndSelectSSHKey() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	sshDir := filepath.Join(home, ".ssh")
	keys := auth.DiscoverSSHKeys(sshDir)
	if len(keys) == 0 {
		fmt.Println("    No SSH keys found in ~/.ssh/ — using default SSH agent")
		return ""
	}

	// Detect non-interactive stdin.
	var reader *os.File
	fi, err := os.Stdin.Stat()
	if err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		reader = os.Stdin
	}

	path, err := PromptSSHKeySelection(keys, reader, os.Stdout)
	if err != nil {
		return ""
	}
	return path
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
