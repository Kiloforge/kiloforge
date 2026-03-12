#!/usr/bin/env python3
# kf-merge-lock — Cross-worktree branch lock helper (CLI wrapper)
#
# Provides a lock on the primary branch to coordinate merges across worktrees.
# Tries HTTP (orchestrator) first, falls back to mkdir (filesystem) seamlessly.
# A single `acquire` call handles mode detection, fallback, and waiting.
#
# USAGE:
#   kf-merge-lock acquire [--holder NAME] [--timeout SECONDS] [--ttl SECONDS] [--pid PID]
#   kf-merge-lock release [--holder NAME]
#   kf-merge-lock heartbeat [--holder NAME] [--ttl SECONDS]
#   kf-merge-lock status
#   kf-merge-lock help
#
# ENVIRONMENT:
#   KF_ORCH_URL     Orchestrator URL (default: http://localhost:39517)
#   KF_LOCK_HOLDER  Default holder name (default: basename of $PWD)
#
# IDEMPOTENT ACQUIRE
#
#   If the same holder already holds the lock, acquire succeeds (re-entry).
#   This supports the conflict resolution flow: kf-merge exits with code 3
#   keeping the lock held, the agent resolves conflicts, then re-runs kf-merge
#   which re-acquires the same lock.
#
# MERGE PROTOCOL — REBASE CONFLICT RESOLUTION
#
#   During `git rebase`, --ours and --theirs have REVERSED semantics:
#     --ours  = the branch being rebased ONTO (e.g., main — the latest state)
#     --theirs = the commit being REPLAYED (the worker's old commit — stale)
#
#   To accept main's version of track state files during rebase conflicts:
#     git checkout --ours .agent/kf/tracks.yaml .agent/kf/tracks/deps.yaml
#     git checkout --ours .agent/kf/tracks/conflicts.yaml
#
#   WRONG: git checkout --theirs (accepts the stale worker commit, reverts
#   other workers' completions — this caused track state regressions).
#
# MAIN WORKTREE CLEANUP
#
#   Before running `git -C <main-worktree> merge --ff-only`, ensure the main
#   worktree is clean. Previous failed merges or stash pops can leave dirty
#   state that blocks ff-merge. Use:
#     git -C <main-worktree> reset --hard HEAD
#   before the merge attempt if the worktree might be dirty.

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from lib import merge_lock

DEFAULT_HOLDER = os.environ.get("KF_LOCK_HOLDER", os.path.basename(os.getcwd()))
DEFAULT_TTL = merge_lock.DEFAULT_TTL


def cmd_acquire(args: list[str]) -> int:
    holder = DEFAULT_HOLDER
    ttl = DEFAULT_TTL
    timeout = 0
    pid = os.getppid()

    i = 0
    while i < len(args):
        if args[i] == "--holder" and i + 1 < len(args):
            holder = args[i + 1]; i += 2
        elif args[i] == "--ttl" and i + 1 < len(args):
            ttl = int(args[i + 1]); i += 2
        elif args[i] == "--timeout" and i + 1 < len(args):
            timeout = int(args[i + 1]); i += 2
        elif args[i] == "--pid" and i + 1 < len(args):
            pid = int(args[i + 1]); i += 2
        else:
            print(f"Unknown option: {args[i]}", file=sys.stderr)
            return 1

    if merge_lock.acquire(holder, ttl=ttl, timeout=timeout, pid=pid):
        print(f"Branch lock acquired")
        return 0
    else:
        print(f"BRANCH LOCK HELD", file=sys.stderr)
        return 1


def cmd_release(args: list[str]) -> int:
    holder = DEFAULT_HOLDER

    i = 0
    while i < len(args):
        if args[i] == "--holder" and i + 1 < len(args):
            holder = args[i + 1]; i += 2
        else:
            print(f"Unknown option: {args[i]}", file=sys.stderr)
            return 1

    if merge_lock.release(holder):
        print("Branch lock released")
        return 0
    else:
        print("ERROR: Cannot release lock — held by another worker", file=sys.stderr)
        return 1


