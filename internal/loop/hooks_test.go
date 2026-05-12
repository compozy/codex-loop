package loop

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUserPromptSubmitCreatesTimeLoopFile(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	now := fixedTime()

	result, err := HandleUserPromptSubmit(paths, UserPromptPayload{
		SessionID: "sess-1",
		CWD:       repoRoot,
		Prompt:    `[[CODEX_LOOP name="release-stress-qa" min="6h"]]` + "\nRun the QA task.",
	}, now)
	if err != nil {
		t.Fatalf("handle user prompt submit: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil hook result, got %#v", result)
	}

	loopFiles, err := IterLoopRecords(paths)
	if err != nil {
		t.Fatalf("iterate loops: %v", err)
	}
	if len(loopFiles) != 1 {
		t.Fatalf("expected 1 loop file, got %d", len(loopFiles))
	}
	record := loopFiles[0].Record
	if record.Status != StatusActive {
		t.Fatalf("expected active status, got %q", record.Status)
	}
	if record.Name != "release-stress-qa" {
		t.Fatalf("unexpected name %q", record.Name)
	}
	if record.TaskPrompt != "Run the QA task." {
		t.Fatalf("unexpected task prompt %q", record.TaskPrompt)
	}
	if record.MinDurationSeconds == nil || *record.MinDurationSeconds != 21600 {
		t.Fatalf("unexpected min duration %#v", record.MinDurationSeconds)
	}
	if record.LimitMode != LimitModeTime {
		t.Fatalf("unexpected limit mode %q", record.LimitMode)
	}
	if record.WorkspaceRoot != repoRoot {
		t.Fatalf("expected workspace root %q, got %q", repoRoot, record.WorkspaceRoot)
	}
}

func TestUserPromptSubmitSupersedesPreviousLoopForSession(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	start := fixedTime()

	_, err := HandleUserPromptSubmit(paths, UserPromptPayload{
		SessionID: "sess-1",
		CWD:       repoRoot,
		Prompt:    `[[CODEX_LOOP name="first-run" min="1h"]]` + "\nFirst task.",
	}, start)
	if err != nil {
		t.Fatalf("first activation: %v", err)
	}
	_, err = HandleUserPromptSubmit(paths, UserPromptPayload{
		SessionID: "sess-1",
		CWD:       repoRoot,
		Prompt:    `[[CODEX_LOOP name="second-run" rounds="2"]]` + "\nSecond task.",
	}, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("second activation: %v", err)
	}

	loopFiles, err := IterLoopRecords(paths)
	if err != nil {
		t.Fatalf("iterate loops: %v", err)
	}
	if len(loopFiles) != 2 {
		t.Fatalf("expected 2 loop files, got %d", len(loopFiles))
	}
	statuses := map[string]string{}
	for _, loopFile := range loopFiles {
		statuses[loopFile.Record.Name] = loopFile.Record.Status
	}
	if statuses["first-run"] != StatusSuperseded {
		t.Fatalf("expected first-run superseded, got %q", statuses["first-run"])
	}
	if statuses["second-run"] != StatusActive {
		t.Fatalf("expected second-run active, got %q", statuses["second-run"])
	}
}

func TestPromptWithoutHeaderIsIgnored(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}

	result, err := HandleUserPromptSubmit(paths, UserPromptPayload{
		SessionID: "sess-1",
		CWD:       repoRoot,
		Prompt:    "Do the normal task without a loop header.",
	}, fixedTime())
	if err != nil {
		t.Fatalf("handle prompt: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	loopFiles, err := IterLoopRecords(paths)
	if err != nil {
		t.Fatalf("iterate loops: %v", err)
	}
	if len(loopFiles) != 0 {
		t.Fatalf("expected no loops, got %d", len(loopFiles))
	}
}

func TestUserPromptSubmitCreatesGoalLoopFile(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}

	result, err := HandleUserPromptSubmit(paths, UserPromptPayload{
		SessionID: "sess-1",
		CWD:       repoRoot,
		Prompt:    `[[CODEX_LOOP name="goal-qa" goal="verify release" confirm_model="gpt-5.5" confirm_reasoning_effort="high"]]` + "\nRun the QA task.",
	}, fixedTime())
	if err != nil {
		t.Fatalf("handle user prompt submit: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil hook result, got %#v", result)
	}

	loopFiles, err := IterLoopRecords(paths)
	if err != nil {
		t.Fatalf("iterate loops: %v", err)
	}
	if len(loopFiles) != 1 {
		t.Fatalf("expected 1 loop file, got %d", len(loopFiles))
	}
	record := loopFiles[0].Record
	if record.LimitMode != LimitModeGoal {
		t.Fatalf("unexpected limit mode %q", record.LimitMode)
	}
	if record.GoalText == nil || *record.GoalText != "verify release" {
		t.Fatalf("unexpected goal text %#v", record.GoalText)
	}
	if record.ConfirmModel == nil || *record.ConfirmModel != "gpt-5.5" {
		t.Fatalf("unexpected confirm model %#v", record.ConfirmModel)
	}
	if record.ConfirmReasoning == nil || *record.ConfirmReasoning != "high" {
		t.Fatalf("unexpected confirm reasoning %#v", record.ConfirmReasoning)
	}
}

