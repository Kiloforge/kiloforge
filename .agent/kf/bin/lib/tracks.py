"""Track registry management for tracks.yaml.

tracks.yaml uses a custom line-per-track format:
    <track-id>: {"title":"...","status":"...","type":"...","created":"...","updated":"..."}

This module provides read/write operations for this format.
"""

import json
import os
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional


HEADER = """\
# Kiloforge Track Registry
#
# FORMAT: <track-id>: {"title":"...","status":"...","type":"...","created":"...","updated":"..."}
# STATUS: pending | in-progress | completed | archived
# ORDER:  Lines sorted alphabetically by track ID. JSON fields in canonical order:
#         title, status, type, created, updated [, archived_at, archive_reason]
# TOOL:   Use `kf-track` to manage entries. Do not edit by hand.
#
"""

CANONICAL_FIELDS = ["title", "status", "type", "created", "updated"]
OPTIONAL_FIELDS = ["archived_at", "archive_reason"]


def today_iso() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%d")


def now_iso() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def normalize_json(data: dict) -> str:
    """Serialize track data with canonical field order for stable diffs."""
    ordered = {}
    for key in CANONICAL_FIELDS:
        if key in data:
            ordered[key] = data[key]
    for key in OPTIONAL_FIELDS:
        if key in data:
            ordered[key] = data[key]
    # Append any extra fields not in canonical/optional lists
    for key in data:
        if key not in ordered:
            ordered[key] = data[key]
    return json.dumps(ordered, separators=(",", ":"))


class TracksRegistry:
    """Read/write interface for tracks.yaml."""

    def __init__(self, path: Path):
        self.path = path
        self._header_lines: list[str] = []
        self._entries: dict[str, dict] = {}
        if self.path.exists():
            self._load()

    def _load(self):
        self._header_lines = []
        self._entries = {}
        for line in self.path.read_text().splitlines():
            if not line or line.startswith("#"):
                self._header_lines.append(line)
                continue
            colon_idx = line.index(":")
            track_id = line[:colon_idx].strip()
            json_str = line[colon_idx + 1:].strip()
            self._entries[track_id] = json.loads(json_str)

    @classmethod
    def from_text(cls, text: str, path: Optional[Path] = None) -> "TracksRegistry":
        """Parse tracks.yaml content from a string (e.g., git show output)."""
        reg = cls.__new__(cls)
        reg.path = path
        reg._header_lines = []
        reg._entries = {}
        for line in text.splitlines():
            if not line or line.startswith("#"):
                reg._header_lines.append(line)
                continue
            colon_idx = line.index(":")
            track_id = line[:colon_idx].strip()
            json_str = line[colon_idx + 1:].strip()
            reg._entries[track_id] = json.loads(json_str)
        return reg

    def ensure(self):
        """Create the file with header if it doesn't exist."""
        if not self.path.exists():
            self.path.parent.mkdir(parents=True, exist_ok=True)
            self.path.write_text(HEADER)
            self._header_lines = HEADER.strip().splitlines()

    def save(self):
        """Write back to disk, sorted alphabetically by track ID."""
        lines = list(self._header_lines)
        if self._entries:
            lines.append("")
            for track_id in sorted(self._entries.keys()):
                lines.append(f"{track_id}: {normalize_json(self._entries[track_id])}")
        self.path.write_text("\n".join(lines) + "\n")

    def exists(self, track_id: str) -> bool:
        return track_id in self._entries

    def get(self, track_id: str) -> Optional[dict]:
        return self._entries.get(track_id)

    def get_field(self, track_id: str, field: str) -> Optional[str]:
        entry = self._entries.get(track_id)
        if entry is None:
            return None
        return entry.get(field)

    def set_field(self, track_id: str, field: str, value: str):
        entry = self._entries.get(track_id)
        if entry is None:
            raise KeyError(f"Track not found: {track_id}")
        entry[field] = value
        if field != "updated":
            entry["updated"] = today_iso()

    def add(self, track_id: str, title: str, type_: str = "feature",
            status: str = "pending") -> dict:
        if track_id in self._entries:
            raise ValueError(f"Track already exists: {track_id}")
        created = today_iso()
        entry = {
            "title": title,
            "status": status,
            "type": type_,
            "created": created,
            "updated": created,
        }
        self._entries[track_id] = entry
        return entry

    def update_status(self, track_id: str, status: str):
        valid = ("pending", "in-progress", "completed", "archived")
        if status not in valid:
            raise ValueError(f"Invalid status: {status} (must be {', '.join(valid)})")
        self.set_field(track_id, "status", status)
        if status == "archived":
            self.set_field(track_id, "archived_at", today_iso())

    def list_by_status(self, *statuses: str) -> dict[str, dict]:
        return {
            tid: data for tid, data in self._entries.items()
            if data.get("status") in statuses
        }

    def list_active(self) -> dict[str, dict]:
        return self.list_by_status("pending", "in-progress")

    def all_entries(self) -> dict[str, dict]:
        return dict(self._entries)

    def ids(self) -> list[str]:
        return list(self._entries.keys())
