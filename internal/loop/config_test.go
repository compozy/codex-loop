package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeConfigDefaultsPreLoopContinue(t *testing.T) {
	t.Parallel()

	cfg := LoadRuntimeConfig(mustPaths(t))

	if cfg.PreLoopContinue.CWD != PreLoopContinueCWDSession {
		t.Fatalf("expected default cwd %q, got %q", PreLoopContinueCWDSession, cfg.PreLoopContinue.CWD)
	}
	if cfg.PreLoopContinue.TimeoutSeconds != DefaultPreLoopContinueTimeoutSeconds {
		t.Fatalf("expected default timeout %d, got %d", DefaultPreLoopContinueTimeoutSeconds, cfg.PreLoopContinue.TimeoutSeconds)
	}
	if cfg.PreLoopContinue.MaxOutputBytes != DefaultPreLoopContinueMaxOutputBytes {
		t.Fatalf("expected default max output %d, got %d", DefaultPreLoopContinueMaxOutputBytes, cfg.PreLoopContinue.MaxOutputBytes)
	}
	if cfg.PreLoopContinue.Command != "" {
		t.Fatalf("expected disabled pre_loop_continue command, got %#v", cfg.PreLoopContinue.Command)
	}
	if cfg.OptionalSkillName != DefaultOptionalSkillName {
		t.Fatalf("expected default optional skill %q, got %q", DefaultOptionalSkillName, cfg.OptionalSkillName)
	}
	if cfg.OptionalSkillPath != "" {
		t.Fatalf("expected default optional skill path empty, got %#v", cfg.OptionalSkillPath)
	}
	if cfg.Hooks.StopTimeoutSeconds != DefaultStopHookTimeoutSeconds {
		t.Fatalf("expected stop timeout %d, got %d", DefaultStopHookTimeoutSeconds, cfg.Hooks.StopTimeoutSeconds)
	}
	if cfg.Goal.ConfirmModel != DefaultGoalConfirmModel {
		t.Fatalf("expected goal model %q, got %q", DefaultGoalConfirmModel, cfg.Goal.ConfirmModel)
	}
	if cfg.Goal.ConfirmReasoningEffort != DefaultGoalConfirmReasoningEffort {
		t.Fatalf("expected goal reasoning %q, got %q", DefaultGoalConfirmReasoningEffort, cfg.Goal.ConfirmReasoningEffort)
	}
	if cfg.Goal.InterpretModel != DefaultGoalInterpretModel {
		t.Fatalf("expected goal interpreter model %q, got %q", DefaultGoalInterpretModel, cfg.Goal.InterpretModel)
	}
	if cfg.Goal.InterpretReasoningEffort != DefaultGoalInterpretReasoningEffort {
		t.Fatalf("expected goal interpreter reasoning %q, got %q", DefaultGoalInterpretReasoningEffort, cfg.Goal.InterpretReasoningEffort)
	}
	if cfg.Goal.InterpretTimeoutSeconds != DefaultGoalInterpretTimeoutSeconds {
		t.Fatalf("expected goal interpreter timeout %d, got %d", DefaultGoalInterpretTimeoutSeconds, cfg.Goal.InterpretTimeoutSeconds)
	}
}

func TestRuntimeConfigParsesPreLoopContinue(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	writeRuntimeConfig(t, paths, `[pre_loop_continue]
command = "scripts/pre-loop --json"
cwd = "workspace_root"
timeout_seconds = 7
max_output_bytes = 42
`)

	cfg := LoadRuntimeConfig(paths)

	if cfg.PreLoopContinue.Command != "scripts/pre-loop --json" {
		t.Fatalf("unexpected command %#v", cfg.PreLoopContinue.Command)
	}
	if cfg.PreLoopContinue.CWD != PreLoopContinueCWDWorkspace {
		t.Fatalf("unexpected cwd %q", cfg.PreLoopContinue.CWD)
	}
	if cfg.PreLoopContinue.TimeoutSeconds != 7 {
		t.Fatalf("unexpected timeout %d", cfg.PreLoopContinue.TimeoutSeconds)
	}
	if cfg.PreLoopContinue.MaxOutputBytes != 42 {
		t.Fatalf("unexpected max output %d", cfg.PreLoopContinue.MaxOutputBytes)
	}
}

