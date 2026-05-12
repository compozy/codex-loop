Goal (incl. success criteria):
- Implement the approved built-in codex-loop tracking skill plan, revised so tracking is built in by default.
- Success: bundled codex-loop skill gains tracking workflow docs/assets/scripts; runtime/install defaults enable optional_skill_name="codex-loop" by default, installs the managed skill copy under CODEX_HOME/skills/codex-loop, preserves custom configured skills, updates docs/tests, and make verify passes.
Constraints/Assumptions:
- No destructive git commands.
- Use RTK for shell commands.
- Use skill-best-practices, skill-load-tips, openai-docs, and golang-pro.
- Accepted plan persisted at .codex/plans/2026-05-12-built-in-tracking-skill.md.
- Durable artifacts must be in English.
- Do not touch unrelated dirty worktree changes.
Key decisions:
- Expand existing plugin skill rather than adding a second skill.
- Default tracking root is .codex/loop/<name>/.
- Runtime tracking skill wiring is built in by default: optional_skill_name="codex-loop".
- The installer must make the default real by copying the embedded skill bundle to ${CODEX_HOME:-$HOME/.codex}/skills/codex-loop/.
- If that skill path already contains unmanaged content, install must fail rather than overwrite it.
- Keep AGH/Compozy/CodeRabbit behavior out of the built-in generic skill.
State:
- Follow-up change complete: built-in tracking is default and codex-loop install physically installs the managed Codex skill.
Done:
- Read RTK, skill instructions, existing plugin skill, cy-codex-loop source, OpenAI docs, runtime config behavior, and relevant tests.
- User approved the proposed plan.
- Persisted accepted plan and session ledger.
- Reworked bundled codex-loop skill into setup/tracking dispatcher with references, assets, and helper scripts.
- Updated runtime optional skill handling to allow name-only plugin skills.
- Added focused runtime and plugin skill tests.
- Updated README and plugin README with tracking and name-only config guidance.
- Fixed blocked-task tracking invariant so blocked tasks force blocked loop state.
- Validation passed: metadata validator, focused tests, go vet ./..., make verify.
- User corrected product behavior: built-in tracking should be default, not opt-in.
- Runtime defaults now set optional_skill_name="codex-loop".
- codex-loop install now creates that default and migrates missing/blank optional_skill_name to codex-loop while preserving non-empty custom skill names.
- Docs and accepted plan updated to remove opt-in language.
- Validation passed after default change: make fmt, go test ./internal/{loop,installer,pluginmeta}, go vet ./..., make verify.
- Added embedded plugin skill package and installer copy/remove behavior for managed ${CODEX_HOME}/skills/codex-loop.
- Added installer tests for skill copy, unmanaged collision refusal, managed uninstall removal, and unmanaged skill preservation.
- Updated README, plugin README, skill references, and accepted plan with installed skill path and uninstall semantics.
- Focused validation passed: make fmt, go test ./internal/installer, go test ./internal/loop, go test ./internal/pluginmeta, go test ./plugins/codex-loop.
- Final validation passed: go vet ./..., make verify.
Now:
- Prepare final response.
Next:
- None.
Open questions (UNCONFIRMED if needed):
- None.
Working set (files/ids/commands):
- .codex/plans/2026-05-12-built-in-tracking-skill.md
- .codex/ledger/2026-05-12-MEMORY-built-in-tracking-skill.md
- plugins/codex-loop/skills/codex-loop/
- plugins/codex-loop/bundle.go
- internal/loop/config.go, internal/loop/hooks.go, internal/loop/*_test.go, internal/pluginmeta/pluginmeta_test.go
- internal/loop/store.go, internal/installer/installer.go, internal/installer/installer_test.go
- README.md, plugins/codex-loop/README.md
- Commands: make fmt, metadata validator, go test ./internal/pluginmeta, go test ./internal/loop, go test ./internal/installer, go test ./plugins/codex-loop, go vet ./..., make verify.
