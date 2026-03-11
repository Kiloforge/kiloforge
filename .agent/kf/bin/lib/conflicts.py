"""Conflict risk pairs management for conflicts.yaml.

conflicts.yaml uses a line-per-pair format:
    <id-a>/<id-b>: {"risk":"high","note":"...","added":"2026-03-10"}

Pair keys are strictly ordered (lower ID / higher ID alphabetically).
"""

import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional


HEADER = """\
# Track Conflict Risk Pairs
#
# PROTOCOL:
#   Each line: <id-a>/<id-b>: {"risk":"high|medium|low","note":"...","added":"..."}
#   Pair key is strictly ordered: lower ID / higher ID (only one record per pair).
#
# RULES:
#   - Architect adds pairs when generating tracks that may conflict.
#   - Pairs auto-cleaned when either track completes or is archived.
#   - Only active (pending/in-progress) tracks should have pairs.
#
# TOOL: Use `kf-track conflicts` to manage entries. Do not edit by hand.
#
"""


def pair_key(id_a: str, id_b: str) -> str:
    """Build the canonical pair key (lower/higher alphabetical order)."""
    return f"{min(id_a, id_b)}/{max(id_a, id_b)}"


class ConflictPairs:
    """Read/write interface for conflicts.yaml."""

    def __init__(self, path: Path):
        self.path = path
        self._header_lines: list[str] = []
        self._pairs: dict[str, dict] = {}  # key -> {risk, note, added}
        if self.path.exists():
            self._load()

    def _load(self):
        self._header_lines = []
        self._pairs = {}
        for line in self.path.read_text().splitlines():
            if not line or line.startswith("#"):
                self._header_lines.append(line)
                continue
            colon_idx = line.index(":")
            key = line[:colon_idx].strip()
            json_str = line[colon_idx + 1:].strip()
            self._pairs[key] = json.loads(json_str)

    def ensure(self):
        """Create the file with header if it doesn't exist."""
        if not self.path.exists():
            self.path.parent.mkdir(parents=True, exist_ok=True)
            self.path.write_text(HEADER)
            self._header_lines = HEADER.strip().splitlines()

    def save(self):
        """Write back to disk, sorted alphabetically by pair key."""
        lines = list(self._header_lines)
        if self._pairs:
            lines.append("")
            for key in sorted(self._pairs.keys()):
                lines.append(f"{key}: {json.dumps(self._pairs[key], separators=(',', ':'))}")
        self.path.write_text("\n".join(lines) + "\n")

    def add(self, id_a: str, id_b: str, risk: str = "medium", note: str = ""):
        if id_a == id_b:
            raise ValueError("Cannot create conflict pair with itself")
        key = pair_key(id_a, id_b)
        today = datetime.now(timezone.utc).strftime("%Y-%m-%d")
        self._pairs[key] = {"risk": risk, "note": note, "added": today}

    def remove(self, id_a: str, id_b: str):
        key = pair_key(id_a, id_b)
        self._pairs.pop(key, None)

    def clean_track(self, track_id: str):
        """Remove all pairs involving a specific track ID."""
        to_remove = [k for k in self._pairs if track_id in k.split("/")]
        for k in to_remove:
            del self._pairs[k]

    def list_pairs(self, filter_id: Optional[str] = None) -> dict[str, dict]:
        if filter_id:
            return {k: v for k, v in self._pairs.items() if filter_id in k.split("/")}
        return dict(self._pairs)

    def clean_completed(self, active_ids: set[str]) -> list[str]:
        """Remove pairs where either track is not in active_ids. Returns removed keys."""
        removed = []
        for key in list(self._pairs.keys()):
            id_a, id_b = key.split("/")
            if id_a not in active_ids or id_b not in active_ids:
                del self._pairs[key]
                removed.append(key)
        return removed
