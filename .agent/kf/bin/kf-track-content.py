#!/Users/benbaldivia/dev-2/kiloforge-wt/kfc-d780f5-worker-1/.agent/kf/.venv/bin/python
"""kf-track-content — Structured track content management for Kiloforge.

Manages per-track track.yaml files that replace spec.md, plan.md, metadata.json, index.md.
Called by kf-track as a delegate for content operations.

TRACK.YAML FORMAT:
  Single structured YAML file per track at .agent/kf/tracks/<id>/track.yaml.
  Sections: header (id, title, type, status, dates), spec, plan, extra.
  Plan tasks track completion state for developer progress.

COMMANDS:
  init <id> --title "..." [--type feature] [--spec-file ...] [--plan-file ...]
  show <id> [--section spec|plan|extra|header] [--json]
  spec <id> [--field summary|context|...] [--set "value"]
  plan <id> [--phase N] [--task N]
  task <id> <phase>.<task> [--done|--pending]
  progress <id>
  migrate <id>    # convert legacy spec.md/plan.md to track.yaml
  migrate-all     # convert all tracks
  claim <id> [--show|--clear]   # record developer claim in extra.claim
  register <id> [--role R] [--show|--clear]  # record creator in extra.created_by
"""

import json
import os
import re
import sys
from datetime import datetime, timezone
from pathlib import Path

# Locate kf directory (KF_DIR env var overrides for --ref support from kf-track)
SCRIPT_DIR = Path(__file__).resolve().parent
KF_DIR = Path(os.environ["KF_DIR"]) if "KF_DIR" in os.environ else SCRIPT_DIR.parent
TRACKS_DIR = KF_DIR / "tracks"

# --- YAML handling (no external deps) ---
# We use a minimal YAML serializer to avoid requiring PyYAML.
# For reading, we use PyYAML if available, otherwise a simple parser.

def _try_import_yaml():
    try:
        import yaml
        return yaml
    except ImportError:
        return None

def yaml_dump(data, stream=None):
    """Dump data to YAML string. Uses PyYAML if available, otherwise manual."""
    yaml = _try_import_yaml()
    if yaml:
        result = yaml.dump(data, default_flow_style=False, sort_keys=False, allow_unicode=True, width=120)
    else:
        result = _manual_yaml_dump(data)
    if stream:
        stream.write(result)
    return result

def yaml_load(text):
    """Load YAML from string. Uses PyYAML if available, otherwise JSON fallback."""
    yaml = _try_import_yaml()
    if yaml:
        return yaml.safe_load(text)
    else:
        # Minimal fallback: try JSON first, then simple key-value
        try:
            return json.loads(text)
        except json.JSONDecodeError:
            return _minimal_yaml_parse(text)

def _manual_yaml_dump(data, indent=0):
    """Minimal YAML serializer for dicts, lists, and scalars."""
    lines = []
    prefix = "  " * indent
    if isinstance(data, dict):
        for key, value in data.items():
            if isinstance(value, (dict, list)) and value:
                lines.append(f"{prefix}{key}:")
                lines.append(_manual_yaml_dump(value, indent + 1))
            elif isinstance(value, dict) and not value:
                lines.append(f"{prefix}{key}: {{}}")
            elif isinstance(value, list) and not value:
                lines.append(f"{prefix}{key}: []")
            elif isinstance(value, bool):
                lines.append(f"{prefix}{key}: {'true' if value else 'false'}")
            elif isinstance(value, (int, float)):
                lines.append(f"{prefix}{key}: {value}")
            elif value is None:
                lines.append(f"{prefix}{key}: null")
            else:
                # String — quote if it contains special chars
                s = str(value)
                if '\n' in s:
                    lines.append(f"{prefix}{key}: |")
                    for line in s.split('\n'):
                        lines.append(f"{prefix}  {line}")
                elif any(c in s for c in '#{}[]&*?|>!%@`') or s.startswith("'"):
                    # Quote strings with special chars, but avoid double-quoting already quoted strings
                    if s.startswith('"') and s.endswith('"'):
                        lines.append(f'{prefix}{key}: {s}')
                    else:
                        lines.append(f'{prefix}{key}: "{s}"')
                else:
                    lines.append(f"{prefix}{key}: {s}")
    elif isinstance(data, list):
        for item in data:
            if isinstance(item, dict):
                first = True
                for key, value in item.items():
                    if first:
                        if isinstance(value, (dict, list)) and value:
                            lines.append(f"{prefix}- {key}:")
                            lines.append(_manual_yaml_dump(value, indent + 2))
                        elif isinstance(value, bool):
                            lines.append(f"{prefix}- {key}: {'true' if value else 'false'}")
                        elif isinstance(value, (int, float)):
                            lines.append(f"{prefix}- {key}: {value}")
                        elif value is None:
                            lines.append(f"{prefix}- {key}: null")
                        else:
                            s = str(value)
                            if '\n' in s:
                                lines.append(f"{prefix}- {key}: |")
                                for l in s.split('\n'):
                                    lines.append(f"{prefix}    {l}")
                            else:
                                lines.append(f"{prefix}- {key}: {s}")
                        first = False
                    else:
                        if isinstance(value, (dict, list)) and value:
                            lines.append(f"{prefix}  {key}:")
                            lines.append(_manual_yaml_dump(value, indent + 2))
                        elif isinstance(value, bool):
                            lines.append(f"{prefix}  {key}: {'true' if value else 'false'}")
                        elif isinstance(value, (int, float)):
                            lines.append(f"{prefix}  {key}: {value}")
                        elif value is None:
                            lines.append(f"{prefix}  {key}: null")
                        else:
                            s = str(value)
                            lines.append(f"{prefix}  {key}: {s}")
            else:
                s = str(item)
                lines.append(f"{prefix}- {s}")
    return '\n'.join(lines)

