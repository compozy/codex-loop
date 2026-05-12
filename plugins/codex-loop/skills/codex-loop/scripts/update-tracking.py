#!/usr/bin/env python3
"""Apply codex-loop tracking state transitions."""

from __future__ import annotations

import argparse
import json
import os
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


VALID_TASK_STATUS = {"pending", "in_progress", "completed", "blocked"}


class TrackingError(ValueError):
    """Invalid requested tracking state transition."""


def now_iso() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def fail(message: str, code: int = 1) -> int:
    print(f"update-tracking: {message}", file=sys.stderr)
    return code


def load_state(path: Path) -> dict[str, Any]:
    with path.open(encoding="utf-8") as handle:
        payload = json.load(handle)
    if not isinstance(payload, dict):
        raise TrackingError("state root must be an object")
    return payload


def atomic_write_json(path: Path, payload: dict[str, Any]) -> None:
    tmp = path.with_name(f".{path.name}.tmp")
    tmp.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    os.replace(tmp, path)


def find_task(state: dict[str, Any], task_id: str) -> dict[str, Any]:
    for task in state.get("tasks") or []:
        if task.get("id") == task_id:
            return task
    raise TrackingError(f"task not found: {task_id}")


def parse_memory_paths(value: str) -> list[str]:
    paths: list[str] = []
    for item in value.split(","):
        text = item.strip()
        if text:
            paths.append(text)
    return paths


def require_memory_files(loop_dir: Path, paths: list[str]) -> None:
    if not paths:
        raise TrackingError("--memory-written is required for task completion or blocker resolution")
    for rel in paths:
        candidate = (loop_dir / rel).resolve(strict=False)
        root = loop_dir.resolve(strict=False)
        try:
            candidate.relative_to(root)
        except ValueError as exc:
            raise TrackingError(f"memory path escapes loop directory: {rel}") from exc
        if not candidate.is_file():
            raise TrackingError(f"memory file does not exist: {rel}")


def set_overall_status(state: dict[str, Any]) -> None:
    blockers = list(state.get("blockers") or [])
    tasks = list(state.get("tasks") or [])
    has_blocked_task = any(task.get("status") == "blocked" for task in tasks)
    if blockers or has_blocked_task:
        state["status"] = "blocked"
        return
    all_complete = bool(tasks) and all(task.get("status") == "completed" for task in tasks)
    verification = state.get("verification") or {}
    if all_complete and verification.get("status") == "PASS":
        state["status"] = "complete"
    else:
        state["status"] = "active"


def append_history(state: dict[str, Any], args: argparse.Namespace, memory_paths: list[str], timestamp: str) -> None:
    state["iteration"] = int(state.get("iteration", 0)) + 1
    verification = state.get("verification") or {}
    entry = {
        "iteration": state["iteration"],
        "timestamp": timestamp,
        "action": args.action or infer_action(args),
        "outcome": args.outcome or infer_outcome(args),
        "task_id": args.start_task or args.complete_task or args.block_task or None,
        "memory_written": memory_paths,
        "verification_status": verification.get("status"),
        "blockers": list(state.get("blockers") or []),
    }
    history = list(state.get("history") or [])
    history.append(entry)
    state["history"] = history[-args.max_history :]


def infer_action(args: argparse.Namespace) -> str:
    if args.start_task:
        return f"start {args.start_task}"
    if args.complete_task:
        return f"complete {args.complete_task}"
    if args.block_task:
        return f"block {args.block_task}"
    if args.verify_pass:
        return "verification PASS"
    if args.verify_fail:
        return "verification FAIL"
    if args.clear_blockers:
        return "clear blockers"
    if args.blocker:
        return "record blocker"
    return "state update"


def infer_outcome(args: argparse.Namespace) -> str:
    if args.blocker or args.block_task or args.verify_fail:
        return "blocked"
    return "completed"


