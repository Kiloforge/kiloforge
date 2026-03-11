"""Git helpers for Kiloforge CLI tools."""

import os
import subprocess
from pathlib import Path
from typing import Optional


def run(
    *args: str,
    cwd: Optional[str] = None,
    check: bool = True,
    capture: bool = True,
) -> subprocess.CompletedProcess:
    """Run a git command and return the result."""
    return subprocess.run(
        ["git", *args],
        cwd=cwd,
        check=check,
        capture_output=capture,
        text=True,
    )


def show(ref: str, path: str) -> Optional[str]:
    """Read a file from a git ref. Returns None if not found."""
    result = run("show", f"{ref}:{path}", check=False)
    if result.returncode != 0:
        return None
    return result.stdout


def current_branch() -> Optional[str]:
    result = run("branch", "--show-current", check=False)
    return result.stdout.strip() if result.returncode == 0 else None


def git_common_dir() -> Optional[str]:
    result = run("rev-parse", "--git-common-dir", check=False)
    if result.returncode != 0:
        return None
    return os.path.realpath(result.stdout.strip())


def toplevel() -> Optional[str]:
    result = run("rev-parse", "--show-toplevel", check=False)
    return result.stdout.strip() if result.returncode == 0 else None


def worktree_list() -> list[dict]:
    """Return list of worktrees with path, commit, and branch."""
    result = run("worktree", "list", "--porcelain", check=False)
    if result.returncode != 0:
        return []
    worktrees = []
    current = {}
    for line in result.stdout.splitlines():
        if line.startswith("worktree "):
            if current:
                worktrees.append(current)
            current = {"path": line[9:]}
        elif line.startswith("HEAD "):
            current["head"] = line[5:]
        elif line.startswith("branch "):
            current["branch"] = line[7:].replace("refs/heads/", "")
        elif line == "bare":
            current["bare"] = True
        elif line == "detached":
            current["detached"] = True
    if current:
        worktrees.append(current)
    return worktrees


def main_worktree() -> Optional[str]:
    """Return the path of the main (first) worktree."""
    wts = worktree_list()
    return wts[0]["path"] if wts else None


def find_worktree_for_branch(branch: str) -> Optional[str]:
    """Find the worktree path that has the given branch checked out."""
    result = run("worktree", "list", check=False)
    if result.returncode != 0:
        return None
    for line in result.stdout.splitlines():
        if f"[{branch}]" in line:
            return line.split()[0]
    # Fallback: return main worktree
    return result.stdout.splitlines()[0].split()[0] if result.stdout else None


def branches_matching(*patterns: str) -> list[str]:
    """List local branches matching the given patterns."""
    args = ["branch", "--list"]
    args.extend(patterns)
    result = run(*args, check=False)
    if result.returncode != 0:
        return []
    return [b.strip().lstrip("* ") for b in result.stdout.splitlines() if b.strip()]
