#!/usr/bin/env python3
"""kf-dispatch — Compute dispatch assignments for idle developer worktrees.

Scans worktree state, reads the track registry and dependency/conflict graphs,
computes priority scores, and outputs a dispatch plan.

USAGE:
  kf-dispatch [OPTIONS]

OPTIONS:
  --ref BRANCH        Read track state from this branch (default: auto-detect via kf-primary-branch)
  --limit N           Max assignments to output (default: unlimited)
  --json              Output as JSON instead of formatted text
  --dry-run           Add dry-run notice to output
  -h, --help          Show this help

PRIORITY SCORING:
  - Unblock factor (3x): tracks that unblock more blocked tracks score higher
  - Conflict penalty (-2x): tracks conflicting with claimed tracks are penalized
  - Type diversity (+1): tracks of types not currently being worked on get a bonus
  - Task count tiebreaker: smaller tracks preferred (unblock pipeline faster)

EXIT CODES:
  0  Success (dispatch plan produced)
  1  Error (missing tools, not initialized, etc.)
"""

import json
import os
import subprocess
import sys
from collections import defaultdict


def run(cmd, **kwargs):
    """Run a shell command, return stdout. Returns empty string on failure."""
    try:
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True, **kwargs)
        return result.stdout.strip()
    except Exception:
        return ""


def run_or_die(cmd, msg):
    """Run a command, exit on failure."""
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    if result.returncode != 0:
        print(f"ERROR: {msg}", file=sys.stderr)
        if result.stderr.strip():
            print(result.stderr.strip(), file=sys.stderr)
        sys.exit(1)
    return result.stdout.strip()


def get_primary_branch():
    """Resolve primary branch via kf-primary-branch."""
    kf_bin = os.path.dirname(os.path.abspath(__file__))
    pb = run(f"{kf_bin}/kf-primary-branch")
    return pb if pb else "main"


def get_worktree_state():
    """Scan git worktrees and classify workers."""
    output = run("git worktree list --porcelain")
    if not output:
        return [], [], []

    worktrees = []
    current = {}
    for line in output.split("\n"):
        if line.startswith("worktree "):
            if current:
                worktrees.append(current)
            current = {"path": line.split(" ", 1)[1]}
        elif line.startswith("HEAD "):
            current["head"] = line.split(" ", 1)[1]
        elif line.startswith("branch "):
            current["branch"] = line.split(" ", 1)[1].replace("refs/heads/", "")
        elif line == "detached":
            current["branch"] = "(detached)"
    if current:
        worktrees.append(current)

    idle = []
    active = []
    unknown = []

    for wt in worktrees:
        folder = os.path.basename(wt["path"])
        branch = wt.get("branch", "")

        # Only consider worker-* or developer-* worktrees
        if not (folder.startswith("worker-") or folder.startswith("developer-")):
            continue

        if branch.startswith("kf/"):
            active.append({"folder": folder, "path": wt["path"], "branch": branch})
        elif branch == folder:
            # On home branch
            idle.append({"folder": folder, "path": wt["path"], "branch": branch})
        else:
            unknown.append({"folder": folder, "path": wt["path"], "branch": branch})

    return idle, active, unknown


def parse_track_status(ref):
    """Parse kf-track status output to get track lists."""
    kf_bin = os.path.dirname(os.path.abspath(__file__))
    output = run(f"{kf_bin}/kf-track list --all --json --ref {ref}")
    if not output:
        return {}, [], [], [], []

    try:
        tracks = json.loads(output)
    except json.JSONDecodeError:
        return {}, [], [], [], []

    all_tracks = {}
    available = []
    blocked = []
    claimed = []
    completed = []

    for track_id, info in tracks.items():
        all_tracks[track_id] = info
        status = info.get("status", "")
        if status == "completed":
            completed.append(track_id)
        elif status == "in-progress":
            claimed.append(track_id)
        elif status == "pending":
            # Will classify as available/blocked after checking deps
            pass

    return all_tracks, available, blocked, claimed, completed


def parse_deps(ref):
    """Parse dependency graph."""
    kf_bin = os.path.dirname(os.path.abspath(__file__))
    output = run(f"{kf_bin}/kf-track deps show --json --ref {ref} 2>/dev/null")
    if not output:
        return {}
    try:
        return json.loads(output)
    except json.JSONDecodeError:
        return {}


