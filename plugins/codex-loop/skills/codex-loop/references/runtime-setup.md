# Runtime Setup

Use this lane for installing, refreshing, checking, or explaining Codex Loop.
Do not bootstrap tracking artifacts unless the user is running an active
restart-safe loop.

## Inspect Current State

1. Read `~/.codex/config.toml` only if it exists and explain whether
   `features.codex_hooks` is enabled.
2. Read `~/.codex/codex-loop/config.toml` only if it exists and summarize
   continuation guidance, goal confirmation settings, Stop timeout, and
   `pre_loop_continue`.
3. When a workspace is known, read the nearest `codex-loop.toml` from the
   current directory up to the workspace root. Treat it as a per-field overlay
   over global runtime config.
4. Do not hand-edit global hook or skill files for normal setup.
   `codex-loop install` syncs managed hooks and the managed built-in skill
   copy while preserving unrelated hooks and unmanaged skills.

## Install, Refresh, or Upgrade

1. If `codex-loop` is not on `PATH`, install it:
   `go install github.com/compozy/codex-loop/cmd/codex-loop@latest`
2. Run `codex-loop install`.
3. For updates, prefer `codex-loop upgrade` or
   `codex-loop upgrade --version v0.1.1` for a pinned release.
4. Report the runtime path, loop state path, installed skill path, managed hook
   config path, config path, and whether Codex plugin marketplace refresh ran
   or was skipped.
5. Tell the user to restart Codex after install or upgrade.

## Activation Header

The activation header must be the first prompt line. The task prompt starts on
the next line. Exactly one limiter is required:

```text
[[CODEX_LOOP name="release-stress-qa" min="6h"]]
[[CODEX_LOOP name="release-stress-qa" rounds="3"]]
[[CODEX_LOOP name="release-stress-qa" goal="ship only after verification"]]
```

Goal loops run a configurable headless confirmation command before continuing
or completing. Model and reasoning are separate fields:

```text
[[CODEX_LOOP name="release-stress-qa" goal="ship only after verification" confirm_model="gpt-5.5" confirm_reasoning_effort="xhigh"]]
```

Loop state is isolated by Codex `session_id`, persists under
`${CODEX_HOME:-$HOME/.codex}/codex-loop/loops/`, and goal-check metadata is
appended to `${CODEX_HOME:-$HOME/.codex}/codex-loop/runs.jsonl`.

## Commands

- `codex-loop install`: install or refresh the local runtime, managed hooks,
  and managed built-in skill at
  `${CODEX_HOME:-$HOME/.codex}/skills/codex-loop/`.
- `codex-loop upgrade`: download and install the latest release.
- `codex-loop status`: print active loop state as JSON.
- `codex-loop status --all`: include completed, superseded, and cut-short loops.
- `codex-loop uninstall`: remove managed runtime artifacts, managed hooks, and
  the managed built-in skill copy only.
- `codex-loop version`: print the CLI version.

## Error Handling

- If `go install` fails, confirm Go is installed and available on `PATH`.
- If `codex-loop install` cannot update config, inspect malformed TOML or
  filesystem permissions.
- If activation does nothing after install, restart Codex and confirm the
  plugin is installed and enabled.
- If continuations name `codex-loop` but the skill is unavailable, rerun
  `codex-loop install` and confirm
  `${CODEX_HOME:-$HOME/.codex}/skills/codex-loop/SKILL.md` exists.
- If install reports an unmanaged `skills/codex-loop` directory, move or rename
  that directory before rerunning install. Do not overwrite it silently.