func TestStopGoalModeCompletesWhenConfirmed(t *testing.T) {
	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	argsPath := filepath.Join(t.TempDir(), "args.txt")
	stdinPath := filepath.Join(t.TempDir(), "stdin.txt")
	fakeCodex := writeFakeCodex(t)
	interpreterArgsPath := filepath.Join(t.TempDir(), "interpreter-args.txt")
	interpreterStdinPath := filepath.Join(t.TempDir(), "interpreter-stdin.txt")
	installFakeInterpreterCodex(t)
	t.Setenv("FAKE_CODEX_ARGS", argsPath)
	t.Setenv("FAKE_CODEX_STDIN", stdinPath)
	t.Setenv("FAKE_CODEX_REVIEW", "The task is complete. Evidence: all work is verified.")
	t.Setenv("FAKE_INTERPRET_ARGS", interpreterArgsPath)
	t.Setenv("FAKE_INTERPRET_STDIN", interpreterStdinPath)
	t.Setenv("FAKE_INTERPRET_VERDICT", `{"completed":true,"confidence":0.98,"reason":"all work is verified","missing_work":[],"next_round_guidance":""}`)
	writeRuntimeConfig(t, paths, `[goal]
`+fakeGoalConfirmCommandConfig(fakeCodex)+`
timeout_seconds = 5
interpret_timeout_seconds = 5
`)

	start := fixedTime()
	path := writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="goal-qa" goal="verify release"]]`+"\nRun the QA task.", start)
	latest := "Done."
	result, err := HandleStop(context.Background(), paths, StopPayload{
		SessionID:            "sess-1",
		CWD:                  repoRoot,
		LastAssistantMessage: &latest,
	}, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}
	if result != nil {
		t.Fatalf("expected completed goal to return nil, got %#v", result)
	}
	updated := readLoop(t, path)
	if updated.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %q", updated.Status)
	}
	if updated.GoalCheckCount != 1 {
		t.Fatalf("expected goal check count 1, got %d", updated.GoalCheckCount)
	}
	assertContains(t, readText(t, argsPath), "--model\ngpt-5.5")
	assertContains(t, readText(t, argsPath), `model_reasoning_effort="high"`)
	assertContains(t, readText(t, argsPath), "--yolo")
	assertNotContains(t, readText(t, argsPath), "--output-schema")
	assertNotContains(t, readText(t, stdinPath), ActivationPrefix)
	assertContains(t, readText(t, interpreterArgsPath), "--model\ngpt-5.4-mini")
	assertContains(t, readText(t, interpreterArgsPath), "--output-schema")
	assertContains(t, readText(t, interpreterStdinPath), "The task is complete. Evidence: all work is verified.")
	assertContains(t, readText(t, paths.RunsLogPath()), `"outcome":"completed"`)
	assertContains(t, readText(t, paths.RunsLogPath()), `"interpret_model":"gpt-5.4-mini"`)
}

func TestStopGoalModeContinuesWhenIncompleteAndRunsPreLoopContinue(t *testing.T) {
	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	fakeCodex := writeFakeCodex(t)
	installFakeInterpreterCodex(t)
	t.Setenv("FAKE_CODEX_ARGS", filepath.Join(t.TempDir(), "args.txt"))
	t.Setenv("FAKE_CODEX_STDIN", filepath.Join(t.TempDir(), "stdin.txt"))
	t.Setenv("FAKE_CODEX_REVIEW", "The task is not complete. Missing work: run integration tests.")
	t.Setenv("FAKE_INTERPRET_VERDICT", `{"completed":false,"confidence":0.35,"reason":"tests are missing","missing_work":["run integration tests"],"next_round_guidance":"add real verification"}`)
	writeRuntimeConfig(t, paths, `[goal]
confirm_command = "/bin/sh -c 'exit 7'"
timeout_seconds = 5
interpret_timeout_seconds = 5
`)
	writeProjectRuntimeConfig(t, repoRoot, `[goal]
`+fakeGoalConfirmCommandConfig(fakeCodex)+`
timeout_seconds = 5
interpret_timeout_seconds = 5

