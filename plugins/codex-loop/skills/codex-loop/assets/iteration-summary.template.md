# codex-loop tracking - Iteration {{ iteration }} summary

- **Loop:** {{ loop_name }}
- **Action:** {{ action }}
- **Task:** {{ task_or_none }}
- **Outcome:** {{ outcome }}
- **Memory written:** {{ memory_paths_csv }}
- **State updated:** `.codex/loop/{{ loop_name }}/state.json`
- **Verification:** {{ verification_status }} ({{ verification_evidence }})
- **Blockers:** {{ blockers_or_none }}
- **Next action:** {{ next_action }}

<!--
Fill the placeholders manually. This block must be the final assistant block
for non-final iterations. When the next action is done, print this summary and
then print the contents of assets/done-signature.txt on a separate final line.
-->
