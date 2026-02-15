package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestDryRunPreToolUseHooks_NoMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		PreToolUse: []PreToolUseHook{
			{Matcher: "Edit", Actions: []Action{{Type: "output", Message: "test", ExitStatus: &[]int{0}[0]}}},
		},
	}

	input := &PreToolUseInput{ToolName: "Write"} // マッチしない

	err := dryRunPreToolUseHooks(config, input, nil)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunPreToolUseHooks() error = %v", err)
	}

	if !strings.Contains(output, "No hooks would be executed") {
		t.Errorf("Expected 'No hooks would be executed', got: %q", output)
	}
}

func TestDryRunPreToolUseHooks_WithMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		PreToolUse: []PreToolUseHook{
			{
				Matcher: "Write",
				Actions: []Action{
					{Type: "command", Command: "echo {.tool_input.file_path}"},
					{Type: "output", Message: "Processing...", ExitStatus: &[]int{0}[0]},
				},
			},
		},
	}

	input := &PreToolUseInput{
		ToolName:  "Write",
		ToolInput: ToolInput{FilePath: "test.go"},
	}

	// JQクエリが動作するようにrawJSONを作成
	rawJSON := map[string]any{
		"tool_name": "Write",
		"tool_input": map[string]any{
			"file_path": "test.go",
		},
	}

	err := dryRunPreToolUseHooks(config, input, rawJSON)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunPreToolUseHooks() error = %v", err)
	}

	expectedStrings := []string{
		"=== PreToolUse Hooks (Dry Run) ===",
		"[Hook 1] Would execute:",
		"Command: echo test.go",
		"Message: Processing...",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got: %q", expected, output)
		}
	}
}

func TestDryRunPreToolUseHooks_ProcessSubstitution(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		PreToolUse: []PreToolUseHook{
			{
				Matcher: "Bash",
				Conditions: []Condition{
					{Type: ConditionGitTrackedFileOperation, Value: "diff"},
				},
				Actions: []Action{
					{
						Type:    "output",
						Message: "Git tracked file operation detected",
					},
				},
			},
		},
	}

	input := &PreToolUseInput{
		BaseInput: BaseInput{
			SessionID:      "test-session-123",
			TranscriptPath: "/path/to/transcript",
			HookEventName:  PreToolUse,
		},
		ToolName:  "Bash",
		ToolInput: ToolInput{Command: "diff -u file1 <(head -48 file2)"},
	}

	rawJSON := map[string]any{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"tool_name":       input.ToolName,
		"tool_input": map[string]any{
			"command": input.ToolInput.Command,
		},
	}

	err := dryRunPreToolUseHooks(config, input, rawJSON)

	// 標準出力を元に戻す
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取る
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	stdoutOutput := buf.String()

	// エラーがないこと
	if err != nil {
		t.Errorf("dryRunPreToolUseHooks() error = %v, want nil", err)
	}

	// 標準出力に"would deny"が含まれることを確認
	if !strings.Contains(stdoutOutput, "would deny") {
		t.Errorf("stdout output = %v, want to contain 'would deny'", stdoutOutput)
	}
}

func TestDryRunPostToolUseHooks_ProcessSubstitution(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		PostToolUse: []PostToolUseHook{
			{
				Matcher: "Bash",
				Conditions: []Condition{
					{Type: ConditionGitTrackedFileOperation, Value: "diff"},
				},
				Actions: []Action{
					{
						Type:    "output",
						Message: "Git tracked file operation detected",
					},
				},
			},
		},
	}

	input := &PostToolUseInput{
		BaseInput: BaseInput{
			SessionID:      "test-session-123",
			TranscriptPath: "/path/to/transcript",
			HookEventName:  PostToolUse,
		},
		ToolName:  "Bash",
		ToolInput: ToolInput{Command: "diff -u file1 <(head -48 file2)"},
	}

	rawJSON := map[string]any{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"tool_name":       input.ToolName,
		"tool_input": map[string]any{
			"command": input.ToolInput.Command,
		},
	}

	err := dryRunPostToolUseHooks(config, input, rawJSON)

	// 標準出力を元に戻す
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取る
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	stdoutOutput := buf.String()

	// エラーがないこと
	if err != nil {
		t.Errorf("dryRunPostToolUseHooks() error = %v, want nil", err)
	}

	// 標準出力に"would warn"が含まれることを確認
	if !strings.Contains(stdoutOutput, "would warn") {
		t.Errorf("stdout output = %v, want to contain 'would warn'", stdoutOutput)
	}
}