[pre_loop_continue]
command = "/bin/sh -c 'printf pre-loop-context'"
`)

	start := fixedTime()
	path := writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="goal-qa" goal="verify release"]]`+"\nRun the QA task.", start)
	result, err := HandleStop(context.Background(), paths, StopPayload{
		SessionID: "sess-1",
		CWD:       repoRoot,
	}, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}
	if result == nil || result["decision"] != "block" {
		t.Fatalf("expected block result, got %#v", result)
	}
	reason := result["reason"].(string)
	assertContains(t, reason, "Goal confirmation verdict:")
	assertContains(t, reason, "run integration tests")
	assertContains(t, reason, "Goal confirmation review text:")
	assertContains(t, reason, "The task is not complete. Missing work: run integration tests.")
	assertContains(t, reason, "pre_loop_continue output:")
	assertContains(t, reason, "pre-loop-context")
	updated := readLoop(t, path)
	if updated.Status != StatusActive {
		t.Fatalf("expected active status, got %q", updated.Status)
	}
	if updated.ContinueCount != 1 {
		t.Fatalf("expected continue count 1, got %d", updated.ContinueCount)
	}
	assertContains(t, readText(t, paths.RunsLogPath()), `"continuation_emitted":true`)
	assertContains(t, readText(t, paths.RunsLogPath()), `"pre_loop_continue_active":true`)
}

func TestStopGoalModeContinuesOnConfirmationFailure(t *testing.T) {
	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	fakeCodex := writeFakeCodex(t)
	t.Setenv("FAKE_CODEX_ARGS", filepath.Join(t.TempDir(), "args.txt"))
	t.Setenv("FAKE_CODEX_STDIN", filepath.Join(t.TempDir(), "stdin.txt"))
	t.Setenv("FAKE_CODEX_EXIT", "7")
	t.Setenv("FAKE_CODEX_REVIEW", "not used")
	writeRuntimeConfig(t, paths, `[goal]
`+fakeGoalConfirmCommandConfig(fakeCodex)+`
timeout_seconds = 5
`)

	start := fixedTime()
	writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="goal-qa" goal="verify release"]]`+"\nRun the QA task.", start)
	result, err := HandleStop(context.Background(), paths, StopPayload{SessionID: "sess-1", CWD: repoRoot}, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}
	if result == nil || result["decision"] != "block" {
		t.Fatalf("expected continuation, got %#v", result)
	}
	reason := result["reason"].(string)
	assertContains(t, reason, "Goal confirmation warning:")
	assertContains(t, reason, "exit status 7")
	assertContains(t, readText(t, paths.RunsLogPath()), `"outcome":"error"`)
}

func TestStopGoalModeUsesPayloadModelWhenConfigBlankAndOmitsBlankReasoning(t *testing.T) {
	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	argsPath := filepath.Join(t.TempDir(), "args.txt")
	fakeCodex := writeFakeCodex(t)
	installFakeInterpreterCodex(t)
	t.Setenv("FAKE_CODEX_ARGS", argsPath)
	t.Setenv("FAKE_CODEX_STDIN", filepath.Join(t.TempDir(), "stdin.txt"))
	t.Setenv("FAKE_CODEX_REVIEW", "The task needs more evidence before completion.")
	t.Setenv("FAKE_INTERPRET_VERDICT", `{"completed":false,"confidence":0.5,"reason":"still checking","missing_work":["more evidence"],"next_round_guidance":"continue"}`)
	writeRuntimeConfig(t, paths, `[goal]
confirm_model = ""
confirm_reasoning_effort = ""
`+fakeGoalConfirmCommandConfig(fakeCodex)+`
timeout_seconds = 5
interpret_timeout_seconds = 5
`)

	start := fixedTime()
	writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="goal-qa" goal="verify release"]]`+"\nRun the QA task.", start)
	_, err := HandleStop(context.Background(), paths, StopPayload{
		SessionID: "sess-1",
		CWD:       repoRoot,
		Model:     "payload-model",
	}, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}
	args := readText(t, argsPath)
	assertContains(t, args, "--model\npayload-model")
	assertNotContains(t, args, "model_reasoning_effort")
}

