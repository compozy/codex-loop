#!/usr/bin/env python3
"""Validate codex-loop tracking artifacts."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any


VALID_TASK_STATUS = {"pending", "in_progress", "completed", "blocked"}
VALID_LOOP_STATUS = {"active", "blocked", "complete"}


def fail(message: str, code: int = 1) -> int:
    print(f"validate-tracking: {message}", file=sys.stderr)
    return code


def load_state(path: Path) -> dict[str, Any]:
    with path.open(encoding="utf-8") as handle:
        payload = json.load(handle)
    if not isinstance(payload, dict):
        raise ValueError("state root must be an object")
    return payload


def require(condition: bool, message: str, errors: list[str]) -> None:
    if not condition:
        errors.append(message)


def validate(loop_dir: Path, state: dict[str, Any], expect_done: bool) -> list[str]:
    errors: list[str] = []
    require(state.get("schema_version") == 1, "schema_version must be 1", errors)
    require(state.get("loop_name") == loop_dir.name, "loop_name must match directory name", errors)
    require(state.get("status") in VALID_LOOP_STATUS, "invalid loop status", errors)
    require(isinstance(state.get("iteration"), int), "iteration must be an integer", errors)
    require((loop_dir / "request.md").is_file(), "request.md missing", errors)
    require((loop_dir / "memory" / "MEMORY.md").is_file(), "memory/MEMORY.md missing", errors)

    tasks = state.get("tasks")
    require(isinstance(tasks, list) and len(tasks) > 0, "tasks must be a non-empty list", errors)
    seen: set[str] = set()
    in_progress = 0
    blocked_tasks = 0
    if isinstance(tasks, list):
        for task in tasks:
            if not isinstance(task, dict):
                errors.append("task must be an object")
                continue
            task_id = task.get("id")
            require(isinstance(task_id, str) and task_id, "task id is required", errors)
            if isinstance(task_id, str):
                require(task_id not in seen, f"duplicate task id: {task_id}", errors)
                seen.add(task_id)
                require((loop_dir / "tasks" / f"{task_id}.md").is_file(), f"task file missing: {task_id}", errors)
            require(bool(str(task.get("title", "")).strip()), f"task title missing: {task_id}", errors)
            status = task.get("status")
            require(status in VALID_TASK_STATUS, f"invalid task status for {task_id}: {status}", errors)
            if status == "in_progress":
                in_progress += 1
            if status == "blocked":
                blocked_tasks += 1
            memory = task.get("memory")
            if status == "completed" and memory:
                require((loop_dir / str(memory)).is_file(), f"completed task memory missing: {memory}", errors)
    require(in_progress <= 1, "at most one task may be in_progress", errors)

    current = state.get("current_task")
    if current is not None:
        require(current in seen, f"current_task not found in tasks: {current}", errors)

    blockers = state.get("blockers")
    require(isinstance(blockers, list), "blockers must be a list", errors)
    if isinstance(blockers, list) and blockers:
        require(state.get("status") == "blocked", "status must be blocked when blockers are open", errors)
    if blocked_tasks > 0:
        require(state.get("status") == "blocked", "status must be blocked when any task is blocked", errors)

    verification = state.get("verification")
    require(isinstance(verification, dict), "verification must be an object", errors)
    if isinstance(verification, dict):
        require(verification.get("status") in {"PASS", "FAIL", None}, "invalid verification status", errors)
        if verification.get("status") == "PASS":
            require(bool(str(verification.get("evidence", "")).strip()), "PASS requires verification evidence", errors)

    history = state.get("history")
    require(isinstance(history, list), "history must be a list", errors)
    if isinstance(history, list):
        for entry in history:
            if not isinstance(entry, dict):
                errors.append("history entry must be an object")
                continue
            for rel in entry.get("memory_written") or []:
                require((loop_dir / str(rel)).is_file(), f"history memory missing: {rel}", errors)

    if expect_done:
        require(state.get("status") == "complete", "expected status complete", errors)
        if isinstance(tasks, list):
            require(all(isinstance(task, dict) and task.get("status") == "completed" for task in tasks), "expected all tasks completed", errors)
        require(not blockers, "expected no blockers", errors)
        if isinstance(verification, dict):
            require(verification.get("status") == "PASS", "expected verification PASS", errors)

    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("name")
    parser.add_argument("--root", default=".codex/loop")
    parser.add_argument("--expect-done", action="store_true")
    args = parser.parse_args()

    loop_dir = Path(args.root) / args.name
    state_path = loop_dir / "state.json"
    if not state_path.exists():
        return fail(f"{state_path} missing", 2)

    try:
        state = load_state(state_path)
    except (OSError, json.JSONDecodeError, ValueError) as exc:
        return fail(str(exc))

    errors = validate(loop_dir, state, args.expect_done)
    if errors:
        for error in errors:
            print(f"validate-tracking: {error}", file=sys.stderr)
        return 1

    print(f"validate-tracking: ok {state_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
