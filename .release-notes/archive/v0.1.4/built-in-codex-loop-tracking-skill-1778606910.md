---
title: Built-in codex-loop tracking skill
type: feature
---

`codex-loop install` now ships and refreshes a bundled `codex-loop` skill at
`${CODEX_HOME:-$HOME/.codex}/skills/codex-loop/`, and new runtime configs
default to `optional_skill_name = "codex-loop"` so every automatic
continuation names the managed skill without needing a workspace path.

What you get out of the box:

- A restart-safe tracking lane that bootstraps `.codex/loop/<name>/` with
  `request.md`, `state.json`, per-task files under `tasks/`, and
  per-iteration memory under `memory/`.
- Stdlib-only helper scripts (`init-tracking.py`, `detect-next.py`,
  `update-tracking.py`, `validate-tracking.py`) for deterministic task,
  blocker, verification, and history transitions.
- Continuation and goal prompts that now honor `optional_skill_name` even
  when no `optional_skill_path` is configured, so the name-only default
  resolves as a normal Codex skill.

Install/uninstall safety:

- `codex-loop install` migrates existing configs from a missing or blank
  `optional_skill_name` to `"codex-loop"` while preserving any non-empty
  custom value.
- The managed skill copy is refreshed on every install via atomic writes.
- A `.codex-loop-managed` marker gates the directory: install refuses to
  overwrite an unmanaged `skills/codex-loop/`, and `codex-loop uninstall`
  only removes the directory when that marker is present.
