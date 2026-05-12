package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	RecordVersion = 4
	RuntimeName   = "codex-loop"

	StatusActive     = "active"
	StatusCompleted  = "completed"
	StatusSuperseded = "superseded"
	StatusCutShort   = "cut_short"
)

type Paths struct {
	CodexHome string
}

type LoopRecord struct {
	Version              int      `json:"version"`
	SessionID            string   `json:"session_id"`
	Name                 string   `json:"name"`
	Slug                 string   `json:"slug"`
	CWD                  string   `json:"cwd"`
	WorkspaceRoot        string   `json:"workspace_root"`
	StartedAt            string   `json:"started_at"`
	TaskPrompt           string   `json:"task_prompt"`
	ActivationPrompt     string   `json:"activation_prompt"`
	Status               string   `json:"status"`
	LimitMode            string   `json:"limit_mode"`
	ContinueCount        int      `json:"continue_count"`
	RapidStopCount       int      `json:"rapid_stop_count"`
	EscalationUsed       bool     `json:"escalation_used"`
	LastStopAt           *string  `json:"last_stop_at"`
	LastContinueAt       *string  `json:"last_continue_at"`
	LastAssistantMessage *string  `json:"last_assistant_message"`
	DurationText         *string  `json:"duration_text"`
	MinDurationSeconds   *int     `json:"min_duration_seconds"`
	DeadlineAt           *string  `json:"deadline_at"`
	RoundsText           *string  `json:"rounds_text"`
	TargetRounds         *int     `json:"target_rounds"`
	CompletedRounds      int      `json:"completed_rounds"`
	GoalText             *string  `json:"goal_text"`
	ConfirmModel         *string  `json:"confirm_model"`
	ConfirmReasoning     *string  `json:"confirm_reasoning_effort"`
	GoalCheckCount       int      `json:"goal_check_count"`
	LastGoalCheckAt      *string  `json:"last_goal_check_at"`
	LastGoalOutcome      *string  `json:"last_goal_outcome"`
	LastGoalConfidence   *float64 `json:"last_goal_confidence"`
	LastGoalReason       *string  `json:"last_goal_reason"`
	LastGoalError        *string  `json:"last_goal_error"`
}

type LoopFile struct {
	Path   string
	Record LoopRecord
}

type StatusRecord struct {
	LoopRecord
	Path string `json:"path"`
}

type StatusFilter struct {
	All           bool
	SessionID     string
	WorkspaceRoot string
}

func DefaultPaths() (Paths, error) {
	if configured := os.Getenv("CODEX_HOME"); configured != "" {
		return NewPaths(configured)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve user home: %w", err)
	}
	return NewPaths(filepath.Join(home, ".codex"))
}

func NewPaths(codexHome string) (Paths, error) {
	if codexHome == "" {
		return Paths{}, fmt.Errorf("codex home is required")
	}
	expanded := codexHome
	if expanded == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, fmt.Errorf("resolve user home: %w", err)
		}
		expanded = home
	} else if len(expanded) > 2 && expanded[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, fmt.Errorf("resolve user home: %w", err)
		}
		expanded = filepath.Join(home, expanded[2:])
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return Paths{}, fmt.Errorf("resolve codex home %q: %w", codexHome, err)
	}
	return Paths{CodexHome: abs}, nil
}

func (p Paths) ConfigPath() string {
	return filepath.Join(p.CodexHome, "config.toml")
}

func (p Paths) HooksPath() string {
	return filepath.Join(p.CodexHome, "hooks.json")
}

func (p Paths) RuntimeRoot() string {
	return filepath.Join(p.CodexHome, RuntimeName)
}

func (p Paths) RuntimeBinDir() string {
	return filepath.Join(p.RuntimeRoot(), "bin")
}

func (p Paths) RuntimeBinaryPath() string {
	return filepath.Join(p.RuntimeBinDir(), RuntimeName)
}

func (p Paths) LoopsDir() string {
	return filepath.Join(p.RuntimeRoot(), "loops")
}

