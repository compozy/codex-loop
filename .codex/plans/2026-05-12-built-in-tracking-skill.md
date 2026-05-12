# Built-In Codex Loop Tracking Skill

## Summary

- Expand the existing bundled `codex-loop` skill into a two-lane skill: one lane keeps the current install/status/activation guidance, and the new lane drives long-running tracked work across Codex Loop restarts.
- Default tracking artifacts live under `.codex/loop/<loop-name>/`, separate from workspace-specific AGENTS ledgers. The skill will still instruct agents to obey any local AGENTS/ledger policy in addition to codex-loop tracking.
- Make runtime behavior built in by default: new installs and missing/blank runtime configs use `optional_skill_name = "codex-loop"`, while non-empty custom skill names remain overrides.
- Make the built-in behavior operational by installing the bundled skill into `${CODEX_HOME:-$HOME/.codex}/skills/codex-loop/`; a name-only `optional_skill_name` must resolve to a real installed skill.
- Do not port AGH/Compozy-specific behavior directly. The built-in version will keep the reusable pattern only: bootstrap, task decomposition, durable state, per-iteration memory, deterministic next-step detection, validation evidence, and done signature.

## Key Changes

- Rewrite `plugins/codex-loop/skills/codex-loop/SKILL.md` as a compact dispatcher with:
  - validated `name: codex-loop`;
  - a broader discovery description covering setup plus tracked loop execution;
  - a Required Reading Router in the first 50 lines;
  - a Reference Index next to the router;
  - hard `STOP. Read ...` directives before tracking, state mutation, continuation config, and finalization steps;
  - no duplicated long checklists inline.
- Add flat one-level skill references:
  - `references/runtime-setup.md`: current install, upgrade, status, uninstall, activation header guidance moved out of the main skill body.
  - `references/tracking-protocol.md`: bootstrap and one-iteration workflow, memory-before-state rule, blocker handling, and final evidence rules.
  - `references/task-decomposition.md`: how to convert a broad user request into concrete tasks when no tasks were provided.
  - `references/state-schema.md`: authoritative JSON state schema and transition invariants.
  - `references/phase-transitions.md`: how `detect-next.py` maps durable state to the next action.
  - `references/continuation-config.md`: `codex-loop.toml`, `optional_skill_name`, optional path behavior, and goal-header examples.
  - `references/checklist.md`: per-iteration self-audit and final done audit.
- Add skill assets:
  - `assets/state.schema.json`: machine-readable schema for `.codex/loop/<name>/state.json`.
  - `assets/memory.template.md`: shared memory section template.
  - `assets/iteration-summary.template.md`: final block required at the end of each tracked iteration.
  - `assets/done-signature.txt`: literal final signature, for example `__CODEX_LOOP_TRACKING_DONE__ state=complete verify=PASS`.
- Add stdlib-only helper scripts under the skill:
  - `scripts/init-tracking.py` (bootstrap, mutating): creates `.codex/loop/<name>/`, writes `request.md`, `state.json`, `memory/MEMORY.md`, and initial task records from agent-provided JSON.
  - `scripts/detect-next.py` (read-only): prints exactly one next action such as `bootstrap`, `execute_task`, `verify`, `resolve_blocker`, or `done`.
  - `scripts/update-tracking.py` (mutating): performs all state transitions, appends bounded history, records memory paths, blockers, and verification status.
  - `scripts/validate-tracking.py` (read-only): checks state/schema/artifact consistency before summaries and final done.
- Update runtime continuation config behavior narrowly:
  - `optional_skill_name = "codex-loop"` without `optional_skill_path` becomes valid and emits `Use the codex-loop skill.` in automatic continuations.
  - Existing `optional_skill_name + optional_skill_path` behavior remains supported, and paths still must resolve inside the active workspace.
  - Default runtime config sets `optional_skill_name = "codex-loop"`, and install migrates missing or blank values to that default.
  - `codex-loop install` copies the embedded skill bundle into `${CODEX_HOME:-$HOME/.codex}/skills/codex-loop/` and refreshes managed copies.
  - `codex-loop install` refuses to overwrite an existing non-empty `skills/codex-loop` directory without the codex-loop management marker.
  - `codex-loop uninstall` removes that skill directory only when the codex-loop management marker is present, preserving unmanaged user skills.
- Update docs:
  - README continuation customization section shows the name-only plugin-skill configuration.
  - Plugin README describes the built-in tracking lane and `.codex/loop/<name>/` artifact layout.
  - Keep all durable docs in English.

## Tracking Behavior

- Bootstrap:
  - Derive `<loop-name>` from the `[[CODEX_LOOP name="..."]]` header when present; otherwise use a short kebab-case task name.
  - If user-supplied tasks exist, normalize them into task objects.
  - If no tasks exist, decompose the request into concrete acceptance-driven tasks before calling `init-tracking.py`.
  - Refuse an empty task list unless the request is truly a single-step validation-only loop; in that case create one explicit validation task.
