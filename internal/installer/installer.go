package installer

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/compozy/codex-loop/internal/loop"
	codexloopplugin "github.com/compozy/codex-loop/plugins/codex-loop"
)

var inlineHooksRE = regexp.MustCompile(`(?m)^\s*\[\[?\s*hooks(?:[.\]])`)

const (
	bundledSkillRoot       = "skills/codex-loop"
	managedSkillMarkerName = ".codex-loop-managed"
)

type Options struct {
	SourceBinary string
}

func Install(paths loop.Paths, opts Options) ([]string, error) {
	messages := make([]string, 0)
	if err := validateBuiltInSkillInstallTarget(paths); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(paths.RuntimeBinDir(), 0o755); err != nil {
		return nil, fmt.Errorf("create runtime bin directory: %w", err)
	}
	if err := os.MkdirAll(paths.LoopsDir(), 0o755); err != nil {
		return nil, fmt.Errorf("create loop state directory: %w", err)
	}

	sourceBinary := opts.SourceBinary
	if strings.TrimSpace(sourceBinary) == "" {
		resolved, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("resolve current executable: %w", err)
		}
		sourceBinary = resolved
	}
	if err := installRuntimeBinary(sourceBinary, paths.RuntimeBinaryPath()); err != nil {
		return nil, err
	}
	messages = append(messages, fmt.Sprintf("Installed runtime binary at %s", paths.RuntimeBinaryPath()))

	if err := installBuiltInSkill(paths); err != nil {
		return nil, err
	}
	messages = append(messages, fmt.Sprintf("Installed built-in codex-loop skill at %s", paths.BuiltInSkillDir()))

	configCreated, runtimeConfigUpdated, err := ensureRuntimeConfig(paths)
	if err != nil {
		return nil, err
	}
	if configCreated {
		messages = append(messages, fmt.Sprintf("Created optional runtime config at %s", paths.RuntimeConfigPath()))
	} else if runtimeConfigUpdated {
		messages = append(messages, fmt.Sprintf("Updated optional runtime config at %s", paths.RuntimeConfigPath()))
	} else {
		messages = append(messages, fmt.Sprintf("Preserved existing runtime config at %s", paths.RuntimeConfigPath()))
	}

	if err := ensureManagedHookConfig(paths); err != nil {
		return nil, err
	}
	messages = append(messages, fmt.Sprintf("Updated managed hook config at %s", paths.HooksPath()))

	configUpdated, err := EnsureCodexHooksEnabled(paths.ConfigPath())
	if err != nil {
		return nil, err
	}
	if configUpdated {
		messages = append(messages, fmt.Sprintf("Enabled features.codex_hooks in %s", paths.ConfigPath()))
	} else {
		messages = append(messages, fmt.Sprintf("features.codex_hooks was already enabled in %s", paths.ConfigPath()))
	}

	messages = append(messages,
		fmt.Sprintf("Ensured loop state directory exists at %s", paths.LoopsDir()),
		fmt.Sprintf("Global Codex home: %s", paths.CodexHome),
		"Restart Codex after installing or updating the codex-loop plugin.",
	)
	return messages, nil
}

func Uninstall(paths loop.Paths) ([]string, error) {
	messages := make([]string, 0, 3)
	removedHooks, err := removeManagedHookConfig(paths)
	if err != nil {
		return nil, err
	}
	if removedHooks {
		messages = append(messages, fmt.Sprintf("Removed managed hook registrations from %s", paths.HooksPath()))
	} else {
		messages = append(messages, fmt.Sprintf("No managed hook registrations were present in %s", paths.HooksPath()))
	}

	runtimeRoot := paths.RuntimeRoot()
	if _, err := os.Stat(runtimeRoot); err != nil {
		if os.IsNotExist(err) {
			messages = append(messages, fmt.Sprintf("No managed runtime directory found at %s", runtimeRoot))
		} else {
			return nil, fmt.Errorf("stat runtime directory %q: %w", runtimeRoot, err)
		}
	} else {
		if err := os.RemoveAll(runtimeRoot); err != nil {
			return nil, fmt.Errorf("remove runtime directory %q: %w", runtimeRoot, err)
		}
		messages = append(messages, fmt.Sprintf("Removed managed runtime directory %s", runtimeRoot))
	}
	removedSkill, err := removeManagedBuiltInSkill(paths)
	if err != nil {
		return nil, err
	}
	if removedSkill {
		messages = append(messages, fmt.Sprintf("Removed managed built-in skill directory %s", paths.BuiltInSkillDir()))
	} else {
		messages = append(messages, fmt.Sprintf("No managed built-in skill directory found at %s", paths.BuiltInSkillDir()))
	}
	messages = append(messages, "Left ~/.codex/config.toml unchanged, including features.codex_hooks.")
	return messages, nil
}