def _minimal_yaml_parse(text):
    """YAML parser for our controlled track.yaml format.

    Handles nested dicts, lists (- item and - key: val), multi-line
    block scalars (|), and quoted strings. Not a general YAML parser.
    """
    lines = text.split('\n')
    root = {}
    _parse_yaml_block(lines, 0, 0, root)
    return root


def _yaml_unquote(s):
    """Strip surrounding quotes from a YAML scalar."""
    s = s.strip()
    if len(s) >= 2 and ((s[0] == '"' and s[-1] == '"') or (s[0] == "'" and s[-1] == "'")):
        return s[1:-1]
    return s


def _yaml_scalar(s):
    """Convert a YAML scalar string to a Python value."""
    s = s.strip()
    if not s or s == '~' or s == 'null':
        return None
    # Unquote first, then check special values
    unquoted = _yaml_unquote(s)
    if unquoted == 'true':
        return True
    if unquoted == 'false':
        return False
    if unquoted == '[]':
        return []
    if unquoted == '{}':
        return {}
    # Try int/float on unquoted value (but only if original was not quoted)
    if not (len(s) >= 2 and s[0] in ('"', "'") and s[-1] == s[0]):
        try:
            return int(unquoted)
        except ValueError:
            pass
        try:
            return float(unquoted)
        except ValueError:
            pass
    return unquoted


def _indent_level(line):
    """Return the number of leading spaces."""
    return len(line) - len(line.lstrip(' '))


def _parse_yaml_block(lines, start, base_indent, target):
    """Parse YAML lines into target dict. Returns next line index."""
    i = start
    while i < len(lines):
        line = lines[i]
        stripped = line.strip()

        # Skip blank lines and comments
        if not stripped or stripped.startswith('#'):
            i += 1
            continue

        indent = _indent_level(line)
        if indent < base_indent:
            return i  # Dedented, return to parent

        # List item
        if stripped.startswith('- '):
            # This is a list — but we should be called from a dict context
            # where the parent key already created the list.
            return i

        # Key: value
        m = re.match(r'^(\s*)([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.*)', line)
        if not m:
            i += 1
            continue

        key = m.group(2)
        rest = m.group(3).strip()
        key_indent = len(m.group(1))

        if key_indent != base_indent:
            if key_indent < base_indent:
                return i
            i += 1
            continue

        if rest == '|':
            # Block scalar — collect indented lines
            i += 1
            block_lines = []
            if i < len(lines):
                first_line = lines[i]
                if first_line.strip():
                    block_indent = _indent_level(first_line)
                else:
                    block_indent = base_indent + 2
                while i < len(lines):
                    bl = lines[i]
                    if bl.strip() == '':
                        block_lines.append('')
                        i += 1
                        continue
                    if _indent_level(bl) < block_indent:
                        break
                    block_lines.append(bl[block_indent:])
                    i += 1
            # Trim trailing empty lines
            while block_lines and block_lines[-1] == '':
                block_lines.pop()
            target[key] = '\n'.join(block_lines)

        elif rest == '' or rest is None:
            # Nested structure — peek at next non-blank line
            i += 1
            # Find next non-blank, non-comment line
            peek = i
            while peek < len(lines) and (not lines[peek].strip() or lines[peek].strip().startswith('#')):
                peek += 1
            if peek >= len(lines):
                target[key] = None
                continue

            next_line = lines[peek]
            next_indent = _indent_level(next_line)
            next_stripped = next_line.strip()

            if next_indent <= base_indent:
                target[key] = None
                continue

            if next_stripped.startswith('- '):
                # It's a list
                lst = []
                i = _parse_yaml_list(lines, peek, next_indent, lst)
                target[key] = lst
            else:
                # It's a nested dict
                child = {}
                i = _parse_yaml_block(lines, peek, next_indent, child)
                target[key] = child

        else:
            # Inline value
            target[key] = _yaml_scalar(rest)
            i += 1

    return i


