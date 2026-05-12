package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/compozy/codex-loop/internal/loop"
)

func TestInstallCreatesManagedFiles(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	sourceBinary := filepath.Join(t.TempDir(), "codex-loop-source")
	if err := os.WriteFile(sourceBinary, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write source binary: %v", err)
	}
	if err := os.MkdirAll(paths.CodexHome, 0o755); err != nil {
		t.Fatalf("create codex home: %v", err)
	}
	if err := os.WriteFile(paths.ConfigPath(), []byte("[features]\nother_flag = true\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	writeJSONDoc(t, paths.HooksPath(), map[string]any{
		"hooks": map[string]any{
			"Stop": []any{
				map[string]any{
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": "python3 ./custom_stop.py",
						},
					},
				},
			},
		},
	})

	messages, err := Install(paths, Options{SourceBinary: sourceBinary})
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	if _, err := os.Stat(paths.RuntimeBinaryPath()); err != nil {
		t.Fatalf("expected runtime binary: %v", err)
	}
	if _, err := os.Stat(paths.LoopsDir()); err != nil {
		t.Fatalf("expected loops dir: %v", err)
	}
	if _, err := os.Stat(paths.RuntimeConfigPath()); err != nil {
		t.Fatalf("expected runtime config: %v", err)
	}
	if _, err := os.Stat(filepath.Join(paths.BuiltInSkillDir(), managedSkillMarkerName)); err != nil {
		t.Fatalf("expected managed built-in skill marker: %v", err)
	}
	assertContains(t, readFile(t, filepath.Join(paths.BuiltInSkillDir(), "SKILL.md")), "name: codex-loop")
	assertContains(t, readFile(t, filepath.Join(paths.BuiltInSkillDir(), "references", "tracking-protocol.md")), "# Tracking Protocol")
	assertContains(t, readFile(t, filepath.Join(paths.BuiltInSkillDir(), "scripts", "detect-next.py")), "Print the next codex-loop tracking action")
	runtimeConfigText := readFile(t, paths.RuntimeConfigPath())
	assertContains(t, runtimeConfigText, "optional_skill_name = \"codex-loop\"")
	configText := readFile(t, paths.ConfigPath())
	assertContains(t, configText, "codex_hooks = true")
	assertContains(t, configText, "other_flag = true")
	hooksDoc := readJSONFile(t, paths.HooksPath())
	assertHookCommandPresent(t, hooksDoc, "Stop", "python3 ./custom_stop.py")
	assertHookCommandPresent(t, hooksDoc, "Stop", managedHookCommand("stop"))
	assertHookCommandPresent(t, hooksDoc, "UserPromptSubmit", managedHookCommand("user-prompt-submit"))

	joined := strings.Join(messages, "\n")
	assertContains(t, joined, "Installed runtime binary")
	assertContains(t, joined, "Installed built-in codex-loop skill")
	assertContains(t, joined, "Updated managed hook config")
	assertContains(t, joined, "Restart Codex")
}

func TestInstallMigratesBlankRuntimeConfigToBuiltInSkill(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	sourceBinary := writeSourceBinary(t)
	if err := os.MkdirAll(filepath.Dir(paths.RuntimeConfigPath()), 0o755); err != nil {
		t.Fatalf("create runtime dir: %v", err)
	}
	customConfig := "optional_skill_name = \"\"\nextra_continuation_guidance = \"keep this\"\n"
	if err := os.WriteFile(paths.RuntimeConfigPath(), []byte(customConfig), 0o644); err != nil {
		t.Fatalf("write runtime config: %v", err)
	}

	if _, err := Install(paths, Options{SourceBinary: sourceBinary}); err != nil {
		t.Fatalf("install: %v", err)
	}
	got := readFile(t, paths.RuntimeConfigPath())
	assertContains(t, got, "optional_skill_name = \"codex-loop\"")
	assertContains(t, got, "extra_continuation_guidance = \"keep this\"")
}

func TestInstallPreservesCustomOptionalSkill(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	sourceBinary := writeSourceBinary(t)
	if err := os.MkdirAll(filepath.Dir(paths.RuntimeConfigPath()), 0o755); err != nil {
		t.Fatalf("create runtime dir: %v", err)
	}
	customConfig := "optional_skill_name = \"focused-qa\"\noptional_skill_path = \".agents/skills/focused-qa\"\nextra_continuation_guidance = \"keep this\"\n"
	if err := os.WriteFile(paths.RuntimeConfigPath(), []byte(customConfig), 0o644); err != nil {
		t.Fatalf("write runtime config: %v", err)
	}

	if _, err := Install(paths, Options{SourceBinary: sourceBinary}); err != nil {
		t.Fatalf("install: %v", err)
	}
	if got := readFile(t, paths.RuntimeConfigPath()); got != customConfig {
		t.Fatalf("expected custom runtime config preserved, got %q", got)
	}
}