func TestDryRunNotificationHooks_NoMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		Notification: []NotificationHook{
			{Matcher: "permission_prompt", Actions: []Action{{Type: "output", Message: "test"}}},
		},
	}

	input := &NotificationInput{
		BaseInput:        BaseInput{HookEventName: Notification},
		NotificationType: "idle_prompt", // マッチしない
	}

	err := dryRunNotificationHooks(config, input, nil)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunNotificationHooks() error = %v", err)
	}

	if !strings.Contains(output, "No hooks would be executed") {
		t.Errorf("Expected 'No hooks would be executed', got: %q", output)
	}
}

func TestDryRunNotificationHooks_WithMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		Notification: []NotificationHook{
			{
				Matcher: "permission_prompt",
				Actions: []Action{
					{Type: "command", Command: "echo {.notification_type}"},
					{Type: "output", Message: "Notification received"},
				},
			},
		},
	}

	input := &NotificationInput{
		BaseInput:        BaseInput{HookEventName: Notification},
		NotificationType: "permission_prompt",
		Title:            "Test Notification",
	}

	rawJSON := map[string]any{
		"hook_event_name":   string(input.HookEventName),
		"notification_type": input.NotificationType,
		"title":             input.Title,
	}

	err := dryRunNotificationHooks(config, input, rawJSON)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunNotificationHooks() error = %v", err)
	}

	expectedStrings := []string{
		"=== Notification Hooks (Dry Run) ===",
		"[Hook 1] Would execute:",
		"Command: echo permission_prompt",
		"Message: Notification received",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got: %q", expected, output)
		}
	}
}

func TestDryRunSubagentStartHooks_NoMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		SubagentStart: []SubagentStartHook{
			{Matcher: "Explore", Actions: []Action{{Type: "output", Message: "test"}}},
		},
	}

	input := &SubagentStartInput{
		BaseInput: BaseInput{HookEventName: SubagentStart},
		AgentType: "Bash", // マッチしない
	}

	err := dryRunSubagentStartHooks(config, input, nil)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunSubagentStartHooks() error = %v", err)
	}

	if !strings.Contains(output, "No hooks would be executed") {
		t.Errorf("Expected 'No hooks would be executed', got: %q", output)
	}
}

func TestDryRunSubagentStartHooks_WithMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		SubagentStart: []SubagentStartHook{
			{
				Matcher: "Explore",
				Actions: []Action{
					{Type: "command", Command: "echo {.agent_type}"},
					{Type: "output", Message: "Subagent started"},
				},
			},
		},
	}

	input := &SubagentStartInput{
		BaseInput: BaseInput{HookEventName: SubagentStart},
		AgentID:   "test-agent-123",
		AgentType: "Explore",
	}

	rawJSON := map[string]any{
		"hook_event_name": string(input.HookEventName),
		"agent_id":        input.AgentID,
		"agent_type":      input.AgentType,
	}

	err := dryRunSubagentStartHooks(config, input, rawJSON)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunSubagentStartHooks() error = %v", err)
	}

	expectedStrings := []string{
		"=== SubagentStart Hooks (Dry Run) ===",
		"[Hook 1] Would execute:",
		"Command: echo Explore",
		"Message: Subagent started",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got: %q", expected, output)
		}
	}
}