def _parse_yaml_list(lines, start, base_indent, target):
    """Parse YAML list items into target list. Returns next line index."""
    i = start
    while i < len(lines):
        line = lines[i]
        stripped = line.strip()

        if not stripped or stripped.startswith('#'):
            i += 1
            continue

        indent = _indent_level(line)
        if indent < base_indent:
            return i

        if not stripped.startswith('- '):
            return i

        # Extract after "- "
        item_text = stripped[2:]

        # Check if it's "- key: value" (dict item in list)
        m = re.match(r'^([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.*)', item_text)
        if m:
            # Dict item in list
            item_dict = {}
            first_key = m.group(1)
            first_rest = m.group(2).strip()

            # The content indent for continuation is indent + 2
            content_indent = indent + 2

            if first_rest == '|':
                # Block scalar for first key
                i += 1
                block_lines = []
                if i < len(lines):
                    first_bl = lines[i]
                    block_indent = _indent_level(first_bl) if first_bl.strip() else content_indent + 2
                    while i < len(lines):
                        bl = lines[i]
                        if bl.strip() == '':
                            block_lines.append('')
                            i += 1
                            continue
                        if _indent_level(bl) < block_indent:
                            break
                        block_lines.append(bl[block_indent:])
                        i += 1
                while block_lines and block_lines[-1] == '':
                    block_lines.pop()
                item_dict[first_key] = '\n'.join(block_lines)
            elif first_rest == '':
                # Nested under first key
                i += 1
                peek = i
                while peek < len(lines) and not lines[peek].strip():
                    peek += 1
                if peek < len(lines) and _indent_level(lines[peek]) > content_indent:
                    next_stripped = lines[peek].strip()
                    next_indent = _indent_level(lines[peek])
                    if next_stripped.startswith('- '):
                        lst = []
                        i = _parse_yaml_list(lines, peek, next_indent, lst)
                        item_dict[first_key] = lst
                    else:
                        child = {}
                        i = _parse_yaml_block(lines, peek, next_indent, child)
                        item_dict[first_key] = child
                else:
                    item_dict[first_key] = None
            else:
                item_dict[first_key] = _yaml_scalar(first_rest)
                i += 1

            # Parse remaining keys at content_indent
            while i < len(lines):
                cl = lines[i]
                cs = cl.strip()
                if not cs or cs.startswith('#'):
                    i += 1
                    continue
                ci = _indent_level(cl)
                if ci < content_indent:
                    break
                if ci == content_indent and cs.startswith('- '):
                    break  # New list item at same level

                km = re.match(r'^([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.*)', cs)
                if km and ci == content_indent:
                    k2 = km.group(1)
                    r2 = km.group(2).strip()
                    if r2 == '|':
                        i += 1
                        block_lines = []
                        if i < len(lines):
                            first_bl = lines[i]
                            block_indent = _indent_level(first_bl) if first_bl.strip() else content_indent + 2
                            while i < len(lines):
                                bl = lines[i]
                                if bl.strip() == '':
                                    block_lines.append('')
                                    i += 1
                                    continue
                                if _indent_level(bl) < block_indent:
                                    break
                                block_lines.append(bl[block_indent:])
                                i += 1
                        while block_lines and block_lines[-1] == '':
                            block_lines.pop()
                        item_dict[k2] = '\n'.join(block_lines)
                    elif r2 == '':
                        i += 1
                        peek = i
                        while peek < len(lines) and not lines[peek].strip():
                            peek += 1
                        if peek < len(lines) and _indent_level(lines[peek]) > content_indent:
                            ns = lines[peek].strip()
                            ni = _indent_level(lines[peek])
                            if ns.startswith('- '):
                                lst = []
                                i = _parse_yaml_list(lines, peek, ni, lst)
                                item_dict[k2] = lst
                            else:
                                child = {}
                                i = _parse_yaml_block(lines, peek, ni, child)
                                item_dict[k2] = child
                        else:
                            item_dict[k2] = None
                    else:
                        item_dict[k2] = _yaml_scalar(r2)
                        i += 1
                else:
                    i += 1

            target.append(item_dict)
        else:
            # Simple list item
            target.append(_yaml_scalar(item_text))
            i += 1

    return i


# --- Track file operations ---

def track_dir(track_id):
    return TRACKS_DIR / track_id

def track_file(track_id):
    return track_dir(track_id) / "track.yaml"

def load_track(track_id):
    """Load a track.yaml file. Returns dict or None."""
    path = track_file(track_id)
    if not path.exists():
        return None
    return yaml_load(path.read_text())

def save_track(track_id, data):
    """Save track data to track.yaml with stable field ordering."""
    path = track_file(track_id)
    path.parent.mkdir(parents=True, exist_ok=True)

    # Enforce canonical section order
    ordered = {}
    for key in ["id", "title", "type", "status", "created", "updated"]:
        if key in data:
            ordered[key] = data[key]

    # Spec section with canonical field order
    if "spec" in data:
        spec = data["spec"]
        ordered_spec = {}
        for key in ["summary", "context", "codebase_analysis", "acceptance_criteria",
                     "out_of_scope", "technical_notes"]:
            if key in spec:
                ordered_spec[key] = spec[key]
        # Any extra spec fields
        for key in spec:
            if key not in ordered_spec:
                ordered_spec[key] = spec[key]
        ordered["spec"] = ordered_spec

    # Plan section
    if "plan" in data:
        ordered["plan"] = data["plan"]

    # Extra section (preserve key order)
    if "extra" in data:
        ordered["extra"] = data["extra"]

    # Any remaining top-level keys
    for key in data:
        if key not in ordered:
            ordered[key] = data[key]

    content = yaml_dump(ordered)
    path.write_text(content)


