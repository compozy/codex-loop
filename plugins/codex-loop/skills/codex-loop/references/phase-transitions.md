# Phase Transitions

`detect-next.py` is the single source of truth for the next tracked action.
Run it before each iteration and follow the printed line.

## Outputs

```text
action=bootstrap name=<name>
action=resolve_blocker name=<name> blocker_count=N
action=execute_task name=<name> task=task-001
action=verify name=<name>
action=done name=<name>
```

## Mapping

1. Missing `state.json` prints `bootstrap`.
2. Any open blocker or `status=blocked` prints `resolve_blocker`.
3. An `in_progress` task prints `execute_task` for that task.
4. The first `pending` task prints `execute_task` for that task.
5. All tasks completed with verification not `PASS` prints `verify`.
6. All tasks completed, no blockers, and verification `PASS` prints `done`.

## Execution Rules

- Execute exactly one printed action per iteration.
- Do not mark a task complete before writing its memory file.
- Do not run verification as a side effect of completing a task unless
  verification itself is the printed action.
- Do not clear blockers without evidence in memory.
- Do not print the done signature unless `validate-tracking.py --expect-done`
  succeeds.

## Failure Handling

If `detect-next.py` cannot parse state, stop and record the parse failure as a
blocker after inspecting the file. Do not rewrite state by hand. If the helper
prints an action that conflicts with visible repository truth, treat the mismatch
as a blocker and reconcile through `update-tracking.py`.
