"""Config management for config.yaml.

config.yaml is a simple flat YAML key-value file.
"""

import yaml
from pathlib import Path
from typing import Optional


CONFIG_SCHEMA = {
    "primary_branch": {"type": "string", "default": "main"},
    "enforce_dep_ordering": {"type": "bool", "default": "true"},
    "max_workers": {"type": "int", "default": "4"},
}

HEADER = """\
# Kiloforge Project Configuration
#
# SCHEMA:
#   primary_branch: string (default: "main")
#     The branch agents read track state from.
#
#   enforce_dep_ordering: bool (default: true)
#     When true, the work queue scheduler skips tracks with unmet dependencies
#     and continues to the next eligible track (drain-loop). When false, tracks
#     are popped in order regardless of dependency status.
#
#   max_workers: int (default: 4)
#     Maximum number of concurrent worker agents the conductor will run.
#     Limits parallel API usage and system resource consumption.
#
# Structured project settings used by kf tooling and agent skills.
# TOOL: Managed by `kf-track config` and agent skills. Hand-editable.

"""


class Config:
    """Read/write interface for config.yaml."""

    def __init__(self, path: Path):
        self.path = path
        self._data: dict = {}
        if self.path.exists():
            self._load()

    def _load(self):
        text = self.path.read_text()
        parsed = yaml.safe_load(text)
        self._data = parsed if isinstance(parsed, dict) else {}

    @classmethod
    def from_text(cls, text: str, path: Optional[Path] = None) -> "Config":
        """Parse config.yaml from a string."""
        cfg = cls.__new__(cls)
        cfg.path = path
        parsed = yaml.safe_load(text)
        cfg._data = parsed if isinstance(parsed, dict) else {}
        return cfg

    def get(self, key: str) -> str:
        """Get a config value, falling back to schema default."""
        if key not in CONFIG_SCHEMA:
            raise KeyError(f"Unknown config key: {key}")
        val = self._data.get(key)
        if val is None:
            return CONFIG_SCHEMA[key]["default"]
        return str(val)

    def set(self, key: str, value: str):
        """Set a config value with type validation."""
        if key not in CONFIG_SCHEMA:
            raise KeyError(f"Unknown config key: {key}")
        schema = CONFIG_SCHEMA[key]
        if schema["type"] == "bool" and value not in ("true", "false"):
            raise ValueError(f"{key} must be a boolean (true|false), got: {value}")
        if schema["type"] == "int":
            try:
                int(value)
            except ValueError:
                raise ValueError(f"{key} must be an integer, got: {value}")
        self._data[key] = value

    def ensure(self):
        """Create the file with header if it doesn't exist."""
        if not self.path.exists():
            self.path.parent.mkdir(parents=True, exist_ok=True)
            self.path.write_text(HEADER)

    def save(self):
        """Write config back to disk, preserving header."""
        text = self.path.read_text() if self.path.exists() else HEADER
        # Extract header (comment lines at top)
        lines = text.splitlines()
        header = []
        for line in lines:
            if line.startswith("#") or not line.strip():
                header.append(line)
            else:
                break
        # Write header + data
        output = "\n".join(header) + "\n"
        for key, value in sorted(self._data.items()):
            output += f"{key}: {value}\n"
        self.path.write_text(output)

    def list_all(self) -> list[dict]:
        """Return all settings with current values and defaults."""
        result = []
        for key, schema in CONFIG_SCHEMA.items():
            current = self._data.get(key, schema["default"])
            result.append({
                "key": key,
                "type": schema["type"],
                "default": schema["default"],
                "current": str(current),
            })
        return result