def today():
    return datetime.now(timezone.utc).strftime("%Y-%m-%d")


# --- Commands ---

def cmd_init(args):
    """Create a new track.yaml from scratch or from spec/plan files."""
    import argparse
    parser = argparse.ArgumentParser(prog="kf-track-content init")
    parser.add_argument("track_id")
    parser.add_argument("--title", required=True)
    parser.add_argument("--type", default="feature", dest="track_type")
    parser.add_argument("--summary", default="")
    parser.add_argument("--spec-file", help="Read spec from file (markdown or yaml)")
    parser.add_argument("--plan-file", help="Read plan from file (markdown or yaml)")
    opts = parser.parse_args(args)

    if track_file(opts.track_id).exists():
        print(f"ERROR: track.yaml already exists for {opts.track_id}", file=sys.stderr)
        return 1

    d = today()
    data = {
        "id": opts.track_id,
        "title": opts.title,
        "type": opts.track_type,
        "status": "pending",
        "created": d,
        "updated": d,
        "spec": {
            "summary": opts.summary,
            "context": "",
            "codebase_analysis": "",
            "acceptance_criteria": [],
            "out_of_scope": "",
            "technical_notes": "",
        },
        "plan": [],
        "extra": {},
    }

    # Import spec from file if provided
    if opts.spec_file:
        spec_data = _parse_spec_md(Path(opts.spec_file).read_text())
        data["spec"].update(spec_data)

    # Import plan from file if provided
    if opts.plan_file:
        plan_data = _parse_plan_md(Path(opts.plan_file).read_text())
        data["plan"] = plan_data

    save_track(opts.track_id, data)
    print(f"Created: {opts.track_id}/track.yaml")
    return 0


def cmd_show(args):
    """Show track content, optionally filtered by section."""
    import argparse
    parser = argparse.ArgumentParser(prog="kf-track-content show")
    parser.add_argument("track_id")
    parser.add_argument("--section", choices=["header", "spec", "plan", "extra"])
    parser.add_argument("--json", action="store_true", dest="as_json")
    opts = parser.parse_args(args)

    data = load_track(opts.track_id)
    if not data:
        print(f"ERROR: No track.yaml for {opts.track_id}", file=sys.stderr)
        return 1

    if opts.section:
        if opts.section == "header":
            section = {k: data[k] for k in ["id", "title", "type", "status", "created", "updated"] if k in data}
        elif opts.section in data:
            section = data[opts.section]
        else:
            section = {}
        if opts.as_json:
            print(json.dumps(section, indent=2))
        else:
            print(yaml_dump({opts.section: section} if opts.section != "header" else section))
    else:
        if opts.as_json:
            print(json.dumps(data, indent=2))
        else:
            print(yaml_dump(data))
    return 0


def cmd_spec(args):
    """Read or update a spec field."""
    import argparse
    parser = argparse.ArgumentParser(prog="kf-track-content spec")
    parser.add_argument("track_id")
    parser.add_argument("--field", help="Specific field to read/set")
    parser.add_argument("--set", help="Value to set (use with --field)", dest="set_value")
    parser.add_argument("--append", help="Append to list field (acceptance_criteria)", dest="append_value")
    opts = parser.parse_args(args)

    data = load_track(opts.track_id)
    if not data:
        print(f"ERROR: No track.yaml for {opts.track_id}", file=sys.stderr)
        return 1

    spec = data.get("spec", {})

    if opts.field and opts.set_value is not None:
        spec[opts.field] = opts.set_value
        data["spec"] = spec
        data["updated"] = today()
        save_track(opts.track_id, data)
        print(f"Set spec.{opts.field}")
        return 0
    elif opts.field and opts.append_value:
        if opts.field not in spec:
            spec[opts.field] = []
        if isinstance(spec[opts.field], list):
            spec[opts.field].append(opts.append_value)
        data["spec"] = spec
        data["updated"] = today()
        save_track(opts.track_id, data)
        print(f"Appended to spec.{opts.field}")
        return 0
    elif opts.field:
        value = spec.get(opts.field, "")
        if isinstance(value, list):
            for i, item in enumerate(value):
                print(f"  {i+1}. {item}")
        else:
            print(value)
        return 0
    else:
        print(yaml_dump({"spec": spec}))
        return 0