func TestInstallAddsBuiltInSkillWhenRuntimeConfigLacksOptionalSkill(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	sourceBinary := writeSourceBinary(t)
	if err := os.MkdirAll(filepath.Dir(paths.RuntimeConfigPath()), 0o755); err != nil {
		t.Fatalf("create runtime dir: %v", err)
	}
	customConfig := "[pre_loop_continue]\ntimeout_seconds = 12\n"
	if err := os.WriteFile(paths.RuntimeConfigPath(), []byte(customConfig), 0o644); err != nil {
		t.Fatalf("write runtime config: %v", err)
	}

	if _, err := Install(paths, Options{SourceBinary: sourceBinary}); err != nil {
		t.Fatalf("install: %v", err)
	}
	got := readFile(t, paths.RuntimeConfigPath())
	assertContains(t, got, "optional_skill_name = \"codex-loop\"")
	assertContains(t, got, "[pre_loop_continue]")
	assertContains(t, got, "timeout_seconds = 12")
}

func TestInstallRefusesToOverwriteUnmanagedBuiltInSkillDirectory(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	sourceBinary := writeSourceBinary(t)
	if err := os.MkdirAll(paths.BuiltInSkillDir(), 0o755); err != nil {
		t.Fatalf("create built-in skill dir: %v", err)
	}
	customSkill := "custom skill"
	customSkillPath := filepath.Join(paths.BuiltInSkillDir(), "SKILL.md")
	if err := os.WriteFile(customSkillPath, []byte(customSkill), 0o644); err != nil {
		t.Fatalf("write custom skill: %v", err)
	}

	_, err := Install(paths, Options{SourceBinary: sourceBinary})
	if err == nil {
		t.Fatal("expected unmanaged skill collision error")
	}
	if !strings.Contains(err.Error(), "unmanaged codex-loop skill directory") {
		t.Fatalf("expected unmanaged skill error, got %v", err)
	}
	if got := readFile(t, customSkillPath); got != customSkill {
		t.Fatalf("expected unmanaged skill preserved, got %q", got)
	}
	if _, statErr := os.Stat(paths.RuntimeBinaryPath()); !os.IsNotExist(statErr) {
		t.Fatalf("expected install to stop before runtime binary write, stat err: %v", statErr)
	}
}

func TestInstallIsIdempotentForManagedHooks(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	sourceBinary := writeSourceBinary(t)

	if _, err := Install(paths, Options{SourceBinary: sourceBinary}); err != nil {
		t.Fatalf("first install: %v", err)
	}
	if _, err := Install(paths, Options{SourceBinary: sourceBinary}); err != nil {
		t.Fatalf("second install: %v", err)
	}

	hooksDoc := readJSONFile(t, paths.HooksPath())
	assertHookCommandCount(t, hooksDoc, "Stop", managedHookCommand("stop"), 1)
	assertHookCommandCount(t, hooksDoc, "UserPromptSubmit", managedHookCommand("user-prompt-submit"), 1)
}

func TestInstallUsesConfiguredStopHookTimeout(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	sourceBinary := writeSourceBinary(t)
	if err := os.MkdirAll(filepath.Dir(paths.RuntimeConfigPath()), 0o755); err != nil {
		t.Fatalf("create runtime config dir: %v", err)
	}
	if err := os.WriteFile(paths.RuntimeConfigPath(), []byte("[hooks]\nstop_timeout_seconds = 1234\n"), 0o644); err != nil {
		t.Fatalf("write runtime config: %v", err)
	}

	if _, err := Install(paths, Options{SourceBinary: sourceBinary}); err != nil {
		t.Fatalf("install: %v", err)
	}

	hooksDoc := readJSONFile(t, paths.HooksPath())
	assertHookTimeout(t, hooksDoc, "Stop", managedHookCommand("stop"), 1234)
	assertHookTimeout(t, hooksDoc, "UserPromptSubmit", managedHookCommand("user-prompt-submit"), 30)
}

func TestEnsureCodexHooksEnabledAddsFeatureSection(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	updated, err := EnsureCodexHooksEnabled(path)
	if err != nil {
		t.Fatalf("ensure hooks enabled: %v", err)
	}
	if !updated {
		t.Fatal("expected update")
	}
	assertContains(t, readFile(t, path), "[features]\ncodex_hooks = true")
}