func TestDryRunStopHooks_NoMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		Stop: []StopHook{
			{
				Conditions: []Condition{{Type: ConditionCwdContains, Value: "/important"}},
				Actions:    []Action{{Type: "output", Message: "test"}},
			},
		},
	}

	input := &StopInput{
		BaseInput: BaseInput{
			HookEventName: Stop,
			Cwd:           "/tmp", // マッチしない
		},
	}

	err := dryRunStopHooks(config, input, nil)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunStopHooks() error = %v", err)
	}

	if !strings.Contains(output, "No hooks would be executed") {
		t.Errorf("Expected 'No hooks would be executed', got: %q", output)
	}
}

func TestDryRunStopHooks_WithMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		Stop: []StopHook{
			{
				Conditions: []Condition{{Type: ConditionCwdContains, Value: "/important"}},
				Actions: []Action{
					{Type: "command", Command: "echo {.cwd}"},
					{Type: "output", Message: "Stop requested", Decision: stringPtr("block"), Reason: stringPtr("Important directory")},
				},
			},
		},
	}

	input := &StopInput{
		BaseInput: BaseInput{
			HookEventName: Stop,
			Cwd:           "/important/project",
		},
		StopHookActive: true,
	}

	rawJSON := map[string]any{
		"hook_event_name":  string(input.HookEventName),
		"cwd":              input.Cwd,
		"stop_hook_active": input.StopHookActive,
	}

	err := dryRunStopHooks(config, input, rawJSON)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunStopHooks() error = %v", err)
	}

	expectedStrings := []string{
		"=== Stop Hooks (Dry Run) ===",
		"[Hook 1] Would execute:",
		"Command: echo /important/project",
		"Message: Stop requested",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got: %q", expected, output)
		}
	}
}

func TestDryRunSubagentStopHooks_NoMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		SubagentStop: []SubagentStopHook{
			{
				Conditions: []Condition{{Type: ConditionCwdContains, Value: "/important"}},
				Actions:    []Action{{Type: "output", Message: "test"}},
			},
		},
	}

	input := &SubagentStopInput{
		BaseInput: BaseInput{
			HookEventName: SubagentStop,
			Cwd:           "/tmp", // マッチしない
		},
	}

	err := dryRunSubagentStopHooks(config, input, nil)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunSubagentStopHooks() error = %v", err)
	}

	if !strings.Contains(output, "No hooks would be executed") {
		t.Errorf("Expected 'No hooks would be executed', got: %q", output)
	}
}

func TestDryRunSubagentStopHooks_WithMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		SubagentStop: []SubagentStopHook{
			{
				Conditions: []Condition{{Type: ConditionCwdContains, Value: "/important"}},
				Actions: []Action{
					{Type: "command", Command: "echo {.cwd}"},
					{Type: "output", Message: "SubagentStop requested", Decision: stringPtr("block"), Reason: stringPtr("Important directory")},
				},
			},
		},
	}

	input := &SubagentStopInput{
		BaseInput: BaseInput{
			HookEventName: SubagentStop,
			Cwd:           "/important/project",
		},
		StopHookActive: true,
	}

	rawJSON := map[string]any{
		"hook_event_name":  string(input.HookEventName),
		"cwd":              input.Cwd,
		"stop_hook_active": input.StopHookActive,
	}

	err := dryRunSubagentStopHooks(config, input, rawJSON)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunSubagentStopHooks() error = %v", err)
	}

	expectedStrings := []string{
		"=== SubagentStop Hooks (Dry Run) ===",
		"[Hook 1] Would execute:",
		"Command: echo /important/project",
		"Message: SubagentStop requested",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got: %q", expected, output)
		}
	}
}

func TestDryRunPreCompactHooks_NoMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		PreCompact: []PreCompactHook{
			{Matcher: "manual", Actions: []Action{{Type: "output", Message: "test"}}},
		},
	}

	input := &PreCompactInput{
		BaseInput: BaseInput{HookEventName: PreCompact},
		Trigger:   "auto", // マッチしない
	}

	err := dryRunPreCompactHooks(config, input, nil)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunPreCompactHooks() error = %v", err)
	}

	if !strings.Contains(output, "No hooks would be executed") {
		t.Errorf("Expected 'No hooks would be executed', got: %q", output)
	}
}

