#!/usr/bin/env python3
# kf-merge-lock — Cross-worktree branch lock helper
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

import json
import os
import shutil
import subprocess
import sys
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

ORCH_URL = os.environ.get("KF_ORCH_URL", "http://localhost:39517")
DEFAULT_HOLDER = os.environ.get("KF_LOCK_HOLDER", os.path.basename(os.getcwd()))
DEFAULT_TTL = 120
DEFAULT_TIMEOUT = 0
STALE_THRESHOLD = 240  # 2x default TTL — auto-clean mkdir locks older than this

# Acquire return codes (internal)
ACQUIRED = 0
HELD = 1       # Lock held by another worker
CONN_ERR = 2   # Connection error (HTTP unavailable)


# --- Mode detection ---

def is_orch_running() -> bool:
    """Check if the orchestrator is reachable via health endpoint."""
    try:
        req = urllib.request.Request(f"{ORCH_URL}/health", method="GET")
        urllib.request.urlopen(req, timeout=5)
        return True
    except Exception:
        return False


def get_lock_dir() -> Path:
    """Return the path to the mkdir-mode lock directory."""
    try:
        common_dir = subprocess.check_output(
            ["git", "rev-parse", "--git-common-dir"],
            stderr=subprocess.DEVNULL, text=True
        ).strip()
    except subprocess.CalledProcessError:
        try:
            common_dir = subprocess.check_output(
                ["git", "rev-parse", "--git-dir"],
                stderr=subprocess.DEVNULL, text=True
            ).strip()
        except subprocess.CalledProcessError:
            common_dir = ".git"
    return Path(common_dir) / "merge.lock"


# --- HTTP mode helpers ---

def _http_json_request(method: str, path: str, data: dict | None = None,
                       timeout: int = 10) -> tuple[int, str]:
    """Make an HTTP request and return (status_code, body)."""
    url = f"{ORCH_URL}{path}"
    body_bytes = json.dumps(data).encode() if data is not None else None
    req = urllib.request.Request(url, data=body_bytes, method=method)
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
    """Attempt HTTP acquire (non-blocking). Returns (ACQUIRED|HELD|CONN_ERR, holder_info)."""
    status, body = _http_json_request(
        "POST", "/api/locks/merge/acquire",
        {"holder": holder, "ttl_seconds": ttl, "timeout_seconds": 0},
        timeout=10,
    )
    if status == 0:
        return CONN_ERR, ""
    if status == 200:
        return ACQUIRED, ""
    # Lock held by someone else
    current_holder = "unknown"
    try:
        parsed = json.loads(body)
        current_holder = parsed.get("current_holder", "unknown")
    except Exception:
        pass
    return HELD, current_holder


def http_release(holder: str) -> int:
    _http_json_request("DELETE", "/api/locks/merge", {"holder": holder})
    print("Branch lock released (HTTP)")
    return 0


def http_heartbeat(holder: str, ttl: int) -> int:
    status, _ = _http_json_request(
        "POST", "/api/locks/merge/heartbeat",
        {"holder": holder, "ttl_seconds": ttl},
    )
    if status == 0 or status >= 400:
        print("WARNING: Heartbeat failed", file=sys.stderr)
        return 1
    return 0


def http_status() -> int:
    status, body = _http_json_request("GET", "/api/locks")
    if status == 0:
        print("Mode:   HTTP (orchestrator unreachable)")
        return 1

    try:
        data = json.loads(body)
    except Exception:
        data = []

    # data could be a list of lock objects or a dict
    locks = data if isinstance(data, list) else [data] if data else []
    count = len(locks)

    print("Mode:   HTTP")
    if count == 0:
        print("Status: No active lock")
        return 0

    print(f"Locks:  {count} active")
    print()
    # Show first merge lock
    for lock in locks:
        if isinstance(lock, dict):
            holder = lock.get("holder", "")
            expires = lock.get("expires_at", "")
            if holder:
                print("  Scope:   branch")
                print(f"  Holder:  {holder}")
                print(f"  Expires: {expires}")
                break
    return 0


# --- mkdir mode helpers ---

def _utcnow_iso() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def _write_lock_info(lock_dir: Path, pid: int, holder: str) -> None:
    info_file = lock_dir / "info"
    info_file.write_text(f"{pid} {_utcnow_iso()} {holder}\n")


