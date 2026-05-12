package loop

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

const DefaultOptionalSkillName = "codex-loop"

const DefaultRuntimeConfig = `# Optional continuation guidance appended to every automatic continuation.
# The bundled codex-loop skill is enabled by default for restart-safe tracking.
optional_skill_name = "codex-loop"
optional_skill_path = ""
extra_continuation_guidance = ""

# Managed Codex lifecycle hook settings. Re-run codex-loop install after changing these.
[hooks]
stop_timeout_seconds = 2700

# Goal-loop confirmation settings.
[goal]
confirm_model = "gpt-5.5"
confirm_reasoning_effort = "high"
confirm_command = "codex exec --cd $WORKSPACE_ROOT --ephemeral --yolo --output-last-message $CONFIRM_OUTPUT_PATH $MODEL_ARGV $REASONING_ARGV --skip-git-repo-check -"
timeout_seconds = 2400
interpret_model = "gpt-5.4-mini"
interpret_reasoning_effort = "low"
interpret_timeout_seconds = 120
max_output_bytes = 12000

# Optional command executed inside codex-loop before each automatic continuation.
[pre_loop_continue]
command = ""
cwd = "session_cwd"
timeout_seconds = 60
max_output_bytes = 12000
`

const projectRuntimeConfigFilename = "codex-loop.toml"

type RuntimeConfig struct {
	OptionalSkillName         string                `toml:"optional_skill_name"`
	OptionalSkillPath         string                `toml:"optional_skill_path"`
	ExtraContinuationGuidance string                `toml:"extra_continuation_guidance"`
	Hooks                     HooksConfig           `toml:"hooks"`
	Goal                      GoalConfig            `toml:"goal"`
	PreLoopContinue           PreLoopContinueConfig `toml:"pre_loop_continue"`
}

type HooksConfig struct {
	StopTimeoutSeconds int `toml:"stop_timeout_seconds"`
}

type GoalConfig struct {
	ConfirmModel             string `toml:"confirm_model"`
	ConfirmReasoningEffort   string `toml:"confirm_reasoning_effort"`
	ConfirmCommand           string `toml:"confirm_command"`
	TimeoutSeconds           int    `toml:"timeout_seconds"`
	InterpretModel           string `toml:"interpret_model"`
	InterpretReasoningEffort string `toml:"interpret_reasoning_effort"`
	InterpretTimeoutSeconds  int    `toml:"interpret_timeout_seconds"`
	MaxOutputBytes           int    `toml:"max_output_bytes"`
}

type PreLoopContinueConfig struct {
	Command        string `toml:"command"`
	CWD            string `toml:"cwd"`
	TimeoutSeconds int    `toml:"timeout_seconds"`
	MaxOutputBytes int    `toml:"max_output_bytes"`
}

type OptionalContinuationConfig struct {
	SkillName     string
	SkillPath     string
	ExtraGuidance string
}

func LoadRuntimeConfig(paths Paths) RuntimeConfig {
	return loadRuntimeConfigFile(paths.RuntimeConfigPath(), defaultRuntimeConfig())
}

func LoadEffectiveRuntimeConfig(paths Paths, cwd string, workspaceRoot string) RuntimeConfig {
	cfg := LoadRuntimeConfig(paths)
	if projectConfigPath, ok := findProjectRuntimeConfig(cwd, workspaceRoot); ok {
		return loadRuntimeConfigFile(projectConfigPath, cfg)
	}
	return cfg
}

func defaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		OptionalSkillName: DefaultOptionalSkillName,
		Hooks: HooksConfig{
			StopTimeoutSeconds: DefaultStopHookTimeoutSeconds,
		},
		Goal: GoalConfig{
			ConfirmModel:             DefaultGoalConfirmModel,
			ConfirmReasoningEffort:   DefaultGoalConfirmReasoningEffort,
			ConfirmCommand:           DefaultGoalConfirmCommand(),
			TimeoutSeconds:           DefaultGoalTimeoutSeconds,
			InterpretModel:           DefaultGoalInterpretModel,
			InterpretReasoningEffort: DefaultGoalInterpretReasoningEffort,
			InterpretTimeoutSeconds:  DefaultGoalInterpretTimeoutSeconds,
			MaxOutputBytes:           DefaultGoalMaxOutputBytes,
		},
		PreLoopContinue: PreLoopContinueConfig{
			CWD:            PreLoopContinueCWDSession,
			TimeoutSeconds: DefaultPreLoopContinueTimeoutSeconds,
			MaxOutputBytes: DefaultPreLoopContinueMaxOutputBytes,
		},
	}
}

