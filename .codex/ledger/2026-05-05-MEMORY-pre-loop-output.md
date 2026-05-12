Goal (incl. success criteria):
- Answer whether the `pre_loop_continue` review/command output must be JSON or can be normal text.
- Success: distinguish Codex `Stop` hook stdout contract from codex-loop `pre_loop_continue` command output contract.
Constraints/Assumptions:
- No destructive git commands.
- Use local project code/docs as authority for codex-loop behavior.
- Use official OpenAI docs for Codex hook stdout behavior.
Key decisions:
- `pre_loop_continue` is an internal codex-loop runtime hook, not a separate Codex lifecycle hook.
- The configured `pre_loop_continue.command` receives JSON input but its stdout is treated as plain prompt text.
State:
- Answer ready.
Done:
- Scanned existing ledgers for cross-agent awareness.
- Read relevant local ledgers for prior pre-loop and command configuration work.
- Checked `internal/loop/pre_loop_continue.go`, README docs, plugin hooks config, and official OpenAI Codex hook docs.
Now:
- Provide concise answer in Portuguese.
Next:
- None.
Open questions (UNCONFIRMED if needed):
- None.
Working set (files/ids/commands):
- `.codex/ledger/2026-05-05-MEMORY-pre-loop-output.md`
- `internal/loop/pre_loop_continue.go`
- `README.md`
- `plugins/codex-loop/hooks/hooks.json`
- Official docs: `https://developers.openai.com/codex/hooks#stop`, `https://developers.openai.com/codex/hooks#common-output-fields`