def apply_update(state: dict[str, Any], loop_dir: Path, args: argparse.Namespace) -> list[str]:
    timestamp = now_iso()
    memory_paths = parse_memory_paths(args.memory_written)

    if args.start_task:
        task = find_task(state, args.start_task)
        if task.get("status") not in {"pending", "in_progress"}:
            raise TrackingError(f"cannot start task with status {task.get('status')}: {args.start_task}")
        for other in state.get("tasks") or []:
            if other.get("id") != args.start_task and other.get("status") == "in_progress":
                raise TrackingError(f"another task is already in progress: {other.get('id')}")
        task["status"] = "in_progress"
        state["current_task"] = args.start_task

    if args.complete_task:
        require_memory_files(loop_dir, memory_paths)
        task = find_task(state, args.complete_task)
        if task.get("status") not in {"pending", "in_progress"}:
            raise TrackingError(f"cannot complete task with status {task.get('status')}: {args.complete_task}")
        task["status"] = "completed"
        task["memory"] = memory_paths[-1]
        task["completed_at"] = timestamp
        if state.get("current_task") == args.complete_task:
            state["current_task"] = None

    if args.block_task:
        task = find_task(state, args.block_task)
        task["status"] = "blocked"
        state["current_task"] = None

    if args.blocker:
        blockers = list(state.get("blockers") or [])
        for blocker in args.blocker:
            text = blocker.strip()
            if text and text not in blockers:
                blockers.append(text)
        state["blockers"] = blockers

    if args.clear_blockers:
        require_memory_files(loop_dir, memory_paths)
        state["blockers"] = []
        for task in state.get("tasks") or []:
            if task.get("status") == "blocked":
                task["status"] = "pending"

    verification = state.setdefault("verification", {})
    if args.verify_pass:
        evidence = args.verify_pass.strip()
        if not evidence:
            raise TrackingError("--verify-pass requires evidence text")
        verification["status"] = "PASS"
        verification["evidence"] = evidence
        verification["last_run"] = timestamp
    if args.verify_fail:
        evidence = args.verify_fail.strip()
        if not evidence:
            raise TrackingError("--verify-fail requires evidence text")
        verification["status"] = "FAIL"
        verification["evidence"] = evidence
        verification["last_run"] = timestamp

    for task in state.get("tasks") or []:
        status = task.get("status")
        if status not in VALID_TASK_STATUS:
            raise TrackingError(f"invalid task status {status!r} for {task.get('id')}")

    state["updated_at"] = timestamp
    set_overall_status(state)
    append_history(state, args, memory_paths, timestamp)
    return memory_paths


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("name")
    parser.add_argument("--root", default=".codex/loop")
    parser.add_argument("--start-task", default="")
    parser.add_argument("--complete-task", default="")
    parser.add_argument("--block-task", default="")
    parser.add_argument("--blocker", action="append", default=[])
    parser.add_argument("--clear-blockers", action="store_true")
    parser.add_argument("--verify-pass", default="")
    parser.add_argument("--verify-fail", default="")
    parser.add_argument("--memory-written", default="")
    parser.add_argument("--action", default="")
    parser.add_argument("--outcome", choices=["completed", "partial", "blocked"], default="")
    parser.add_argument("--max-history", type=int, default=50)
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    loop_dir = Path(args.root) / args.name
    state_path = loop_dir / "state.json"
    if not state_path.exists():
        return fail(f"{state_path} missing; run init-tracking.py first", 2)
    if args.max_history < 1:
        return fail("--max-history must be positive")

    try:
        state = load_state(state_path)
        memory_paths = apply_update(state, loop_dir, args)
        atomic_write_json(state_path, state)
    except (OSError, json.JSONDecodeError, TrackingError) as exc:
        return fail(str(exc))

    memory_csv = ",".join(memory_paths) if memory_paths else "-"
    print(f"update-tracking: wrote {state_path} iteration={state['iteration']} status={state['status']} memory={memory_csv}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
