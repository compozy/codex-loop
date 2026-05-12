package pluginmeta

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundledCodexLoopSkillShape(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	skillDir := filepath.Join(root, "plugins", "codex-loop", "skills", "codex-loop")
	skillPath := filepath.Join(skillDir, "SKILL.md")
	skill := readTextFile(t, skillPath)
	frontmatter := parseSkillFrontmatter(t, skill)

	if frontmatter["name"] != filepath.Base(skillDir) {
		t.Fatalf("skill name %q must match directory %q", frontmatter["name"], filepath.Base(skillDir))
	}
	description := frontmatter["description"]
	if len(description) > 1024 {
		t.Fatalf("description too long: %d", len(description))
	}
	if !strings.Contains(description, "Use for") {
		t.Fatalf("description must include positive trigger guidance: %q", description)
	}
	if !strings.Contains(description, "Do not use") {
		t.Fatalf("description must include negative trigger guidance: %q", description)
	}
	for _, banned := range []string{" you ", " your ", " I ", " me ", " my "} {
		if strings.Contains(" "+description+" ", banned) {
			t.Fatalf("description contains first/second-person wording %q: %q", banned, description)
		}
	}

	requireWithinFirstLines(t, skill, "## Required Reading Router", 50)
	if !strings.Contains(skill, "## Reference Index") {
		t.Fatal("skill missing Reference Index")
	}
	lowerSkill := strings.ToLower(skill)
	if strings.Contains(lowerSkill, "for depth") || strings.Contains(lowerSkill, "read ") && strings.Contains(lowerSkill, " for more") {
		t.Fatal("skill contains soft reference-loading phrasing")
	}
	if count := strings.Count(skill, "**STOP. Read "); count < 3 {
		t.Fatalf("expected at least 3 hard STOP directives, got %d", count)
	}

	expectedRefs := []string{
		"runtime-setup.md",
		"continuation-config.md",
		"tracking-protocol.md",
		"task-decomposition.md",
		"state-schema.md",
		"phase-transitions.md",
		"checklist.md",
	}
	for _, name := range expectedRefs {
		path := filepath.Join(skillDir, "references", name)
		content := readTextFile(t, path)
		if !strings.Contains(skill, "references/"+name) {
			t.Fatalf("SKILL.md does not route reference %s", name)
		}
		if lineCount(content) > 100 && !strings.Contains(content, "## Contents") {
			t.Fatalf("reference over 100 lines must include Contents: %s", name)
		}
	}

	expectedScripts := []string{"init-tracking.py", "detect-next.py", "update-tracking.py", "validate-tracking.py"}
	for _, name := range expectedScripts {
		if _, err := os.Stat(filepath.Join(skillDir, "scripts", name)); err != nil {
			t.Fatalf("expected script %s: %v", name, err)
		}
		if !strings.Contains(skill, "scripts/"+name) {
			t.Fatalf("SKILL.md does not mention script %s", name)
		}
	}
	assertFlatSkillDir(t, filepath.Join(skillDir, "references"))
	assertFlatSkillDir(t, filepath.Join(skillDir, "assets"))
	assertFlatSkillDir(t, filepath.Join(skillDir, "scripts"))

	var schema map[string]any
	if err := json.Unmarshal([]byte(readTextFile(t, filepath.Join(skillDir, "assets", "state.schema.json"))), &schema); err != nil {
		t.Fatalf("state schema must be valid JSON: %v", err)
	}
}