func TestResolveOptionalContinuationConfigAllowsNameWithoutPath(t *testing.T) {
	t.Parallel()

	cfg := RuntimeConfig{
		OptionalSkillName:         " codex-loop ",
		ExtraContinuationGuidance: " Resume tracked state. ",
	}

	resolved := ResolveOptionalContinuationConfig(cfg, t.TempDir())

	if resolved.SkillName != "codex-loop" {
		t.Fatalf("expected skill name, got %#v", resolved.SkillName)
	}
	if resolved.SkillPath != "" {
		t.Fatalf("expected no skill path, got %#v", resolved.SkillPath)
	}
	if resolved.ExtraGuidance != "Resume tracked state." {
		t.Fatalf("unexpected guidance %#v", resolved.ExtraGuidance)
	}
}

func TestResolveOptionalContinuationConfigDropsInvalidPathEvenWithName(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	cfg := RuntimeConfig{
		OptionalSkillName: "focused-qa",
		OptionalSkillPath: "../outside/SKILL.md",
	}

	resolved := ResolveOptionalContinuationConfig(cfg, workspaceRoot)

	if resolved.SkillName != "" {
		t.Fatalf("expected invalid path to drop skill name, got %#v", resolved.SkillName)
	}
	if resolved.SkillPath != "" {
		t.Fatalf("expected invalid path to be empty, got %#v", resolved.SkillPath)
	}
}

func TestRuntimeConfigParsesGoalAndHookSettings(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	writeRuntimeConfig(t, paths, `[hooks]
stop_timeout_seconds = 160

[goal]
confirm_model = "gpt-5.4"
confirm_reasoning_effort = "xhigh"
confirm_command = "bin/codex exec $MODEL_ARGV $REASONING_ARGV $PROMPT_FILE"
timeout_seconds = 90
interpret_model = "gpt-5.4-mini"
interpret_reasoning_effort = "medium"
interpret_timeout_seconds = 20
max_output_bytes = 77
`)

	cfg := LoadRuntimeConfig(paths)

	if cfg.Hooks.StopTimeoutSeconds != 160 {
		t.Fatalf("unexpected stop timeout %d", cfg.Hooks.StopTimeoutSeconds)
	}
	if cfg.Goal.ConfirmModel != "gpt-5.4" {
		t.Fatalf("unexpected model %q", cfg.Goal.ConfirmModel)
	}
	if cfg.Goal.ConfirmReasoningEffort != "xhigh" {
		t.Fatalf("unexpected reasoning %q", cfg.Goal.ConfirmReasoningEffort)
	}
	if cfg.Goal.ConfirmCommand != "bin/codex exec $MODEL_ARGV $REASONING_ARGV $PROMPT_FILE" {
		t.Fatalf("unexpected confirm command %#v", cfg.Goal.ConfirmCommand)
	}
	if cfg.Goal.TimeoutSeconds != 90 {
		t.Fatalf("unexpected goal timeout %d", cfg.Goal.TimeoutSeconds)
	}
	if cfg.Goal.InterpretModel != "gpt-5.4-mini" {
		t.Fatalf("unexpected interpreter model %q", cfg.Goal.InterpretModel)
	}
	if cfg.Goal.InterpretReasoningEffort != "medium" {
		t.Fatalf("unexpected interpreter reasoning %q", cfg.Goal.InterpretReasoningEffort)
	}
	if cfg.Goal.InterpretTimeoutSeconds != 20 {
		t.Fatalf("unexpected interpreter timeout %d", cfg.Goal.InterpretTimeoutSeconds)
	}
	if cfg.Goal.MaxOutputBytes != 77 {
		t.Fatalf("unexpected max output %d", cfg.Goal.MaxOutputBytes)
	}
}