func TestStopGoalModeTimesOutConfirmation(t *testing.T) {
	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	fakeCodex := writeFakeCodex(t)
	t.Setenv("FAKE_CODEX_ARGS", filepath.Join(t.TempDir(), "args.txt"))
	t.Setenv("FAKE_CODEX_STDIN", filepath.Join(t.TempDir(), "stdin.txt"))
	t.Setenv("FAKE_CODEX_SLEEP", "2")
	t.Setenv("FAKE_CODEX_REVIEW", "late")
	writeRuntimeConfig(t, paths, `[goal]
`+fakeGoalConfirmCommandConfig(fakeCodex)+`
timeout_seconds = 1
`)

	start := fixedTime()
	writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="goal-qa" goal="verify release"]]`+"\nRun the QA task.", start)
	result, err := HandleStop(context.Background(), paths, StopPayload{SessionID: "sess-1", CWD: repoRoot}, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}
	if result == nil || result["decision"] != "block" {
		t.Fatalf("expected continuation, got %#v", result)
	}
	assertContains(t, result["reason"].(string), "timed out")
	assertContains(t, readText(t, paths.RunsLogPath()), `"outcome":"timeout"`)
}

func TestStopGoalModeContinuesOnInterpretationInvalidOutput(t *testing.T) {
	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	fakeCodex := writeFakeCodex(t)
	installFakeInterpreterCodex(t)
	review := "The task is not complete. Missing work: run release verification."
	t.Setenv("FAKE_CODEX_REVIEW", review)
	t.Setenv("FAKE_INTERPRET_VERDICT", "not-json")
	writeRuntimeConfig(t, paths, `[goal]
`+fakeGoalConfirmCommandConfig(fakeCodex)+`
timeout_seconds = 5
interpret_timeout_seconds = 5
`)

	start := fixedTime()
	writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="goal-qa" goal="verify release"]]`+"\nRun the QA task.", start)
	result, err := HandleStop(context.Background(), paths, StopPayload{SessionID: "sess-1", CWD: repoRoot}, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}
	if result == nil || result["decision"] != "block" {
		t.Fatalf("expected continuation, got %#v", result)
	}
	reason := result["reason"].(string)
	assertContains(t, reason, "Goal confirmation warning:")
	assertContains(t, reason, "decode goal interpretation output")
	assertContains(t, reason, "Goal confirmation review text:")
	assertContains(t, reason, review)
	logText := readText(t, paths.RunsLogPath())
	assertContains(t, logText, `"outcome":"invalid_output"`)
	assertNotContains(t, logText, review)
}

func TestStopGoalModeCustomCommandCanReturnReviewOnStdout(t *testing.T) {
	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	promptArgPath := filepath.Join(t.TempDir(), "prompt-arg.txt")
	promptFileCopyPath := filepath.Join(t.TempDir(), "prompt-file.txt")
	stdinPath := filepath.Join(t.TempDir(), "stdin.txt")
	fakeConfirm := writeFakeStdoutConfirm(t)
	installFakeInterpreterCodex(t)
	t.Setenv("FAKE_CONFIRM_PROMPT_ARG", promptArgPath)
	t.Setenv("FAKE_CONFIRM_PROMPT_FILE_COPY", promptFileCopyPath)
	t.Setenv("FAKE_CONFIRM_STDIN", stdinPath)
	t.Setenv("FAKE_CONFIRM_REVIEW", "This is complete based on stdout review evidence.")
	t.Setenv("FAKE_INTERPRET_VERDICT", `{"completed":true,"confidence":0.92,"reason":"stdout review","missing_work":[],"next_round_guidance":""}`)
	writeRuntimeConfig(t, paths, `[goal]
confirm_command = "`+fakeConfirm+` $PROMPT $PROMPT_FILE"
timeout_seconds = 5
interpret_timeout_seconds = 5
`)

	start := fixedTime()
	path := writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="goal-qa" goal="verify release"]]`+"\nRun the QA task.", start)
	result, err := HandleStop(context.Background(), paths, StopPayload{SessionID: "sess-1", CWD: repoRoot}, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}
	if result != nil {
		t.Fatalf("expected stdout review to complete goal, got %#v", result)
	}
	updated := readLoop(t, path)
	if updated.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %q", updated.Status)
	}
	assertContains(t, readText(t, promptArgPath), "Original task:")
	assertContains(t, readText(t, promptFileCopyPath), "Run the QA task.")
	assertContains(t, readText(t, stdinPath), "Decision rules:")
	assertContains(t, readText(t, paths.RunsLogPath()), `"outcome":"completed"`)
}

func TestStopContinuesBeforeDeadlineWithOptionalGuidance(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	skillDir := filepath.Join(repoRoot, ".agents", "skills", "focused-qa")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("create skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("placeholder"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	writeRuntimeConfig(t, paths, `optional_skill_name = "focused-qa"
optional_skill_path = ".agents/skills/focused-qa"
extra_continuation_guidance = "Capture concrete evidence before you stop."
`)

	start := fixedTime()
	path := writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="release-stress-qa" min="6h"]]`+"\nRun the QA task.", start)
	latest := "Task looks complete."
	result, err := HandleStop(context.Background(), paths, StopPayload{
		SessionID:            "sess-1",
		CWD:                  repoRoot,
		LastAssistantMessage: &latest,
	}, start.Add(30*time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}
	if result == nil || result["decision"] != "block" {
		t.Fatalf("expected block result, got %#v", result)
	}
	reason, ok := result["reason"].(string)
	if !ok {
		t.Fatalf("expected string reason, got %#v", result["reason"])
	}
	assertContains(t, reason, "Remaining time")
	assertContains(t, reason, "focused-qa")
	assertContains(t, reason, filepath.Join(skillDir, "SKILL.md"))
	assertContains(t, reason, "Capture concrete evidence before you stop.")

	updated := readLoop(t, path)
	if updated.ContinueCount != 1 {
		t.Fatalf("expected continue_count 1, got %d", updated.ContinueCount)
	}
	if updated.Status != StatusActive {
		t.Fatalf("expected active status, got %q", updated.Status)
	}
}

