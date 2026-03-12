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
	Use:   "add <remote-url-or-local-path>",
	Short: "Clone a repo and register it as a project",
	Long: `Clones a git remote URL or local repo into a managed directory and registers it
as a kiloforge project.

The repo name is derived from the URL or directory name.
Use --name to override the derived name.

Examples:
  kf add git@github.com:user/my-project.git
  kf add https://github.com/user/my-project.git
  kf add /path/to/local/repo
  kf add ./relative/path
  kf add ~/my-projects/repo
  kf add git@github.com:user/my-project.git --name custom-name`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

var (
	flagAddName   string
	flagAddSSHKey string
	flagAddOutput string
)

func init() {
	addCmd.Flags().StringVar(&flagAddName, "name", "", "Project slug (defaults to repo name from URL)")
	addCmd.Flags().StringVar(&flagAddSSHKey, "ssh-key", "", "Path to SSH private key for this project's git operations")
	addCmd.Flags().StringVar(&flagAddOutput, "output", "", "Directory for the browseable mirror clone (defaults to ~/.kiloforge/output/{slug}/)")
}

func runAdd(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	arg := args[0]

	// Load global config.
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("%s", notInitializedError())
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

	// Resolve --output path.
	var outputDir string
	if flagAddOutput != "" {
		outputDir, err = expandPath(flagAddOutput)
		if err != nil {
			return fmt.Errorf("resolve output path: %w", err)
		}
		outputDir, err = filepath.Abs(outputDir)
		if err != nil {
			return fmt.Errorf("resolve output path: %w", err)
		}
		// If path exists, it must be either a git repo or an empty directory.
		if info, statErr := os.Stat(outputDir); statErr == nil {
			if !info.IsDir() {
				return fmt.Errorf("--output path exists but is not a directory: %s", outputDir)
			}
			entries, _ := os.ReadDir(outputDir)
			if len(entries) > 0 {
				// Non-empty — must be a git repo.
				if _, gitErr := os.Stat(filepath.Join(outputDir, ".git")); gitErr != nil {
					return fmt.Errorf("--output path exists and is not a git repo: %s", outputDir)
				}
			}
		}
	}

	// Determine if this is a remote URL or local path.
	if isRemoteURL(arg) {
		return addFromRemote(ctx, arg, projectSvc, rt, outputDir)
	}
	return addFromLocal(ctx, arg, projectSvc, rt, outputDir)
}

func addFromRemote(ctx context.Context, remoteURL string, projectSvc *service.ProjectService, rt *CLIRuntime, outputDir string) error {
	repoName, err := repoNameFromURL(remoteURL)
	if err != nil {
		return fmt.Errorf("parse remote URL: %w", err)
	}
	slug := repoName
	if flagAddName != "" {
		slug = flagAddName
	}

	// Resolve SSH key path.
	var sshKeyPath string
	if flagAddSSHKey != "" {
		sshKeyPath, err = expandPath(flagAddSSHKey)
		if err != nil {
			return fmt.Errorf("resolve SSH key path: %w", err)
		}
		if _, err := os.Stat(sshKeyPath); err != nil {
			return fmt.Errorf("SSH key not found: %s", sshKeyPath)
		}
	} else if isSSHRemote(remoteURL) {
		sshKeyPath = discoverAndSelectSSHKey()
	}

	if p, err := rt.Projects.GetProject(slug); err == nil {
		fmt.Printf("Project %q is already registered.\n", slug)
		fmt.Printf("  Path:   %s\n", p.ProjectDir)
		return nil
	}

	fmt.Printf("==> Adding project %q from %s...\n", slug, remoteURL)
	result, err := projectSvc.AddProject(ctx, remoteURL, flagAddName, domain.AddProjectOpts{
		SSHKeyPath: sshKeyPath,
		OutputDir:  outputDir,
	})
	if err != nil {
		return fmt.Errorf("add project: %w", err)
	}

	return printAddResult(result)
}

func addFromLocal(ctx context.Context, localPath string, projectSvc *service.ProjectService, rt *CLIRuntime, outputDir string) error {
	// Expand ~ and resolve to absolute path.
	resolved, err := expandPath(localPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	slug := filepath.Base(resolved)
	if flagAddName != "" {
		slug = flagAddName
	}

	if p, err := rt.Projects.GetProject(slug); err == nil {
		fmt.Printf("Project %q is already registered.\n", slug)
		fmt.Printf("  Path:   %s\n", p.ProjectDir)
		return nil
	}

	fmt.Printf("==> Adding project %q from %s...\n", slug, resolved)
	result, err := projectSvc.AddLocalProject(ctx, resolved, flagAddName, domain.AddProjectOpts{
		OutputDir: outputDir,
	})
	if err != nil {
		return fmt.Errorf("add project: %w", err)
	}

	return printAddResult(result)
}

func printAddResult(result *domain.AddProjectResult) error {
	if result.EmptyRepo {
		fmt.Println("==> Repository has no commits — push commits to get started.")
	}

	p := result.Project
	fmt.Println()
	fmt.Printf("Project '%s' registered!\n", p.Slug)
	fmt.Printf("  Path:   %s\n", p.ProjectDir)
	fmt.Printf("  Mirror: %s\n", p.MirrorDir)
	if p.OriginRemote != "" {
		fmt.Printf("  Origin: %s\n", p.OriginRemote)
	}
	if p.PrimaryBranch != "" {
		fmt.Printf("  Branch: %s\n", p.PrimaryBranch)
	}
	fmt.Println()

	// Install embedded skills locally into the project.
	fmt.Println("==> Transforming your agent into a high-productivity track-slinging machine...")
	installed, err := installLocalSkills(p.ProjectDir)
	if err != nil {
		fmt.Printf("    Warning: local skills installation failed: %v\n", err)
	} else if len(installed) == 0 {
		fmt.Println("    Skills already up to date")
	} else {
		fmt.Printf("    Installed %d skill(s) to %s/.claude/skills/\n", len(installed), p.ProjectDir)
	}

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