func TestDryRunPreCompactHooks_WithMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		PreCompact: []PreCompactHook{
			{
				Matcher: "manual",
				Actions: []Action{
					{Type: "command", Command: "echo {.trigger}"},
					{Type: "output", Message: "Pre-compaction processing"},
				},
			},
		},
	}

	input := &PreCompactInput{
		BaseInput:          BaseInput{HookEventName: PreCompact},
		Trigger:            "manual",
		CustomInstructions: "Test instructions",
	}

	rawJSON := map[string]any{
		"hook_event_name":     string(input.HookEventName),
		"trigger":             input.Trigger,
		"custom_instructions": input.CustomInstructions,
	}

	err := dryRunPreCompactHooks(config, input, rawJSON)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunPreCompactHooks() error = %v", err)
	}

	expectedStrings := []string{
		"=== PreCompact Hooks (Dry Run) ===",
		"[Hook 1] Would execute:",
		"Command: echo manual",
		"Message: Pre-compaction processing",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got: %q", expected, output)
		}
	}
}

func TestDryRunSessionStartHooks_NoMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		SessionStart: []SessionStartHook{
			{Matcher: "startup", Actions: []Action{{Type: "output", Message: "test"}}},
		},
	}

	input := &SessionStartInput{
		BaseInput: BaseInput{HookEventName: SessionStart},
		Source:    "resume", // マッチしない
	}

	err := dryRunSessionStartHooks(config, input, nil)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunSessionStartHooks() error = %v", err)
	}

	if !strings.Contains(output, "No matching SessionStart hooks found") {
		t.Errorf("Expected 'No matching SessionStart hooks found', got: %q", output)
	}
}

func TestDryRunSessionStartHooks_WithMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		SessionStart: []SessionStartHook{
			{
				Matcher: "startup",
				Actions: []Action{
					{Type: "command", Command: "echo {.source}"},
					{Type: "output", Message: "Session started"},
				},
			},
		},
	}

	input := &SessionStartInput{
		BaseInput: BaseInput{HookEventName: SessionStart},
		Source:    "startup",
		AgentType: "default",
		Model:     "claude-sonnet-4",
	}

	rawJSON := map[string]any{
		"hook_event_name": string(input.HookEventName),
		"source":          input.Source,
		"agent_type":      input.AgentType,
		"model":           input.Model,
	}

	err := dryRunSessionStartHooks(config, input, rawJSON)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunSessionStartHooks() error = %v", err)
	}

	expectedStrings := []string{
		"=== SessionStart Hooks (Dry Run) ===",
		"[Hook 1] Matcher: startup, Source: startup",
		"Command: echo startup",
		"Message: Session started",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got: %q", expected, output)
		}
	}
}

func TestDryRunUserPromptSubmitHooks_NoMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		UserPromptSubmit: []UserPromptSubmitHook{
			{
				Conditions: []Condition{{Type: ConditionPromptRegex, Value: "delete"}},
				Actions:    []Action{{Type: "output", Message: "test"}},
			},
		},
	}

	input := &UserPromptSubmitInput{
		BaseInput: BaseInput{HookEventName: UserPromptSubmit},
		Prompt:    "hello world", // マッチしない
	}

	err := dryRunUserPromptSubmitHooks(config, input, nil)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunUserPromptSubmitHooks() error = %v", err)
	}

	if !strings.Contains(output, "No matching UserPromptSubmit hooks found") {
		t.Errorf("Expected 'No matching UserPromptSubmit hooks found', got: %q", output)
	}
}

func TestDryRunUserPromptSubmitHooks_WithMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		UserPromptSubmit: []UserPromptSubmitHook{
			{
				Conditions: []Condition{{Type: ConditionPromptRegex, Value: "delete"}},
				Actions: []Action{
					{Type: "command", Command: "echo {.prompt}"},
					{Type: "output", Message: "Dangerous command detected", Decision: stringPtr("block")},
				},
			},
		},
	}

	input := &UserPromptSubmitInput{
		BaseInput: BaseInput{HookEventName: UserPromptSubmit},
		Prompt:    "delete all files",
	}

	rawJSON := map[string]any{
		"hook_event_name": string(input.HookEventName),
		"prompt":          input.Prompt,
	}

	err := dryRunUserPromptSubmitHooks(config, input, rawJSON)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunUserPromptSubmitHooks() error = %v", err)
	}

	expectedStrings := []string{
		"=== UserPromptSubmit Hooks (Dry Run) ===",
		"[Hook 1] Prompt: delete all files",
		"Command: echo delete all files",
		"Message: Dangerous command detected",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got: %q", expected, output)
		}
	}
}