- Per iteration:
  - Run `detect-next.py` first and follow its single printed action.
  - Execute only one coherent task/slice before stopping.
  - Write/update the current iteration memory before calling `update-tracking.py`.
  - Run the repo-appropriate verification command before marking verification as `PASS`; in this repo, code changes still require `make verify`.
  - End with the iteration summary template as the last block, except final done also prints the done signature on the last line.
- Finalization:
  - `done` requires all tasks completed, no open blockers, validation status `PASS`, and `validate-tracking.py` success.
  - The skill does not invent CodeRabbit, Compozy, Claude/Opus delegation, or project-specific QA gates. Those can be added by project-local skills or explicit user instructions.

## Public Interfaces

- Skill interface:
  - Bundled skill remains `codex-loop` at `plugins/codex-loop/skills/codex-loop/`.
  - Installed managed skill lives at `${CODEX_HOME:-$HOME/.codex}/skills/codex-loop/`.
  - The skill gains `scripts/`, `references/`, and `assets/` folders; all paths are flat and one level deep.
- Runtime config interface:
  - `optional_skill_name` can now be used alone for plugin-provided skills.
  - `optional_skill_path` remains optional; when set, it keeps current workspace-bound path validation.
  - No activation header fields change.
  - No hook JSON event shape changes.
- Artifact interface:
  - `.codex/loop/<name>/request.md`
  - `.codex/loop/<name>/state.json`
  - `.codex/loop/<name>/memory/MEMORY.md`
  - `.codex/loop/<name>/memory/iter-NNN.md`
  - optional `.codex/loop/<name>/tasks/task-NNN.md` only if the implementation chooses human-readable task files in addition to `state.json`; `state.json` remains authoritative.

## Test Plan

- Extend the existing canonical plugin metadata suite under `internal/pluginmeta` rather than creating an unrelated test owner.
- Add metadata/load tests:
  - skill directory name matches `name`;
  - description validates with the skill-best-practices validator rules;
  - `SKILL.md` has one Required Reading Router near the top;
  - references/assets/scripts are flat one-level folders;
  - helper paths referenced in `SKILL.md` exist;
  - no soft "for depth, see" phrasing replaces mandatory STOP directives.
- Add helper-script integration tests from Go:
  - bootstrap creates the expected `.codex/loop/<name>/` structure and valid `state.json`;
  - detecting immediately after bootstrap returns the first pending task;
  - completing a task advances to the next task;
  - all tasks complete without verification returns `verify`;
  - verification `PASS` plus no blockers returns `done`;
  - invalid transitions fail with non-zero status and do not corrupt the previous state file;
  - state writes are atomic enough that a failed update leaves parseable JSON.
- Add runtime config tests:
  - `optional_skill_name = "codex-loop"` with no path emits a continuation prompt naming the skill;
  - existing name+path behavior still includes the resolved `SKILL.md` path;
  - paths outside the workspace are still ignored.
- Add installer tests:
  - `codex-loop install` writes `SKILL.md`, references, scripts, and a codex-loop management marker under `${CODEX_HOME}/skills/codex-loop/`;
  - reinstall refreshes managed copies;
  - install fails instead of overwriting an unmanaged `skills/codex-loop` directory;
  - uninstall removes marker-managed skill directories;
  - uninstall preserves unmanaged `skills/codex-loop` directories.
- Run final gates:
  - `make fmt`
  - `go vet ./...`
  - `make verify`

## Revision: Built-In By Default

- User corrected the earlier opt-in assumption: the bundled `codex-loop` tracking skill must be built in by default.
- Runtime defaults should set `optional_skill_name = "codex-loop"`.
- `codex-loop install` should create config with the built-in skill enabled and migrate missing/blank `optional_skill_name` values to `codex-loop` while preserving non-empty custom skill names and unrelated config.
- Documentation should describe custom skill names as overrides, not as the default path to enable tracking.

## Revision: Installed Built-In Skill

- User clarified that default naming is insufficient unless the skill is physically available to Codex.
- `codex-loop install` must copy the embedded plugin skill to `${CODEX_HOME:-$HOME/.codex}/skills/codex-loop/`.
- The installed skill directory is managed with a codex-loop marker so updates can refresh it and uninstall can remove only codex-loop-owned copies.
- If `skills/codex-loop` already exists without that marker, install must stop rather than overwrite user-managed skill content.
- Runtime config remains name-only (`optional_skill_name = "codex-loop"`, empty `optional_skill_path`) because the installer owns global skill availability.

## Assumptions

- The unanswered decisions use the recommended defaults: expand the existing `codex-loop` skill and use `.codex/loop/<name>/`.
- Implementation will use `golang-pro` because runtime config tests and Go code changes are included.
- The accepted Plan Mode plan must be persisted under `.codex/plans/` before implementation starts, per this workspace policy.
- The Memory Ledger should be created/updated during implementation, but this planning turn stayed read-only because Plan Mode forbids mutating workspace files before acceptance.