def parse_conflicts(ref):
    """Parse conflict pairs."""
    kf_bin = os.path.dirname(os.path.abspath(__file__))
    output = run(f"{kf_bin}/kf-track conflicts list --all --json --ref {ref} 2>/dev/null")
    if not output:
        return {}
    try:
        return json.loads(output)
    except json.JSONDecodeError:
        return {}


def classify_pending(all_tracks, deps, completed_set):
    """Classify pending tracks as available or blocked based on deps."""
    available = []
    blocked = []

    for track_id, info in all_tracks.items():
        if info.get("status") != "pending":
            continue

        track_deps = deps.get(track_id, [])
        unmet = [d for d in track_deps if d not in completed_set]

        if unmet:
            blocked.append({"id": track_id, "unmet_deps": unmet, **info})
        else:
            available.append({"id": track_id, **info})

    return available, blocked


def compute_priority(track, all_tracks, deps, conflicts, claimed_ids, active_types):
    """Compute priority score for an available track."""
    track_id = track["id"]
    score = 0

    # 4a. Unblock factor (3x) — count blocked tracks that depend on this one
    unblock_count = 0
    for other_id, other_deps in deps.items():
        if track_id in other_deps:
            other_status = all_tracks.get(other_id, {}).get("status", "")
            if other_status == "pending":
                unblock_count += 1
    score += unblock_count * 3

    # 4b. Conflict penalty (-2x) — penalize if conflicts with claimed tracks
    conflict_with_claimed = 0
    for pair_key, pair_info in conflicts.items():
        ids = pair_key.split("/")
        if len(ids) == 2:
            if track_id in ids:
                other = ids[0] if ids[1] == track_id else ids[1]
                if other in claimed_ids:
                    conflict_with_claimed += 1
    score -= conflict_with_claimed * 2

    # 4c. Type diversity (+1)
    track_type = track.get("type", "")
    if track_type and track_type not in active_types:
        score += 1

    # Task count for tiebreaker (lower is better, stored as negative fraction)
    task_count = track.get("task_count", track.get("tasks", 99))
    if isinstance(task_count, str):
        try:
            task_count = int(task_count.split("/")[0]) if "/" in task_count else int(task_count)
        except ValueError:
            task_count = 99

    return score, -task_count, track_id  # tuple for stable sort


def get_conflict_pairs(track_id, conflicts):
    """Get all track IDs that conflict with the given track."""
    conflicting = set()
    for pair_key in conflicts:
        ids = pair_key.split("/")
        if len(ids) == 2 and track_id in ids:
            other = ids[0] if ids[1] == track_id else ids[1]
            conflicting.add(other)
    return conflicting


