package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestShouldExecutePreToolUseHook(t *testing.T) {
	tests := []struct {
		name    string
		hook    PreToolUseHook
		input   *PreToolUseInput
		want    bool
		wantErr bool
	}{
		{
			"Match with no conditions",
			PreToolUseHook{Matcher: "Write"},
			&PreToolUseInput{ToolName: "Write"},
			true,
			false,
		},
		{
			"No match with matcher",
			PreToolUseHook{Matcher: "Edit"},
			&PreToolUseInput{ToolName: "Write"},
			false,
			false,
		},
		{
			"Match with satisfied condition",
			PreToolUseHook{
				Matcher: "Write",
				Conditions: []Condition{
					{Type: ConditionFileExtension, Value: ".go"},
				},
			},
			&PreToolUseInput{
				ToolName:  "Write",
				ToolInput: ToolInput{FilePath: "main.go"},
			},
			true,
			false,
		},
		{
			"Match but condition not satisfied",
			PreToolUseHook{
				Matcher: "Write",
				Conditions: []Condition{
					{Type: ConditionFileExtension, Value: ".py"},
				},
			},
			&PreToolUseInput{
				ToolName:  "Write",
				ToolInput: ToolInput{FilePath: "main.go"},
			},
			false,
			false,
		},
		{
			"Multiple conditions - all satisfied",
			PreToolUseHook{
				Matcher: "Write",
				Conditions: []Condition{
					{Type: ConditionFileExtension, Value: ".go"},
					{Type: ConditionCommandContains, Value: "test"},
				},
			},
			&PreToolUseInput{
				ToolName: "Write",
				ToolInput: ToolInput{
					FilePath: "test.go",
					Command:  "go test",
				},
			},
			true,
			false,
		},
		{
			"Multiple conditions - one not satisfied",
			PreToolUseHook{
				Matcher: "Write",
				Conditions: []Condition{
					{Type: ConditionFileExtension, Value: ".go"},
					{Type: ConditionCommandContains, Value: "build"},
				},
			},
			&PreToolUseInput{
				ToolName: "Write",
				ToolInput: ToolInput{
					FilePath: "test.go",
					Command:  "go test",
				},
			},
			false,
			false,
		},
		{
			"Empty matcher matches all",
			PreToolUseHook{
				Matcher: "",
				Conditions: []Condition{
					{Type: ConditionFileExtension, Value: ".go"},
				},
			},
			&PreToolUseInput{
				ToolName:  "AnyTool",
				ToolInput: ToolInput{FilePath: "main.go"},
			},
			true,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := shouldExecutePreToolUseHook(tt.hook, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("shouldExecutePreToolUseHook() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("shouldExecutePreToolUseHook() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecuteSessionStartHooks(t *testing.T) {
	config := &Config{
		SessionStart: []SessionStartHook{
			{
				Matcher: "startup",
				Actions: []Action{
					{
						Type:    "output",
						Message: "Session started: {.session_id}",
					},
				},
			},
			{
				Matcher: "resume",
				Actions: []Action{
					{
						Type:    "output",
						Message: "Session resumed",
					},
				},
			},
		},
	}

	tests := []struct {
		name           string
		input          *SessionStartInput
		expectedOutput string
		shouldMatch    bool
	}{
		{
			name: "Startup matcher matches",
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-session-123",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  SessionStart,
				},
				Source: "startup",
			},
			expectedOutput: "Session started: test-session-123",
			shouldMatch:    true,
		},
		{
			name: "Resume matcher matches",
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-session-456",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  SessionStart,
				},
				Source: "resume",
			},
			expectedOutput: "Session resumed",
			shouldMatch:    true,
		},
		{
			name: "Clear source doesn't match",
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-session-789",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  SessionStart,
				},
				Source: "clear",
			},
			expectedOutput: "",
			shouldMatch:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// キャプチャ用バッファ
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// rawJSON作成
			rawJSON := map[string]interface{}{
				"session_id":      tt.input.SessionID,
				"transcript_path": tt.input.TranscriptPath,
				"hook_event_name": string(tt.input.HookEventName),
				"source":          tt.input.Source,
			}

			// フック実行
			err := executeSessionStartHooks(config, tt.input, rawJSON)

			// 出力キャプチャ
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := strings.TrimSpace(buf.String())

			// エラーチェック
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// 出力チェック
			if tt.shouldMatch {
				if output != tt.expectedOutput {
					t.Errorf("Expected output '%s', got '%s'", tt.expectedOutput, output)
				}
			} else {
				if output != "" {
					t.Errorf("Expected no output, got '%s'", output)
				}
			}
		})
	}
}

