#!/usr/bin/env python3
# kf-preflight — Verify Kiloforge is initialized before running any skill.
#
# Checks that required metadata files and CLI tools exist on the primary
# branch. Run this at the start of every kf-* skill.
#
# USAGE:
#   eval "$(.agent/kf/bin/kf-preflight.py)"
#
# On success: prints shell commands for eval:
#   - source .agent/kf/.venv/bin/activate  (activates venv for session)
#   - PRIMARY_BRANCH=<branch>
#
# On failure: prints error message to stderr and exits with code 1
#
# CHECKS:
#   1. Ensures project venv exists with dependencies
#   2. Resolves primary branch
#   3. Verifies required metadata files exist on primary branch
#   4. Verifies CLI tools are installed locally

import os
import subprocess
import sys

KF_BIN = os.path.dirname(os.path.abspath(__file__))
KF_DIR = os.path.dirname(KF_BIN)  # .agent/kf
VENV_DIR = os.path.join(KF_DIR, ".venv")


def ensure_venv():
    """Create project-local venv and install PyYAML if missing.

    This is the safety net — if the venv doesn't exist or PyYAML is missing,
    fix it here so no script ever falls back to system pip.
    """
    venv_python = os.path.join(VENV_DIR, "bin", "python")
    if not os.path.exists(venv_python):
        venv_python = os.path.join(VENV_DIR, "Scripts", "python")  # Windows

    # Check if venv exists and has PyYAML
    if os.path.exists(venv_python):
        result = subprocess.run(
            [venv_python, "-c", "import yaml"],
            capture_output=True,
        )
        if result.returncode == 0:
            return  # All good

    # Create venv if missing
    if not os.path.isdir(VENV_DIR):
        print("Creating project-local venv at .agent/kf/.venv...", file=sys.stderr)
        result = subprocess.run(
            [sys.executable, "-m", "venv", VENV_DIR],
            capture_output=True, text=True,
        )
        if result.returncode != 0:
            print(f"ERROR: Failed to create venv: {result.stderr}", file=sys.stderr)
            print("Install Python 3 venv support and retry.", file=sys.stderr)
            sys.exit(1)

    # Install PyYAML into venv
    venv_pip = os.path.join(VENV_DIR, "bin", "pip")
    if not os.path.exists(venv_pip):
        venv_pip = os.path.join(VENV_DIR, "Scripts", "pip")
    print("Installing PyYAML into .agent/kf/.venv...", file=sys.stderr)
    subprocess.run(
        [venv_pip, "install", "-q", "pyyaml"],
        capture_output=True,
    )


REQUIRED_FILES = [
    ".agent/kf/config.yaml",
    ".agent/kf/product.yaml",
    ".agent/kf/tech-stack.yaml",
    ".agent/kf/workflow.yaml",
    ".agent/kf/tracks.yaml",
]

REQUIRED_TOOLS = [
    "kf-primary-branch.py",
    "kf-track.py",
    "kf-track-content.py",
    "kf-merge.py",
    "kf-merge-lock.py",
]


def resolve_primary_branch() -> str:
    """Resolve the primary branch by calling the sibling kf-primary-branch script."""
    script = os.path.join(KF_BIN, "kf-primary-branch.py")
    result = subprocess.run(
        [script],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        print("ERROR: Failed to resolve primary branch.", file=sys.stderr)
        if result.stderr:
            print(result.stderr.rstrip(), file=sys.stderr)
        sys.exit(1)
    return result.stdout.strip()


def check_metadata_files(primary_branch: str) -> None:
    """Verify required metadata files exist on the primary branch."""
    missing = []
    for file in REQUIRED_FILES:
        result = subprocess.run(
            ["git", "show", f"{primary_branch}:{file}"],
            capture_output=True,
        )
        if result.returncode != 0:
            missing.append(file)

    if missing:
        print(
            f"ERROR: Kiloforge is not initialized. Missing files on {primary_branch}:",
            file=sys.stderr,
        )
        for f in missing:
            print(f"  - {f}", file=sys.stderr)
        print("", file=sys.stderr)
        print("Run /kf-setup to initialize Kiloforge for this project.", file=sys.stderr)
        sys.exit(1)


def check_cli_tools() -> None:
    """Verify CLI tools are installed locally in the bin directory."""
    missing = []
    for tool in REQUIRED_TOOLS:
        path = os.path.join(KF_BIN, tool)
        if not (os.path.isfile(path) and os.access(path, os.X_OK)):
            missing.append(tool)

    if missing:
        print("WARNING: Missing CLI tools in .agent/kf/bin/:", file=sys.stderr)
        for t in missing:
            print(f"  - {t}", file=sys.stderr)
        print("Re-run /kf-setup to install tools.", file=sys.stderr)
        # Warning only — don't block, some tools may not be needed


def main() -> None:
    ensure_venv()
    primary_branch = resolve_primary_branch()
    check_metadata_files(primary_branch)
    check_cli_tools()

    # Output shell commands for eval — activates venv and sets PRIMARY_BRANCH.
    # The venv activation puts .venv/bin/python on PATH so all kf-* scripts
    # (which use #!/usr/bin/env python3) pick up the correct interpreter.
    venv_activate = os.path.join(KF_DIR, ".venv", "bin", "activate")
    if not os.path.exists(venv_activate):
        venv_activate = os.path.join(KF_DIR, ".venv", "Scripts", "activate")
    if os.path.exists(venv_activate):
        print(f"source {venv_activate}")
    print(f"PRIMARY_BRANCH={primary_branch}")


if __name__ == "__main__":
    main()
