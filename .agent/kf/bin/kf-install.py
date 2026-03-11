#!/Users/benbaldivia/dev-2/kiloforge-wt/worker-3/.agent/kf/.venv/bin/python
"""kf-install — Initialize a Kiloforge project or update its CLI tools.

Sets up the full .agent/kf/ directory structure: venv, CLI tools, and
empty metadata scaffolding. Existing metadata files are never overwritten.

USAGE:
    kf-install [OPTIONS]
    kf-install --update    # Update CLI tools only (skip scaffolding)

OPTIONS:
    --project-dir DIR   Target project root (default: cwd)
    --skills-dir DIR    Path to kiloforge-skills repo (default: auto-detect)
    --update            Update mode: only copy scripts and lib (skip venv/metadata)
    --skip-venv         Skip venv creation/update
    --primary-branch B  Primary branch name for config.yaml (default: main)

Run from anywhere — it auto-detects the skills repo from its own location.
"""

import argparse
import os
import shutil
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path


# Scripts that were renamed to .py — clean up old versions
LEGACY_NAMES = [
    "kf-preflight",
    "kf-primary-branch",
    "kf-merge",
    "kf-merge-lock",
    "kf-track",
    "kf-track-content",
    "kf-worktree-env",
    "kf-dispatch",
    "kf-claim",
]

DEPS_YAML_HEADER = """\
# Track Dependency Graph
#
# PROTOCOL:
#   Canonical source for track dependency ordering (adjacency list).
#   Each key is a track ID; its value is a list of prerequisite track IDs.
#
# RULES:
#   - Only pending/in-progress tracks listed. Completed tracks pruned on cleanup.
#   - Architect appends entries when creating tracks.
#   - Developer checks deps before claiming: all deps must be completed.
#   - Cycles are forbidden.
"""

CONFLICTS_YAML_HEADER = """\
# Track Conflict Pairs
#
# PROTOCOL:
#   Records pairs of tracks that risk merge conflicts if worked in parallel.
#   Each key is "{lower-id}/{higher-id}" (alphabetical).
#
# RULES:
#   - Architect adds pairs when genuine file overlap exists.
#   - Pairs auto-cleaned when either track completes.
"""


def detect_skills_dir(script_path: Path) -> Path:
    """Detect the kiloforge-skills repo root from this script's location."""
    # This script lives at kf-bin/scripts/kf-install.py
    # Skills repo root is two levels up
    return script_path.parent.parent.parent


def ensure_venv(project_dir: Path) -> Path:
    """Create or update the project-local venv at .agent/kf/.venv."""
    venv_dir = project_dir / ".agent" / "kf" / ".venv"

    if not venv_dir.is_dir():
        print(f"Creating venv at {venv_dir}...")
        venv_dir.parent.mkdir(parents=True, exist_ok=True)
        result = subprocess.run(
            [sys.executable, "-m", "venv", str(venv_dir)],
            capture_output=True, text=True,
        )
        if result.returncode != 0:
            print(f"ERROR: Failed to create venv: {result.stderr}", file=sys.stderr)
            sys.exit(1)
    else:
        print(f"Venv exists at {venv_dir}")

    # Install/upgrade PyYAML
    pip = venv_dir / "bin" / "pip"
    if not pip.exists():
        pip = venv_dir / "Scripts" / "pip"  # Windows
    print("Installing PyYAML...")
    subprocess.run(
        [str(pip), "install", "-q", "pyyaml"],
        capture_output=True, text=True,
    )

    # Verify
    python = venv_dir / "bin" / "python"
    if not python.exists():
        python = venv_dir / "Scripts" / "python"  # Windows
    result = subprocess.run(
        [str(python), "-c", "import yaml; print('PyYAML', yaml.__version__)"],
        capture_output=True, text=True,
    )
    if result.returncode == 0:
        print(f"  {result.stdout.strip()}")
    else:
        print(f"WARNING: PyYAML verification failed: {result.stderr}", file=sys.stderr)

    return venv_dir


def ensure_gitignore(project_dir: Path):
    """Ensure .agent/kf/.gitignore contains .venv entry."""
    gitignore = project_dir / ".agent" / "kf" / ".gitignore"
    gitignore.parent.mkdir(parents=True, exist_ok=True)

    entries_needed = [".venv"]
    existing = set()
    if gitignore.exists():
        existing = set(gitignore.read_text().splitlines())

    missing = [e for e in entries_needed if e not in existing]
    if missing:
        with gitignore.open("a") as f:
            for entry in missing:
                f.write(f"{entry}\n")
        print(f"Updated .agent/kf/.gitignore: added {', '.join(missing)}")