func TestRuntimeConfigAllowsBlankGoalModelAndReasoning(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	writeRuntimeConfig(t, paths, `[goal]
confirm_model = ""
confirm_reasoning_effort = ""
interpret_model = ""
interpret_reasoning_effort = ""
`)

	cfg := LoadRuntimeConfig(paths)

	if cfg.Goal.ConfirmModel != "" {
		t.Fatalf("expected blank goal model, got %q", cfg.Goal.ConfirmModel)
	}
	if cfg.Goal.ConfirmReasoningEffort != "" {
		t.Fatalf("expected blank goal reasoning, got %q", cfg.Goal.ConfirmReasoningEffort)
	}
	if cfg.Goal.InterpretModel != "" {
		t.Fatalf("expected blank goal interpreter model, got %q", cfg.Goal.InterpretModel)
	}
	if cfg.Goal.InterpretReasoningEffort != "" {
		t.Fatalf("expected blank goal interpreter reasoning, got %q", cfg.Goal.InterpretReasoningEffort)
	}
}

func TestRuntimeConfigDefaultGoalConfirmCommandUsesYolo(t *testing.T) {
	t.Parallel()

	cfg := LoadRuntimeConfig(mustPaths(t))

	if !strings.Contains(cfg.Goal.ConfirmCommand, "--yolo") {
		t.Fatalf("expected default confirm command to include --yolo, got %#v", cfg.Goal.ConfirmCommand)
	}
	if !strings.Contains(cfg.Goal.ConfirmCommand, "$MODEL_ARGV") {
		t.Fatalf("expected default confirm command to include model argv placeholder, got %#v", cfg.Goal.ConfirmCommand)
	}
	if !strings.Contains(cfg.Goal.ConfirmCommand, "$REASONING_ARGV") {
		t.Fatalf("expected default confirm command to include reasoning argv placeholder, got %#v", cfg.Goal.ConfirmCommand)
	}
	if strings.Contains(cfg.Goal.ConfirmCommand, "--output-schema") {
		t.Fatalf("expected default confirm command to return text without structured output, got %#v", cfg.Goal.ConfirmCommand)
	}
	if !strings.Contains(cfg.Goal.ConfirmCommand, "$CONFIRM_OUTPUT_PATH") {
		t.Fatalf("expected default confirm command to include confirmation output placeholder, got %#v", cfg.Goal.ConfirmCommand)
	}
	interpretCommand := defaultGoalInterpreterCodexExec()
	if !strings.Contains(interpretCommand, "--output-schema") {
		t.Fatalf("expected fixed interpreter command to include structured output, got %#v", interpretCommand)
	}
	if !strings.Contains(interpretCommand, "$INTERPRET_SCHEMA_PATH") {
		t.Fatalf("expected fixed interpreter command to include interpreter schema placeholder, got %#v", interpretCommand)
	}
	if !strings.Contains(interpretCommand, "$INTERPRET_OUTPUT_PATH") {
		t.Fatalf("expected fixed interpreter command to include interpreter output placeholder, got %#v", interpretCommand)
	}
}