func TestStopContinuesBeforeDeadlineWithNameOnlyOptionalSkill(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	writeRuntimeConfig(t, paths, "optional_skill_name = \"codex-loop\"\nextra_continuation_guidance = \"Resume tracked state before stopping.\"\n")

	start := fixedTime()
	writeLoop(t, paths, "sess-1", repoRoot, "[[CODEX_LOOP name=\"release-stress-qa\" min=\"6h\"]]\nRun the QA task.", start)
	result, err := HandleStop(context.Background(), paths, StopPayload{
		SessionID: "sess-1",
		CWD:       repoRoot,
	}, start.Add(30*time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}
	if result == nil || result["decision"] != "block" {
		t.Fatalf("expected block result, got %#v", result)
	}
	reason, ok := result["reason"].(string)
	if !ok {
		t.Fatalf("expected string reason, got %#v", result["reason"])
	}
	assertContains(t, reason, "- explicit use of the codex-loop skill")
	assertContains(t, reason, "Resume tracked state before stopping.")
}

func TestStopMarksTimeLoopCompletedAfterDeadline(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	start := fixedTime()
	path := writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="release-stress-qa" min="5m"]]`+"\nRun the QA task.", start)
	latest := "Done."

	result, err := HandleStop(context.Background(), paths, StopPayload{
		SessionID:            "sess-1",
		CWD:                  repoRoot,
		LastAssistantMessage: &latest,
	}, start.Add(6*time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	updated := readLoop(t, path)
	if updated.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %q", updated.Status)
	}
}

func TestStopRoundsModeCompletesTargetRounds(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	start := fixedTime()
	path := writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="release-stress-qa" rounds="3"]]`+"\nRun the QA task.", start)

	firstMessage := "Round one done."
	first, err := HandleStop(context.Background(), paths, StopPayload{SessionID: "sess-1", CWD: repoRoot, LastAssistantMessage: &firstMessage}, start.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("first stop: %v", err)
	}
	secondMessage := "Round two done."
	second, err := HandleStop(context.Background(), paths, StopPayload{SessionID: "sess-1", CWD: repoRoot, LastAssistantMessage: &secondMessage}, start.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("second stop: %v", err)
	}
	thirdMessage := "Round three done."
	third, err := HandleStop(context.Background(), paths, StopPayload{SessionID: "sess-1", CWD: repoRoot, LastAssistantMessage: &thirdMessage}, start.Add(15*time.Minute))
	if err != nil {
		t.Fatalf("third stop: %v", err)
	}

	if first == nil || first["decision"] != "block" {
		t.Fatalf("expected first block, got %#v", first)
	}
	assertContains(t, first["reason"].(string), "Round 2 of 3 begins now.")
	if second == nil || second["decision"] != "block" {
		t.Fatalf("expected second block, got %#v", second)
	}
	assertContains(t, second["reason"].(string), "Round 3 of 3 begins now.")
	if third != nil {
		t.Fatalf("expected third nil, got %#v", third)
	}
	updated := readLoop(t, path)
	if updated.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %q", updated.Status)
	}
	if updated.CompletedRounds != 3 {
		t.Fatalf("expected completed rounds 3, got %d", updated.CompletedRounds)
	}
}