func (p Paths) RuntimeConfigPath() string {
	return filepath.Join(p.RuntimeRoot(), "config.toml")
}

func (p Paths) SkillsDir() string {
	return filepath.Join(p.CodexHome, "skills")
}

func (p Paths) BuiltInSkillDir() string {
	return filepath.Join(p.SkillsDir(), RuntimeName)
}

func (p Paths) RunsLogPath() string {
	return filepath.Join(p.RuntimeRoot(), "runs.jsonl")
}

func ResolveWorkspaceRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root from %q: %w", start, err)
	}
	for {
		if info, statErr := os.Stat(filepath.Join(current, ".codex")); statErr == nil && info.IsDir() {
			return current, nil
		}
		if _, statErr := os.Stat(filepath.Join(current, ".git")); statErr == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return filepath.Abs(start)
		}
		current = parent
	}
}

func ISOFormat(value time.Time) string {
	return value.UTC().Truncate(time.Second).Format(time.RFC3339)
}

func ParseISO8601(value *string) (*time.Time, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		return nil, fmt.Errorf("parse timestamp %q: %w", *value, err)
	}
	utc := parsed.UTC()
	return &utc, nil
}

func BuildLoopRecord(sessionID string, cwd string, workspaceRoot string, activation Activation, now time.Time) LoopRecord {
	startedAt := ISOFormat(now)
	record := LoopRecord{
		Version:          RecordVersion,
		SessionID:        sessionID,
		Name:             activation.Name,
		Slug:             activation.Slug,
		CWD:              cwd,
		WorkspaceRoot:    workspaceRoot,
		StartedAt:        startedAt,
		TaskPrompt:       activation.TaskPrompt,
		ActivationPrompt: activation.ActivationPrompt,
		Status:           StatusActive,
		LimitMode:        activation.LimitMode,
	}
	if activation.LimitMode == LimitModeTime {
		durationText := activation.DurationText
		minDurationSeconds := activation.MinDurationSeconds
		deadlineAt := ISOFormat(now.Add(time.Duration(minDurationSeconds) * time.Second))
		record.DurationText = &durationText
		record.MinDurationSeconds = &minDurationSeconds
		record.DeadlineAt = &deadlineAt
		return record
	}
	if activation.LimitMode == LimitModeGoal {
		goalText := activation.GoalText
		record.GoalText = &goalText
		if activation.ConfirmModel != "" {
			confirmModel := activation.ConfirmModel
			record.ConfirmModel = &confirmModel
		}
		if activation.ConfirmReasoning != "" {
			confirmReasoning := activation.ConfirmReasoning
			record.ConfirmReasoning = &confirmReasoning
		}
		return record
	}

	roundsText := activation.RoundsText
	targetRounds := activation.TargetRounds
	record.RoundsText = &roundsText
	record.TargetRounds = &targetRounds
	return record
}

func CreateLoopPath(paths Paths, slug string, now time.Time) string {
	stamp := now.UTC().Format("20060102T150405Z")
	return filepath.Join(paths.LoopsDir(), fmt.Sprintf("%s_%s.json", stamp, slug))
}

func ReplaceLoopFile(path string, record LoopRecord) error {
	record.Status = NormalizeStatus(record.Status)
	return AtomicWriteJSON(path, record)
}

