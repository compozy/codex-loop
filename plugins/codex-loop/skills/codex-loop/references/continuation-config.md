# Continuation Config

Codex Loop continuation guidance names the bundled `codex-loop` tracking skill
by default. Installing the runtime creates or migrates the runtime config so
`optional_skill_name = "codex-loop"` unless a non-empty custom skill name is
already configured, and copies the managed skill to
`${CODEX_HOME:-$HOME/.codex}/skills/codex-loop/`.

## Global Config

The global runtime config lives at `~/.codex/codex-loop/config.toml`.

```toml
optional_skill_name = "codex-loop"
optional_skill_path = ""
extra_continuation_guidance = "Use the tracked execution lane and update .codex/loop state before stopping."
```

`optional_skill_name` can be used alone for plugin-provided skills. When
`optional_skill_path` is set for a custom workspace skill, the path must
resolve inside the active workspace and may point to a skill directory or
directly to `SKILL.md`.

## Project Config

Projects may define `codex-loop.toml` in the workspace. During an active loop,
Codex Loop searches from the hook CWD up to the resolved workspace root and uses
the nearest file. It never searches above the workspace root.

```toml
# ./codex-loop.toml
optional_skill_name = "codex-loop"
extra_continuation_guidance = "Resume from .codex/loop/<name>/state.json and run one tracked action."

[pre_loop_continue]
command = ".codex/scripts/loop-context.sh --input $INPUT_FILE"
cwd = "workspace_root"
timeout_seconds = 30
max_output_bytes = 8000
```

Effective config precedence is:

1. built-in defaults;
2. global `~/.codex/codex-loop/config.toml`;
3. nearest project `codex-loop.toml`.

Project fields overlay per field. Omitted fields inherit. A project
`[pre_loop_continue] command = ""` disables a global pre-loop command for that
project.

## Goal Header With Tracking

Use a goal header when the loop should continue until independently confirmed:

```text
[[CODEX_LOOP name="<name>" goal="complete <name> with tracked codex-loop state, all tasks done, blockers closed, and verification PASS"]]

Use the codex-loop skill. Track the work under .codex/loop/<name>/.
```

For long or high-risk work, raise the confirmation effort:

```text
[[CODEX_LOOP name="<name>" goal="..." confirm_model="gpt-5.5" confirm_reasoning_effort="xhigh"]]
```

## Install Behavior

- New installs write `optional_skill_name = "codex-loop"`.
- Existing runtime configs with missing or blank `optional_skill_name` are
  migrated to `codex-loop`.
- Existing non-empty custom skill names are preserved.
- `optional_skill_path` remains empty for the bundled skill because
  `codex-loop install` installs it into Codex's global skill directory.
- Do not set `optional_skill_path` to plugin cache paths in project config.
- Do not use project config to rewrite managed global hook registrations.
- `codex-loop uninstall` removes the installed skill directory only when the
  codex-loop management marker is present.
- If the target skill directory exists without that marker, install stops rather
  than overwriting unmanaged skill content.
