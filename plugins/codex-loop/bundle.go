package codexloopplugin

import "embed"

// SkillFS contains the bundled Codex Loop skill files installed by the runtime.
//
//go:embed skills/codex-loop
var SkillFS embed.FS
