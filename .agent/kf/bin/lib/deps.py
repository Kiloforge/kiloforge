"""Dependency graph management for deps.yaml.

deps.yaml is a standard YAML adjacency list:
    track-id-a: []
    track-id-b:
      - track-id-a
"""

import yaml
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional


HEADER = """\
# Track Dependency Graph
#
# PROTOCOL:
#   Canonical source for track dependency ordering (adjacency list).
#   Each key is a track ID; its value is a list of prerequisite track IDs.
#
# RULES:
#   - Only pending/in-progress tracks listed. Completed tracks pruned on cleanup.
#   - Architect appends entries when creating tracks.
#   - Developer checks deps before claiming: all deps must be completed.
#   - Cycles are forbidden.
#
# UPDATED: {timestamp}
"""


class DepsGraph:
    """Read/write interface for deps.yaml."""

    def __init__(self, path: Path):
        self.path = path
        self._header: str = ""
        self._graph: dict[str, list[str]] = {}
        if self.path.exists():
            self._load()

    def _load(self):
        text = self.path.read_text()
        # Separate header comments from data
        header_lines = []
        data_lines = []
        for line in text.splitlines():
            if line.startswith("#") or (not data_lines and not line.strip()):
                header_lines.append(line)
            else:
                data_lines.append(line)
        self._header = "\n".join(header_lines)
        data_text = "\n".join(data_lines)
        parsed = yaml.safe_load(data_text) if data_text.strip() else None
        self._graph = parsed if isinstance(parsed, dict) else {}
        # Normalize: ensure all values are lists
        for key in self._graph:
            if self._graph[key] is None:
                self._graph[key] = []

    @classmethod
    def from_text(cls, text: str, path: Optional[Path] = None) -> "DepsGraph":
        """Parse deps.yaml content from a string."""
        dg = cls.__new__(cls)
        dg.path = path
        header_lines = []
        data_lines = []
        for line in text.splitlines():
            if line.startswith("#") or (not data_lines and not line.strip()):
                header_lines.append(line)
            else:
                data_lines.append(line)
        dg._header = "\n".join(header_lines)
        data_text = "\n".join(data_lines)
        parsed = yaml.safe_load(data_text) if data_text.strip() else None
        dg._graph = parsed if isinstance(parsed, dict) else {}
        for key in dg._graph:
            if dg._graph[key] is None:
                dg._graph[key] = []
        return dg

    def ensure(self):
        """Create the file with header if it doesn't exist."""
        if not self.path.exists():
            self.path.parent.mkdir(parents=True, exist_ok=True)
            ts = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
            self.path.write_text(HEADER.format(timestamp=ts))
            self._header = HEADER.format(timestamp=ts).strip()

    def save(self):
        """Write back to disk, sorted alphabetically."""
        lines = [self._header, ""]
        for tid in sorted(self._graph.keys()):
            deps = sorted(self._graph[tid])
            if not deps:
                lines.append(f"{tid}: []")
            else:
                lines.append(f"{tid}:")
                for dep in deps:
                    lines.append(f"  - {dep}")
            lines.append("")
        self.path.write_text("\n".join(lines).rstrip() + "\n")

    def get_deps(self, track_id: str) -> list[str]:
        return list(self._graph.get(track_id, []))

    def add_track(self, track_id: str, deps: Optional[list[str]] = None):
        """Add a track to the graph with optional dependencies."""
        self._graph[track_id] = list(deps) if deps else []

    def add_dep(self, track_id: str, dep_id: str):
        if track_id not in self._graph:
            self._graph[track_id] = []
        if dep_id not in self._graph[track_id]:
            self._graph[track_id].append(dep_id)

    def remove_dep(self, track_id: str, dep_id: str):
        if track_id in self._graph:
            self._graph[track_id] = [d for d in self._graph[track_id] if d != dep_id]

    def remove_track(self, track_id: str):
        """Remove a track from the graph entirely."""
        self._graph.pop(track_id, None)

    def all_satisfied(self, track_id: str, completed_ids: set[str]) -> bool:
        """Check if all dependencies for a track are in the completed set."""
        deps = self._graph.get(track_id, [])
        return all(d in completed_ids for d in deps)

    def dep_summary(self, track_id: str, completed_ids: set[str]) -> str:
        """Return 'N/M met' or '-' for no deps."""
        deps = self._graph.get(track_id, [])
        if not deps:
            return "-"
        met = sum(1 for d in deps if d in completed_ids)
        total = len(deps)
        check = " ✓" if met == total else ""
        return f"{met}/{total}{check}"

    def graph(self) -> dict[str, list[str]]:
        return dict(self._graph)
