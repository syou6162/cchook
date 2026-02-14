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
	rawJSON := map[string]interface{}{
		"tool_name": "Write",
		"tool_input": map[string]interface{}{
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

	rawJSON := map[string]interface{}{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"tool_name":       input.ToolName,
		"tool_input": map[string]interface{}{
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

	rawJSON := map[string]interface{}{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"tool_name":       input.ToolName,
		"tool_input": map[string]interface{}{
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