def scaffold_metadata(project_dir: Path, primary_branch: str) -> list[str]:
    """Create empty metadata files if they don't already exist.

    Never overwrites existing files — safe to run on an existing project.
    Returns list of files created.
    """
    kf_dir = project_dir / ".agent" / "kf"
    tracks_dir = kf_dir / "tracks"
    tracks_dir.mkdir(parents=True, exist_ok=True)

    created = []

    def write_if_missing(path: Path, content: str) -> bool:
        if path.exists():
            return False
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)
        return True

    # config.yaml
    if write_if_missing(kf_dir / "config.yaml",
        f'project_name: ""\nprimary_branch: "{primary_branch}"\n'):
        created.append("config.yaml")

    # product.yaml
    if write_if_missing(kf_dir / "product.yaml",
        "# Product Definition\n# Populated by /kf-setup interactive Q&A\n"
        "name: \"\"\ndescription: \"\"\nproblem: \"\"\nusers: \"\"\ngoals: []\n"):
        created.append("product.yaml")

    # product-guidelines.yaml
    if write_if_missing(kf_dir / "product-guidelines.yaml",
        "# Product Guidelines\n# Populated by /kf-setup interactive Q&A\n"
        "voice: \"\"\nprinciples: []\n"):
        created.append("product-guidelines.yaml")

    # tech-stack.yaml
    if write_if_missing(kf_dir / "tech-stack.yaml",
        "# Tech Stack\n# Populated by /kf-setup interactive Q&A\n"
        "languages: []\nfrontend: \"\"\nbackend: \"\"\ndatabase: \"\"\n"
        "infrastructure: \"\"\ndependencies: []\n"):
        created.append("tech-stack.yaml")

    # workflow.yaml
    if write_if_missing(kf_dir / "workflow.yaml",
        "# Workflow Configuration\n# Populated by /kf-setup interactive Q&A\n"
        "tdd:\n  strictness: flexible\n"
        "commits:\n  strategy: conventional\n"
        "review: optional\n"
        "verification:\n  checkpoints: track_completion\n"
        "  commands: []\n"):
        created.append("workflow.yaml")

    # tracks.yaml
    if write_if_missing(kf_dir / "tracks.yaml", ""):
        created.append("tracks.yaml")

    # tracks/deps.yaml
    if write_if_missing(tracks_dir / "deps.yaml", DEPS_YAML_HEADER):
        created.append("tracks/deps.yaml")

    # tracks/conflicts.yaml
    if write_if_missing(tracks_dir / "conflicts.yaml", CONFLICTS_YAML_HEADER):
        created.append("tracks/conflicts.yaml")

    # setup_state.json
    now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    if write_if_missing(kf_dir / "setup_state.json",
        '{\n'
        '  "status": "scaffolded",\n'
        f'  "primary_branch": "{primary_branch}",\n'
        '  "auto_commit": true,\n'
        '  "project_type": "",\n'
        '  "current_section": "product",\n'
        '  "current_question": 1,\n'
        '  "completed_sections": [],\n'
        '  "answers": {},\n'
        '  "files_created": [],\n'
        f'  "started_at": "{now}",\n'
        f'  "last_updated": "{now}"\n'
        '}\n'):
        created.append("setup_state.json")

    return created


def copy_scripts(skills_dir: Path, project_dir: Path) -> list[str]:
    """Copy scripts and lib from skills repo to project."""
    src = skills_dir / "kf-bin" / "scripts"
    dst = project_dir / ".agent" / "kf" / "bin"

    if not src.is_dir():
        print(f"ERROR: Scripts source not found at {src}", file=sys.stderr)
        sys.exit(1)

    dst.mkdir(parents=True, exist_ok=True)

    copied = []

    # Copy Python scripts
    for f in sorted(src.glob("*.py")):
        shutil.copy2(f, dst / f.name)
        os.chmod(dst / f.name, 0o755)
        copied.append(f.name)

    # Copy lib/ directory
    src_lib = src / "lib"
    dst_lib = dst / "lib"
    if src_lib.is_dir():
        if dst_lib.exists():
            shutil.rmtree(dst_lib)
        shutil.copytree(src_lib, dst_lib)
        lib_files = list(src_lib.rglob("*.py"))
        copied.append(f"lib/ ({len(lib_files)} files)")

    return copied


