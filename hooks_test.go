package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestShouldExecutePreToolUseHook(t *testing.T) {
	tests := []struct {
		name  string
		hook  PreToolUseHook
		input *PreToolUseInput
		want  bool
	}{
		{
			"Match with no conditions",
			PreToolUseHook{Matcher: "Write"},
			&PreToolUseInput{ToolName: "Write"},
			true,
		},
		{
			"No match with matcher",
			PreToolUseHook{Matcher: "Edit"},
			&PreToolUseInput{ToolName: "Write"},
			false,
		},
		{
			"Match with satisfied condition",
			PreToolUseHook{
				Matcher: "Write",
				Conditions: []PreToolUseCondition{
					{Type: "file_extension", Value: ".go"},
				},
			},
			&PreToolUseInput{
				ToolName:  "Write",
				ToolInput: ToolInput{FilePath: "main.go"},
			},
			true,
		},
		{
			"Match but condition not satisfied",
			PreToolUseHook{
				Matcher: "Write",
				Conditions: []PreToolUseCondition{
					{Type: "file_extension", Value: ".py"},
				},
			},
			&PreToolUseInput{
				ToolName:  "Write",
				ToolInput: ToolInput{FilePath: "main.go"},
			},
			false,
		},
		{
			"Multiple conditions - all satisfied",
			PreToolUseHook{
				Matcher: "Write",
				Conditions: []PreToolUseCondition{
					{Type: "file_extension", Value: ".go"},
					{Type: "command_contains", Value: "test"},
				},
			},
			&PreToolUseInput{
				ToolName: "Write",
				ToolInput: ToolInput{
					FilePath: "main.go",
					Command:  "test command",
				},
			},
			true,
		},
		{
			"Multiple conditions - one not satisfied",
			PreToolUseHook{
				Matcher: "Write",
				Conditions: []PreToolUseCondition{
					{Type: "file_extension", Value: ".go"},
					{Type: "command_contains", Value: "build"},
				},
			},
			&PreToolUseInput{
				ToolName: "Write",
				ToolInput: ToolInput{
					FilePath: "main.go",
					Command:  "test command",
				},
			},
			false,
		},
		{
			"Empty matcher matches all",
			PreToolUseHook{Matcher: ""},
			&PreToolUseInput{ToolName: "AnyTool"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldExecutePreToolUseHook(tt.hook, tt.input); got != tt.want {
				t.Errorf("shouldExecutePreToolUseHook() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldExecutePostToolUseHook(t *testing.T) {
	tests := []struct {
		name  string
		hook  PostToolUseHook
		input *PostToolUseInput
		want  bool
	}{
		{
			"Match with condition",
			PostToolUseHook{
				Matcher: "Edit",
				Conditions: []PostToolUseCondition{
					{Type: "file_extension", Value: ".go"},
				},
			},
			&PostToolUseInput{
				ToolName:  "Edit",
				ToolInput: ToolInput{FilePath: "test.go"},
			},
			true,
		},
		{
			"No match",
			PostToolUseHook{Matcher: "Write"},
			&PostToolUseInput{ToolName: "Edit"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldExecutePostToolUseHook(tt.hook, tt.input); got != tt.want {
				t.Errorf("shouldExecutePostToolUseHook() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecutePreToolUseHook_OutputAction(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	hook := PreToolUseHook{
		Actions: []PreToolUseAction{
			{Type: "output", Message: "Test message"},
		},
	}
	
	input := &PreToolUseInput{ToolName: "Write"}
	
	err := executePreToolUseHook(hook, input)
	
	// 標準出力を復元
	w.Close()
	os.Stdout = oldStdout
	
	// 出力を読み取り
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	
	if err != nil {
		t.Errorf("executePreToolUseHook() error = %v", err)
	}
	
	if !strings.Contains(output, "Test message") {
		t.Errorf("Expected output to contain 'Test message', got: %q", output)
	}
}

func TestExecutePreToolUseHook_CommandAction(t *testing.T) {
	hook := PreToolUseHook{
		Actions: []PreToolUseAction{
			{Type: "command", Command: "echo test"},
		},
	}

	input := &PreToolUseInput{ToolName: "Write"}

	err := executePreToolUseHook(hook, input, nil)
	if err != nil {
		t.Errorf("executePreToolUseHook() error = %v", err)
	}
}

func TestExecutePreToolUseHook_CommandWithVariables(t *testing.T) {
	hook := PreToolUseHook{
		Actions: []PreToolUseAction{
			{Type: "command", Command: "echo {tool_input.file_path}"},
		},
	}
	
	input := &PreToolUseInput{
		ToolName:  "Write",
		ToolInput: ToolInput{FilePath: "test.go"},
	}
	
	err := executePreToolUseHook(hook, input)
	if err != nil {
		t.Errorf("executePreToolUseHook() error = %v", err)
	}
}

func TestExecutePreToolUseHook_FailingCommand(t *testing.T) {
	hook := PreToolUseHook{
		Actions: []PreToolUseAction{
			{Type: "command", Command: "false"}, // 常に失敗するコマンド
		},
	}

	input := &PreToolUseInput{ToolName: "Write"}

	err := executePreToolUseHook(hook, input, nil)
	if err == nil {
		t.Error("Expected error for failing command, got nil")
	}
}

func TestExecutePostToolUseHook_Success(t *testing.T) {
	hook := PostToolUseHook{
		Actions: []PostToolUseAction{
			{Type: "output", Message: "Post-processing complete"},
		},
	}
	
	input := &PostToolUseInput{ToolName: "Edit"}
	
	err := executePostToolUseHook(hook, input)
	if err != nil {
		t.Errorf("executePostToolUseHook() error = %v", err)
	}
}

func TestExecutePreToolUseHooks_Integration(t *testing.T) {
	// 標準エラーをキャプチャして、フック実行エラーをテスト
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	config := &Config{
		PreToolUse: []PreToolUseHook{
			{
				Matcher: "Write",
				Actions: []PreToolUseAction{
					{Type: "command", Command: "false"}, // 失敗するコマンド
				},
			},
		},
	}
	
	input := &PreToolUseInput{ToolName: "Write"}
	
	err := executePreToolUseHooks(config, input)
	
	// 標準エラーを復元
	w.Close()
	os.Stderr = oldStderr
	
	// エラー出力を読み取り
	var buf bytes.Buffer
	buf.ReadFrom(r)
	stderrOutput := buf.String()
	
	// executePreToolUseHooksはエラーを返さず、stdErr に出力するだけ
	if err != nil {
		t.Errorf("executePreToolUseHooks() error = %v", err)
	}
	
	if !strings.Contains(stderrOutput, "PreToolUse hook 0 failed") {
		t.Errorf("Expected stderr to contain hook failure message, got: %q", stderrOutput)
	}
}

func TestExecutePostToolUseHooks_Integration(t *testing.T) {
	config := &Config{
		PostToolUse: []PostToolUseHook{
			{
				Matcher: "Edit",
				Actions: []PostToolUseAction{
					{Type: "output", Message: "File processed"},
				},
			},
		},
	}

	input := &PostToolUseInput{ToolName: "Edit"}

	err := executePostToolUseHooks(config, input, nil)
	if err != nil {
		t.Errorf("executePostToolUseHooks() error = %v", err)
	}
}

func TestExecuteNotificationHooks(t *testing.T) {
	config := &Config{}
	input := &NotificationInput{Message: "test"}

	err := executeNotificationHooks(config, input, nil)
	if err != nil {
		t.Errorf("executeNotificationHooks() error = %v, expected nil", err)
	}
}

func TestExecuteStopHooks(t *testing.T) {
	config := &Config{}
	input := &StopInput{}
	
	err := executeStopHooks(config, input)
	if err != nil {
		t.Errorf("executeStopHooks() error = %v, expected nil", err)
	}
}

func TestExecuteSubagentStopHooks(t *testing.T) {
	config := &Config{}
	input := &SubagentStopInput{}

	err := executeSubagentStopHooks(config, input, nil)
	if err != nil {
		t.Errorf("executeSubagentStopHooks() error = %v, expected nil", err)
	}
}

func TestExecutePreCompactHooks(t *testing.T) {
	config := &Config{}
	input := &PreCompactInput{}
	
	err := executePreCompactHooks(config, input)
	if err != nil {
		t.Errorf("executePreCompactHooks() error = %v, expected nil", err)
	}
}

func TestDryRunUnimplementedHooks(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	err := dryRunUnimplementedHooks(Notification)
	
	// 標準出力を復元
	w.Close()
	os.Stdout = oldStdout
	
	// 出力を読み取り
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	
	if err != nil {
		t.Errorf("dryRunUnimplementedHooks() error = %v", err)
	}
	
	if !strings.Contains(output, "=== Notification Hooks (Dry Run) ===") {
		t.Errorf("Expected dry run header, got: %q", output)
	}
	
	if !strings.Contains(output, "No hooks implemented yet") {
		t.Errorf("Expected 'No hooks implemented yet' message, got: %q", output)
	}
}

func TestDryRunPreToolUseHooks_NoMatch(t *testing.T) {
	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := &Config{
		PreToolUse: []PreToolUseHook{
			{Matcher: "Edit", Actions: []PreToolUseAction{{Type: "output", Message: "test"}}},
		},
	}

	input := &PreToolUseInput{ToolName: "Write"} // マッチしない

	err := dryRunPreToolUseHooks(config, input, nil)

	// 標準出力を復元
	w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	buf.ReadFrom(r)
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
				Actions: []PreToolUseAction{
					{Type: "command", Command: "echo {tool_input.file_path}"},
					{Type: "output", Message: "Processing..."},
				},
			},
		},
	}
	
	input := &PreToolUseInput{
		ToolName:  "Write",
		ToolInput: ToolInput{FilePath: "test.go"},
	}
	
	err := dryRunPreToolUseHooks(config, input)
	
	// 標準出力を復元
	w.Close()
	os.Stdout = oldStdout
	
	// 出力を読み取り
	var buf bytes.Buffer
	buf.ReadFrom(r)
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