def cmd_plan(args):
    """Show plan, optionally filtered to a phase."""
    import argparse
    parser = argparse.ArgumentParser(prog="kf-track-content plan")
    parser.add_argument("track_id")
    parser.add_argument("--phase", type=int, help="Show specific phase (1-based)")
    opts = parser.parse_args(args)

    data = load_track(opts.track_id)
    if not data:
        print(f"ERROR: No track.yaml for {opts.track_id}", file=sys.stderr)
        return 1

    plan = data.get("plan", [])
    if not plan:
        print("(no plan)")
        return 0

    if opts.phase:
        idx = opts.phase - 1
        if idx < 0 or idx >= len(plan):
            print(f"ERROR: Phase {opts.phase} not found (have {len(plan)} phases)", file=sys.stderr)
            return 1
        phase = plan[idx]
        print(f"Phase {opts.phase}: {phase.get('phase', 'Unnamed')}")
        for i, task in enumerate(phase.get("tasks", [])):
            text = task.get("text", task) if isinstance(task, dict) else str(task)
            done = task.get("done", False) if isinstance(task, dict) else False
            marker = "[x]" if done else "[ ]"
            print(f"  {marker} {opts.phase}.{i+1}: {text}")
    else:
        for pi, phase in enumerate(plan):
            phase_name = phase.get("phase", "Unnamed")
            tasks = phase.get("tasks", [])
            done_count = sum(1 for t in tasks if isinstance(t, dict) and t.get("done", False))
            print(f"Phase {pi+1}: {phase_name} ({done_count}/{len(tasks)})")
            for ti, task in enumerate(tasks):
                text = task.get("text", task) if isinstance(task, dict) else str(task)
                done = task.get("done", False) if isinstance(task, dict) else False
                marker = "[x]" if done else "[ ]"
                print(f"  {marker} {pi+1}.{ti+1}: {text}")
    return 0


def cmd_task(args):
    """Update task completion status."""
    import argparse
    parser = argparse.ArgumentParser(prog="kf-track-content task")
    parser.add_argument("track_id")
    parser.add_argument("task_ref", help="Phase.Task reference like 1.3 or 2.1")
    parser.add_argument("--done", action="store_true")
    parser.add_argument("--pending", action="store_true")
    opts = parser.parse_args(args)

    if not opts.done and not opts.pending:
        print("ERROR: Must specify --done or --pending", file=sys.stderr)
        return 1

    data = load_track(opts.track_id)
    if not data:
        print(f"ERROR: No track.yaml for {opts.track_id}", file=sys.stderr)
        return 1

    # Parse task reference
    parts = opts.task_ref.split(".")
    if len(parts) != 2:
        print(f"ERROR: Invalid task ref '{opts.task_ref}'. Use phase.task (e.g. 1.3)", file=sys.stderr)
        return 1
    phase_idx = int(parts[0]) - 1
    task_idx = int(parts[1]) - 1

    plan = data.get("plan", [])
    if phase_idx < 0 or phase_idx >= len(plan):
        print(f"ERROR: Phase {parts[0]} not found", file=sys.stderr)
        return 1

    tasks = plan[phase_idx].get("tasks", [])
    if task_idx < 0 or task_idx >= len(tasks):
        print(f"ERROR: Task {parts[1]} not found in phase {parts[0]}", file=sys.stderr)
        return 1

    task = tasks[task_idx]
    if isinstance(task, str):
        # Upgrade from plain string to dict
        task = {"text": task, "done": False}
        tasks[task_idx] = task

    new_status = opts.done
    task["done"] = new_status
    data["updated"] = today()
    save_track(opts.track_id, data)

    text = task.get("text", "")
    marker = "[x]" if new_status else "[ ]"
    print(f"{marker} {opts.task_ref}: {text}")
    return 0


def cmd_progress(args):
    """Show completion progress for a track."""
    import argparse
    parser = argparse.ArgumentParser(prog="kf-track-content progress")
    parser.add_argument("track_id")
    parser.add_argument("--json", action="store_true", dest="as_json")
    opts = parser.parse_args(args)

    data = load_track(opts.track_id)
    if not data:
        print(f"ERROR: No track.yaml for {opts.track_id}", file=sys.stderr)
        return 1

    plan = data.get("plan", [])
    total_phases = len(plan)
    total_tasks = 0
    done_tasks = 0
    done_phases = 0

    phase_stats = []
    for pi, phase in enumerate(plan):
        tasks = phase.get("tasks", [])
        phase_total = len(tasks)
        phase_done = sum(1 for t in tasks if isinstance(t, dict) and t.get("done", False))
        total_tasks += phase_total
        done_tasks += phase_done
        if phase_total > 0 and phase_done == phase_total:
            done_phases += 1
        phase_stats.append({
            "phase": pi + 1,
            "name": phase.get("phase", "Unnamed"),
            "total": phase_total,
            "done": phase_done,
        })

    result = {
        "id": opts.track_id,
        "phases": {"total": total_phases, "completed": done_phases},
        "tasks": {"total": total_tasks, "completed": done_tasks},
        "percent": round(done_tasks / total_tasks * 100) if total_tasks > 0 else 0,
        "phase_detail": phase_stats,
    }

    if opts.as_json:
        print(json.dumps(result))
    else:
        print(f"Track: {opts.track_id}")
        print(f"Progress: {done_tasks}/{total_tasks} tasks ({result['percent']}%), {done_phases}/{total_phases} phases")
        for ps in phase_stats:
            marker = "[x]" if ps["done"] == ps["total"] and ps["total"] > 0 else "[ ]"
            print(f"  {marker} Phase {ps['phase']}: {ps['name']} ({ps['done']}/{ps['total']})")
    return 0