def _parse_lock_info(lock_dir: Path) -> tuple[str, str, str] | None:
    """Return (pid_str, timestamp, holder) or None."""
    info_file = lock_dir / "info"
    if not info_file.is_file():
        return None
    try:
        text = info_file.read_text().strip()
    except OSError:
        return None
    parts = text.split(None, 2)
    if len(parts) < 3:
        return None
    return parts[0], parts[1], parts[2]


def _pid_alive(pid: int) -> bool:
    """Check if a process is alive via os.kill(pid, 0)."""
    try:
        os.kill(pid, 0)
        return True
    except (OSError, ProcessLookupError):
        return False


def _parse_iso_timestamp(ts: str) -> float | None:
    """Parse an ISO 8601 UTC timestamp to epoch seconds."""
    try:
        dt = datetime.strptime(ts, "%Y-%m-%dT%H:%M:%SZ")
        dt = dt.replace(tzinfo=timezone.utc)
        return dt.timestamp()
    except (ValueError, TypeError):
        return None


def _mkdir_check_stale(lock_dir: Path) -> bool:
    """Check if a lock is stale (dead PID + old enough). Returns True if cleaned."""
    info = _parse_lock_info(lock_dir)
    if info is None:
        return False

    pid_str, timestamp, holder = info

    # Validate PID is numeric
    if not pid_str.isdigit():
        return False
    lock_pid = int(pid_str)

    # Check if PID is alive
    if _pid_alive(lock_pid):
        return False  # Process alive — lock is valid

    # PID is dead — check age threshold
    lock_epoch = _parse_iso_timestamp(timestamp)
    if lock_epoch is None:
        return False
    now_epoch = time.time()
    age = int(now_epoch - lock_epoch)

    if age >= STALE_THRESHOLD:
        print(
            f"Auto-cleaning stale lock: holder={holder} pid={lock_pid} age={age}s (dead process)",
            file=sys.stderr,
        )
        shutil.rmtree(lock_dir, ignore_errors=True)
        return True

    # PID dead but lock is recent — don't auto-clean (might be a race)
    print(
        f"WARNING: Lock holder PID {lock_pid} is dead (age={age}s < {STALE_THRESHOLD}s threshold) — not auto-cleaning",
        file=sys.stderr,
    )
    return False


def _try_mkdir(lock_dir: Path) -> bool:
    """Attempt atomic mkdir. Returns True on success."""
    try:
        lock_dir.mkdir(parents=False, exist_ok=False)
        return True
    except FileExistsError:
        return False


def _mkdir_try_acquire(lock_dir: Path, holder: str, pid: int) -> tuple[int, str]:
    """Attempt mkdir acquire once. Returns (ACQUIRED|HELD, holder_info)."""
    if _try_mkdir(lock_dir):
        _write_lock_info(lock_dir, pid, holder)
        return ACQUIRED, ""

    # Lock exists — check for stale lock
    if _mkdir_check_stale(lock_dir):
        if _try_mkdir(lock_dir):
            _write_lock_info(lock_dir, pid, holder)
            return ACQUIRED, ""

    # Lock is held
    info = _parse_lock_info(lock_dir)
    info_str = f"{info[2]}" if info else "unknown"
    return HELD, info_str


def mkdir_release(holder: str) -> int:
    lock_dir = get_lock_dir()

    if not lock_dir.is_dir():
        print("Branch lock released (mkdir, already unlocked)")
        return 0

    # Validate holder before releasing
    info = _parse_lock_info(lock_dir)
    if info is not None:
        lock_holder = info[2]
        if lock_holder and lock_holder != holder:
            print(
                f"ERROR: Cannot release lock — held by '{lock_holder}', not '{holder}'",
                file=sys.stderr,
            )
            return 1

    shutil.rmtree(lock_dir, ignore_errors=True)
    print("Branch lock released (mkdir)")
    return 0


def mkdir_heartbeat(holder: str) -> int:
    lock_dir = get_lock_dir()
    info_file = lock_dir / "info"
    if lock_dir.is_dir() and info_file.is_file():
        info = _parse_lock_info(lock_dir)
        if info is not None:
            original_pid = info[0]
            try:
                info_file.write_text(f"{original_pid} {_utcnow_iso()} {holder}\n")
            except OSError:
                pass
    return 0