func normalizeRuntimeConfig(cfg RuntimeConfig) RuntimeConfig {
	if cfg.Hooks.StopTimeoutSeconds <= 0 {
		cfg.Hooks.StopTimeoutSeconds = DefaultStopHookTimeoutSeconds
	}
	if strings.TrimSpace(cfg.Goal.ConfirmModel) == "" {
		cfg.Goal.ConfirmModel = DefaultGoalConfirmModel
	}
	if strings.TrimSpace(cfg.Goal.ConfirmReasoningEffort) == "" || !ValidReasoningEffort(cfg.Goal.ConfirmReasoningEffort) {
		cfg.Goal.ConfirmReasoningEffort = DefaultGoalConfirmReasoningEffort
	}
	if strings.TrimSpace(cfg.Goal.ConfirmCommand) == "" {
		cfg.Goal.ConfirmCommand = DefaultGoalConfirmCommand()
	}
	if cfg.Goal.TimeoutSeconds <= 0 {
		cfg.Goal.TimeoutSeconds = DefaultGoalTimeoutSeconds
	}
	if strings.TrimSpace(cfg.Goal.InterpretModel) == "" {
		cfg.Goal.InterpretModel = DefaultGoalInterpretModel
	}
	if strings.TrimSpace(cfg.Goal.InterpretReasoningEffort) == "" || !ValidReasoningEffort(cfg.Goal.InterpretReasoningEffort) {
		cfg.Goal.InterpretReasoningEffort = DefaultGoalInterpretReasoningEffort
	}
	if cfg.Goal.InterpretTimeoutSeconds <= 0 {
		cfg.Goal.InterpretTimeoutSeconds = DefaultGoalInterpretTimeoutSeconds
	}
	maxGoalBudget := cfg.Hooks.StopTimeoutSeconds - GoalHookTimeoutGraceSeconds
	if maxGoalBudget <= 1 {
		cfg.Goal.TimeoutSeconds = 1
		cfg.Goal.InterpretTimeoutSeconds = 1
	} else {
		if cfg.Goal.TimeoutSeconds >= maxGoalBudget {
			cfg.Goal.TimeoutSeconds = maxGoalBudget - 1
		}
		maxInterpretTimeout := maxGoalBudget - cfg.Goal.TimeoutSeconds
		if maxInterpretTimeout < 1 {
			maxInterpretTimeout = 1
		}
		if cfg.Goal.InterpretTimeoutSeconds > maxInterpretTimeout {
			cfg.Goal.InterpretTimeoutSeconds = maxInterpretTimeout
		}
	}
	if cfg.Goal.MaxOutputBytes <= 0 {
		cfg.Goal.MaxOutputBytes = DefaultGoalMaxOutputBytes
	}
	if strings.TrimSpace(cfg.PreLoopContinue.CWD) == "" {
		cfg.PreLoopContinue.CWD = PreLoopContinueCWDSession
	}
	if cfg.PreLoopContinue.TimeoutSeconds <= 0 {
		cfg.PreLoopContinue.TimeoutSeconds = DefaultPreLoopContinueTimeoutSeconds
	}
	if cfg.PreLoopContinue.MaxOutputBytes <= 0 {
		cfg.PreLoopContinue.MaxOutputBytes = DefaultPreLoopContinueMaxOutputBytes
	}
	return cfg
}