func TestEnsureCodexHooksEnabledUpdatesExistingValue(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[features]\ncodex_hooks = false\nother_flag = true\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	updated, err := EnsureCodexHooksEnabled(path)
	if err != nil {
		t.Fatalf("ensure hooks enabled: %v", err)
	}
	if !updated {
		t.Fatal("expected update")
	}
	configText := readFile(t, path)
	assertContains(t, configText, "codex_hooks = true")
	assertContains(t, configText, "other_flag = true")
}

func TestUninstallRemovesOnlyManagedHooksAndRuntime(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	if err := os.MkdirAll(paths.RuntimeRoot(), 0o755); err != nil {
		t.Fatalf("create runtime root: %v", err)
	}
	if err := os.MkdirAll(paths.BuiltInSkillDir(), 0o755); err != nil {
		t.Fatalf("create built-in skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(paths.BuiltInSkillDir(), managedSkillMarkerName), []byte("managed_by = \"codex-loop\"\n"), 0o644); err != nil {
		t.Fatalf("write built-in skill marker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(paths.BuiltInSkillDir(), "SKILL.md"), []byte("managed skill"), 0o644); err != nil {
		t.Fatalf("write built-in skill: %v", err)
	}
	if err := os.MkdirAll(paths.CodexHome, 0o755); err != nil {
		t.Fatalf("create codex home: %v", err)
	}
	configText := "[features]\ncodex_hooks = true\n"
	if err := os.WriteFile(paths.ConfigPath(), []byte(configText), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	writeJSONDoc(t, paths.HooksPath(), map[string]any{
		"hooks": map[string]any{
			"Stop": []any{
				map[string]any{
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": "python3 ./custom_stop.py",
						},
						map[string]any{
							"type":    "command",
							"command": managedHookCommand("stop"),
						},
					},
				},
			},
			"UserPromptSubmit": []any{
				map[string]any{
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": managedHookCommand("user-prompt-submit"),
						},
					},
				},
			},
		},
	})

	messages, err := Uninstall(paths)
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if _, err := os.Stat(paths.RuntimeRoot()); !os.IsNotExist(err) {
		t.Fatalf("expected runtime removed, stat err: %v", err)
	}
	if _, err := os.Stat(paths.BuiltInSkillDir()); !os.IsNotExist(err) {
		t.Fatalf("expected managed built-in skill removed, stat err: %v", err)
	}
	if got := readFile(t, paths.ConfigPath()); got != configText {
		t.Fatalf("expected config preserved, got %q", got)
	}
	hooksDoc := readJSONFile(t, paths.HooksPath())
	assertHookCommandPresent(t, hooksDoc, "Stop", "python3 ./custom_stop.py")
	assertHookCommandCount(t, hooksDoc, "Stop", managedHookCommand("stop"), 0)
	assertHookCommandCount(t, hooksDoc, "UserPromptSubmit", managedHookCommand("user-prompt-submit"), 0)

	joined := strings.Join(messages, "\n")
	assertContains(t, joined, "Removed managed hook registrations")
	assertContains(t, joined, "Removed managed runtime directory")
	assertContains(t, joined, "Removed managed built-in skill directory")
}

func TestUninstallPreservesUnmanagedSkillDirectory(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	if err := os.MkdirAll(paths.BuiltInSkillDir(), 0o755); err != nil {
		t.Fatalf("create built-in skill dir: %v", err)
	}
	customSkill := "custom skill"
	if err := os.WriteFile(filepath.Join(paths.BuiltInSkillDir(), "SKILL.md"), []byte(customSkill), 0o644); err != nil {
		t.Fatalf("write custom skill: %v", err)
	}

	if _, err := Uninstall(paths); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if got := readFile(t, filepath.Join(paths.BuiltInSkillDir(), "SKILL.md")); got != customSkill {
		t.Fatalf("expected unmanaged skill preserved, got %q", got)
	}
}

func TestStatusRecordsJSONRoundTrip(t *testing.T) {
	t.Parallel()

	record := loop.StatusRecord{
		LoopRecord: loop.LoopRecord{
			Version:   loop.RecordVersion,
			SessionID: "sess-1",
			Name:      "qa",
			Status:    loop.StatusActive,
		},
		Path: "/tmp/loop.json",
	}

	content, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal status record: %v", err)
	}
	assertContains(t, string(content), `"path":"/tmp/loop.json"`)
}

func TestManagedHooksTemplateMatchesBundledPluginHooks(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	expected := readJSONFile(t, filepath.Join(root, "plugins", "codex-loop", "hooks", "hooks.json"))
	actual := normalizeJSONDoc(t, managedHooksTemplate(loop.DefaultStopHookTimeoutSeconds))
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("managed hooks template diverged from bundled plugin hooks\nactual: %#v\nexpected: %#v", actual, expected)
	}
}

