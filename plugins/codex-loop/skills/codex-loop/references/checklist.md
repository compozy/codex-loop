# Tracking Checklist

Run this checklist before printing an iteration summary. A failed item means the
iteration is not complete.

## Every Iteration

- [ ] `detect-next.py <name>` ran first and its printed action was followed.
- [ ] Exactly one tracked action was attempted.
- [ ] Relevant request, state, task, and memory artifacts were read before work.
- [ ] Memory was written before any state transition.
- [ ] `update-tracking.py` was the only writer for `state.json`.
- [ ] `validate-tracking.py <name>` passed before the summary.
- [ ] The iteration summary block is the final assistant block for non-final iterations.

## Bootstrap

- [ ] The loop name came from the activation header or a stable kebab-case task name.
- [ ] The original request was captured in `request.md`.
- [ ] User-provided tasks were preserved, or broad work was decomposed into concrete task objects.
- [ ] Empty task lists were rejected unless this is a validation-only loop with one explicit validation task.

## Task Execution

- [ ] Only the printed task was worked.
- [ ] Acceptance criteria were checked before marking the task complete.
- [ ] The current memory file names touched surfaces, validation evidence, and next-run notes.
- [ ] Production or artifact bugs found by tests were fixed instead of weakening tests.

## Verification

- [ ] The repository-appropriate verification command ran.
- [ ] Failure output was investigated and fixed before any `PASS` transition.
- [ ] `--verify-pass` included concrete evidence text.

## Final Done

- [ ] Every task status is `completed`.
- [ ] No blockers remain.
- [ ] `verification.status` is `PASS`.
- [ ] `validate-tracking.py <name> --expect-done` passed.
- [ ] The literal contents of `assets/done-signature.txt` are the final line.