func TestDryRunSessionEndHooks_NoMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		SessionEnd: []SessionEndHook{
			{
				Conditions: []Condition{{Type: ConditionReasonIs, Value: "clear"}},
				Actions:    []Action{{Type: "output", Message: "test"}},
			},
		},
	}

	input := &SessionEndInput{
		BaseInput: BaseInput{HookEventName: SessionEnd},
		Reason:    "logout", // マッチしない
	}

	err := dryRunSessionEndHooks(config, input, nil)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunSessionEndHooks() error = %v", err)
	}

	if !strings.Contains(output, "No matching SessionEnd hooks found") {
		t.Errorf("Expected 'No matching SessionEnd hooks found', got: %q", output)
	}
}

func TestDryRunSessionEndHooks_WithMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		SessionEnd: []SessionEndHook{
			{
				Conditions: []Condition{{Type: ConditionReasonIs, Value: "clear"}},
				Actions: []Action{
					{Type: "command", Command: "echo {.reason}"},
					{Type: "output", Message: "Session cleanup complete"},
				},
			},
		},
	}

	input := &SessionEndInput{
		BaseInput: BaseInput{HookEventName: SessionEnd},
		Reason:    "clear",
	}

	rawJSON := map[string]any{
		"hook_event_name": string(input.HookEventName),
		"reason":          input.Reason,
	}

	err := dryRunSessionEndHooks(config, input, rawJSON)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunSessionEndHooks() error = %v", err)
	}

	expectedStrings := []string{
		"=== SessionEnd Hooks (Dry Run) ===",
		"[Hook 1] Reason: clear",
		"Command: echo clear",
		"Message: Session cleanup complete",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got: %q", expected, output)
		}
	}
}

func TestDryRunPermissionRequestHooks_NoMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		PermissionRequest: []PermissionRequestHook{
			{Matcher: "Write", Actions: []Action{{Type: "output", Message: "test"}}},
		},
	}

	input := &PermissionRequestInput{
		BaseInput: BaseInput{HookEventName: PermissionRequest},
		ToolName:  "Read", // マッチしない
	}

	err := dryRunPermissionRequestHooks(config, input, nil)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunPermissionRequestHooks() error = %v", err)
	}

	if !strings.Contains(output, "No matching PermissionRequest hooks found") {
		t.Errorf("Expected 'No matching PermissionRequest hooks found', got: %q", output)
	}
}

func TestDryRunPermissionRequestHooks_WithMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		PermissionRequest: []PermissionRequestHook{
			{
				Matcher: "Write",
				Actions: []Action{
					{Type: "command", Command: "echo {.tool_input.file_path}"},
					{Type: "output", Message: "Permission granted", Behavior: stringPtr("allow")},
				},
			},
		},
	}

	input := &PermissionRequestInput{
		BaseInput: BaseInput{HookEventName: PermissionRequest},
		ToolName:  "Write",
		ToolInput: ToolInput{FilePath: "test.txt"},
	}

	rawJSON := map[string]any{
		"hook_event_name": string(input.HookEventName),
		"tool_name":       input.ToolName,
		"tool_input": map[string]any{
			"file_path": input.ToolInput.FilePath,
		},
	}

	err := dryRunPermissionRequestHooks(config, input, rawJSON)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunPermissionRequestHooks() error = %v", err)
	}

	expectedStrings := []string{
		"=== PermissionRequest Hooks ===",
		"[Hook 1] Tool: Write",
		"Command: echo test.txt",
		"Message: Permission granted",
		"Behavior: allow",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got: %q", expected, output)
		}
	}
}
