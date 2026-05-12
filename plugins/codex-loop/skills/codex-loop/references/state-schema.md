# State Schema

`.codex/loop/<name>/state.json` is the authoritative tracking file. It is
mutated only by `init-tracking.py` and `update-tracking.py`; hand edits void
resume guarantees.

## Top-Level Fields

| Field | Type | Meaning |
| --- | --- | --- |
| `schema_version` | integer | Current schema version, always `1`. |
| `loop_name` | string | Tracking name and artifact directory name. |
| `created_at` | RFC3339 UTC string | Bootstrap timestamp. |
| `updated_at` | RFC3339 UTC string | Last state mutation timestamp. |
| `iteration` | integer | Monotonic update counter. |
| `status` | string | `active`, `blocked`, or `complete`. |
| `request` | object | Original request text and optional goal text. |
| `verification` | object | Command, status, evidence, and last run timestamp. |
| `tasks` | array | Ordered task records. |
| `current_task` | string or null | Current task id while in progress. |
| `blockers` | array | Open blocker strings. |
| `history` | array | Bounded append-only transition log. |

## Task Fields

| Field | Type | Meaning |
| --- | --- | --- |
| `id` | string | Stable id such as `task-001`. |
| `title` | string | Human task name. |
| `description` | string | Scope and context. |
| `acceptance` | array | Acceptance criteria strings. |
| `status` | string | `pending`, `in_progress`, `completed`, or `blocked`. |
| `memory` | string or null | Latest memory path for this task. |
| `created_at` | RFC3339 UTC string | Task creation timestamp. |
| `completed_at` | RFC3339 UTC string or null | Completion timestamp. |

## Invariants

1. Task ids are unique and ordered.
2. At most one task is `in_progress`.
3. `current_task` is null unless a task is `in_progress`.
4. `status=blocked` when `blockers` is non-empty.
5. `status=complete` only when every task is completed, blockers are empty,
   and `verification.status=PASS`.
6. `history` is append-only and capped by `update-tracking.py`.
7. Verification `PASS` requires concrete evidence text.

## Artifact Layout

```text
.codex/loop/<name>/
  request.md
  state.json
  tasks/task-001.md
  memory/MEMORY.md
  memory/iter-001.md
```

`state.json` remains authoritative even when task Markdown files exist for
human scanning.