def rewrite_shebangs(project_dir: Path, venv_dir: Path):
    """Rewrite Python script shebangs to use the project-local venv."""
    bin_dir = project_dir / ".agent" / "kf" / "bin"
    python_path = str(venv_dir / "bin" / "python")
    if not Path(python_path).exists():
        python_path = str(venv_dir / "Scripts" / "python")  # Windows

    count = 0
    for f in sorted(bin_dir.glob("*.py")):
        text = f.read_text()
        lines = text.split("\n", 1)
        if lines and "python" in lines[0]:
            new_shebang = f"#!{python_path}"
            if lines[0] != new_shebang:
                new_text = new_shebang + "\n" + (lines[1] if len(lines) > 1 else "")
                f.write_text(new_text)
                count += 1

    print(f"Rewrote shebangs in {count} script(s)")


def clean_legacy(project_dir: Path) -> list[str]:
    """Remove old non-.py scripts that have been superseded."""
    bin_dir = project_dir / ".agent" / "kf" / "bin"
    removed = []

    for name in LEGACY_NAMES:
        old_file = bin_dir / name  # without .py
        py_file = bin_dir / f"{name}.py"
        if old_file.exists() and not old_file.suffix and py_file.exists():
            old_file.unlink()
            removed.append(name)

    return removed


def main():
    parser = argparse.ArgumentParser(
        description="Initialize a Kiloforge project or update its CLI tools.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__,
    )
    parser.add_argument(
        "--project-dir", default=os.getcwd(),
        help="Target project root (default: cwd)",
    )
    parser.add_argument(
        "--skills-dir", default=None,
        help="Path to kiloforge-skills repo (default: auto-detect)",
    )
    parser.add_argument(
        "--update", action="store_true",
        help="Update mode: only copy scripts and lib (skip venv/metadata)",
    )
    parser.add_argument(
        "--skip-venv", action="store_true",
        help="Skip venv creation/update",
    )
    parser.add_argument(
        "--primary-branch", default="main",
        help="Primary branch name for config.yaml (default: main)",
    )
    args = parser.parse_args()

    project_dir = Path(args.project_dir).resolve()
    if args.skills_dir:
        skills_dir = Path(args.skills_dir).resolve()
    else:
        skills_dir = detect_skills_dir(Path(__file__).resolve())

    mode = "Update" if args.update else "Install"

    print("=" * 60)
    print(f"  Kiloforge — {mode}")
    print("=" * 60)
    print(f"  Skills repo:  {skills_dir}")
    print(f"  Project:      {project_dir}")
    print(f"  Target:       {project_dir / '.agent' / 'kf'}")
    print("=" * 60)
    print()

    # Step 1: Venv (skip in update mode)
    skip_venv = args.skip_venv or args.update
    if not skip_venv:
        venv_dir = ensure_venv(project_dir)
        ensure_gitignore(project_dir)
        print()
    else:
        venv_dir = project_dir / ".agent" / "kf" / ".venv"
        if args.update:
            print("Update mode — skipping venv setup")
        else:
            print("Skipping venv setup (--skip-venv)")

    # Step 2: Scaffold metadata (skip in update mode)
    if not args.update:
        print("Scaffolding metadata files...")
        created = scaffold_metadata(project_dir, args.primary_branch)
        if created:
            for name in created:
                print(f"  created .agent/kf/{name}")
        else:
            print("  (all metadata files already exist)")
        print()

    # Step 3: Copy scripts
    print("Copying scripts...")
    copied = copy_scripts(skills_dir, project_dir)
    for name in copied:
        print(f"  {name}")
    print()

    # Step 4: Rewrite shebangs
    if venv_dir.is_dir():
        rewrite_shebangs(project_dir, venv_dir)
    else:
        print("WARNING: Venv not found — shebangs not updated", file=sys.stderr)
    print()

    # Step 5: Clean legacy
    removed = clean_legacy(project_dir)
    if removed:
        print(f"Cleaned {len(removed)} legacy script(s):")
        for name in removed:
            print(f"  removed {name} (superseded by {name}.py)")
    else:
        print("No legacy scripts to clean")

    print()
    print("=" * 60)
    print(f"  {mode} complete")
    if not args.update:
        print()
        print("  Next: run /kf-setup to configure project metadata")
    print("=" * 60)


if __name__ == "__main__":
    main()
