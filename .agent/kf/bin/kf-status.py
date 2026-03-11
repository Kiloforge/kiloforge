#!/Users/benbaldivia/dev-2/kiloforge-wt/worker-3/.agent/kf/.venv/bin/python
"""kf-status — Full project status in a single command.

Combines pre-flight, current workers, track status, and dispatch
recommendations into one output. Designed for /kf-status skill.

USAGE:
    kf-status [--ref BRANCH] [--json]

EXIT CODES:
    0  Success
    1  Error (not initialized, etc.)
"""

import json
import os
import subprocess
import sys

BIN_DIR = os.path.dirname(os.path.realpath(__file__))


def run_script(name, *args, check=False):
    """Run a sibling script and return (returncode, stdout, stderr)."""
    cmd = [os.path.join(BIN_DIR, name)] + list(args)
    result = subprocess.run(cmd, capture_output=True, text=True)
    if check and result.returncode != 0:
        return result.returncode, result.stdout, result.stderr
    return result.returncode, result.stdout, result.stderr


def main():
    # Parse args
    ref = None
    json_mode = False
    args = sys.argv[1:]
    i = 0
    while i < len(args):
        if args[i] == "--ref" and i + 1 < len(args):
            ref = args[i + 1]; i += 2
        elif args[i] == "--json":
            json_mode = True; i += 1
        elif args[i] in ("--help", "-h"):
            print(__doc__)
            return 0
        else:
            print(f"Unknown option: {args[i]}", file=sys.stderr)
            return 1

    # Step 1: Pre-flight — resolve primary branch
    rc, stdout, stderr = run_script("kf-preflight.py")
    if rc != 0:
        print(stderr or "ERROR: Pre-flight check failed. Run /kf-setup first.", file=sys.stderr)
        return 1

    # Parse PRIMARY_BRANCH from preflight output
    primary_branch = ref
    for line in stdout.splitlines():
        if line.startswith("PRIMARY_BRANCH="):
            primary_branch = primary_branch or line.split("=", 1)[1].strip().strip("'\"")
    if not primary_branch:
        primary_branch = "main"

    # Step 2: Current workers (instant — filesystem claims)
    rc, claim_out, _ = run_script("kf-claim.py", "list")
    has_claims = rc == 0 and "(no active claims)" not in claim_out

    # Step 3: Track status
    status_args = ["status"]
    if primary_branch:
        status_args += ["--ref", primary_branch]
    rc, status_out, status_err = run_script("kf-track.py", *status_args)
    if rc != 0:
        print(status_err or status_out, file=sys.stderr)
        return 1

    # Step 4: Dispatch recommendations
    dispatch_out = ""
    # Check if worktrees exist (more than just the main one)
    wt_result = subprocess.run(
        ["git", "worktree", "list"],
        capture_output=True, text=True,
    )
    worktree_count = len(wt_result.stdout.strip().splitlines()) if wt_result.stdout else 0
    if worktree_count > 1:
        dispatch_args = []
        if primary_branch:
            dispatch_args += ["--ref", primary_branch]
        rc, dispatch_out, _ = run_script("kf-dispatch.py", *dispatch_args)

    # --- Output ---
    if has_claims:
        print("=" * 80)
        print("                            CURRENT WORKERS")
        print("=" * 80)
        print()
        print(claim_out.strip())
        print()

    print(status_out, end="")

    if dispatch_out and dispatch_out.strip():
        print()
        print("-" * 80)
        print("                        DISPATCH RECOMMENDATIONS")
        print("-" * 80)
        print()
        print(dispatch_out.strip())
        print()

    return 0


if __name__ == "__main__":
    sys.exit(main())