func TestSessionStartHooksWithConditions(t *testing.T) {
	// go.modは既に存在するはず
	config := &Config{
		SessionStart: []SessionStartHook{
			{
				Matcher: "startup",
				Conditions: []Condition{
					{Type: ConditionFileExists, Value: "go.mod"},
				},
				Actions: []Action{
					{
						Type:    "output",
						Message: "Go project detected",
					},
				},
			},
			{
				Matcher: "startup",
				Conditions: []Condition{
					{Type: ConditionFileExists, Value: "nonexistent.file"},
				},
				Actions: []Action{
					{
						Type:    "output",
						Message: "This should not appear",
					},
				},
			},
			{
				Matcher: "startup",
				Conditions: []Condition{
					{Type: ConditionFileExistsRecursive, Value: "hooks_test.go"},
				},
				Actions: []Action{
					{
						Type:    "output",
						Message: "Test file found recursively",
					},
				},
			},
		},
	}

	tests := []struct {
		name           string
		input          *SessionStartInput
		expectedOutput []string
	}{
		{
			name: "Conditions check - file_exists and file_exists_recursive",
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  SessionStart,
				},
				Source: "startup",
			},
			expectedOutput: []string{"Go project detected", "Test file found recursively"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// キャプチャ用バッファ
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// rawJSON作成
			rawJSON := map[string]interface{}{
				"session_id":      tt.input.SessionID,
				"transcript_path": tt.input.TranscriptPath,
				"hook_event_name": string(tt.input.HookEventName),
				"source":          tt.input.Source,
			}

			// フック実行
			err := executeSessionStartHooks(config, tt.input, rawJSON)

			// 出力キャプチャ
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := strings.TrimSpace(buf.String())

			// エラーチェック
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// 出力チェック
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', got '%s'", expected, output)
				}
			}

			// "This should not appear"が出力されていないことを確認
			if strings.Contains(output, "This should not appear") {
				t.Errorf("Output should not contain 'This should not appear', got '%s'", output)
			}
		})
	}
}

