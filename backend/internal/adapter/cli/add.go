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
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/service"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <remote-url>",
	Short: "Clone a remote repo and register it as a project",
	Long: `Clones a git remote URL into a managed directory and registers it
as a kiloforge project.

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
	// Load global config.
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'kf init' first")
	}

	// Open database and wire up project service.
	rt, err := NewCLIRuntimeFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = rt.Close() }()

	projectSvc := service.NewProjectService(
		rt.Projects.Store(),
		service.ProjectServiceConfig{
			DataDir:          cfg.DataDir,
			OrchestratorPort: cfg.OrchestratorPort,
		},
	)

	if p, err := rt.Projects.GetProject(slug); err == nil {
		fmt.Printf("Project %q is already registered.\n", slug)
		fmt.Printf("  Path:   %s\n", p.ProjectDir)
		return nil
	}

	fmt.Printf("==> Adding project %q from %s...\n", slug, remoteURL)
	result, err := projectSvc.AddProject(ctx, remoteURL, flagAddName, domain.AddProjectOpts{
		SSHKeyPath: sshKeyPath,
	})
	if err != nil {
		return fmt.Errorf("add project: %w", err)
	}

	if result.EmptyRepo {
		fmt.Println("==> Repository has no commits — push commits to get started.")
	}

	p := result.Project
	fmt.Println()
	fmt.Printf("Project '%s' registered!\n", p.Slug)
	fmt.Printf("  Path:   %s\n", p.ProjectDir)
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