func TestRuntimeConfigNormalizesInvalidGoalValues(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	writeRuntimeConfig(t, paths, `[hooks]
stop_timeout_seconds = 40

[goal]
confirm_reasoning_effort = "invalid"
timeout_seconds = 100
interpret_reasoning_effort = "also-invalid"
interpret_timeout_seconds = 100
max_output_bytes = 0
`)

	cfg := LoadRuntimeConfig(paths)

	if cfg.Goal.ConfirmReasoningEffort != DefaultGoalConfirmReasoningEffort {
		t.Fatalf("expected default reasoning, got %q", cfg.Goal.ConfirmReasoningEffort)
	}
	if cfg.Goal.TimeoutSeconds != 9 {
		t.Fatalf("expected goal timeout normalized to 9, got %d", cfg.Goal.TimeoutSeconds)
	}
	if cfg.Goal.InterpretReasoningEffort != DefaultGoalInterpretReasoningEffort {
		t.Fatalf("expected default interpreter reasoning, got %q", cfg.Goal.InterpretReasoningEffort)
	}
	if cfg.Goal.InterpretTimeoutSeconds != 1 {
		t.Fatalf("expected interpreter timeout normalized to 1, got %d", cfg.Goal.InterpretTimeoutSeconds)
	}
	if cfg.Goal.MaxOutputBytes != DefaultGoalMaxOutputBytes {
		t.Fatalf("expected default max output, got %d", cfg.Goal.MaxOutputBytes)
	}
}

func TestRuntimeConfigNormalizesPreLoopContinueDefaults(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	writeRuntimeConfig(t, paths, `[pre_loop_continue]
command = "/bin/echo"
timeout_seconds = 0
max_output_bytes = -1
`)

	cfg := LoadRuntimeConfig(paths)

	if cfg.PreLoopContinue.CWD != PreLoopContinueCWDSession {
		t.Fatalf("expected default cwd, got %q", cfg.PreLoopContinue.CWD)
	}
	if cfg.PreLoopContinue.TimeoutSeconds != DefaultPreLoopContinueTimeoutSeconds {
		t.Fatalf("expected default timeout, got %d", cfg.PreLoopContinue.TimeoutSeconds)
	}
	if cfg.PreLoopContinue.MaxOutputBytes != DefaultPreLoopContinueMaxOutputBytes {
		t.Fatalf("expected default max output, got %d", cfg.PreLoopContinue.MaxOutputBytes)
	}
}

func TestEffectiveRuntimeConfigUsesGlobalWhenProjectConfigIsMissing(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	nested := filepath.Join(repoRoot, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("create nested repo dir: %v", err)
	}
	writeRuntimeConfig(t, paths, `[pre_loop_continue]
command = "/bin/echo global"
cwd = "workspace_root"
timeout_seconds = 11
max_output_bytes = 22
`)

	cfg := LoadEffectiveRuntimeConfig(paths, nested, repoRoot)

	if cfg.PreLoopContinue.Command != "/bin/echo global" {
		t.Fatalf("unexpected command %#v", cfg.PreLoopContinue.Command)
	}
	if cfg.PreLoopContinue.CWD != PreLoopContinueCWDWorkspace {
		t.Fatalf("unexpected cwd %q", cfg.PreLoopContinue.CWD)
	}
	if cfg.PreLoopContinue.TimeoutSeconds != 11 {
		t.Fatalf("unexpected timeout %d", cfg.PreLoopContinue.TimeoutSeconds)
	}
	if cfg.PreLoopContinue.MaxOutputBytes != 22 {
		t.Fatalf("unexpected max output %d", cfg.PreLoopContinue.MaxOutputBytes)
	}
}

func TestEffectiveRuntimeConfigOverlaysProjectFields(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	nested := filepath.Join(repoRoot, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("create nested repo dir: %v", err)
	}
	writeRuntimeConfig(t, paths, `[goal]
confirm_model = ""
confirm_reasoning_effort = ""

[pre_loop_continue]
command = "/bin/echo global"
cwd = "workspace_root"
timeout_seconds = 11
max_output_bytes = 22
`)
	writeProjectRuntimeConfig(t, repoRoot, `[pre_loop_continue]
timeout_seconds = 5
`)

	cfg := LoadEffectiveRuntimeConfig(paths, nested, repoRoot)

	if cfg.PreLoopContinue.Command != "/bin/echo global" {
		t.Fatalf("expected inherited command, got %#v", cfg.PreLoopContinue.Command)
	}
	if cfg.PreLoopContinue.CWD != PreLoopContinueCWDWorkspace {
		t.Fatalf("expected inherited cwd, got %q", cfg.PreLoopContinue.CWD)
	}
	if cfg.PreLoopContinue.TimeoutSeconds != 5 {
		t.Fatalf("expected project timeout, got %d", cfg.PreLoopContinue.TimeoutSeconds)
	}
	if cfg.PreLoopContinue.MaxOutputBytes != 22 {
		t.Fatalf("expected inherited max output, got %d", cfg.PreLoopContinue.MaxOutputBytes)
	}
	if cfg.Goal.ConfirmModel != "" {
		t.Fatalf("expected inherited blank goal model, got %q", cfg.Goal.ConfirmModel)
	}
	if cfg.Goal.ConfirmReasoningEffort != "" {
		t.Fatalf("expected inherited blank goal reasoning, got %q", cfg.Goal.ConfirmReasoningEffort)
	}
}