func TestStopEscalatesOnceThenCutsShortInRoundsMode(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	start := fixedTime()
	path := writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="release-stress-qa" rounds="5"]]`+"\nRun the QA task.", start)
	record := readLoop(t, path)
	record.RapidStopCount = 2
	lastStopAt := record.StartedAt
	record.LastStopAt = &lastStopAt
	if err := ReplaceLoopFile(path, record); err != nil {
		t.Fatalf("replace loop: %v", err)
	}

	escalationMessage := "Stopped again."
	escalation, err := HandleStop(context.Background(), paths, StopPayload{SessionID: "sess-1", CWD: repoRoot, LastAssistantMessage: &escalationMessage}, start.Add(90*time.Second))
	if err != nil {
		t.Fatalf("escalation stop: %v", err)
	}
	cutShortMessage := "Stopped quickly again."
	cutShort, err := HandleStop(context.Background(), paths, StopPayload{SessionID: "sess-1", CWD: repoRoot, LastAssistantMessage: &cutShortMessage}, start.Add(150*time.Second))
	if err != nil {
		t.Fatalf("cut-short stop: %v", err)
	}

	if escalation == nil || escalation["decision"] != "block" {
		t.Fatalf("expected escalation block, got %#v", escalation)
	}
	assertContains(t, escalation["reason"].(string), "Broaden the scope materially")
	if cutShort == nil || cutShort["continue"] != false {
		t.Fatalf("expected cut-short warning, got %#v", cutShort)
	}
	updated := readLoop(t, path)
	if updated.Status != StatusCutShort {
		t.Fatalf("expected cut_short status, got %q", updated.Status)
	}
	if updated.CompletedRounds != 2 {
		t.Fatalf("expected completed rounds 2, got %d", updated.CompletedRounds)
	}
}

func TestStopRunsPreLoopContinueWithSessionCWDAndJSONInput(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	sessionCWD := filepath.Join(repoRoot, "nested")
	if err := os.MkdirAll(sessionCWD, 0o755); err != nil {
		t.Fatalf("create session cwd: %v", err)
	}
	writeRuntimeConfig(t, paths, `[pre_loop_continue]
command = '''/bin/sh -c "printf 'cwd=%s\n' \"$PWD\"; printf 'file='; cat \"$1\"; printf '\nstdin='; cat" sh $INPUT_FILE'''
`)

	start := fixedTime()
	writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="release-stress-qa" rounds="2"]]`+"\nRun the QA task.", start)
	latest := "First pass done."
	result, err := HandleStop(context.Background(), paths, StopPayload{
		SessionID:            "sess-1",
		CWD:                  sessionCWD,
		LastAssistantMessage: &latest,
	}, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}

	reason := result["reason"].(string)
	assertContains(t, reason, "pre_loop_continue output:")
	resolvedSessionCWD, err := filepath.EvalSymlinks(sessionCWD)
	if err != nil {
		t.Fatalf("resolve session cwd symlinks: %v", err)
	}
	assertContains(t, reason, "cwd="+resolvedSessionCWD)
	assertContains(t, reason, "file=")
	assertContains(t, reason, "stdin=")
	assertContains(t, reason, `"event_name":"pre_loop_continue"`)
	assertContains(t, reason, `"session_id":"sess-1"`)
}

func TestStopRunsPreLoopContinueWithWorkspaceRoot(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	sessionCWD := filepath.Join(repoRoot, "nested")
	if err := os.MkdirAll(sessionCWD, 0o755); err != nil {
		t.Fatalf("create session cwd: %v", err)
	}
	writeRuntimeConfig(t, paths, `[pre_loop_continue]
command = "/bin/pwd"
cwd = "workspace_root"
`)

	start := fixedTime()
	writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="release-stress-qa" rounds="2"]]`+"\nRun the QA task.", start)
	result, err := HandleStop(context.Background(), paths, StopPayload{
		SessionID: "sess-1",
		CWD:       sessionCWD,
	}, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}

	reason := result["reason"].(string)
	assertContains(t, reason, "pre_loop_continue output:")
	assertContains(t, reason, repoRoot)
}

func TestStopUsesProjectPreLoopContinueOverGlobal(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	sessionCWD := filepath.Join(repoRoot, "nested")
	if err := os.MkdirAll(sessionCWD, 0o755); err != nil {
		t.Fatalf("create session cwd: %v", err)
	}
	writeRuntimeConfig(t, paths, `[pre_loop_continue]
command = "/bin/sh -c 'printf global-pre-loop'"
cwd = "session_cwd"
`)
	writeProjectRuntimeConfig(t, repoRoot, `[pre_loop_continue]
command = "/bin/pwd"
cwd = "workspace_root"
`)

	start := fixedTime()
	writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="release-stress-qa" rounds="2"]]`+"\nRun the QA task.", start)
	result, err := HandleStop(context.Background(), paths, StopPayload{
		SessionID: "sess-1",
		CWD:       sessionCWD,
	}, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}

	reason := result["reason"].(string)
	assertContains(t, reason, "pre_loop_continue output:")
	assertContains(t, reason, repoRoot)
	assertNotContains(t, reason, "global-pre-loop")
}