def main():
    import argparse
    parser = argparse.ArgumentParser(description="Kiloforge dispatch — compute worker assignments")
    parser.add_argument("--ref", default=None, help="Branch to read track state from")
    parser.add_argument("--limit", type=int, default=0, help="Max assignments (0=unlimited)")
    parser.add_argument("--json", action="store_true", help="Output as JSON")
    parser.add_argument("--dry-run", action="store_true", help="Add dry-run notice")
    args = parser.parse_args()

    ref = args.ref or get_primary_branch()

    # Scan worktrees
    idle, active, unknown = get_worktree_state()

    # Read track state
    all_tracks, _, _, claimed, completed = parse_track_status(ref)
    if not all_tracks:
        if args.json:
            print(json.dumps({"error": "no_tracks", "message": "No tracks found"}))
        else:
            print("=" * 80)
            print("                    KILOFORGE DISPATCH — NO TRACKS EXIST")
            print("=" * 80)
            print()
            print("No tracks found. Create tracks first:")
            print()
            print("  /kf-architect <feature description>")
            print()
            print("=" * 80)
        return

    claimed_set = set(claimed)
    completed_set = set(completed)

    # Read deps and conflicts
    deps = parse_deps(ref)
    conflicts = parse_conflicts(ref)

    # Classify pending tracks
    available, blocked = classify_pending(all_tracks, deps, completed_set)

    # Get active worker types
    active_types = set()
    for w in active:
        # Try to extract track ID from branch name (kf/{type}/{trackId})
        parts = w["branch"].split("/")
        if len(parts) >= 2:
            active_types.add(parts[1])  # type is second segment

    # Compute priority scores
    scored = []
    for track in available:
        priority = compute_priority(track, all_tracks, deps, conflicts, claimed_set, active_types)
        scored.append((priority, track))

    scored.sort(key=lambda x: x[0], reverse=True)

    # Match workers to tracks
    assignments = []
    deferred = []
    assigned_ids = set()
    assigned_conflicts = set()  # track IDs that conflict with assigned tracks

    for worker in idle:
        if args.limit and len(assignments) >= args.limit:
            break

        best = None
        for priority, track in scored:
            tid = track["id"]
            if tid in assigned_ids:
                continue
            if tid in assigned_conflicts:
                deferred.append({"track": track, "reason": "conflicts with assigned track"})
                continue

            # Check conflict with claimed tracks
            track_conflicts = get_conflict_pairs(tid, conflicts)
            if track_conflicts & claimed_set:
                deferred.append({
                    "track": track,
                    "reason": f"conflicts with claimed: {', '.join(track_conflicts & claimed_set)}"
                })
                continue

            best = (priority, track)
            break

        if best:
            priority, track = best
            tid = track["id"]
            assigned_ids.add(tid)
            # Mark conflicting tracks for deferral
            assigned_conflicts |= get_conflict_pairs(tid, conflicts)

            track_type = track.get("type", "feature")
            unblock_count = priority[0] // 3 if priority[0] > 0 else 0

            assignments.append({
                "worker": worker["folder"],
                "track_id": tid,
                "track_title": track.get("title", ""),
                "track_type": track_type,
                "branch": f"kf/{track_type}/{tid}",
                "priority_score": priority[0],
                "unblocks": unblock_count,
                "command": f'claude --worktree {worker["folder"]} -p "/kf-developer {tid}"',
            })

    unassigned = [w for w in idle if w["folder"] not in {a["worker"] for a in assignments}]

    # Output
    if args.json:
        print(json.dumps({
            "workers": {
                "idle": len(idle),
                "active": len(active),
                "unknown": len(unknown),
                "total": len(idle) + len(active) + len(unknown),
            },
            "tracks": {
                "available": len(available),
                "blocked": len(blocked),
                "claimed": len(claimed),
                "completed": len(completed),
            },
            "active_workers": [
                {"folder": w["folder"], "branch": w["branch"]}
                for w in active
            ],
            "assignments": assignments,
            "deferred": [
                {"track_id": d["track"]["id"], "reason": d["reason"]}
                for d in deferred
            ],
            "blocked_tracks": [
                {"track_id": b["id"], "waiting_on": b["unmet_deps"]}
                for b in blocked
            ],
            "unassigned_workers": [w["folder"] for w in unassigned],
            "dry_run": args.dry_run,
        }, indent=2))
        return

    # Formatted output
    print("=" * 80)
    print("                    KILOFORGE DISPATCH — SWARM ASSIGNMENTS")
    print("=" * 80)
    print()
    print(f"Workers: {len(idle)} idle, {len(active)} active, {len(idle) + len(active) + len(unknown)} total")
    print(f"Tracks:  {len(available)} available, {len(blocked)} blocked, {len(claimed)} in-progress")
    print()

    if active:
        print("--- ACTIVE WORKERS " + "-" * 59)
        print()
        for w in active:
            print(f"  {w['folder']}  →  ACTIVE on {w['branch']}")
        print()

    if assignments:
        print("--- DISPATCH PLAN " + "-" * 60)
        print()
        for a in assignments:
            reason_parts = []
            if a["unblocks"]:
                reason_parts.append(f"unblocks {a['unblocks']} track(s)")
            if a["priority_score"] < 0:
                reason_parts.append("conflict-penalized")
            if not reason_parts:
                reason_parts.append("independent")
            reason = ", ".join(reason_parts)

            print(f"  {a['worker']}  →  {a['branch']}")
            print(f"                   Priority: {reason}")
            print(f"                   Command: {a['command']}")
            print()

    if unassigned:
        print("--- UNASSIGNED " + "-" * 64)
        print()
        for w in unassigned:
            print(f"  {w['folder']}  →  no available tracks")
        print()

    if deferred:
        print("--- DEFERRED (conflict risk) " + "-" * 50)
        print()
        for d in deferred:
            print(f"  {d['track']['id']} — {d['reason']}")
        print()

    if blocked:
        print("--- BLOCKED " + "-" * 67)
        print()
        for b in blocked:
            waiting = ", ".join(b["unmet_deps"])
            print(f"  {b['id']} — waiting on: {waiting}")
        print()

    if not idle:
        print("All workers are active. No dispatch needed.")
        print(f"To add capacity: git worktree add developer-N {ref}")
        print()

    print("=" * 80)

    if args.dry_run:
        print()
        print("DRY RUN — no actions taken. Review the plan above and run the commands manually.")


if __name__ == "__main__":
    main()