func TestEffectiveRuntimeConfigProjectBlankCommandDisablesGlobalCommand(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	writeRuntimeConfig(t, paths, `[pre_loop_continue]
command = "/bin/echo global"
timeout_seconds = 11
`)
	writeProjectRuntimeConfig(t, repoRoot, `[pre_loop_continue]
command = ""
`)

	cfg := LoadEffectiveRuntimeConfig(paths, repoRoot, repoRoot)

	if cfg.PreLoopContinue.Command != "" {
		t.Fatalf("expected local blank command to disable global command, got %#v", cfg.PreLoopContinue.Command)
	}
	if cfg.PreLoopContinue.TimeoutSeconds != 11 {
		t.Fatalf("expected inherited timeout, got %d", cfg.PreLoopContinue.TimeoutSeconds)
	}
}

func TestEffectiveRuntimeConfigUsesNearestProjectConfigOnly(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	nested := filepath.Join(repoRoot, "nested")
	deeper := filepath.Join(nested, "deeper")
	if err := os.MkdirAll(deeper, 0o755); err != nil {
		t.Fatalf("create nested repo dir: %v", err)
	}
	writeRuntimeConfig(t, paths, `[pre_loop_continue]
command = "/bin/echo global"
timeout_seconds = 11
`)
	writeProjectRuntimeConfig(t, repoRoot, `[pre_loop_continue]
command = "/bin/echo root"
`)
	writeProjectRuntimeConfig(t, nested, `[pre_loop_continue]
timeout_seconds = 3
`)

	cfg := LoadEffectiveRuntimeConfig(paths, deeper, repoRoot)

	if cfg.PreLoopContinue.Command != "/bin/echo global" {
		t.Fatalf("expected nearest partial config to inherit from global instead of root config, got %#v", cfg.PreLoopContinue.Command)
	}
	if cfg.PreLoopContinue.TimeoutSeconds != 3 {
		t.Fatalf("expected nearest timeout, got %d", cfg.PreLoopContinue.TimeoutSeconds)
	}
}

func TestEffectiveRuntimeConfigDoesNotReadAboveWorkspaceRoot(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	parent := t.TempDir()
	repoRoot := filepath.Join(parent, "repo")
	nested := filepath.Join(repoRoot, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("create nested repo dir: %v", err)
	}
	writeRuntimeConfig(t, paths, `[pre_loop_continue]
command = "/bin/echo global"
`)
	writeProjectRuntimeConfig(t, parent, `[pre_loop_continue]
command = "/bin/echo parent"
`)

	cfg := LoadEffectiveRuntimeConfig(paths, nested, repoRoot)

	if cfg.PreLoopContinue.Command != "/bin/echo global" {
		t.Fatalf("expected global command because parent config is above workspace, got %#v", cfg.PreLoopContinue.Command)
	}
}

func writeProjectRuntimeConfig(t *testing.T, dir string, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create project config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, projectRuntimeConfigFilename), []byte(content), 0o644); err != nil {
		t.Fatalf("write project runtime config: %v", err)
	}
}
