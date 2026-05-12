---
name: codex-loop
description: Guides Codex Loop setup and long-running tracked execution by installing or refreshing the runtime, explaining activation headers, bootstrapping durable tracking artifacts, decomposing broad requests into tasks, and maintaining per-iteration memory and state. Use for Codex Loop installation, status, activation, or tracked restart-safe implementation, QA, review, and remediation loops. Do not use for one-off coding tasks that do not need Codex Loop or persistent tracking.
---

# Codex Loop

Use this skill for two lanes: runtime setup and restart-safe tracked execution.
The tracking lane creates project-local artifacts under `.codex/loop/<name>/`
so every continuation can resume from durable state instead of chat memory.

## Required Reading Router

Match the task to the row. Read the listed files in full before producing
output or running helper scripts. Inline content in this file is a dispatcher,
not the contract.

| Task | MUST read |
| --- | --- |
| Install, refresh, upgrade, status, uninstall, or explain activation | `references/runtime-setup.md` |
| Configure automatic continuations or project `codex-loop.toml` | `references/continuation-config.md` + `references/runtime-setup.md` |
| Bootstrap a tracked loop or create tasks from a broad request | `references/tracking-protocol.md` + `references/task-decomposition.md` + `references/state-schema.md` |
| Resume a tracked loop after a Codex Loop continuation | `references/tracking-protocol.md` + `references/phase-transitions.md` + `references/state-schema.md` |
| Audit or finalize a tracked loop | `references/checklist.md` + `references/phase-transitions.md` |

## Reference Index

- `references/runtime-setup.md`: install, upgrade, status, uninstall, activation header, and current runtime behavior.
- `references/continuation-config.md`: default built-in skill wiring with `optional_skill_name`, `optional_skill_path`, project config, and goal-header examples.
- `references/tracking-protocol.md`: bootstrap, one-action-per-iteration execution, memory-before-state sequencing, blockers, summaries, and done signature.
- `references/task-decomposition.md`: conversion of broad requests into task objects accepted by `init-tracking.py`.
- `references/state-schema.md`: authoritative `state.json` fields, status values, and invariants.
- `references/phase-transitions.md`: deterministic next-action mapping used by `detect-next.py`.
- `references/checklist.md`: required self-audit before iteration summaries and final completion.

## Helper Scripts

Resolve `<codex-loop-skill-dir>` to the directory containing this `SKILL.md`.
Invoke helpers with `python3 <codex-loop-skill-dir>/scripts/<name>.py`; do
not rely on the current working directory matching the skill directory.

| Script | Role | Purpose |
| --- | --- | --- |
| `scripts/init-tracking.py` | bootstrap, mutating | Create `.codex/loop/<name>/`, `request.md`, `state.json`, memory, and task files. |
| `scripts/detect-next.py` | read-only | Print the single next action for the current tracked loop. |
| `scripts/update-tracking.py` | mutating | Apply task, blocker, verification, and history transitions atomically. |
| `scripts/validate-tracking.py` | read-only | Validate tracking artifacts before summaries or final completion. |

## Procedures

**Step 1: Route the request**
1. Inspect whether the user needs setup/status help or active tracked execution.
2. For setup/status help, read `references/runtime-setup.md` and follow that lane only.
3. For continuation configuration, read `references/continuation-config.md` before advising or editing config.
4. For tracked execution, continue to Step 2.

**Step 2: Bootstrap or resume tracking**
1. Determine the loop name from the `[[CODEX_LOOP name="..."]]` header when present; otherwise derive a short kebab-case name from the task.
2. Run `python3 <codex-loop-skill-dir>/scripts/detect-next.py <name>` from the workspace root.
3. If the action is `bootstrap`, gather the original request and task list. If the user did not provide tasks, decompose the request first.
4. Run `python3 <codex-loop-skill-dir>/scripts/init-tracking.py <name> --request-file <path> --tasks-json '<json>'` or use `--request "<text>"` for short prompts.
5. **STOP. Read `references/tracking-protocol.md`, `references/task-decomposition.md`, and `references/state-schema.md` in full before bootstrapping or resuming tracked execution.** The bullets above are route markers, not the tracking contract.

**Step 3: Execute exactly one tracked action**
1. Re-run `detect-next.py` after bootstrap or at the start of every continuation.
2. Follow the printed action exactly: `execute_task`, `verify`, `resolve_blocker`, or `done`.
3. For `execute_task`, work on only the printed task. Write the iteration memory file before calling `update-tracking.py --complete-task`.
4. For `verify`, run the repository-appropriate validation command and only pass `--verify-pass` when real evidence supports it.
5. For `resolve_blocker`, clear blockers only after the underlying issue is actually resolved and recorded in memory.
6. **STOP. Read `references/phase-transitions.md` in full before updating state.** The helper output is authoritative for the next action.

**Step 4: Audit, summarize, and stop**
1. Run `python3 <codex-loop-skill-dir>/scripts/validate-tracking.py <name>` before printing an iteration summary.
2. Fill `assets/iteration-summary.template.md` and make it the final block of the assistant message for non-final iterations.
3. For `done`, run `validate-tracking.py <name> --expect-done`, print the summary, then print the literal contents of `assets/done-signature.txt` on the final line.
4. **STOP. Read `references/checklist.md` in full before claiming an iteration or loop is complete.** The checklist is the completion gate.

## Error Handling

- If `init-tracking.py` reports that state already exists, switch to resume mode and run `detect-next.py`.
- If a helper reports invalid JSON or an invalid transition, fix the input or complete the missing prerequisite; do not hand-edit `state.json`.
- If validation fails, record the failure as a blocker with `update-tracking.py --blocker "<summary>"`, print an iteration summary with `outcome=blocked`, and stop.
- If the repository verification fails, fix production code or task artifacts rather than weakening checks; then rerun verification before marking `PASS`.