def cmd_extra(args):
    """Read or set extra metadata sections."""
    import argparse
    parser = argparse.ArgumentParser(prog="kf-track-content extra")
    parser.add_argument("track_id")
    parser.add_argument("--key", help="Extra section key")
    parser.add_argument("--set", help="Value to set", dest="set_value")
    parser.add_argument("--delete", action="store_true")
    opts = parser.parse_args(args)

    data = load_track(opts.track_id)
    if not data:
        print(f"ERROR: No track.yaml for {opts.track_id}", file=sys.stderr)
        return 1

    extra = data.get("extra", {})

    if opts.key and opts.set_value is not None:
        extra[opts.key] = opts.set_value
        data["extra"] = extra
        data["updated"] = today()
        save_track(opts.track_id, data)
        print(f"Set extra.{opts.key}")
    elif opts.key and opts.delete:
        extra.pop(opts.key, None)
        data["extra"] = extra
        data["updated"] = today()
        save_track(opts.track_id, data)
        print(f"Deleted extra.{opts.key}")
    elif opts.key:
        print(extra.get(opts.key, ""))
    else:
        if extra:
            print(yaml_dump({"extra": extra}))
        else:
            print("(no extra sections)")
    return 0


# --- Agent identity helpers ---

def _discover_session_id():
    """Discover Claude session ID from filesystem (most recent .jsonl file)."""
    cwd = os.getcwd()
    sanitized = cwd.replace("/", "-").lstrip("-")
    sessions_dir = Path.home() / ".claude" / "projects" / f"-{sanitized}"
    if not sessions_dir.exists():
        return None
    jsonl_files = sorted(sessions_dir.glob("*.jsonl"), key=lambda p: p.stat().st_mtime, reverse=True)
    if not jsonl_files:
        return None
    return jsonl_files[0].stem  # filename without .jsonl


def _discover_identity():
    """Discover agent identity from env vars with filesystem fallback."""
    import subprocess
    identity = {}

    # Agent ID — only from orchestrator env var
    agent_id = os.environ.get("KF_AGENT_ID", "")
    if agent_id:
        identity["agent_id"] = agent_id

    # Session ID — env var first, then filesystem discovery
    session_id = os.environ.get("KF_SESSION_ID", "")
    if not session_id:
        session_id = _discover_session_id() or ""
    if session_id:
        identity["session_id"] = session_id

    # Role — env var or default
    role = os.environ.get("KF_AGENT_ROLE", "")
    if role:
        identity["role"] = role

    # Worktree — basename of cwd
    identity["worktree"] = os.path.basename(os.getcwd())

    # Branch — from git
    try:
        result = subprocess.run(
            ["git", "branch", "--show-current"],
            capture_output=True, text=True, timeout=5
        )
        branch = result.stdout.strip()
        if branch:
            identity["branch"] = branch
    except (subprocess.SubprocessError, FileNotFoundError):
        pass

    # Model — from env or claude CLI
    model = os.environ.get("ANTHROPIC_MODEL", "")
    if model:
        identity["model"] = model

    return identity


def cmd_claim(args):
    """Record developer claim metadata in extra.claim."""
    import argparse
    parser = argparse.ArgumentParser(prog="kf-track-content claim")
    parser.add_argument("track_id")
    parser.add_argument("--show", action="store_true", help="Display the claim record")
    parser.add_argument("--clear", action="store_true", help="Remove the claim record")
    parser.add_argument("--role", default=None, help="Override role (default: from env or 'developer')")
    opts = parser.parse_args(args)

    data = load_track(opts.track_id)
    if not data:
        print(f"ERROR: No track.yaml for {opts.track_id}", file=sys.stderr)
        return 1

    extra = data.get("extra", {})

    if opts.show:
        claim = extra.get("claim")
        if claim:
            print(yaml_dump({"claim": claim}))
        else:
            print("(no claim record)")
        return 0

    if opts.clear:
        extra.pop("claim", None)
        data["extra"] = extra
        data["updated"] = today()
        save_track(opts.track_id, data)
        print("Claim record cleared")
        return 0

    # Write claim record
    identity = _discover_identity()
    claim = {}
    if "agent_id" in identity:
        claim["agent_id"] = identity["agent_id"]
    claim["role"] = opts.role or identity.get("role", "developer")
    if "session_id" in identity:
        claim["session_id"] = identity["session_id"]
    if "worktree" in identity:
        claim["worktree"] = identity["worktree"]
    if "branch" in identity:
        claim["branch"] = identity["branch"]
    claim["claimed_at"] = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    if "model" in identity:
        claim["model"] = identity["model"]

    extra["claim"] = claim
    data["extra"] = extra
    data["updated"] = today()
    save_track(opts.track_id, data)

    session_str = claim.get("session_id", "(not discovered)")
    print(f"Claim recorded for {opts.track_id}")
    print(f"  session_id: {session_str}")
    if "worktree" in claim:
        print(f"  worktree: {claim['worktree']}")
    if "branch" in claim:
        print(f"  branch: {claim['branch']}")
    return 0


