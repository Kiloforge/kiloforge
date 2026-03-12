#!/Users/benbaldivia/dev-2/kiloforge-wt/kfc-d780f5-worker-1/.agent/kf/.venv/bin/python
"""kf-merge — Unified merge protocol for Kiloforge agents.

Merges the current branch into the primary branch using a lock-protected,
rebase-based, fast-forward-only strategy. Lock is always released on error.

USAGE:
  kf-merge [OPTIONS]

OPTIONS:
  --holder NAME         Lock holder identity (default: basename of cwd)
  --timeout SECONDS     Lock acquisition timeout; 0=fail if held (default: 0)
  --verify CMD          Verification command to run post-rebase (skip if omitted)
  --registry-cmd CMD    Registry update command to run post-rebase. Runs against
                        clean rebased state, then stages and commits registry files.
                        Used by architects to avoid registry conflicts.
  --reapply CMD         Re-apply track state after conflict resolution during rebase.
                        Used by developers when registry conflicts occur.
  --conflict-strategy   How to resolve track state conflicts during rebase:
                          "theirs" (default) — accept primary branch version
                          "ours"  — accept current branch version (for reports)
  --cleanup-branch NAME Delete this branch after merge (implementation branch)

EXIT CODES:
  0  Merge succeeded
  1  Merge failed (lock released, safe to retry)
  2  Lock held by another worker (not acquired)

FLOWS:
  Metadata merge (architect — commit track content first, registry under lock):
    kf-merge --holder architect-1 --timeout 0 \
      --registry-cmd ".agent/kf/bin/kf-track.py add X --title '...' --type feature"

  Implementation merge (developer — with verification):
    kf-merge --holder developer-1 --timeout 300 \
      --verify "make test && make build && make lint" \
      --reapply ".agent/kf/bin/kf-track.py update X --status completed" \
      --cleanup-branch kf/feature/my-track
"""

import argparse
import atexit
import os
import re
import subprocess
import sys
import threading

# ── Globals ──────────────────────────────────────────────────────────────────

KF_BIN = os.path.dirname(os.path.realpath(__file__))

STATE_FILES = [
    ".agent/kf/tracks.yaml",
    ".agent/kf/tracks/deps.yaml",
    ".agent/kf/tracks/conflicts.yaml",
]
REPORT_FILES = ".agent/kf/_reports/"

lock_acquired = False
heartbeat_stop = threading.Event()
heartbeat_thread = None


# ── Helpers ──────────────────────────────────────────────────────────────────