func TestExecuteUserPromptSubmitHooks(t *testing.T) {
	config := &Config{
		UserPromptSubmit: []UserPromptSubmitHook{
			{
				Conditions: []Condition{
					{Type: ConditionPromptRegex, Value: "block"},
				},
				Actions: []Action{
					{
						Type:       "output",
						Message:    "Blocked prompt",
						ExitStatus: intPtr(2),
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		input       *UserPromptSubmitInput
		shouldError bool
	}{
		{
			name: "Blocked prompt",
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "This contains block keyword",
			},
			shouldError: true,
		},
		{
			name: "Allowed prompt",
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test456",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "This is allowed",
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawJSON := map[string]interface{}{
				"session_id":      tt.input.SessionID,
				"hook_event_name": string(tt.input.HookEventName),
				"prompt":          tt.input.Prompt,
			}

			err := executeUserPromptSubmitHooks(config, tt.input, rawJSON)

			if tt.shouldError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if tt.shouldError && err != nil {
				exitErr, ok := err.(*ExitError)
				if !ok {
					t.Errorf("Expected ExitError, got %T", err)
				} else if exitErr.Code != 2 {
					t.Errorf("Expected exit code 2, got %d", exitErr.Code)
				}
			}
		})
	}
}

func TestShouldExecutePostToolUseHook(t *testing.T) {
	tests := []struct {
		name    string
		hook    PostToolUseHook
		input   *PostToolUseInput
		want    bool
		wantErr bool
	}{
		{
			"Match with condition",
			PostToolUseHook{
				Matcher: "Edit",
				Conditions: []Condition{
					{Type: ConditionFileExtension, Value: ".go"},
				},
			},
			&PostToolUseInput{
				ToolName:  "Edit",
				ToolInput: ToolInput{FilePath: "test.go"},
			},
			true,
			false,
		},
		{
			"No match",
			PostToolUseHook{Matcher: "Write"},
			&PostToolUseInput{ToolName: "Edit"},
			false,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := shouldExecutePostToolUseHook(tt.hook, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("shouldExecutePostToolUseHook() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
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

	exitStatus := 0
	hook := PreToolUseHook{
		Actions: []Action{
			{Type: "output", Message: "Test message", ExitStatus: &exitStatus}},
	}

	input := &PreToolUseInput{ToolName: "Write"}

	err := executePreToolUseHook(hook, input, nil)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
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
		Actions: []Action{
			{Type: "command", Command: "echo test"}},
	}

	input := &PreToolUseInput{ToolName: "Write"}

	err := executePreToolUseHook(hook, input, nil)
	if err != nil {
		t.Errorf("executePreToolUseHook() error = %v", err)
	}
}

func TestExecutePreToolUseHook_CommandWithVariables(t *testing.T) {
	hook := PreToolUseHook{
		Actions: []Action{
			{Type: "command", Command: "echo {.tool_input.file_path}"}},
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

	err := executePreToolUseHook(hook, input, rawJSON)
	if err != nil {
		t.Errorf("executePreToolUseHook() error = %v", err)
	}
}

func TestExecutePreToolUseHook_FailingCommand(t *testing.T) {
	hook := PreToolUseHook{
		Actions: []Action{
			{Type: "command", Command: "false"}, // 常に失敗するコマンド
		},
	}

	input := &PreToolUseInput{ToolName: "Write"}

	err := executePreToolUseHook(hook, input, nil)
	if err == nil {
		t.Error("Expected error for failing command, got nil")
	}
}

func TestExecutePreToolUseHook_FailingCommandReturnsExit2(t *testing.T) {
	hook := PreToolUseHook{
		Actions: []Action{
			{Type: "command", Command: "false"}, // 失敗するコマンド
		},
	}

	input := &PreToolUseInput{ToolName: "Write"}

	err := executePreToolUseHook(hook, input, nil)

	// エラーが返されることを確認
	if err == nil {
		t.Fatal("Expected error for failing command, got nil")
	}

	// ExitError型であることを確認
	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("Expected ExitError, got %T", err)
	}

	// exit code 2であることを確認
	if exitErr.Code != 2 {
		t.Errorf("Expected exit code 2, got %d", exitErr.Code)
	}

	// stderrに出力されることを確認
	if !exitErr.Stderr {
		t.Error("Expected stderr to be true")
	}

	// エラーメッセージにCommand failedが含まれることを確認
	if !strings.Contains(exitErr.Message, "Command failed") {
		t.Errorf("Expected message to contain 'Command failed', got: %s", exitErr.Message)
	}
}

func TestExecuteUserPromptSubmitHook_FailingCommandReturnsExit2(t *testing.T) {
	config := &Config{
		UserPromptSubmit: []UserPromptSubmitHook{
			{
				Actions: []Action{
					{Type: "command", Command: "false"}, // 失敗するコマンド
				},
			},
		},
	}

	input := &UserPromptSubmitInput{
		BaseInput: BaseInput{
			SessionID:     "test123",
			HookEventName: UserPromptSubmit,
		},
		Prompt: "test prompt",
	}

	err := executeUserPromptSubmitHooks(config, input, nil)

	// ExitError型でexit code 2であることを確認
	if err == nil {
		t.Fatal("Expected error for failing command, got nil")
	}

	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("Expected ExitError, got %T", err)
	}

	if exitErr.Code != 2 {
		t.Errorf("Expected exit code 2, got %d", exitErr.Code)
	}
}

func TestExecuteStopHook_FailingCommandReturnsExit2(t *testing.T) {
	config := &Config{
		Stop: []StopHook{
			{
				Actions: []Action{
					{Type: "command", Command: "false"}, // 失敗するコマンド
				},
			},
		},
	}

	input := &StopInput{
		BaseInput: BaseInput{
			SessionID:     "test123",
			HookEventName: Stop,
		},
	}

	err := executeStopHooks(config, input, nil)

	// ExitError型でexit code 2であることを確認
	if err == nil {
		t.Fatal("Expected error for failing command, got nil")
	}

	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("Expected ExitError, got %T", err)
	}

	if exitErr.Code != 2 {
		t.Errorf("Expected exit code 2, got %d", exitErr.Code)
	}
}

func TestExecuteSubagentStopHook_FailingCommandReturnsExit2(t *testing.T) {
	config := &Config{
		SubagentStop: []SubagentStopHook{
			{
				Actions: []Action{
					{Type: "command", Command: "false"}, // 失敗するコマンド
				},
			},
		},
	}

	input := &SubagentStopInput{
		BaseInput: BaseInput{
			SessionID:     "test123",
			HookEventName: SubagentStop,
		},
	}

	err := executeSubagentStopHooks(config, input, nil)

	// ExitError型でexit code 2であることを確認
	if err == nil {
		t.Fatal("Expected error for failing command, got nil")
	}

	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("Expected ExitError, got %T", err)
	}

	if exitErr.Code != 2 {
		t.Errorf("Expected exit code 2, got %d", exitErr.Code)
	}
}

func TestExecutePostToolUseHook_Success(t *testing.T) {
	exitStatus := 0
	hook := PostToolUseHook{
		Actions: []Action{
			{Type: "output", Message: "Post-processing complete", ExitStatus: &exitStatus}},
	}

	input := &PostToolUseInput{ToolName: "Edit"}

	err := executePostToolUseHook(hook, input, nil)
	if err != nil {
		t.Errorf("executePostToolUseHook() error = %v", err)
	}
}

func TestExecutePreToolUseHooks_Integration(t *testing.T) {
	config := &Config{
		PreToolUse: []PreToolUseHook{
			{
				Matcher: "Write",
				Actions: []Action{
					{Type: "command", Command: "false"}, // 失敗するコマンド
				},
			},
		},
	}

	input := &PreToolUseInput{ToolName: "Write"}

	err := executePreToolUseHooks(config, input, nil)

	// executePreToolUseHooksはフック失敗時にエラーを返す
	if err == nil {
		t.Error("Expected executePreToolUseHooks to return error for failing command")
	}

	// エラーメッセージに"PreToolUse hook 0 failed"が含まれることを確認
	if !strings.Contains(err.Error(), "PreToolUse hook 0 failed") {
		t.Errorf("Expected error message to contain hook failure message, got: %q", err.Error())
	}
}

func TestExecutePostToolUseHooks_Integration(t *testing.T) {
	config := &Config{
		PostToolUse: []PostToolUseHook{
			{
				Matcher: "Edit",
				Actions: []Action{
					{Type: "output", Message: "File processed", ExitStatus: &[]int{0}[0]},
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

	err := executeStopHooks(config, input, nil)
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

	err := executePreCompactHooks(config, input, nil)
	if err != nil {
		t.Errorf("executePreCompactHooks() error = %v, expected nil", err)
	}
}

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

func TestExecuteSessionEndHooks(t *testing.T) {
	// Create temporary test file for file_exists condition
	tmpFile, err := os.CreateTemp("", "sessionend_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Config for tests that expect hooks to match
	config := &Config{
		SessionEnd: []SessionEndHook{
			{
				Conditions: []Condition{
					{
						Type:  ConditionReasonIs,
						Value: "clear",
					},
				},
				Actions: []Action{
					{
						Type:    "output",
						Message: "Session cleared: {.session_id}",
					},
				},
			},
			{
				Conditions: []Condition{
					{
						Type:  ConditionReasonIs,
						Value: "logout",
					},
				},
				Actions: []Action{
					{
						Type:    "output",
						Message: "Logged out from session",
					},
				},
			},
			{
				Conditions: []Condition{
					{
						Type:  ConditionFileExists,
						Value: tmpFile.Name(),
					},
				},
				Actions: []Action{
					{
						Type:    "output",
						Message: "Cleanup: temp file exists",
					},
				},
			},
		},
	}

	// Config for tests that expect no hooks to match
	noMatchConfig := &Config{
		SessionEnd: []SessionEndHook{
			{
				Conditions: []Condition{
					{
						Type:  ConditionReasonIs,
						Value: "clear",
					},
				},
				Actions: []Action{
					{
						Type:    "output",
						Message: "This should not be printed",
					},
				},
			},
		},
	}

	tests := []struct {
		name           string
		config         *Config
		input          *SessionEndInput
		expectedOutput string
		shouldMatch    bool
	}{
		{
			name:   "Reason clear matches",
			config: config,
			input: &SessionEndInput{
				BaseInput: BaseInput{
					SessionID:      "test-session-123",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  SessionEnd,
				},
				Reason: "clear",
			},
			expectedOutput: "Session cleared: test-session-123",
			shouldMatch:    true,
		},
		{
			name:   "Reason logout matches",
			config: config,
			input: &SessionEndInput{
				BaseInput: BaseInput{
					SessionID:      "test-session-456",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  SessionEnd,
				},
				Reason: "logout",
			},
			expectedOutput: "Logged out from session",
			shouldMatch:    true,
		},
		{
			name:   "File exists condition matches",
			config: config,
			input: &SessionEndInput{
				BaseInput: BaseInput{
					SessionID:      "test-session-789",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  SessionEnd,
				},
				Reason: "other",
			},
			expectedOutput: "Cleanup: temp file exists",
			shouldMatch:    true,
		},
		{
			name:   "Reason prompt_input_exit doesn't match",
			config: config,
			input: &SessionEndInput{
				BaseInput: BaseInput{
					SessionID:      "test-session-999",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  SessionEnd,
				},
				Reason: "prompt_input_exit",
			},
			expectedOutput: "Cleanup: temp file exists",
			shouldMatch:    true,
		},
		{
			name:   "No hooks match - reason mismatch",
			config: noMatchConfig,
			input: &SessionEndInput{
				BaseInput: BaseInput{
					SessionID:      "test-session-000",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  SessionEnd,
				},
				Reason: "unknown_reason",
			},
			expectedOutput: "",
			shouldMatch:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Create rawJSON
			rawJSON := map[string]interface{}{
				"session_id":      tt.input.SessionID,
				"transcript_path": tt.input.TranscriptPath,
				"hook_event_name": string(tt.input.HookEventName),
				"reason":          tt.input.Reason,
			}

			// Execute hooks
			err := executeSessionEndHooks(tt.config, tt.input, rawJSON)

			// Restore stdout and capture output
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := strings.TrimSpace(buf.String())

			// Check error
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check output
			if tt.shouldMatch {
				if !strings.Contains(output, tt.expectedOutput) {
					t.Errorf("Expected output to contain %q, got %q", tt.expectedOutput, output)
				}
			} else {
				if output != "" {
					t.Errorf("Expected no output, got %q", output)
				}
			}
		})
	}
}

func TestExecuteSessionEndHooks_CommandAction(t *testing.T) {
	// Create temporary test file for verification
	tmpFile, err := os.CreateTemp("", "sessionend_cmd_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	config := &Config{
		SessionEnd: []SessionEndHook{
			{
				Conditions: []Condition{
					{
						Type:  ConditionReasonIs,
						Value: "clear",
					},
				},
				Actions: []Action{
					{
						Type:    "command",
						Command: "echo 'Session cleared' > " + tmpFile.Name(),
					},
				},
			},
		},
	}

	input := &SessionEndInput{
		BaseInput: BaseInput{
			SessionID:      "test-cmd-session",
			TranscriptPath: "/path/to/transcript",
			HookEventName:  SessionEnd,
		},
		Reason: "clear",
	}

	rawJSON := map[string]interface{}{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"reason":          input.Reason,
	}

	// Execute hooks
	err = executeSessionEndHooks(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify command was executed
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	expected := "Session cleared\n"
	if string(content) != expected {
		t.Errorf("Expected file content %q, got %q", expected, string(content))
	}
}
