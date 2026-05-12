# Codex Loop Plugin

Codex plugin bundle for the `codex-loop` CLI.

The plugin contributes:

- `skills/codex-loop/SKILL.md`: setup, activation, continuation config, and restart-safe tracking workflow.
- `skills/codex-loop/{references,assets,scripts}/`: tracking protocol docs, templates, JSON schema, and stdlib-only helper scripts.
- `hooks/hooks.json`: Codex lifecycle hooks for `UserPromptSubmit` and `Stop`.

The hook commands call the runtime binary installed by:

```bash
go install github.com/compozy/codex-loop/cmd/codex-loop@latest
codex-loop install
```

Existing installs can be updated with:

```bash
codex-loop upgrade              # latest GitHub release
codex-loop upgrade --version v0.1.1
```

`codex-loop install` also mirrors those managed hook registrations into
`~/.codex/hooks.json` and copies the bundled skill into
`${CODEX_HOME:-$HOME/.codex}/skills/codex-loop/` so current Codex builds can
resolve `optional_skill_name = "codex-loop"` by name without overwriting
unrelated user hooks or unmanaged skills. The bundled Stop hook default timeout
is 2700 seconds so goal confirmation can run slow reasoning models;
user-specific timeout changes live in `~/.codex/codex-loop/config.toml` and
require rerunning `codex-loop install`.

Activation supports exactly one limiter:

```text
[[CODEX_LOOP name="qa" min="6h"]]
[[CODEX_LOOP name="qa" rounds="3"]]
[[CODEX_LOOP name="qa" goal="finish only when verified"]]
```

Goal loops confirm completion with a configurable headless command that returns normal text. The default confirmation command is `codex exec --yolo`, `gpt-5.5`, and reasoning effort `high`; custom runners can be configured with `[goal].confirm_command` as a shell-like string that codex-loop parses to argv before direct execution. codex-loop then privately interprets the text with a fixed `codex exec --output-schema` step; the default interpreter uses `gpt-5.4-mini` and reasoning effort `low`.

## Built-In Tracking Skill

The bundled `codex-loop` skill can also drive restart-safe tracked execution.
It creates project-local artifacts under `.codex/loop/<name>/`:

```text
.codex/loop/<name>/
  request.md
  state.json
  tasks/task-001.md
  memory/MEMORY.md
  memory/iter-001.md
```

The helper scripts are deliberately local and stdlib-only:

- `init-tracking.py`: bootstrap state, request, memory, and task files.
- `detect-next.py`: print the single next tracked action.
- `update-tracking.py`: apply task, blocker, verification, and history transitions.
- `validate-tracking.py`: check artifact consistency before summaries or final completion.

Continuation prompts name the installed built-in skill by default without a
workspace path:

```toml
optional_skill_name = "codex-loop"
extra_continuation_guidance = "Resume from .codex/loop/<name>/state.json and run one tracked action."
```

`codex-loop install` creates new runtime configs with that default and migrates
missing or blank `optional_skill_name` values to `codex-loop`. Non-empty
custom skill names are preserved. When `optional_skill_path` is set for a
custom workspace skill, the existing inside-workspace validation still applies.
The managed `skills/codex-loop` copy is refreshed on install and removed by
`codex-loop uninstall` only when the codex-loop management marker is present.
If `skills/codex-loop` already exists without that marker, install stops instead
of overwriting the unmanaged skill.

After installing or updating the plugin or runtime, restart Codex.