def run(cmd, *, check=True, capture=False, cwd=None, shell=False):
    """Run a command, returning CompletedProcess."""
    kwargs = dict(check=check, cwd=cwd)
    if capture:
        kwargs.update(stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    if shell:
        return subprocess.run(cmd, shell=True, **kwargs)
    return subprocess.run(cmd, **kwargs)


def run_quiet(cmd, **kwargs):
    """Run a command, suppressing stderr, not raising on failure."""
    return subprocess.run(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        text=True,
        check=False,
        **kwargs,
    )


def cleanup():
    """Release heartbeat and lock on any exit."""
    global lock_acquired, heartbeat_thread

    # Stop heartbeat thread
    if heartbeat_thread is not None:
        heartbeat_stop.set()
        heartbeat_thread.join(timeout=5)
        heartbeat_thread = None

    # Release lock
    if lock_acquired:
        run_quiet([os.path.join(KF_BIN, "kf-merge-lock.py"), "release"])
        lock_acquired = False
        print("Lock released.")


atexit.register(cleanup)


def die(msg):
    print(f"ERROR: {msg}", file=sys.stderr)
    sys.exit(1)


def heartbeat_loop():
    """Background loop: send heartbeat every 30 seconds until stopped."""
    lock_script = os.path.join(KF_BIN, "kf-merge-lock.py")
    while not heartbeat_stop.wait(30):
        run_quiet([lock_script, "heartbeat"])


# ── Argument parsing ─────────────────────────────────────────────────────────

def parse_args():
    parser = argparse.ArgumentParser(
        description="Unified merge protocol for Kiloforge agents.",
        add_help=True,
    )
    parser.add_argument(
        "--holder", default=os.path.basename(os.getcwd()),
        help="Lock holder identity (default: basename of cwd)",
    )
    parser.add_argument(
        "--timeout", type=int, default=0,
        help="Lock acquisition timeout; 0=fail if held (default: 0)",
    )
    parser.add_argument(
        "--verify", dest="verify_cmd", default="",
        help="Verification command to run post-rebase",
    )
    parser.add_argument(
        "--registry-cmd", dest="registry_cmd", default="",
        help="Registry update command to run post-rebase",
    )
    parser.add_argument(
        "--reapply", dest="reapply_cmd", default="",
        help="Re-apply track state after conflict resolution during rebase",
    )
    parser.add_argument(
        "--conflict-strategy", dest="conflict_strategy", default="theirs",
        choices=["theirs", "ours"],
        help='How to resolve track state conflicts: "theirs" (default) or "ours"',
    )
    parser.add_argument(
        "--cleanup-branch", dest="cleanup_branch", default="",
        help="Delete this branch after merge",
    )
    return parser.parse_args()


# ── Main ─────────────────────────────────────────────────────────────────────

def main():
    global lock_acquired, heartbeat_thread

    args = parse_args()

    # ── Step 1: Resolve primary branch ───────────────────────────────────────
    result = run(
        [os.path.join(KF_BIN, "kf-primary-branch.py")],
        capture=True,
    )
    primary_branch = result.stdout.strip()

    result = run(["git", "branch", "--show-current"], capture=True)
    current_branch = result.stdout.strip()

    print(f"Merge: {current_branch} -> {primary_branch} (holder: {args.holder})")

    # ── Step 2: Locate primary branch worktree ───────────────────────────────
    result = run(["git", "worktree", "list"], capture=True)
    main_worktree = ""
    first_worktree = ""
    for line in result.stdout.splitlines():
        if not first_worktree:
            first_worktree = line.split()[0]
        if re.search(rf"\[{re.escape(primary_branch)}\]", line):
            main_worktree = line.split()[0]
            break
    if not main_worktree:
        main_worktree = first_worktree

    print(f"Primary worktree: {main_worktree}")

    # ── Step 3: Acquire branch lock ──────────────────────────────────────────
    print(f"Acquiring branch lock (timeout: {args.timeout}s)...")
    lock_result = run(
        [os.path.join(KF_BIN, "kf-merge-lock.py"), "acquire", "--timeout", str(args.timeout)],
        check=False,
    )
    if lock_result.returncode != 0:
        print("BRANCH LOCK HELD — another worker is merging. Retry later.")
        sys.exit(2)

    lock_acquired = True
    print("Lock acquired.")

    # ── Step 4: Start heartbeat ──────────────────────────────────────────────
    heartbeat_thread = threading.Thread(target=heartbeat_loop, daemon=True)
    heartbeat_thread.start()

    # ── Step 5: Rebase onto primary branch ───────────────────────────────────
    print(f"Rebasing onto {primary_branch}...")
    rebase_result = run_quiet(["git", "rebase", primary_branch])

    if rebase_result.returncode != 0:
        print("Rebase conflict detected — resolving track state files...")

        if args.conflict_strategy == "theirs":
            for f in STATE_FILES:
                r = run_quiet(["git", "checkout", "--theirs", f])
                if r.returncode == 0:
                    run_quiet(["git", "add", f])

        elif args.conflict_strategy == "ours":
            r = run_quiet(["git", "checkout", "--ours", REPORT_FILES])
            if r.returncode == 0:
                run_quiet(["git", "add", REPORT_FILES])
            for f in STATE_FILES:
                r = run_quiet(["git", "checkout", "--theirs", f])
                if r.returncode == 0:
                    run_quiet(["git", "add", f])

        continue_result = run_quiet(["git", "rebase", "--continue"])
        if continue_result.returncode != 0:
            die("Rebase failed after conflict resolution — non-state file conflict. Manual intervention required.")

        # Re-apply track state changes after accepting theirs (developer flow)
        if args.reapply_cmd:
            print("Re-applying track state changes...")
            r = run(args.reapply_cmd, shell=True, check=False)
            if r.returncode != 0:
                die(f"Reapply command failed: {args.reapply_cmd}")
            for f in STATE_FILES:
                run_quiet(["git", "add", f])
            run_quiet(["git", "commit", "--amend", "--no-edit"])

    print("Rebase complete.")

    # ── Step 6: Registry update (metadata merge — post-rebase, conflict-free)
    if args.registry_cmd:
        print("Updating registry against rebased state...")
        r = run(args.registry_cmd, shell=True, check=False)
        if r.returncode != 0:
            die(f"Registry command failed: {args.registry_cmd}")
        for f in STATE_FILES:
            run_quiet(["git", "add", f])
        run_quiet(["git", "commit", "-m", "chore(kf): update track registry"])
        print("Registry updated.")

    # ── Step 7: Post-rebase verification (implementation merge) ──────────────
    if args.verify_cmd:
        print(f"Running verification: {args.verify_cmd}")
        r = run(args.verify_cmd, shell=True, check=False)
        if r.returncode != 0:
            die("Verification failed. Fix issues and retry.")
        print("Verification passed.")

    # ── Step 8: Fast-forward merge ───────────────────────────────────────────
    print(f"Merging into {primary_branch} (fast-forward only)...")
    ff_result = run(
        ["git", "-C", main_worktree, "merge", current_branch, "--ff-only"],
        check=False,
    )
    if ff_result.returncode != 0:
        die("Fast-forward merge failed — primary branch has diverged.")

    print("MERGE SUCCEEDED.")

    # ── Step 9: Cleanup ──────────────────────────────────────────────────────
    # Lock + heartbeat released by atexit handler

    if args.cleanup_branch:
        run_quiet(["git", "branch", "-d", args.cleanup_branch])

    print("Done.")


if __name__ == "__main__":
    main()