def mkdir_status() -> int:
    lock_dir = get_lock_dir()

    print("Mode:   mkdir (fallback)")

    if not lock_dir.is_dir():
        print("Status: No active lock")
        return 0

    info = _parse_lock_info(lock_dir)
    if info is None:
        print("Status: LOCKED")
        print()
        print("  Info: unknown")
        return 0

    pid_str, timestamp, holder = info

    print("Status: LOCKED")
    print()
    print(f"  Holder:    {holder}")
    print(f"  PID:       {pid_str}")
    print(f"  Acquired:  {timestamp}")

    if not pid_str.isdigit():
        print("  PID alive: unknown (not a valid PID)")
    elif _pid_alive(int(pid_str)):
        print("  PID alive: yes")
    else:
        print("  PID alive: NO — process is dead (lock may be stale)")

    return 0


# --- Unified acquire ---

def _try_acquire_once(holder: str, ttl: int, pid: int) -> tuple[int, str, str]:
    """Single acquire attempt: HTTP first, mkdir fallback.

    Returns (ACQUIRED|HELD|CONN_ERR, holder_info, mode).
    """
    lock_dir = get_lock_dir()

    # Try HTTP first
    if is_orch_running():
        result, held_by = _http_try_acquire(holder, ttl)
        if result == ACQUIRED:
            return ACQUIRED, "", "HTTP"
        if result == HELD:
            return HELD, held_by, "HTTP"
        # CONN_ERR — fall through to mkdir

    # mkdir fallback
    result, held_by = _mkdir_try_acquire(lock_dir, holder, pid)
    if result == ACQUIRED:
        return ACQUIRED, "", "mkdir"
    return HELD, held_by, "mkdir"


def unified_acquire(holder: str, ttl: int, timeout: int, pid: int) -> int:
    """Try HTTP first, fall back to mkdir. Poll at 1s intervals if held.

    Returns 0 on success, 1 on held/timeout.
    """
    result, held_by, mode = _try_acquire_once(holder, ttl, pid)
    if result == ACQUIRED:
        print(f"Branch lock acquired ({mode})")
        return 0

    # Lock held — fail immediately if no timeout
    if timeout == 0:
        print(f"BRANCH LOCK HELD by {held_by}", file=sys.stderr)
        return 1

    # Poll at 1s intervals
    elapsed = 0
    interval = 1
    log_interval = 10  # only log every 10s to avoid spam
    while elapsed < timeout:
        if elapsed % log_interval == 0:
            print(f"Branch lock held by {held_by} — waiting... ({elapsed}s/{timeout}s)", file=sys.stderr)
        time.sleep(interval)
        elapsed += interval

        result, held_by, mode = _try_acquire_once(holder, ttl, pid)
        if result == ACQUIRED:
            print(f"Branch lock acquired ({mode}) after {elapsed}s")
            return 0

    print(f"BRANCH LOCK TIMEOUT — could not acquire after {timeout}s", file=sys.stderr)
    return 1


# --- Main command dispatch ---

def cmd_acquire(args: list[str]) -> int:
    holder = DEFAULT_HOLDER
    ttl = DEFAULT_TTL
    timeout = DEFAULT_TIMEOUT
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

    return unified_acquire(holder, ttl, timeout, pid)


def cmd_release(args: list[str]) -> int:
    holder = DEFAULT_HOLDER

    i = 0
    while i < len(args):
        if args[i] == "--holder" and i + 1 < len(args):
            holder = args[i + 1]; i += 2
        else:
            print(f"Unknown option: {args[i]}", file=sys.stderr)
            return 1

    if is_orch_running():
        return http_release(holder)
    else:
        return mkdir_release(holder)


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

    if is_orch_running():
        return http_heartbeat(holder, ttl)
    else:
        return mkdir_heartbeat(holder)


def cmd_status() -> int:
    print("===============================================")
    print("          BRANCH LOCK STATUS")
    print("===============================================")
    print()

    if is_orch_running():
        rc = http_status()
    else:
        rc = mkdir_status()

    print()
    print("===============================================")
    return rc


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

  Acquire tries HTTP first. If the orchestrator is unreachable (or becomes
  unreachable during a wait), it falls back to mkdir automatically.

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