func mustPaths(t *testing.T) loop.Paths {
	t.Helper()
	paths, err := loop.NewPaths(filepath.Join(t.TempDir(), ".codex-home"))
	if err != nil {
		t.Fatalf("new paths: %v", err)
	}
	return paths
}

func writeSourceBinary(t *testing.T) string {
	t.Helper()
	sourceBinary := filepath.Join(t.TempDir(), "codex-loop-source")
	if err := os.WriteFile(sourceBinary, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write source binary: %v", err)
	}
	return sourceBinary
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func assertContains(t *testing.T, haystack string, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected %q to contain %q", haystack, needle)
	}
}

func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return payload
}

func writeJSONDoc(t *testing.T, path string, payload map[string]any) {
	t.Helper()
	content, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, append(content, '\n'), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertHookCommandPresent(t *testing.T, hooksDoc map[string]any, eventName string, command string) {
	t.Helper()
	if countHookCommand(t, hooksDoc, eventName, command) == 0 {
		t.Fatalf("expected command %q under hooks.%s", command, eventName)
	}
}

func assertHookCommandCount(t *testing.T, hooksDoc map[string]any, eventName string, command string, expected int) {
	t.Helper()
	if count := countHookCommand(t, hooksDoc, eventName, command); count != expected {
		t.Fatalf("expected command %q under hooks.%s %d time(s), got %d", command, eventName, expected, count)
	}
}

func assertHookTimeout(t *testing.T, hooksDoc map[string]any, eventName string, command string, expected int) {
	t.Helper()
	hook := findHookCommand(t, hooksDoc, eventName, command)
	timeout, ok := hook["timeout"].(float64)
	if !ok {
		t.Fatalf("expected timeout for command %q under hooks.%s, got %#v", command, eventName, hook["timeout"])
	}
	if int(timeout) != expected {
		t.Fatalf("expected timeout %d for command %q under hooks.%s, got %v", expected, command, eventName, timeout)
	}
}

func countHookCommand(t *testing.T, hooksDoc map[string]any, eventName string, command string) int {
	t.Helper()
	hooksRoot, ok := hooksDoc["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("hooks document missing hooks object: %#v", hooksDoc["hooks"])
	}
	matcherGroups, ok := hooksRoot[eventName].([]any)
	if !ok {
		return 0
	}
	count := 0
	for _, matcherGroupAny := range matcherGroups {
		matcherGroup, ok := matcherGroupAny.(map[string]any)
		if !ok {
			t.Fatalf("hooks.%s entry is not an object: %#v", eventName, matcherGroupAny)
		}
		hooks, ok := matcherGroup["hooks"].([]any)
		if !ok {
			t.Fatalf("hooks.%s.hooks is not a list: %#v", eventName, matcherGroup["hooks"])
		}
		for _, hookAny := range hooks {
			hook, ok := hookAny.(map[string]any)
			if !ok {
				t.Fatalf("hooks.%s hook entry is not an object: %#v", eventName, hookAny)
			}
			if hook["command"] == command {
				count++
			}
		}
	}
	return count
}

func findHookCommand(t *testing.T, hooksDoc map[string]any, eventName string, command string) map[string]any {
	t.Helper()
	hooksRoot, ok := hooksDoc["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("hooks document missing hooks object: %#v", hooksDoc["hooks"])
	}
	matcherGroups, ok := hooksRoot[eventName].([]any)
	if !ok {
		t.Fatalf("hooks.%s missing", eventName)
	}
	for _, matcherGroupAny := range matcherGroups {
		matcherGroup, ok := matcherGroupAny.(map[string]any)
		if !ok {
			t.Fatalf("unexpected matcher group %#v", matcherGroupAny)
		}
		hooks, ok := matcherGroup["hooks"].([]any)
		if !ok {
			continue
		}
		for _, hookAny := range hooks {
			hook, ok := hookAny.(map[string]any)
			if !ok {
				t.Fatalf("unexpected hook %#v", hookAny)
			}
			if hook["command"] == command {
				return hook
			}
		}
	}
	t.Fatalf("expected command %q under hooks.%s", command, eventName)
	return nil
}

func normalizeJSONDoc(t *testing.T, payload any) map[string]any {
	t.Helper()
	content, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal JSON doc: %v", err)
	}
	var normalized map[string]any
	if err := json.Unmarshal(content, &normalized); err != nil {
		t.Fatalf("unmarshal normalized JSON doc: %v", err)
	}
	return normalized
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatal("could not find repo root")
		}
		wd = parent
	}
}
