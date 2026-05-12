#!/usr/bin/env python3
"""Bootstrap codex-loop tracking artifacts.

Creates .codex/loop/<name>/ with request.md, state.json, task files, and
memory/MEMORY.md. Refuses to overwrite existing state.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


NAME_RE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]*$")


def now_iso() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def fail(message: str, code: int = 1) -> int:
    print(f"init-tracking: {message}", file=sys.stderr)
    return code


def atomic_write_text(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = path.with_name(f".{path.name}.tmp")
    tmp.write_text(text, encoding="utf-8")
    os.replace(tmp, path)


def atomic_write_json(path: Path, payload: dict[str, Any]) -> None:
    atomic_write_text(path, json.dumps(payload, indent=2, sort_keys=True) + "\n")


def read_request(args: argparse.Namespace) -> str:
    if args.request_file:
        return Path(args.request_file).read_text(encoding="utf-8")
    if args.request:
        return args.request
    return ""


def read_tasks(args: argparse.Namespace) -> list[Any]:
    raw = ""
    if args.tasks_file:
        raw = Path(args.tasks_file).read_text(encoding="utf-8")
    elif args.tasks_json:
        raw = args.tasks_json
    if not raw.strip():
        raise ValueError("tasks JSON is required")
    payload = json.loads(raw)
    if not isinstance(payload, list):
        raise ValueError("tasks JSON must be a list")
    return payload


def normalize_acceptance(value: Any) -> list[str]:
    if value is None:
        return []
    if isinstance(value, str):
        text = value.strip()
        return [text] if text else []
    if isinstance(value, list):
        items: list[str] = []
        for item in value:
            text = str(item).strip()
            if text:
                items.append(text)
        return items
    text = str(value).strip()
    return [text] if text else []


def normalize_tasks(payload: list[Any], timestamp: str) -> list[dict[str, Any]]:
    tasks: list[dict[str, Any]] = []
    for index, item in enumerate(payload, start=1):
        if isinstance(item, str):
            title = item.strip()
            description = ""
            acceptance: list[str] = []
        elif isinstance(item, dict):
            title = str(item.get("title", "")).strip()
            description = str(item.get("description", "")).strip()
            acceptance = normalize_acceptance(item.get("acceptance"))
        else:
            raise ValueError(f"task {index} must be a string or object")
        if not title:
            raise ValueError(f"task {index} is missing title")
        tasks.append(
            {
                "id": f"task-{index:03d}",
                "title": title,
                "description": description,
                "acceptance": acceptance,
                "status": "pending",
                "memory": None,
                "created_at": timestamp,
                "completed_at": None,
            }
        )
    if not tasks:
        raise ValueError("at least one task is required")
    return tasks


def markdown_list(items: list[str]) -> str:
    if not items:
        return "- No explicit acceptance criteria recorded.\n"
    return "".join(f"- {item}\n" for item in items)


def write_task_files(loop_dir: Path, tasks: list[dict[str, Any]]) -> None:
    tasks_dir = loop_dir / "tasks"
    tasks_dir.mkdir(parents=True, exist_ok=True)
    for task in tasks:
        body = [
            f"# {task['id']}: {task['title']}",
            "",
            "## Description",
            "",
            task["description"] or "No description recorded.",
            "",
            "## Acceptance",
            "",
            markdown_list(task["acceptance"]).rstrip(),
            "",
        ]
        atomic_write_text(tasks_dir / f"{task['id']}.md", "\n".join(body))


def write_request(loop_dir: Path, name: str, request_text: str, goal: str, tasks: list[dict[str, Any]]) -> None:
    lines = [
        f"# Codex Loop Request: {name}",
        "",
        "## Goal",
        "",
        goal or "No separate goal recorded.",
        "",
        "## Original Request",
        "",
        request_text or "No request text recorded.",
        "",
        "## Tasks",
        "",
    ]
    for task in tasks:
        lines.append(f"- {task['id']}: {task['title']}")
    atomic_write_text(loop_dir / "request.md", "\n".join(lines) + "\n")


def write_memory(loop_dir: Path, name: str) -> None:
    skill_dir = Path(__file__).resolve().parents[1]
    template = (skill_dir / "assets" / "memory.template.md").read_text(encoding="utf-8")
    atomic_write_text(loop_dir / "memory" / "MEMORY.md", template.replace("{{ loop_name }}", name))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("name")
    parser.add_argument("--root", default=".codex/loop")
    parser.add_argument("--request", default="")
    parser.add_argument("--request-file", default="")
    parser.add_argument("--goal", default="")
    parser.add_argument("--tasks-json", default="")
    parser.add_argument("--tasks-file", default="")
    parser.add_argument("--verification-command", default="")
    args = parser.parse_args()

    if not NAME_RE.match(args.name):
        return fail("name must start with an alphanumeric character and contain only letters, numbers, dots, underscores, or hyphens")

    loop_dir = Path(args.root) / args.name
    state_path = loop_dir / "state.json"
    if state_path.exists():
        return fail(f"{state_path} already exists; refusing to overwrite", 2)

    try:
        request_text = read_request(args)
        tasks_payload = read_tasks(args)
        timestamp = now_iso()
        tasks = normalize_tasks(tasks_payload, timestamp)
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        return fail(str(exc))

    loop_dir.mkdir(parents=True, exist_ok=True)
    write_request(loop_dir, args.name, request_text, args.goal, tasks)
    write_memory(loop_dir, args.name)
    write_task_files(loop_dir, tasks)

    state = {
        "schema_version": 1,
        "loop_name": args.name,
        "created_at": timestamp,
        "updated_at": timestamp,
        "iteration": 0,
        "status": "active",
        "request": {"text": request_text, "goal": args.goal},
        "verification": {
            "command": args.verification_command,
            "status": None,
            "evidence": "",
            "last_run": None,
        },
        "tasks": tasks,
        "current_task": None,
        "blockers": [],
        "history": [],
    }
    atomic_write_json(state_path, state)
    print(f"init-tracking: wrote {state_path} with {len(tasks)} task(s)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
