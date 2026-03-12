"""Cross-worktree merge lock for Kiloforge agents.

Coordinates merges to the primary branch across worktrees. Tries HTTP
(orchestrator) first, falls back to mkdir (filesystem) seamlessly.

The lock directory lives at:
    $(git rev-parse --git-common-dir)/merge.lock/

Usage from Python:
    from lib import merge_lock

    acquired = merge_lock.acquire("my-holder")
    if not acquired:
        print("lock held")
    # ... do merge ...
    merge_lock.release("my-holder")

    # Non-blocking status check (no acquire):
    info = merge_lock.status()
    # info is None (unlocked) or {"holder": ..., "pid": ..., ...}
"""

import json
import os
import shutil
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

from . import git

ORCH_URL = os.environ.get("KF_ORCH_URL", "http://localhost:39517")
DEFAULT_TTL = 120
STALE_THRESHOLD = 240  # 2x default TTL


# ---------------------------------------------------------------------------
# Internal constants
# ---------------------------------------------------------------------------

_ACQUIRED = 0
_HELD = 1
_CONN_ERR = 2


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _lock_dir() -> Path:
    common = git.git_common_dir()
    if not common:
        return Path(".git") / "merge.lock"
    return Path(common) / "merge.lock"


def _utcnow_iso() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def _pid_alive(pid: int) -> bool:
    try:
        os.kill(pid, 0)
        return True
    except (OSError, ProcessLookupError):
        return False


def _parse_iso(ts: str) -> Optional[float]:
    try:
        dt = datetime.strptime(ts, "%Y-%m-%dT%H:%M:%SZ")
        dt = dt.replace(tzinfo=timezone.utc)
        return dt.timestamp()
    except (ValueError, TypeError):
        return None


def _is_orch_running() -> bool:
    try:
        req = urllib.request.Request(f"{ORCH_URL}/health", method="GET")
        urllib.request.urlopen(req, timeout=5)
        return True
    except Exception:
        return False


# ---------------------------------------------------------------------------
# HTTP mode
# ---------------------------------------------------------------------------

def _http_request(method: str, path: str, data: Optional[dict] = None,
                  timeout: int = 10) -> tuple[int, str]:
    url = f"{ORCH_URL}{path}"
    body = json.dumps(data).encode() if data is not None else None
    req = urllib.request.Request(url, data=body, method=method)
    if data is not None:
        req.add_header("Content-Type", "application/json")
    try:
        resp = urllib.request.urlopen(req, timeout=timeout)
        return resp.status, resp.read().decode()
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode()
    except Exception:
        return 0, ""


def _http_try_acquire(holder: str, ttl: int) -> tuple[int, str]:
    status, body = _http_request(
        "POST", "/api/locks/merge/acquire",
        {"holder": holder, "ttl_seconds": ttl, "timeout_seconds": 0},
    )
    if status == 0:
        return _CONN_ERR, ""
    if status == 200:
        return _ACQUIRED, ""
    current_holder = "unknown"
    try:
        current_holder = json.loads(body).get("current_holder", "unknown")
    except Exception:
        pass
    return _HELD, current_holder


def _http_release(holder: str) -> bool:
    _http_request("DELETE", "/api/locks/merge", {"holder": holder})
    return True


def _http_heartbeat(holder: str, ttl: int) -> bool:
    status, _ = _http_request(
        "POST", "/api/locks/merge/heartbeat",
        {"holder": holder, "ttl_seconds": ttl},
    )
    return status != 0 and status < 400


def _http_status() -> Optional[dict]:
    status, body = _http_request("GET", "/api/locks")
    if status == 0:
        return None
    try:
        data = json.loads(body)
    except Exception:
        return None
    locks = data if isinstance(data, list) else [data] if data else []
    for lock in locks:
        if isinstance(lock, dict) and lock.get("holder"):
            return {
                "holder": lock.get("holder", "unknown"),
                "expires": lock.get("expires_at", ""),
                "mode": "http",
            }
    return None


# ---------------------------------------------------------------------------
# mkdir mode
# ---------------------------------------------------------------------------

def _parse_lock_info(lock_dir: Path) -> Optional[tuple[str, str, str]]:
    """Return (pid_str, timestamp, holder) or None."""
    info_file = lock_dir / "info"
    if not info_file.is_file():
        return None
    try:
        parts = info_file.read_text().strip().split(None, 2)
        return (parts[0], parts[1], parts[2]) if len(parts) >= 3 else None
    except OSError:
        return None


def _mkdir_check_stale(lock_dir: Path) -> bool:
    """Check if lock is stale (dead PID + old enough). Returns True if cleaned."""
    info = _parse_lock_info(lock_dir)
    if info is None:
        return False
    pid_str, timestamp, _holder = info
    if not pid_str.isdigit():
        return False
    if _pid_alive(int(pid_str)):
        return False
    lock_epoch = _parse_iso(timestamp)
    if lock_epoch is None:
        return False
    if (time.time() - lock_epoch) >= STALE_THRESHOLD:
        shutil.rmtree(lock_dir, ignore_errors=True)
        return True
    return False