func AtomicWriteJSON(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directory for %q: %w", path, err)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*.json")
	if err != nil {
		return fmt.Errorf("create temp JSON file for %q: %w", path, err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	encoder := json.NewEncoder(temp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		_ = temp.Close()
		return fmt.Errorf("encode JSON %q: %w", path, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temp JSON file for %q: %w", path, err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("replace JSON file %q: %w", path, err)
	}
	return nil
}

func LoadLoop(path string) (LoopRecord, error) {
	handle, err := os.Open(path)
	if err != nil {
		return LoopRecord{}, fmt.Errorf("open loop file %q: %w", path, err)
	}
	defer handle.Close()

	var record LoopRecord
	if err := json.NewDecoder(handle).Decode(&record); err != nil {
		return LoopRecord{}, fmt.Errorf("decode loop file %q: %w", path, err)
	}
	record.Status = NormalizeStatus(record.Status)
	return record, nil
}

func IterLoopRecords(paths Paths) ([]LoopFile, error) {
	entries, err := os.ReadDir(paths.LoopsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read loops directory %q: %w", paths.LoopsDir(), err)
	}

	records := make([]LoopFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(paths.LoopsDir(), entry.Name())
		record, loadErr := LoadLoop(path)
		if loadErr != nil {
			continue
		}
		records = append(records, LoopFile{Path: path, Record: record})
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Path < records[j].Path
	})
	return records, nil
}

func SupersedeActiveLoops(paths Paths, sessionID string, keepPath string) error {
	records, err := IterLoopRecords(paths)
	if err != nil {
		return err
	}
	for _, loopFile := range records {
		if loopFile.Path == keepPath {
			continue
		}
		if loopFile.Record.SessionID != sessionID || NormalizeStatus(loopFile.Record.Status) != StatusActive {
			continue
		}
		loopFile.Record.Status = StatusSuperseded
		if err := ReplaceLoopFile(loopFile.Path, loopFile.Record); err != nil {
			return err
		}
	}
	return nil
}

func ResolveActiveLoop(paths Paths, sessionID string) (*LoopFile, error) {
	records, err := IterLoopRecords(paths)
	if err != nil {
		return nil, err
	}
	active := make([]LoopFile, 0)
	for _, loopFile := range records {
		if loopFile.Record.SessionID == sessionID && NormalizeStatus(loopFile.Record.Status) == StatusActive {
			active = append(active, loopFile)
		}
	}
	if len(active) == 0 {
		return nil, nil
	}
	sort.Slice(active, func(i, j int) bool {
		left := active[i].Record.StartedAt
		right := active[j].Record.StartedAt
		if left == right {
			return active[i].Path < active[j].Path
		}
		return left < right
	})
	keep := active[len(active)-1]
	for _, stale := range active[:len(active)-1] {
		stale.Record.Status = StatusSuperseded
		if err := ReplaceLoopFile(stale.Path, stale.Record); err != nil {
			return nil, err
		}
	}
	return &keep, nil
}

func ListStatusRecords(paths Paths, filter StatusFilter) ([]StatusRecord, error) {
	workspaceRoot := ""
	if filter.WorkspaceRoot != "" {
		abs, err := filepath.Abs(filter.WorkspaceRoot)
		if err != nil {
			return nil, fmt.Errorf("resolve workspace root filter %q: %w", filter.WorkspaceRoot, err)
		}
		workspaceRoot = abs
	}

	records, err := IterLoopRecords(paths)
	if err != nil {
		return nil, err
	}
	statusRecords := make([]StatusRecord, 0, len(records))
	for _, loopFile := range records {
		record := loopFile.Record
		if filter.SessionID != "" && record.SessionID != filter.SessionID {
			continue
		}
		if workspaceRoot != "" {
			candidate := record.WorkspaceRoot
			if candidate == "" {
				candidate = record.CWD
			}
			abs, absErr := filepath.Abs(candidate)
			if absErr != nil || abs != workspaceRoot {
				continue
			}
		}
		if !filter.All && NormalizeStatus(record.Status) != StatusActive {
			continue
		}
		statusRecords = append(statusRecords, StatusRecord{
			LoopRecord: record,
			Path:       loopFile.Path,
		})
	}
	return statusRecords, nil
}

func NormalizeStatus(value string) string {
	switch value {
	case StatusActive, StatusCompleted, StatusSuperseded, StatusCutShort:
		return value
	default:
		return StatusActive
	}
}

func ResolveLimitMode(record LoopRecord) string {
	if record.LimitMode == LimitModeTime || record.LimitMode == LimitModeRounds || record.LimitMode == LimitModeGoal {
		return record.LimitMode
	}
	if record.GoalText != nil {
		return LimitModeGoal
	}
	if record.TargetRounds != nil {
		return LimitModeRounds
	}
	return LimitModeTime
}