func TestStopPreLoopContinueFailureContinuesWithWarning(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	writeRuntimeConfig(t, paths, `[pre_loop_continue]
command = "/bin/sh -c 'printf secret >&2; exit 7'"
`)

	start := fixedTime()
	writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="release-stress-qa" rounds="2"]]`+"\nRun the QA task.", start)
	result, err := HandleStop(context.Background(), paths, StopPayload{
		SessionID: "sess-1",
		CWD:       repoRoot,
	}, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}

	if result["decision"] != "block" {
		t.Fatalf("expected continuation despite failure, got %#v", result)
	}
	reason := result["reason"].(string)
	assertContains(t, reason, "pre_loop_continue warning:")
	assertContains(t, reason, "exit status 7")
	assertNotContains(t, reason, "secret")
}

func TestStopPreLoopContinueTimeoutContinuesWithWarning(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	writeRuntimeConfig(t, paths, `[pre_loop_continue]
command = "/bin/sh -c 'sleep 5'"
timeout_seconds = 1
`)

	start := fixedTime()
	writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="release-stress-qa" rounds="2"]]`+"\nRun the QA task.", start)
	result, err := HandleStop(context.Background(), paths, StopPayload{
		SessionID: "sess-1",
		CWD:       repoRoot,
	}, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}

	reason := result["reason"].(string)
	assertContains(t, reason, "pre_loop_continue warning:")
	assertContains(t, reason, "timed out")
}

func TestStopPreLoopContinueTruncatesOutput(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	writeRuntimeConfig(t, paths, `[pre_loop_continue]
command = "/bin/sh -c 'printf abcdef'"
max_output_bytes = 4
`)

	start := fixedTime()
	writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="release-stress-qa" rounds="2"]]`+"\nRun the QA task.", start)
	result, err := HandleStop(context.Background(), paths, StopPayload{
		SessionID: "sess-1",
		CWD:       repoRoot,
	}, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}

	reason := result["reason"].(string)
	assertContains(t, reason, "abcd")
	assertContains(t, reason, "[output truncated after 4 bytes]")
	assertNotContains(t, reason, "abcdef")
}

func TestStopPreLoopContinueInvalidCWDContinuesWithWarning(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	writeRuntimeConfig(t, paths, `[pre_loop_continue]
command = "/bin/sh -c 'printf should-not-run'"
cwd = "elsewhere"
`)

	start := fixedTime()
	writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="release-stress-qa" rounds="2"]]`+"\nRun the QA task.", start)
	result, err := HandleStop(context.Background(), paths, StopPayload{
		SessionID: "sess-1",
		CWD:       repoRoot,
	}, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}

	reason := result["reason"].(string)
	assertContains(t, reason, "pre_loop_continue warning:")
	assertContains(t, reason, `invalid cwd "elsewhere"`)
	assertNotContains(t, reason, "should-not-run")
}

func TestStopPreLoopContinueDoesNotRunWhenLoopCompletes(t *testing.T) {
	t.Parallel()

	paths := mustPaths(t)
	repoRoot := filepath.Join(t.TempDir(), "repo")
	marker := filepath.Join(t.TempDir(), "marker")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("create repo root: %v", err)
	}
	writeRuntimeConfig(t, paths, `[pre_loop_continue]
command = "/bin/sh -c 'touch \"$1\"' sh `+marker+`"
`)

	start := fixedTime()
	writeLoop(t, paths, "sess-1", repoRoot, `[[CODEX_LOOP name="release-stress-qa" rounds="1"]]`+"\nRun the QA task.", start)
	result, err := HandleStop(context.Background(), paths, StopPayload{
		SessionID: "sess-1",
		CWD:       repoRoot,
	}, start.Add(time.Minute))
	if err != nil {
		t.Fatalf("handle stop: %v", err)
	}
	if result != nil {
		t.Fatalf("expected completed loop to return nil, got %#v", result)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("expected marker not to exist, stat err: %v", err)
	}
}

func mustPaths(t *testing.T) Paths {
	t.Helper()
	paths, err := NewPaths(filepath.Join(t.TempDir(), ".codex-home"))
	if err != nil {
		t.Fatalf("new paths: %v", err)
	}
	return paths
}

func fixedTime() time.Time {
	return time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
}

func writeRuntimeConfig(t *testing.T, paths Paths, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(paths.RuntimeConfigPath()), 0o755); err != nil {
		t.Fatalf("create runtime config dir: %v", err)
	}
	if err := os.WriteFile(paths.RuntimeConfigPath(), []byte(content), 0o644); err != nil {
		t.Fatalf("write runtime config: %v", err)
	}
}