def cmd_register(args):
    """Record architect/creator identity in extra.created_by."""
    import argparse
    parser = argparse.ArgumentParser(prog="kf-track-content register")
    parser.add_argument("track_id")
    parser.add_argument("--show", action="store_true", help="Display the created_by record")
    parser.add_argument("--clear", action="store_true", help="Remove the created_by record")
    parser.add_argument("--role", default=None, help="Override role (default: from env or 'architect')")
    opts = parser.parse_args(args)

    data = load_track(opts.track_id)
    if not data:
        print(f"ERROR: No track.yaml for {opts.track_id}", file=sys.stderr)
        return 1

    extra = data.get("extra", {})

    if opts.show:
        created_by = extra.get("created_by")
        if created_by:
            print(yaml_dump({"created_by": created_by}))
        else:
            print("(no created_by record)")
        return 0

    if opts.clear:
        extra.pop("created_by", None)
        data["extra"] = extra
        data["updated"] = today()
        save_track(opts.track_id, data)
        print("Created_by record cleared")
        return 0

    # Write created_by record
    identity = _discover_identity()
    created_by = {}
    if "agent_id" in identity:
        created_by["agent_id"] = identity["agent_id"]
    created_by["role"] = opts.role or identity.get("role", "architect")
    if "session_id" in identity:
        created_by["session_id"] = identity["session_id"]
    created_by["timestamp"] = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    extra["created_by"] = created_by
    data["extra"] = extra
    data["updated"] = today()
    save_track(opts.track_id, data)

    session_str = created_by.get("session_id", "(not discovered)")
    print(f"Register recorded for {opts.track_id}")
    print(f"  role: {created_by['role']}")
    print(f"  session_id: {session_str}")
    return 0


# --- Migration from legacy format ---

def _parse_spec_md(text):
    """Parse a spec.md markdown file into a spec dict."""
    spec = {}
    current_section = None
    current_lines = []

    section_map = {
        "summary": "summary",
        "context": "context",
        "codebase analysis": "codebase_analysis",
        "acceptance criteria": "acceptance_criteria",
        "conflict risk": "conflict_risk",
        "out of scope": "out_of_scope",
        "technical notes": "technical_notes",
    }
    # These sections are skipped — dependencies live in deps.yaml, not per-track
    skip_sections = {"dependencies", "blockers", "conflict risk"}

    def flush():
        nonlocal current_section, current_lines
        if current_section and current_lines:
            content = '\n'.join(current_lines).strip()
            if current_section == "acceptance_criteria":
                # Parse checklist items
                items = []
                for line in current_lines:
                    line = line.strip()
                    m = re.match(r'^-\s*\[[ x]\]\s*(.*)', line)
                    if m:
                        items.append(m.group(1))
                    elif line and not line.startswith('#'):
                        items.append(line)
                spec[current_section] = items
            else:
                spec[current_section] = content
        current_section = None
        current_lines = []

    for line in text.split('\n'):
        # Match ## Section headers
        m = re.match(r'^##\s+(.*)', line)
        if m:
            flush()
            header = m.group(1).strip().lower()
            if header in skip_sections:
                current_section = None  # discard — deps live in deps.yaml
            elif header in section_map:
                current_section = section_map[header]
            else:
                # Unknown section goes to extra
                current_section = header.replace(' ', '_').replace('-', '_')
            continue

        # Skip title line and metadata lines at top
        if line.startswith('# ') or line.startswith('**Track ID:') or line.startswith('**Type:') or \
           line.startswith('**Created:') or line.startswith('**Status:'):
            continue

        if line.startswith('---') and not current_section:
            continue
        if line.startswith('_Generated by'):
            continue

        if current_section:
            current_lines.append(line)

    flush()
    return spec


def _parse_plan_md(text):
    """Parse a plan.md markdown file into a plan list."""
    phases = []
    current_phase = None

    for line in text.split('\n'):
        # Match ## Phase N: Name
        m = re.match(r'^##\s+Phase\s+\d+[.:]\s*(.*)', line)
        if m:
            if current_phase:
                phases.append(current_phase)
            current_phase = {"phase": m.group(1).strip(), "tasks": []}
            continue

        # Match task lines: - [ ] Task N.M: description
        m = re.match(r'^-\s*\[([x ])\]\s*Task\s+[\d.]+[.:]\s*(.*)', line)
        if m:
            if current_phase is None:
                current_phase = {"phase": "Unnamed", "tasks": []}
            done = m.group(1) == 'x'
            text = m.group(2).strip()
            current_phase["tasks"].append({"text": text, "done": done})
            continue

        # Simpler task format: - [ ] description (without Task N.M prefix)
        m = re.match(r'^-\s*\[([x ])\]\s*(.*)', line)
        if m:
            if current_phase is None:
                current_phase = {"phase": "Unnamed", "tasks": []}
            done = m.group(1) == 'x'
            text = m.group(2).strip()
            current_phase["tasks"].append({"text": text, "done": done})

    if current_phase:
        phases.append(current_phase)

    return phases