func EnsureCodexHooksEnabled(configPath string) (bool, error) {
	contentBytes, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("read config %q: %w", configPath, err)
	}
	original := string(contentBytes)
	lines := []string{}
	if original != "" {
		lines = strings.Split(strings.ReplaceAll(original, "\r\n", "\n"), "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
	}

	sectionStart := -1
	sectionEnd := len(lines)
	headerRE := regexp.MustCompile(`^\s*\[([^\[\]]+)\]\s*$`)
	for index, line := range lines {
		match := headerRE.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		if strings.TrimSpace(match[1]) == "features" {
			sectionStart = index
			for probe := index + 1; probe < len(lines); probe++ {
				if headerRE.MatchString(lines[probe]) || inlineHooksRE.MatchString(lines[probe]) {
					sectionEnd = probe
					break
				}
			}
			break
		}
	}

	if sectionStart == -1 {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "[features]", "codex_hooks = true")
	} else {
		replaced := false
		codexHooksRE := regexp.MustCompile(`^\s*codex_hooks\s*=`)
		for index := sectionStart + 1; index < sectionEnd; index++ {
			if codexHooksRE.MatchString(lines[index]) {
				lines[index] = "codex_hooks = true"
				replaced = true
				break
			}
		}
		if !replaced {
			nextLines := append([]string{}, lines[:sectionEnd]...)
			nextLines = append(nextLines, "codex_hooks = true")
			nextLines = append(nextLines, lines[sectionEnd:]...)
			lines = nextLines
		}
	}

	updated := strings.Join(lines, "\n")
	if updated != "" {
		updated = strings.TrimRight(updated, "\n") + "\n"
	}
	if updated == original {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return false, fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(configPath, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("write config %q: %w", configPath, err)
	}
	return true, nil
}

func ensureRuntimeConfig(paths loop.Paths) (bool, bool, error) {
	if _, err := os.Stat(paths.RuntimeConfigPath()); err == nil {
		updated, updateErr := ensureBuiltInOptionalSkill(paths.RuntimeConfigPath())
		return false, updated, updateErr
	} else if !os.IsNotExist(err) {
		return false, false, fmt.Errorf("stat runtime config %q: %w", paths.RuntimeConfigPath(), err)
	}
	if err := os.MkdirAll(filepath.Dir(paths.RuntimeConfigPath()), 0o755); err != nil {
		return false, false, fmt.Errorf("create runtime config directory: %w", err)
	}
	if err := os.WriteFile(paths.RuntimeConfigPath(), []byte(loop.DefaultRuntimeConfig), 0o644); err != nil {
		return false, false, fmt.Errorf("write runtime config: %w", err)
	}
	return true, false, nil
}

