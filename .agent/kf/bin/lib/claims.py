"""Track claim detection via worktree locks, orchestrator API, and git branches.

Detection priority (fastest first):
1. Worktree claim locks (instant — filesystem read)
2. Orchestrator API (fast — HTTP call)
3. Git branch scan (slow — lists branches + worktrees)
"""

import json
import os
import urllib.request
from typing import Optional
from . import git
from . import worktree_lock


def worktree_lock_claimed() -> list[tuple[str, str]]:
    """Read claims from worktree lock files (instant).

    Returns list of (track_id, worker_name) tuples.
    """
    return worktree_lock.claimed_track_ids()


def branch_scan_claimed() -> list[tuple[str, str]]:
    """Scan git branches for implementation branches (slow fallback).

    Returns list of (track_id, worker_name) tuples.
    Worker name is extracted from worktree folder name if identifiable.
    """
    branches = git.branches_matching(
        "kf/feature/*", "kf/bug/*", "kf/chore/*", "kf/refactor/*",
        "feature/*", "bug/*", "chore/*", "refactor/*",
    )
    if not branches:
        return []

    # Build worktree map: branch -> worktree path
    worktree_map = {}
    for wt in git.worktree_list():
        branch = wt.get("branch", "")
        if branch:
            worktree_map[branch] = wt["path"]

    results = []
    for branch in branches:
        # Extract track ID: strip type prefix (feature/, kf/feature/, etc.)
        parts = branch.split("/")
        track_id = parts[-1] if len(parts) > 1 else branch

        worker = ""
        wt_path = worktree_map.get(branch, "")
        if wt_path:
            worker = os.path.basename(wt_path)

        results.append((track_id, worker))

    return results


def server_query_claims() -> list[tuple[str, str]]:
    """Query orchestrator claim API for claimed tracks.

    Returns list of (track_id, worker_name) tuples.
    Returns empty list if server is unreachable.
    """
    orch_url = os.environ.get("KF_ORCH_URL", "http://localhost:39517")
    try:
        req = urllib.request.Request(
            f"{orch_url}/api/tracks/claims",
            method="GET",
        )
        with urllib.request.urlopen(req, timeout=1) as resp:
            data = json.loads(resp.read())
            return [(c["track_id"], c.get("worker", "")) for c in data]
    except Exception:
        return []


def get_claimed_tracks() -> list[tuple[str, str]]:
    """Get claimed tracks from all sources, deduplicated.

    Priority: worktree locks > server > branch scan.
    Returns list of (track_id, worker_name) tuples.
    """
    seen = set()
    results = []

    # 1. Worktree claim locks (instant)
    for tid, worker in worktree_lock_claimed():
        if tid not in seen:
            seen.add(tid)
            results.append((tid, worker))

    # 2. Orchestrator API
    for tid, worker in server_query_claims():
        if tid not in seen:
            seen.add(tid)
            results.append((tid, worker))

    # 3. Branch scan (slowest — only if no claims found yet)
    if not results:
        for tid, worker in branch_scan_claimed():
            if tid not in seen:
                seen.add(tid)
                results.append((tid, worker))

    return results


def is_track_claimed(track_id: str) -> tuple[bool, Optional[str]]:
    """Check if a specific track is claimed.

    Returns (is_claimed, worker_name).
    """
    # Fast path: check worktree locks first
    claim = worktree_lock.find_track_claim(track_id)
    if claim:
        wt_name, info = claim
        return True, info.get("holder", wt_name)

    # Full search
    claims = get_claimed_tracks()
    for tid, worker in claims:
        if tid == track_id:
            return True, worker
    return False, None