def cmd_migrate(args):
    """Migrate a single track from legacy format to track.yaml."""
    import argparse
    parser = argparse.ArgumentParser(prog="kf-track-content migrate")
    parser.add_argument("track_id")
    parser.add_argument("--force", action="store_true", help="Overwrite existing track.yaml")
    parser.add_argument("--keep", action="store_true", help="Keep legacy files after migration")
    opts = parser.parse_args(args)

    tdir = track_dir(opts.track_id)
    if not tdir.exists():
        print(f"ERROR: Track directory not found: {tdir}", file=sys.stderr)
        return 1

    tfile = track_file(opts.track_id)
    if tfile.exists() and not opts.force:
        print(f"SKIP: track.yaml already exists for {opts.track_id} (use --force to overwrite)")
        return 0

    # Read legacy files
    metadata = {}
    meta_path = tdir / "metadata.json"
    if meta_path.exists():
        metadata = json.loads(meta_path.read_text())

    spec = {}
    spec_path = tdir / "spec.md"
    if spec_path.exists():
        spec = _parse_spec_md(spec_path.read_text())

    plan = []
    plan_path = tdir / "plan.md"
    if plan_path.exists():
        plan = _parse_plan_md(plan_path.read_text())

    # Build track.yaml
    data = {
        "id": opts.track_id,
        "title": metadata.get("title", spec.get("title", opts.track_id)),
        "type": metadata.get("type", "feature"),
        "status": metadata.get("status", "pending"),
        "created": metadata.get("created", today()),
        "updated": metadata.get("updated", today()),
        "spec": spec if spec else {
            "summary": "",
            "context": "",
            "acceptance_criteria": [],
            "out_of_scope": "",
            "technical_notes": "",
        },
        "plan": plan,
        "extra": {},
    }

    save_track(opts.track_id, data)
    print(f"Migrated: {opts.track_id}/track.yaml")

    # Remove legacy files if not keeping
    if not opts.keep:
        for f in ["spec.md", "plan.md", "metadata.json", "index.md"]:
            p = tdir / f
            if p.exists():
                p.unlink()
        print(f"  Removed legacy files")

    return 0


def cmd_migrate_all(args):
    """Migrate all tracks with legacy files to track.yaml."""
    import argparse
    parser = argparse.ArgumentParser(prog="kf-track-content migrate-all")
    parser.add_argument("--force", action="store_true")
    parser.add_argument("--keep", action="store_true")
    parser.add_argument("--dry-run", action="store_true")
    opts = parser.parse_args(args)

    if not TRACKS_DIR.exists():
        print("ERROR: No tracks directory found", file=sys.stderr)
        return 1

    migrated = 0
    skipped = 0
    errors = 0

    for tdir in sorted(TRACKS_DIR.iterdir()):
        if not tdir.is_dir():
            continue
        if tdir.name.startswith('_'):  # skip _archive
            continue

        # Check if has legacy files
        has_legacy = (tdir / "spec.md").exists() or (tdir / "plan.md").exists() or (tdir / "metadata.json").exists()
        has_track_yaml = (tdir / "track.yaml").exists()

        if not has_legacy:
            continue

        if has_track_yaml and not opts.force:
            skipped += 1
            continue

        if opts.dry_run:
            print(f"  Would migrate: {tdir.name}")
            migrated += 1
            continue

        try:
            migrate_args = [tdir.name]
            if opts.force:
                migrate_args.append("--force")
            if opts.keep:
                migrate_args.append("--keep")
            result = cmd_migrate(migrate_args)
            if result == 0:
                migrated += 1
            else:
                errors += 1
        except Exception as e:
            print(f"  ERROR migrating {tdir.name}: {e}", file=sys.stderr)
            errors += 1

    print(f"\nMigration complete: {migrated} migrated, {skipped} skipped, {errors} errors")
    return 0 if errors == 0 else 1


# --- Main dispatch ---

COMMANDS = {
    "init": cmd_init,
    "show": cmd_show,
    "spec": cmd_spec,
    "plan": cmd_plan,
    "task": cmd_task,
    "progress": cmd_progress,
    "extra": cmd_extra,
    "migrate": cmd_migrate,
    "migrate-all": cmd_migrate_all,
    "claim": cmd_claim,
    "register": cmd_register,
}

def main():
    if len(sys.argv) < 2 or sys.argv[1] in ("-h", "--help", "help"):
        print(__doc__)
        return 0

    cmd = sys.argv[1]
    if cmd not in COMMANDS:
        print(f"Unknown command: {cmd}", file=sys.stderr)
        print(f"Available: {', '.join(COMMANDS.keys())}", file=sys.stderr)
        return 1

    return COMMANDS[cmd](sys.argv[2:])

if __name__ == "__main__":
    sys.exit(main() or 0)
