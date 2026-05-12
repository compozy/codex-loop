# Task Decomposition

Use this reference when a tracked loop starts from a broad request and no task
list was provided.

## Task Object Shape

```json
[
  {
    "title": "Implement tracked state helpers",
    "description": "Add helper scripts that create, inspect, update, and validate loop state.",
    "acceptance": [
      "Bootstrap creates request.md, state.json, memory, and task files.",
      "Invalid transitions fail without corrupting state."
    ]
  }
]
```

`title` is required. `description` and `acceptance` are optional but strongly
preferred. `acceptance` may be a string or a list of strings.

## Decomposition Rules

1. Convert the request into acceptance-driven tasks that can each finish in one
   Codex Loop iteration.
2. Keep tasks independent enough that a future continuation can understand the
   next action from `state.json`, the task file, and memory.
3. Create a verification task when the request has implementation work but no
   explicit validation step.
4. Do not create tasks that only freeze implementation details, static prose,
   generated output, snapshots, config shape, or file existence unless that
   artifact is the product contract.
5. If the request is truly validation-only, create exactly one validation task
   with concrete evidence requirements.

## Task Sizing

A good task has:

- one clear behavioral outcome;
- concrete files or surfaces to inspect or modify;
- acceptance criteria that can be verified by commands, tests, review, or user
  visible behavior;
- enough context for a restarted agent to continue without reading chat history.

Split a task when it crosses unrelated subsystems, requires different validation
gates, or would make one iteration too broad to summarize accurately.

## No-Task Blocker

If no meaningful task can be derived, do not bootstrap an empty state file.
Explain the blocker and ask for the missing product goal or acceptance criteria.