func TestCodexLoopTrackingHelperLifecycle(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	scriptDir := filepath.Join(root, "plugins", "codex-loop", "skills", "codex-loop", "scripts")
	workspace := t.TempDir()
	tasksJSON := "[{\"title\":\"Implement helper lifecycle\",\"description\":\"Exercise task completion.\",\"acceptance\":[\"First task can be completed with memory evidence.\"]},{\"title\":\"Verify helper lifecycle\",\"acceptance\":\"Final detection reaches verification.\"}]"

	runPython(t, workspace, filepath.Join(scriptDir, "init-tracking.py"), "demo-loop",
		"--request", "Ship a tracked loop helper lifecycle.",
		"--goal", "state complete with verification PASS",
		"--tasks-json", tasksJSON,
		"--verification-command", "make verify",
	)
	assertCommandOutput(t, runPython(t, workspace, filepath.Join(scriptDir, "detect-next.py"), "demo-loop"), "action=execute_task name=demo-loop task=task-001")

	stateBefore := readTextFile(t, filepath.Join(workspace, ".codex", "loop", "demo-loop", "state.json"))
	assertPythonFails(t, workspace, filepath.Join(scriptDir, "update-tracking.py"), "demo-loop", "--complete-task", "task-001")
	stateAfterFailure := readTextFile(t, filepath.Join(workspace, ".codex", "loop", "demo-loop", "state.json"))
	if stateAfterFailure != stateBefore {
		t.Fatal("invalid transition corrupted state")
	}

	writeLoopMemory(t, workspace, "demo-loop", "iter-001.md", "task one evidence")
	runPython(t, workspace, filepath.Join(scriptDir, "update-tracking.py"), "demo-loop",
		"--complete-task", "task-001",
		"--memory-written", "memory/iter-001.md",
	)
	stateAfterSuccess := readTextFile(t, filepath.Join(workspace, ".codex", "loop", "demo-loop", "state.json"))
	if stateAfterSuccess == stateBefore {
		t.Fatal("expected valid task completion to update state")
	}
	assertCommandOutput(t, runPython(t, workspace, filepath.Join(scriptDir, "detect-next.py"), "demo-loop"), "action=execute_task name=demo-loop task=task-002")

	writeLoopMemory(t, workspace, "demo-loop", "iter-002.md", "task two evidence")
	runPython(t, workspace, filepath.Join(scriptDir, "update-tracking.py"), "demo-loop",
		"--complete-task", "task-002",
		"--memory-written", "memory/iter-002.md",
	)
	assertCommandOutput(t, runPython(t, workspace, filepath.Join(scriptDir, "detect-next.py"), "demo-loop"), "action=verify name=demo-loop")

	runPython(t, workspace, filepath.Join(scriptDir, "update-tracking.py"), "demo-loop",
		"--verify-pass", "make verify passed",
	)
	assertCommandOutput(t, runPython(t, workspace, filepath.Join(scriptDir, "detect-next.py"), "demo-loop"), "action=done name=demo-loop")
	runPython(t, workspace, filepath.Join(scriptDir, "validate-tracking.py"), "demo-loop", "--expect-done")

	var state map[string]any
	decodeJSONFile(t, filepath.Join(workspace, ".codex", "loop", "demo-loop", "state.json"), &state)
	if state["status"] != "complete" {
		t.Fatalf("expected complete state, got %#v", state["status"])
	}
	verification, ok := state["verification"].(map[string]any)
	if !ok {
		t.Fatalf("verification must be an object: %#v", state["verification"])
	}
	if verification["status"] != "PASS" {
		t.Fatalf("expected verification PASS, got %#v", verification["status"])
	}
	if state["iteration"] != float64(3) {
		t.Fatalf("expected 3 iterations, got %#v", state["iteration"])
	}
	history, ok := state["history"].([]any)
	if !ok || len(history) != 3 {
		t.Fatalf("expected 3 history entries, got %#v", state["history"])
	}
}

