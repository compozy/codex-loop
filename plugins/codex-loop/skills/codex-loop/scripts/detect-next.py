#!/usr/bin/env python3
"""Print the next codex-loop tracking action."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any


def load_state(path: Path) -> dict[str, Any]:
    with path.open(encoding="utf-8") as handle:
        payload = json.load(handle)
    if not isinstance(payload, dict):
        raise ValueError("state root must be an object")
    return payload


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("name")
    parser.add_argument("--root", default=".codex/loop")
    args = parser.parse_args()

    state_path = Path(args.root) / args.name / "state.json"
    if not state_path.exists():
        print(f"action=bootstrap name={args.name}")
        return 0

    try:
        state = load_state(state_path)
    except (OSError, json.JSONDecodeError, ValueError) as exc:
        print(f"detect-next: failed to read {state_path}: {exc}", file=sys.stderr)
        return 1

    blockers = list(state.get("blockers") or [])
    if state.get("status") == "blocked" or blockers:
        print(f"action=resolve_blocker name={args.name} blocker_count={len(blockers)}")
        return 0

    tasks = list(state.get("tasks") or [])
    for task in tasks:
        if task.get("status") == "in_progress":
            print(f"action=execute_task name={args.name} task={task.get('id')}")
            return 0

    for task in tasks:
        if task.get("status") == "pending":
            print(f"action=execute_task name={args.name} task={task.get('id')}")
            return 0

    if not tasks:
        print(f"detect-next: {state_path} has no tasks", file=sys.stderr)
        return 1

    verification = state.get("verification") or {}
    if verification.get("status") != "PASS":
        print(f"action=verify name={args.name}")
        return 0

    print(f"action=done name={args.name}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
