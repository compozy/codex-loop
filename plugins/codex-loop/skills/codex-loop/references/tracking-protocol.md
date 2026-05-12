# Tracking Protocol

The tracking lane keeps one durable task state under `.codex/loop/<name>/`.
Each continuation performs exactly one coherent action, updates memory, updates
state through the helper, prints an iteration summary, and stops.

## Bootstrap

1. Derive `<name>` from `[[CODEX_LOOP name="..."]]` when present.
2. Capture the original user request in `request.md`.
3. Use the user-provided tasks when they exist. Otherwise decompose the request
   into task objects before bootstrapping.
4. Run `init-tracking.py <name> --request-file <path> --tasks-json '<json>'`
   from the workspace root.
5. Re-run `detect-next.py <name>`; the next action should be the first task.

## One Iteration

1. Run `detect-next.py <name>` first.
2. Read `.codex/loop/<name>/state.json` and the printed task file, if any.
3. Do only the printed action. Do not chain multiple task completions or a task
   plus verification in one iteration.
4. Write the current memory file before changing state.
5. Call `update-tracking.py` with the state transition and memory path.
6. Run `validate-tracking.py <name>`.
7. Fill `assets/iteration-summary.template.md` and make it the final block of
   the assistant response for non-final iterations.

## Memory

The shared memory file is `.codex/loop/<name>/memory/MEMORY.md`. Per-iteration
memory files use `memory/iter-NNN.md`, where `NNN` matches the state iteration
that completes the action.

Every memory update should capture:

- objective snapshot;
- important decisions;
- files and surfaces touched;
- validation evidence;
- errors, corrections, or blockers;
- ready-for-next-run notes.

Promote information to `MEMORY.md` only when future iterations need it, it is
durable, and it is not already obvious from the request or repository.

## Blockers

Record blockers with `update-tracking.py --blocker "<summary>"` when the loop
cannot advance safely. The next action becomes `resolve_blocker` until blockers
are cleared. Clear blockers only after resolving the underlying issue and
recording evidence in memory.

## Verification

Mark verification `PASS` only after running the repository-appropriate gate.
For this repository, production code, plugin, or behavior changes require
`make verify`. If verification fails, fix the production issue or task artifact
instead of weakening the gate.

## Done

Done requires all tasks completed, no open blockers, verification `PASS`, and
`validate-tracking.py <name> --expect-done` success. The final assistant message
prints the iteration summary, then the literal done signature on its own final
line.