def _mkdir_try_acquire(lock_dir: Path, holder: str, pid: int) -> tuple[int, str]:
    try:
        lock_dir.mkdir(parents=False, exist_ok=False)
    except FileExistsError:
        # Re-entry: if same holder already holds the lock, succeed (idempotent)
        info = _parse_lock_info(lock_dir)
        if info is not None and info[2] == holder:
            # Refresh timestamp
            (lock_dir / "info").write_text(f"{pid} {_utcnow_iso()} {holder}\n")
            return _ACQUIRED, ""
        if _mkdir_check_stale(lock_dir):
            try:
                lock_dir.mkdir(parents=False, exist_ok=False)
            except FileExistsError:
                info = _parse_lock_info(lock_dir)
                return _HELD, info[2] if info else "unknown"
        else:
            return _HELD, info[2] if info else "unknown"

    (lock_dir / "info").write_text(f"{pid} {_utcnow_iso()} {holder}\n")
    return _ACQUIRED, ""


def _mkdir_release(holder: str) -> bool:
    lock_dir = _lock_dir()
    if not lock_dir.is_dir():
        return True
    info = _parse_lock_info(lock_dir)
    if info is not None:
        lock_holder = info[2]
        if lock_holder and lock_holder != holder:
            return False
    shutil.rmtree(lock_dir, ignore_errors=True)
    return True


def _mkdir_heartbeat(holder: str) -> bool:
    lock_dir = _lock_dir()
    info = _parse_lock_info(lock_dir)
    if info is not None:
        try:
            (lock_dir / "info").write_text(f"{info[0]} {_utcnow_iso()} {holder}\n")
        except OSError:
            return False
    return True


def _mkdir_status() -> Optional[dict]:
    lock_dir = _lock_dir()
    if not lock_dir.is_dir():
        return None
    info = _parse_lock_info(lock_dir)
    if info is None:
        return {"holder": "unknown", "mode": "mkdir"}
    pid_str, timestamp, holder = info
    return {
        "holder": holder,
        "pid": int(pid_str) if pid_str.isdigit() else None,
        "acquired": timestamp,
        "pid_alive": _pid_alive(int(pid_str)) if pid_str.isdigit() else None,
        "mode": "mkdir",
    }


# ---------------------------------------------------------------------------
# Unified acquire attempt (try HTTP, fall back to mkdir)
# ---------------------------------------------------------------------------

def _try_once(holder: str, ttl: int, pid: int) -> tuple[int, str, str]:
    """Single attempt. Returns (_ACQUIRED|_HELD, holder_info, mode)."""
    if _is_orch_running():
        result, held_by = _http_try_acquire(holder, ttl)
        if result == _ACQUIRED:
            return _ACQUIRED, "", "http"
        if result == _HELD:
            return _HELD, held_by, "http"
    # mkdir fallback
    result, held_by = _mkdir_try_acquire(_lock_dir(), holder, pid)
    if result == _ACQUIRED:
        return _ACQUIRED, "", "mkdir"
    return _HELD, held_by, "mkdir"


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------

def acquire(
    holder: str,
    ttl: int = DEFAULT_TTL,
    timeout: int = 0,
    pid: Optional[int] = None,
) -> bool:
    """Acquire the merge lock.

    Args:
        holder: Identity of the lock holder.
        ttl: Lock time-to-live in seconds (HTTP mode).
        timeout: 0 = fail immediately if held, >0 = poll up to this many seconds.
        pid: PID to record for stale detection (default: current PID).

    Returns True if acquired, False if held by another worker.
    """
    if pid is None:
        pid = os.getpid()

    result, _held_by, _mode = _try_once(holder, ttl, pid)
    if result == _ACQUIRED:
        return True

    if timeout <= 0:
        return False

    elapsed = 0
    while elapsed < timeout:
        time.sleep(1)
        elapsed += 1
        result, _held_by, _mode = _try_once(holder, ttl, pid)
        if result == _ACQUIRED:
            return True

    return False


def release(holder: str) -> bool:
    """Release the merge lock. Returns True if released."""
    if _is_orch_running():
        return _http_release(holder)
    return _mkdir_release(holder)


def heartbeat(holder: str, ttl: int = DEFAULT_TTL) -> bool:
    """Send a heartbeat to keep the lock alive. Returns True on success."""
    if _is_orch_running():
        return _http_heartbeat(holder, ttl)
    return _mkdir_heartbeat(holder)


def status() -> Optional[dict]:
    """Check lock status without acquiring.

    Returns None if unlocked, or a dict with at least {"holder": str, "mode": str}.
    """
    if _is_orch_running():
        return _http_status()
    return _mkdir_status()


def is_locked() -> bool:
    """Quick check: is the merge lock currently held?"""
    return status() is not None
