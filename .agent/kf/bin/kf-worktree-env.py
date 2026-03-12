#!/Users/benbaldivia/dev-2/kiloforge-wt/kfc-d780f5-worker-1/.agent/kf/.venv/bin/python
# kf-worktree-env — Detect git worktree context and print env vars.
#
# Usage:
#   eval "$(kf-worktree-env)"              (export env vars into current shell)
#   kf-worktree-env                        (print VAR=value lines)
#   kf-worktree-env --help                 (show help)
#   kf-worktree-env --show-branches        (list worktrees and branches)
#
# Printed variables:
#   GIT_DIR          — Path to the shared .git directory
#   GIT_WORK_TREE    — Path to the worktree root
#   KF_IS_WORKTREE   — 1 if in a worktree, 0 if in a normal repo
#   KF_WORKTREE_NAME — Basename of the current directory (e.g., "developer-1")
#   KF_MAIN_WORKTREE — Path to the main worktree

import os
import subprocess
import sys


def git(*args: str) -> str:
    """Run a git command and return stripped stdout, or raise on failure."""
    result = subprocess.run(
        ["git", *args],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise subprocess.CalledProcessError(
            result.returncode, ["git", *args], result.stdout, result.stderr
        )
    return result.stdout.strip()


HELP_TEXT = """\
kf-worktree-env — Detect git worktree context and print env vars.

Usage:
  eval "$(kf-worktree-env)"              Export env vars into current shell
  kf-worktree-env                        Print worktree info as VAR=value
  kf-worktree-env --show-branches        List all worktrees and their branches
  kf-worktree-env --help                 Show this help

Printed variables:
  GIT_DIR            Path to the shared .git directory (git-common-dir)
  GIT_WORK_TREE      Path to the worktree root (show-toplevel)
  KF_IS_WORKTREE     1 if running in a worktree, 0 if normal repo
  KF_WORKTREE_NAME   Basename of the current directory
  KF_MAIN_WORKTREE   Path to the main worktree

Why this matters:
  Git worktrees have a .git file (not directory), which breaks Go's VCS
  stamping. Exporting GIT_DIR and GIT_WORK_TREE fixes this."""


def main() -> int:
    show_branches = "--show-branches" in sys.argv[1:]
    show_help = "--help" in sys.argv[1:] or "-h" in sys.argv[1:]

    if show_help:
        print(HELP_TEXT)
        return 0

    # Detect git repo
    try:
        git_toplevel = git("rev-parse", "--show-toplevel")
    except subprocess.CalledProcessError:
        print("kf-worktree-env: not in a git repository", file=sys.stderr)
        return 1

    # Get the shared .git directory (resolved to absolute path)
    git_common_dir_raw = subprocess.run(
        ["git", "rev-parse", "--git-common-dir"],
        capture_output=True,
        text=True,
        cwd=git_toplevel,
    ).stdout.strip()
    git_common_dir = os.path.realpath(os.path.join(git_toplevel, git_common_dir_raw))

    # Get main worktree path (first line of porcelain output)
    worktree_porcelain = git("worktree", "list", "--porcelain")
    main_worktree = ""
    for line in worktree_porcelain.splitlines():
        if line.startswith("worktree "):
            main_worktree = line[len("worktree "):]
            break

    # In a worktree, .git is a file; in the main repo, .git is a directory
    is_worktree = 1 if os.path.isfile(os.path.join(git_toplevel, ".git")) else 0

    worktree_name = os.path.basename(git_toplevel)

    quiet = os.environ.get("KF_QUIET") == "1"

    if not quiet:
        if is_worktree:
            print("# kf-worktree-env: worktree detected", file=sys.stderr)
        else:
            print("# kf-worktree-env: normal repo (not a worktree)", file=sys.stderr)

    # Always print VAR=value lines to stdout for eval consumption
    print(f"GIT_DIR={git_common_dir}")
    print(f"GIT_WORK_TREE={git_toplevel}")
    print(f"KF_IS_WORKTREE={is_worktree}")
    print(f"KF_WORKTREE_NAME={worktree_name}")
    print(f"KF_MAIN_WORKTREE={main_worktree}")

    if show_branches:
        print("", file=sys.stderr)
        print("Worktrees and branches:", file=sys.stderr)
        subprocess.run(["git", "worktree", "list"])

    return 0


if __name__ == "__main__":
    sys.exit(main())
