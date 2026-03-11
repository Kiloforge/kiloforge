"""Per-worktree claim lock for Kiloforge agents.

Each worktree can claim at most one track at a time. Claims are recorded
as lock directories under `$(git rev-parse --git-common-dir)/kf-claims/`
so they are visible to all worktrees in the repo.

Structure:
    <git-common-dir>/kf-claims/<worktree-name>/
        info    — JSON: {"track_id": "...", "pid": N, "holder": "...", "started": "..."}

This provides instant claim detection (read files) instead of slow
branch scanning. A worktree lock is acquired before branch checkout
and released after merge or abandonment.

Stale lock detection: if the PID in the info file is dead and the lock
is older than STALE_THRESHOLD seconds, it is auto-cleaned.
"""

import json
import os
import signal
import sys
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

from . import git

STALE_THRESHOLD = 300  # 5 minutes — auto-clean dead-PID locks older than this


def _claims_dir() -> Path:
    """Return the kf-claims directory under the git common dir."""
    common = git.git_common_dir()
    if not common:
        print("ERROR: not in a git repository", file=sys.stderr)
        sys.exit(1)
    return Path(common) / "kf-claims"


def _lock_dir(worktree_name: str) -> Path:
    return _claims_dir() / worktree_name


def _info_path(worktree_name: str) -> Path:
    return _lock_dir(worktree_name) / "info"


def _pid_alive(pid: int) -> bool:
    """Check if a process is alive."""
    try:
        os.kill(pid, 0)
        return True
    except (OSError, ProcessLookupError):
        return False


def _parse_iso(ts: str) -> Optional[float]:
    """Parse ISO timestamp to epoch seconds."""
    try:
        dt = datetime.fromisoformat(ts.replace("Z", "+00:00"))
        return dt.timestamp()
    except (ValueError, TypeError):
        return None


def _check_stale(worktree_name: str) -> bool:
    """Check if a claim lock is stale (dead PID + old enough).

    Returns True if the lock was stale and has been cleaned.
    """
    info_file = _info_path(worktree_name)
    if not info_file.exists():
        return False

    try:
        info = json.loads(info_file.read_text())
    except (json.JSONDecodeError, OSError):
        return False

    pid = info.get("pid")
    if not isinstance(pid, int):
        return False

    if _pid_alive(pid):
        return False  # Process alive — lock is valid

    started = _parse_iso(info.get("started", ""))
    if started is None:
        return False

    age = time.time() - started
    if age >= STALE_THRESHOLD:
        lock_dir = _lock_dir(worktree_name)
        holder = info.get("holder", "unknown")
        track_id = info.get("track_id", "unknown")
        print(
            f"Auto-cleaning stale claim: worktree={worktree_name} "
            f"holder={holder} track={track_id} pid={pid} age={int(age)}s",
            file=sys.stderr,
        )
        try:
            info_file.unlink(missing_ok=True)
            lock_dir.rmdir()
        except OSError:
            pass
        return True

    return False


def acquire(
    worktree_name: str,
    track_id: str,
    holder: Optional[str] = None,
    pid: Optional[int] = None,
) -> bool:
    """Acquire a worktree claim lock for a track.

    Returns True if acquired, False if the worktree is already claimed.
    """
    if holder is None:
        holder = worktree_name
    if pid is None:
        pid = os.getpid()

    lock_dir = _lock_dir(worktree_name)

    # Try to create the lock directory
    try:
        lock_dir.mkdir(parents=True, exist_ok=False)
    except FileExistsError:
        # Lock exists — check for stale
        if _check_stale(worktree_name):
            # Stale lock cleaned — retry
            try:
                lock_dir.mkdir(parents=True, exist_ok=False)
            except FileExistsError:
                return False
        else:
            return False

    # Write claim info
    now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    info = {
        "track_id": track_id,
        "pid": pid,
        "holder": holder,
        "started": now,
    }
    _info_path(worktree_name).write_text(json.dumps(info, indent=2) + "\n")
    return True


def release(worktree_name: str, holder: Optional[str] = None) -> bool:
    """Release a worktree claim lock.

    If holder is specified, validates it matches before releasing.
    Returns True if released, False if not found or holder mismatch.
    """
    info_file = _info_path(worktree_name)
    lock_dir = _lock_dir(worktree_name)

    if not lock_dir.exists():
        return True  # Already unlocked

    # Validate holder if specified
    if holder and info_file.exists():
        try:
            info = json.loads(info_file.read_text())
            lock_holder = info.get("holder", "")
            if lock_holder and lock_holder != holder:
                print(
                    f"ERROR: Cannot release claim — held by '{lock_holder}', not '{holder}'",
                    file=sys.stderr,
                )
                return False
        except (json.JSONDecodeError, OSError):
            pass

    try:
        info_file.unlink(missing_ok=True)
        lock_dir.rmdir()
    except OSError:
        pass
    return True


def read_claim(worktree_name: str) -> Optional[dict]:
    """Read the claim info for a worktree.

    Returns the info dict or None if no claim exists.
    """
    info_file = _info_path(worktree_name)
    if not info_file.exists():
        return None
    try:
        return json.loads(info_file.read_text())
    except (json.JSONDecodeError, OSError):
        return None


def list_claims() -> dict[str, dict]:
    """List all active worktree claims.

    Returns {worktree_name: info_dict} for all claimed worktrees.
    Auto-cleans stale locks during enumeration.
    """
    claims_dir = _claims_dir()
    if not claims_dir.exists():
        return {}

    result = {}
    for entry in sorted(claims_dir.iterdir()):
        if not entry.is_dir():
            continue
        wt_name = entry.name

        # Auto-clean stale locks during scan
        _check_stale(wt_name)

        info = read_claim(wt_name)
        if info:
            result[wt_name] = info

    return result


def find_track_claim(track_id: str) -> Optional[tuple[str, dict]]:
    """Find which worktree (if any) has claimed a specific track.

    Returns (worktree_name, info_dict) or None.
    """
    for wt_name, info in list_claims().items():
        if info.get("track_id") == track_id:
            return wt_name, info
    return None


def claimed_track_ids() -> list[tuple[str, str]]:
    """Return all claimed (track_id, worktree_name) pairs.

    This is the fast replacement for branch_scan_claimed().
    """
    return [
        (info["track_id"], wt_name)
        for wt_name, info in list_claims().items()
        if "track_id" in info
    ]