func ensureBuiltInOptionalSkill(configPath string) (bool, error) {
	contentBytes, err := os.ReadFile(configPath)
	if err != nil {
		return false, fmt.Errorf("read runtime config %q: %w", configPath, err)
	}
	content := strings.ReplaceAll(string(contentBytes), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	optionalSkillRE := regexp.MustCompile(`^\s*optional_skill_name\s*=`)
	for index, line := range lines {
		if !optionalSkillRE.MatchString(line) {
			continue
		}
		_, rawValue, _ := strings.Cut(line, "=")
		value, ok := quotedConfigString(rawValue)
		if !ok || strings.TrimSpace(value) != "" {
			return false, nil
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		lines[index] = fmt.Sprintf(`%soptional_skill_name = %q`, indent, loop.DefaultOptionalSkillName)
		return writeRuntimeConfigLines(configPath, lines)
	}

	inserted := append([]string{fmt.Sprintf(`optional_skill_name = %q`, loop.DefaultOptionalSkillName)}, lines...)
	return writeRuntimeConfigLines(configPath, inserted)
}

func quotedConfigString(rawValue string) (string, bool) {
	valueText := strings.TrimSpace(rawValue)
	if valueText == "" {
		return "", false
	}
	value, err := strconv.Unquote(valueText)
	if err == nil {
		return value, true
	}
	return "", false
}

func writeRuntimeConfigLines(configPath string, lines []string) (bool, error) {
	updated := strings.Join(lines, "\n")
	updated = strings.TrimRight(updated, "\n") + "\n"
	if err := os.WriteFile(configPath, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("write runtime config %q: %w", configPath, err)
	}
	return true, nil
}

func installBuiltInSkill(paths loop.Paths) error {
	destinationRoot := paths.BuiltInSkillDir()
	if hasManagedSkillMarker(destinationRoot) {
		if err := os.RemoveAll(destinationRoot); err != nil {
			return fmt.Errorf("remove previous built-in skill directory %q: %w", destinationRoot, err)
		}
	}
	if err := fs.WalkDir(codexloopplugin.SkillFS, bundledSkillRoot, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if sourcePath == bundledSkillRoot {
			return nil
		}
		relative := strings.TrimPrefix(sourcePath, bundledSkillRoot+"/")
		destination := filepath.Join(destinationRoot, filepath.FromSlash(relative))
		if entry.IsDir() {
			if err := os.MkdirAll(destination, 0o755); err != nil {
				return fmt.Errorf("create skill directory %q: %w", destination, err)
			}
			return nil
		}
		content, err := codexloopplugin.SkillFS.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("read bundled skill file %q: %w", sourcePath, err)
		}
		if err := writeFileAtomic(destination, content, 0o644); err != nil {
			return fmt.Errorf("write built-in skill file %q: %w", destination, err)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := writeFileAtomic(filepath.Join(destinationRoot, managedSkillMarkerName), []byte("managed_by = \"codex-loop\"\n"), 0o644); err != nil {
		return fmt.Errorf("write built-in skill marker: %w", err)
	}
	return nil
}

func validateBuiltInSkillInstallTarget(paths loop.Paths) error {
	destinationRoot := paths.BuiltInSkillDir()
	if hasManagedSkillMarker(destinationRoot) {
		return nil
	}
	entries, err := os.ReadDir(destinationRoot)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read built-in skill directory %q: %w", destinationRoot, err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("unmanaged codex-loop skill directory already exists at %s; move it aside before running codex-loop install", destinationRoot)
	}
	return nil
}

func removeManagedBuiltInSkill(paths loop.Paths) (bool, error) {
	destinationRoot := paths.BuiltInSkillDir()
	if !hasManagedSkillMarker(destinationRoot) {
		return false, nil
	}
	if err := os.RemoveAll(destinationRoot); err != nil {
		return false, fmt.Errorf("remove managed built-in skill directory %q: %w", destinationRoot, err)
	}
	return true, nil
}

func hasManagedSkillMarker(skillDir string) bool {
	info, err := os.Stat(filepath.Join(skillDir, managedSkillMarkerName))
	return err == nil && !info.IsDir()
}

func installRuntimeBinary(source string, destination string) error {
	absSource, err := filepath.Abs(source)
	if err != nil {
		return fmt.Errorf("resolve source binary %q: %w", source, err)
	}
	if _, err := os.Stat(absSource); err != nil {
		return fmt.Errorf("stat source binary %q: %w", absSource, err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create runtime binary directory: %w", err)
	}
	if samePath(absSource, destination) {
		return nil
	}
	return copyFile(absSource, destination, 0o755)
}

func copyFile(source string, destination string, mode os.FileMode) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open source %q: %w", source, err)
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(destination), ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file for %q: %w", destination, err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	if _, err := io.Copy(temp, sourceFile); err != nil {
		_ = temp.Close()
		return fmt.Errorf("copy %q to %q: %w", source, destination, err)
	}
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		return fmt.Errorf("chmod temp file for %q: %w", destination, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temp file for %q: %w", destination, err)
	}
	if err := os.Rename(tempName, destination); err != nil {
		return fmt.Errorf("replace %q: %w", destination, err)
	}
	return nil
}

func writeFileAtomic(destination string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(destination), ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file for %q: %w", destination, err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write temp file for %q: %w", destination, err)
	}
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		return fmt.Errorf("chmod temp file for %q: %w", destination, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temp file for %q: %w", destination, err)
	}
	if err := os.Rename(tempName, destination); err != nil {
		return fmt.Errorf("replace %q: %w", destination, err)
	}
	return nil
}

func samePath(left string, right string) bool {
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	if leftErr == nil && rightErr == nil {
		return os.SameFile(leftInfo, rightInfo)
	}
	leftAbs, leftAbsErr := filepath.Abs(left)
	rightAbs, rightAbsErr := filepath.Abs(right)
	return leftAbsErr == nil && rightAbsErr == nil && leftAbs == rightAbs
}
