#!/usr/bin/env python3
# Resolve the Kiloforge primary branch from config.yaml.
# Tries local file first, then git HEAD, defaults to "main".
#
# Usage: PRIMARY_BRANCH=$(kf-primary-branch)

import subprocess
import sys

import yaml

CONFIG_PATH = ".agent/kf/config.yaml"
DEFAULT_BRANCH = "main"


def read_local_config():
    """Try reading config.yaml from the working directory."""
    try:
        with open(CONFIG_PATH) as f:
            return yaml.safe_load(f)
    except (OSError, yaml.YAMLError):
        return None


def read_git_config():
    """Fall back to git show HEAD:.agent/kf/config.yaml."""
    try:
        result = subprocess.run(
            ["git", "show", f"HEAD:{CONFIG_PATH}"],
            capture_output=True,
            text=True,
            check=True,
        )
        return yaml.safe_load(result.stdout)
    except (subprocess.CalledProcessError, yaml.YAMLError):
        return None


def main():
    for reader in (read_local_config, read_git_config):
        config = reader()
        if isinstance(config, dict):
            branch = config.get("primary_branch")
            if branch:
                print(branch)
                return
    print(DEFAULT_BRANCH)


if __name__ == "__main__":
    main()