def cmd_heartbeat(args: list[str]) -> int:
    holder = DEFAULT_HOLDER
    ttl = DEFAULT_TTL

    i = 0
    while i < len(args):
        if args[i] == "--holder" and i + 1 < len(args):
            holder = args[i + 1]; i += 2
        elif args[i] == "--ttl" and i + 1 < len(args):
            ttl = int(args[i + 1]); i += 2
        else:
            print(f"Unknown option: {args[i]}", file=sys.stderr)
            return 1

    if merge_lock.heartbeat(holder, ttl=ttl):
        return 0
    print("WARNING: Heartbeat failed", file=sys.stderr)
    return 1


def cmd_status() -> int:
    print("===============================================")
    print("          BRANCH LOCK STATUS")
    print("===============================================")
    print()

    info = merge_lock.status()
    if info is None:
        print("Status: No active lock")
    else:
        mode = info.get("mode", "unknown")
        print(f"Mode:   {mode}")
        print("Status: LOCKED")
        print()
        print(f"  Holder:   {info.get('holder', 'unknown')}")
        if "pid" in info and info["pid"] is not None:
            print(f"  PID:      {info['pid']}")
            if info.get("pid_alive") is not None:
                print(f"  PID alive: {'yes' if info['pid_alive'] else 'NO — may be stale'}")
        if "acquired" in info:
            print(f"  Acquired: {info['acquired']}")
        if "expires" in info:
            print(f"  Expires:  {info['expires']}")

    print()
    print("===============================================")
    return 0


def cmd_help() -> int:
    print("""\
kf-merge-lock — Cross-worktree branch lock

Coordinates merges to the primary branch across worktrees.
Tries HTTP (orchestrator) first, falls back to mkdir (filesystem).
A single acquire call handles mode detection, fallback, and waiting.

USAGE:
  kf-merge-lock acquire [--holder NAME] [--timeout SECONDS] [--ttl SECONDS] [--pid PID]
  kf-merge-lock release [--holder NAME]
  kf-merge-lock heartbeat [--holder NAME] [--ttl SECONDS]
  kf-merge-lock status
  kf-merge-lock help

OPTIONS:
  --holder NAME     Lock holder identity (default: basename of $PWD)
  --timeout SECONDS Acquire timeout: 0=fail if held, >0=wait (default: 0)
  --ttl SECONDS     Lock TTL in seconds (default: 120)
  --pid PID         PID to record for stale detection (default: parent PID)

MODES (automatic — no user configuration needed):
  HTTP   — Used when orchestrator is reachable ($KF_ORCH_URL/health responds).
           Uses TTL, heartbeat, server-side long-poll for waiting.
  mkdir  — Used when orchestrator is unavailable.
           Uses PID + timestamp for stale detection. Auto-cleans dead-PID locks
           older than 240s.

ENVIRONMENT:
  KF_ORCH_URL      Orchestrator URL (default: http://localhost:39517)
  KF_LOCK_HOLDER   Default holder name (default: basename of $PWD)

EXAMPLES:
  kf-merge-lock acquire --holder developer-1 --timeout 300
  kf-merge-lock heartbeat --holder developer-1
  kf-merge-lock status
  kf-merge-lock release --holder developer-1""")
    return 0


def main() -> int:
    args = sys.argv[1:]
    cmd = args[0] if args else "help"
    rest = args[1:] if len(args) > 1 else []

    commands = {
        "acquire": lambda: cmd_acquire(rest),
        "release": lambda: cmd_release(rest),
        "heartbeat": lambda: cmd_heartbeat(rest),
        "status": lambda: cmd_status(),
        "help": lambda: cmd_help(),
        "--help": lambda: cmd_help(),
        "-h": lambda: cmd_help(),
    }

    if cmd in commands:
        return commands[cmd]()
    else:
        print(f"Unknown command: {cmd}", file=sys.stderr)
        cmd_help()
        return 1


if __name__ == "__main__":
    sys.exit(main())