func TestCodexLoopTrackingBlockedTaskRequiresResolution(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	scriptDir := filepath.Join(root, "plugins", "codex-loop", "skills", "codex-loop", "scripts")
	workspace := t.TempDir()
	tasksJSON := "[{\"title\":\"Investigate blocker\",\"acceptance\":[\"Blocker can be cleared with memory evidence.\"]}]"

	runPython(t, workspace, filepath.Join(scriptDir, "init-tracking.py"), "blocked-loop",
		"--request", "Exercise blocked task transitions.",
		"--tasks-json", tasksJSON,
	)
	runPython(t, workspace, filepath.Join(scriptDir, "update-tracking.py"), "blocked-loop",
		"--block-task", "task-001",
		"--blocker", "external dependency unavailable",
	)
	assertCommandOutput(t, runPython(t, workspace, filepath.Join(scriptDir, "detect-next.py"), "blocked-loop"), "action=resolve_blocker name=blocked-loop blocker_count=1")
	runPython(t, workspace, filepath.Join(scriptDir, "validate-tracking.py"), "blocked-loop")

	writeLoopMemory(t, workspace, "blocked-loop", "iter-001.md", "blocker resolved")
	runPython(t, workspace, filepath.Join(scriptDir, "update-tracking.py"), "blocked-loop",
		"--clear-blockers",
		"--memory-written", "memory/iter-001.md",
	)
	assertCommandOutput(t, runPython(t, workspace, filepath.Join(scriptDir, "detect-next.py"), "blocked-loop"), "action=execute_task name=blocked-loop task=task-001")
}

func parseSkillFrontmatter(t *testing.T, content string) map[string]string {
	t.Helper()
	if !strings.HasPrefix(content, "---\n") {
		t.Fatal("skill missing frontmatter")
	}
	parts := strings.SplitN(content, "\n---\n", 2)
	if len(parts) != 2 {
		t.Fatal("skill frontmatter is not closed")
	}
	fields := make(map[string]string)
	for _, line := range strings.Split(strings.TrimPrefix(parts[0], "---\n"), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), "\"'")
	}
	return fields
}

func requireWithinFirstLines(t *testing.T, content string, needle string, maxLine int) {
	t.Helper()
	for index, line := range strings.Split(content, "\n") {
		if strings.Contains(line, needle) {
			if index+1 > maxLine {
				t.Fatalf("%q appears at line %d, expected within first %d", needle, index+1, maxLine)
			}
			return
		}
	}
	t.Fatalf("missing %q", needle)
}

func assertFlatSkillDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			t.Fatalf("skill folder must be flat; unexpected nested directory %s", filepath.Join(dir, entry.Name()))
		}
	}
}

func lineCount(content string) int {
	if content == "" {
		return 0
	}
	return strings.Count(content, "\n") + 1
}

func runPython(t *testing.T, dir string, script string, args ...string) string {
	t.Helper()
	cmdArgs := append([]string{script}, args...)
	cmd := exec.Command("python3", cmdArgs...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("python3 %s failed: %v\nstdout: %s\nstderr: %s", strings.Join(cmdArgs, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

func assertPythonFails(t *testing.T, dir string, script string, args ...string) {
	t.Helper()
	cmdArgs := append([]string{script}, args...)
	cmd := exec.Command("python3", cmdArgs...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatalf("expected python3 %s to fail", strings.Join(cmdArgs, " "))
	}
	if stderr.Len() == 0 {
		t.Fatalf("expected failure to explain itself for python3 %s", strings.Join(cmdArgs, " "))
	}
}

func assertCommandOutput(t *testing.T, got string, want string) {
	t.Helper()
	if strings.TrimSpace(got) != want {
		t.Fatalf("unexpected output\nwant: %s\n got: %s", want, strings.TrimSpace(got))
	}
}

func writeLoopMemory(t *testing.T, workspace string, loopName string, fileName string, body string) {
	t.Helper()
	path := filepath.Join(workspace, ".codex", "loop", loopName, "memory", fileName)
	if err := os.WriteFile(path, []byte(fmt.Sprintf("# Memory\n\n%s\n", body)), 0o644); err != nil {
		t.Fatalf("write memory %s: %v", path, err)
	}
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func decodeJSONFile(t *testing.T, path string, target any) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(content, target); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}