func fakeGoalConfirmCommandConfig(fakeCodex string) string {
	return `confirm_command = "` + fakeCodex + ` exec --yolo --output-last-message $CONFIRM_OUTPUT_PATH $MODEL_ARGV $REASONING_ARGV --skip-git-repo-check -"`
}

func writeFakeCodex(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "codex")
	script := `#!/bin/sh
set -eu
out=""
prev=""
if [ -n "${FAKE_CODEX_ARGS:-}" ]; then
  : > "$FAKE_CODEX_ARGS"
fi
for arg in "$@"; do
  if [ -n "${FAKE_CODEX_ARGS:-}" ]; then
    printf '%s\n' "$arg" >> "$FAKE_CODEX_ARGS"
  fi
  if [ "$prev" = "--output-last-message" ]; then
    out="$arg"
  fi
  prev="$arg"
done
if [ -n "${FAKE_CODEX_STDIN:-}" ]; then
  cat > "$FAKE_CODEX_STDIN"
else
  cat >/dev/null
fi
if [ -n "${FAKE_CODEX_SLEEP:-}" ]; then
  sleep "$FAKE_CODEX_SLEEP"
fi
if [ -n "$out" ]; then
  printf '%s\n' "${FAKE_CODEX_REVIEW:-}" > "$out"
fi
exit "${FAKE_CODEX_EXIT:-0}"
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	return path
}

func installFakeInterpreterCodex(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "codex")
	script := `#!/bin/sh
set -eu
out=""
prev=""
if [ -n "${FAKE_INTERPRET_ARGS:-}" ]; then
  : > "$FAKE_INTERPRET_ARGS"
fi
for arg in "$@"; do
  if [ -n "${FAKE_INTERPRET_ARGS:-}" ]; then
    printf '%s\n' "$arg" >> "$FAKE_INTERPRET_ARGS"
  fi
  if [ "$prev" = "--output-last-message" ]; then
    out="$arg"
  fi
  prev="$arg"
done
if [ -n "${FAKE_INTERPRET_STDIN:-}" ]; then
  cat > "$FAKE_INTERPRET_STDIN"
else
  cat >/dev/null
fi
if [ -n "${FAKE_INTERPRET_SLEEP:-}" ]; then
  sleep "$FAKE_INTERPRET_SLEEP"
fi
if [ -n "$out" ]; then
  printf '%s\n' "${FAKE_INTERPRET_VERDICT:-}" > "$out"
fi
exit "${FAKE_INTERPRET_EXIT:-0}"
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake interpreter codex: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func writeFakeStdoutConfirm(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "confirm")
	script := `#!/bin/sh
set -eu
if [ -n "${FAKE_CONFIRM_PROMPT_ARG:-}" ]; then
  printf '%s' "${1:-}" > "$FAKE_CONFIRM_PROMPT_ARG"
fi
if [ -n "${FAKE_CONFIRM_PROMPT_FILE_COPY:-}" ]; then
  cat "${2:-/dev/null}" > "$FAKE_CONFIRM_PROMPT_FILE_COPY"
fi
if [ -n "${FAKE_CONFIRM_STDIN:-}" ]; then
  cat > "$FAKE_CONFIRM_STDIN"
else
  cat >/dev/null
fi
printf '%s\n' "${FAKE_CONFIRM_REVIEW:-The task is complete based on stdout review.}"
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake stdout confirm: %v", err)
	}
	return path
}

func writeLoop(t *testing.T, paths Paths, sessionID string, repoRoot string, prompt string, now time.Time) string {
	t.Helper()
	activation, ok, err := ExtractActivation(prompt)
	if err != nil {
		t.Fatalf("extract activation: %v", err)
	}
	if !ok {
		t.Fatal("expected activation")
	}
	record := BuildLoopRecord(sessionID, repoRoot, repoRoot, activation, now)
	path := CreateLoopPath(paths, record.Slug, now)
	if err := ReplaceLoopFile(path, record); err != nil {
		t.Fatalf("write loop: %v", err)
	}
	return path
}

func readLoop(t *testing.T, path string) LoopRecord {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read loop: %v", err)
	}
	var record LoopRecord
	if err := json.Unmarshal(content, &record); err != nil {
		t.Fatalf("decode loop: %v", err)
	}
	return record
}

func readText(t *testing.T, path string) string {
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

func assertNotContains(t *testing.T, haystack string, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Fatalf("expected %q not to contain %q", haystack, needle)
	}
}
