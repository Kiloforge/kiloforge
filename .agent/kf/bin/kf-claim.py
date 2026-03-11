#!/Users/benbaldivia/dev-2/kiloforge-wt/worker-3/.agent/kf/.venv/bin/python
"""kf-claim — Per-worktree track claim management.

Manages claim locks that record which track a worktree is working on.
Claims are instant to read (filesystem) and visible to all worktrees.

USAGE:
    kf-claim acquire <track-id> [--worktree NAME] [--holder NAME]
    kf-claim release [--worktree NAME] [--holder NAME]
    kf-claim show [--worktree NAME]
    kf-claim list
    kf-claim find <track-id>
    kf-claim help

LOCATION:
    Claims are stored under $(git rev-parse --git-common-dir)/kf-claims/
    as lock directories, one per worktree.

EXIT CODES:
    0  Success
    1  Error (claim held, not found, etc.)
"""

import argparse
import json
import os
import sys

# Add parent directory to path so lib/ is importable
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from lib import worktree_lock


def default_worktree_name() -> str:
    """Infer worktree name from current directory."""
    return os.path.basename(os.getcwd())


def cmd_acquire(args):
    wt = args.worktree or default_worktree_name()
    holder = args.holder or wt

    # Check if this worktree already has a claim
    existing = worktree_lock.read_claim(wt)
    if existing:
        if existing.get("track_id") == args.track_id:
            print(f"Already claimed: {args.track_id} (by {wt})")
            return 0
        print(
            f"ERROR: Worktree '{wt}' already has a claim on "
            f"'{existing.get('track_id')}'. Release it first.",
            file=sys.stderr,
        )
        return 1

    # Check if another worktree already claimed this track
    claim = worktree_lock.find_track_claim(args.track_id)
    if claim:
        other_wt, info = claim
        print(
            f"ERROR: Track '{args.track_id}' already claimed by "
            f"worktree '{other_wt}' (holder: {info.get('holder', 'unknown')})",
            file=sys.stderr,
        )
        return 1

    if worktree_lock.acquire(wt, args.track_id, holder=holder):
        print(f"Claimed: {args.track_id} (worktree: {wt})")
        return 0
    else:
        print(f"ERROR: Failed to acquire claim for worktree '{wt}'", file=sys.stderr)
        return 1


def cmd_release(args):
    wt = args.worktree or default_worktree_name()
    holder = args.holder

    existing = worktree_lock.read_claim(wt)
    if not existing:
        print(f"No active claim for worktree '{wt}'")
        return 0

    if worktree_lock.release(wt, holder=holder):
        print(f"Released: {existing.get('track_id', 'unknown')} (worktree: {wt})")
        return 0
    else:
        return 1


def cmd_show(args):
    wt = args.worktree or default_worktree_name()
    info = worktree_lock.read_claim(wt)
    if not info:
        print(f"No active claim for worktree '{wt}'")
        return 0
    if args.json:
        print(json.dumps({"worktree": wt, **info}, indent=2))
    else:
        print(f"Worktree:  {wt}")
        print(f"Track:     {info.get('track_id', 'unknown')}")
        print(f"Holder:    {info.get('holder', 'unknown')}")
        print(f"PID:       {info.get('pid', 'unknown')}")
        print(f"Started:   {info.get('started', 'unknown')}")
    return 0


def cmd_list(args):
    claims = worktree_lock.list_claims()
    if not claims:
        print("(no active claims)")
        return 0

    if args.json:
        out = [{"worktree": wt, **info} for wt, info in claims.items()]
        print(json.dumps(out, indent=2))
    else:
        fmt = "%-20s %-50s %-20s %s"
        print(fmt % ("WORKTREE", "TRACK", "HOLDER", "STARTED"))
        print(fmt % ("--------", "-----", "------", "-------"))
        for wt, info in claims.items():
            print(fmt % (
                wt,
                info.get("track_id", "?"),
                info.get("holder", "?"),
                info.get("started", "?"),
            ))
        print(f"\n{len(claims)} active claim(s)")
    return 0


def cmd_find(args):
    claim = worktree_lock.find_track_claim(args.track_id)
    if not claim:
        print(f"Track '{args.track_id}' is not claimed by any worktree")
        return 1
    wt_name, info = claim
    if args.json:
        print(json.dumps({"worktree": wt_name, **info}, indent=2))
    else:
        print(f"Track '{args.track_id}' claimed by worktree '{wt_name}'")
        print(f"  Holder:  {info.get('holder', 'unknown')}")
        print(f"  PID:     {info.get('pid', 'unknown')}")
        print(f"  Started: {info.get('started', 'unknown')}")
    return 0


def main():
    parser = argparse.ArgumentParser(
        description="Per-worktree track claim management",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__,
    )
    sub = parser.add_subparsers(dest="command")

    # acquire
    p_acq = sub.add_parser("acquire", help="Claim a track for this worktree")
    p_acq.add_argument("track_id", help="Track ID to claim")
    p_acq.add_argument("--worktree", help="Worktree name (default: cwd basename)")
    p_acq.add_argument("--holder", help="Holder identity (default: worktree name)")

    # release
    p_rel = sub.add_parser("release", help="Release the current worktree claim")
    p_rel.add_argument("--worktree", help="Worktree name (default: cwd basename)")
    p_rel.add_argument("--holder", help="Validate holder before releasing")

    # show
    p_show = sub.add_parser("show", help="Show current worktree claim")
    p_show.add_argument("--worktree", help="Worktree name (default: cwd basename)")
    p_show.add_argument("--json", action="store_true", help="Output as JSON")

    # list
    p_list = sub.add_parser("list", help="List all active claims")
    p_list.add_argument("--json", action="store_true", help="Output as JSON")

    # find
    p_find = sub.add_parser("find", help="Find which worktree claimed a track")
    p_find.add_argument("track_id", help="Track ID to search for")
    p_find.add_argument("--json", action="store_true", help="Output as JSON")

    # help
    sub.add_parser("help", help="Show help")

    args = parser.parse_args()

    if not args.command or args.command == "help":
        parser.print_help()
        return 0

    handlers = {
        "acquire": cmd_acquire,
        "release": cmd_release,
        "show": cmd_show,
        "list": cmd_list,
        "find": cmd_find,
    }
    return handlers[args.command](args)


if __name__ == "__main__":
    sys.exit(main())