func ResolveOptionalContinuationConfig(cfg RuntimeConfig, workspaceRoot string) OptionalContinuationConfig {
	skillName := strings.TrimSpace(cfg.OptionalSkillName)
	skillPathText := strings.TrimSpace(cfg.OptionalSkillPath)
	extraGuidance := strings.TrimSpace(cfg.ExtraContinuationGuidance)

	resolvedSkillPath := ""
	if skillPathText != "" {
		candidate := filepath.Clean(skillPathText)
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(workspaceRoot, candidate)
		}
		absCandidate, err := filepath.Abs(candidate)
		if err == nil {
			if info, statErr := os.Stat(absCandidate); statErr == nil && info.IsDir() {
				absCandidate = filepath.Join(absCandidate, "SKILL.md")
			}
			if pathIsInside(workspaceRoot, absCandidate) {
				if info, statErr := os.Stat(absCandidate); statErr == nil && !info.IsDir() {
					resolvedSkillPath = absCandidate
				}
			}
		}
		if resolvedSkillPath == "" {
			skillName = ""
		}
	}

	return OptionalContinuationConfig{
		SkillName:     skillName,
		SkillPath:     resolvedSkillPath,
		ExtraGuidance: extraGuidance,
	}
}

func loadRuntimeConfigFile(path string, base RuntimeConfig) RuntimeConfig {
	if _, err := os.Stat(path); err != nil {
		return base
	}
	cfg := base
	metadata, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return parseRuntimeConfigFallback(path, base)
	}
	return normalizeRuntimeConfigWithMetadata(cfg, base, metadata)
}

func normalizeRuntimeConfigWithMetadata(cfg RuntimeConfig, base RuntimeConfig, metadata toml.MetaData) RuntimeConfig {
	normalized := normalizeRuntimeConfig(cfg)
	if shouldPreserveBlankGoalValue(base.Goal.ConfirmModel, cfg.Goal.ConfirmModel, metadata, "confirm_model") {
		normalized.Goal.ConfirmModel = ""
	}
	if shouldPreserveBlankGoalValue(base.Goal.ConfirmReasoningEffort, cfg.Goal.ConfirmReasoningEffort, metadata, "confirm_reasoning_effort") {
		normalized.Goal.ConfirmReasoningEffort = ""
	}
	if shouldPreserveBlankGoalValue(base.Goal.InterpretModel, cfg.Goal.InterpretModel, metadata, "interpret_model") {
		normalized.Goal.InterpretModel = ""
	}
	if shouldPreserveBlankGoalValue(base.Goal.InterpretReasoningEffort, cfg.Goal.InterpretReasoningEffort, metadata, "interpret_reasoning_effort") {
		normalized.Goal.InterpretReasoningEffort = ""
	}
	return normalized
}

func shouldPreserveBlankGoalValue(baseValue string, value string, metadata toml.MetaData, key string) bool {
	return strings.TrimSpace(value) == "" && (strings.TrimSpace(baseValue) == "" || metadata.IsDefined("goal", key))
}

func parseRuntimeConfigFallback(path string, base RuntimeConfig) RuntimeConfig {
	content, err := os.ReadFile(path)
	if err != nil {
		return base
	}
	cfg := base
	for _, rawLine := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		key, rawValue, _ := strings.Cut(line, "=")
		value, err := strconv.Unquote(strings.TrimSpace(rawValue))
		if err != nil {
			continue
		}
		switch strings.TrimSpace(key) {
		case "optional_skill_name":
			cfg.OptionalSkillName = value
		case "optional_skill_path":
			cfg.OptionalSkillPath = value
		case "extra_continuation_guidance":
			cfg.ExtraContinuationGuidance = value
		}
	}
	return normalizeRuntimeConfig(cfg)
}

func findProjectRuntimeConfig(cwd string, workspaceRoot string) (string, bool) {
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" {
		return "", false
	}
	absWorkspaceRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return "", false
	}

	current := strings.TrimSpace(cwd)
	if current == "" || !pathIsInside(absWorkspaceRoot, current) {
		current = absWorkspaceRoot
	}
	current, err = filepath.Abs(current)
	if err != nil {
		current = absWorkspaceRoot
	}
	if !pathIsInside(absWorkspaceRoot, current) {
		current = absWorkspaceRoot
	}

	for {
		candidate := filepath.Join(current, projectRuntimeConfigFilename)
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, true
		}
		if filepath.Clean(current) == filepath.Clean(absWorkspaceRoot) {
			return "", false
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

func pathIsInside(root string, candidate string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absCandidate)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." && !filepath.IsAbs(rel))
}
