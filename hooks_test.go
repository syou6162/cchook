package main

import (
	"bytes"
	"encoding/json"
	"errors"
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
		name                  string
		input                 *SessionStartInput
		expectedAddContext    string
		expectedHookEventName string
		shouldMatch           bool
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
			expectedAddContext:    "Session started: test-session-123",
			expectedHookEventName: "SessionStart",
			shouldMatch:           true,
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
			expectedAddContext:    "Session resumed",
			expectedHookEventName: "SessionStart",
			shouldMatch:           true,
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
			expectedAddContext:    "",
			expectedHookEventName: "",
			shouldMatch:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// rawJSON作成
			rawJSON := map[string]interface{}{
				"session_id":      tt.input.SessionID,
				"transcript_path": tt.input.TranscriptPath,
				"hook_event_name": string(tt.input.HookEventName),
				"source":          tt.input.Source,
			}

			// フック実行
			output, err := executeSessionStartHooks(config, tt.input, rawJSON)

			// エラーチェック
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// JSON出力チェック
			if tt.shouldMatch {
				if output.HookSpecificOutput == nil {
					t.Fatal("HookSpecificOutput is nil")
				}
				if output.HookSpecificOutput.AdditionalContext != tt.expectedAddContext {
					t.Errorf("Expected additionalContext '%s', got '%s'", tt.expectedAddContext, output.HookSpecificOutput.AdditionalContext)
				}
				if output.HookSpecificOutput.HookEventName != tt.expectedHookEventName {
					t.Errorf("Expected hookEventName '%s', got '%s'", tt.expectedHookEventName, output.HookSpecificOutput.HookEventName)
				}
			} else {
				if output.HookSpecificOutput != nil {
					t.Errorf("Expected no HookSpecificOutput, got %+v", output.HookSpecificOutput)
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
		name                   string
		input                  *SessionStartInput
		expectedAddContextPart []string
		notExpectedPart        string
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
			expectedAddContextPart: []string{"Go project detected", "Test file found recursively"},
			notExpectedPart:        "This should not appear",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// rawJSON作成
			rawJSON := map[string]interface{}{
				"session_id":      tt.input.SessionID,
				"transcript_path": tt.input.TranscriptPath,
				"hook_event_name": string(tt.input.HookEventName),
				"source":          tt.input.Source,
			}

			// フック実行
			output, err := executeSessionStartHooks(config, tt.input, rawJSON)

			// エラーチェック
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// JSON出力チェック
			if output.HookSpecificOutput == nil {
				t.Fatal("HookSpecificOutput is nil")
			}

			// AdditionalContextに期待される文字列が含まれているか確認
			addContext := output.HookSpecificOutput.AdditionalContext
			for _, expected := range tt.expectedAddContextPart {
				if !strings.Contains(addContext, expected) {
					t.Errorf("Expected additionalContext to contain '%s', got '%s'", expected, addContext)
				}
			}

			// "This should not appear"が含まれていないことを確認
			if strings.Contains(addContext, tt.notExpectedPart) {
				t.Errorf("additionalContext should not contain '%s', got '%s'", tt.notExpectedPart, addContext)
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

func TestExecutePreToolUseHooks_ConditionErrorAggregation(t *testing.T) {
	// 無効な条件タイプを含む設定（prompt_regex はPreToolUseでは使えない）
	config := &Config{
		PreToolUse: []PreToolUseHook{
			{
				Matcher: "Write",
				Conditions: []Condition{
					{Type: ConditionPromptRegex, Value: "test"}, // PreToolUseでは無効
				},
				Actions: []Action{
					{Type: "output", Message: "test"},
				},
			},
			{
				Matcher: "Write", // 2件目もWriteにマッチするように変更
				Conditions: []Condition{
					{Type: ConditionEveryNPrompts, Value: "5"}, // これもPreToolUseでは無効
				},
				Actions: []Action{
					{Type: "output", Message: "test"},
				},
			},
		},
	}

	input := &PreToolUseInput{
		BaseInput: BaseInput{Cwd: "/tmp"},
		ToolName:  "Write",
	}

	err := executePreToolUseHooks(config, input, nil)

	// エラーが返されることを確認
	if err == nil {
		t.Fatal("Expected error for invalid condition types, got nil")
	}

	// エラーメッセージに両方のフックのエラーが含まれることを確認（errors.Joinによる集約）
	errMsg := err.Error()
	if !strings.Contains(errMsg, "hook[PreToolUse][0]") {
		t.Errorf("Expected error message to contain first hook error, got: %q", errMsg)
	}
	// 2件目のフックのエラーも含まれることを確認（Editツールにマッチするフックもある）
	if !strings.Contains(errMsg, "hook[PreToolUse][1]") {
		t.Errorf("Expected error message to contain second hook error, got: %q", errMsg)
	}
	if !strings.Contains(errMsg, "unknown condition type") {
		t.Errorf("Expected error message to contain 'unknown condition type', got: %q", errMsg)
	}
}

func TestExecutePostToolUseHooks_ConditionErrorAggregation(t *testing.T) {
	// 複数の無効な条件タイプを含む設定
	config := &Config{
		PostToolUse: []PostToolUseHook{
			{
				Matcher: "Write",
				Conditions: []Condition{
					{Type: ConditionReasonIs, Value: "clear"}, // PostToolUseでは無効
				},
				Actions: []Action{
					{Type: "output", Message: "test"},
				},
			},
			{
				Matcher: "Write", // 2件目もWriteにマッチするように変更
				Conditions: []Condition{
					{Type: ConditionPromptRegex, Value: "test"}, // PostToolUseでは無効
				},
				Actions: []Action{
					{Type: "output", Message: "test"},
				},
			},
		},
	}

	input := &PostToolUseInput{
		BaseInput: BaseInput{Cwd: "/tmp"},
		ToolName:  "Write",
	}

	err := executePostToolUseHooks(config, input, nil)

	// エラーが返されることを確認
	if err == nil {
		t.Fatal("Expected error for invalid condition types, got nil")
	}

	// 複数のエラーが集約されていることを確認
	errMsg := err.Error()
	if !strings.Contains(errMsg, "hook[PostToolUse][0]") {
		t.Errorf("Expected error message to contain first hook error, got: %q", errMsg)
	}
	// 2件目のフックのエラーも含まれることを確認
	if !strings.Contains(errMsg, "hook[PostToolUse][1]") {
		t.Errorf("Expected error message to contain second hook error, got: %q", errMsg)
	}
	if !strings.Contains(errMsg, "unknown condition type") {
		t.Errorf("Expected error message to contain 'unknown condition type', got: %q", errMsg)
	}
}

func TestExecuteNotificationHooks_ConditionErrorAggregation(t *testing.T) {
	// Notificationでは使えない条件タイプを設定
	config := &Config{
		Notification: []NotificationHook{
			{
				Conditions: []Condition{
					{Type: ConditionFileExtension, Value: ".go"}, // Notificationでは無効
				},
				Actions: []Action{
					{Type: "output", Message: "test"},
				},
			},
		},
	}

	input := &NotificationInput{
		BaseInput: BaseInput{Cwd: "/tmp"},
		Message:   "test notification",
	}

	err := executeNotificationHooks(config, input, nil)

	// エラーが返されることを確認
	if err == nil {
		t.Fatal("Expected error for invalid condition type, got nil")
	}

	// エラーメッセージの確認
	errMsg := err.Error()
	if !strings.Contains(errMsg, "hook[Notification][0]") {
		t.Errorf("Expected error message to contain hook identifier, got: %q", errMsg)
	}
	if !strings.Contains(errMsg, "unknown condition type") {
		t.Errorf("Expected error message to contain 'unknown condition type', got: %q", errMsg)
	}
}

func TestExecutePreToolUseHooks_ConditionErrorAndExitError(t *testing.T) {
	// 条件エラーとアクション実行エラー（ExitError）が同時に発生するケース
	exitStatus := 10
	config := &Config{
		PreToolUse: []PreToolUseHook{
			{
				Matcher: "Write",
				Conditions: []Condition{
					{Type: ConditionPromptRegex, Value: "test"}, // PreToolUseでは無効
				},
				Actions: []Action{
					{Type: "output", Message: "test"},
				},
			},
			{
				Matcher: "Write",
				Actions: []Action{
					{Type: "output", Message: "will fail", ExitStatus: &exitStatus},
				},
			},
		},
	}

	input := &PreToolUseInput{
		BaseInput: BaseInput{Cwd: "/tmp"},
		ToolName:  "Write",
	}

	err := executePreToolUseHooks(config, input, nil)

	// エラーが返されることを確認
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// errors.Asを使ってExitErrorを取り出せることを確認
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatal("Expected ExitError to be extractable with errors.As, but it wasn't")
	}

	// ExitErrorの情報が保持されていることを確認
	if exitErr.Code != 10 {
		t.Errorf("Expected exit code 10, got %d", exitErr.Code)
	}
	if exitErr.Stderr != false {
		t.Errorf("Expected Stderr=false, got %v", exitErr.Stderr)
	}

	// エラーメッセージに条件エラーとアクションエラーの両方が含まれることを確認
	errMsg := err.Error()
	if !strings.Contains(errMsg, "hook[PreToolUse][0]") {
		t.Errorf("Expected error message to contain condition error, got: %q", errMsg)
	}
	if !strings.Contains(errMsg, "PreToolUse hook 1 failed") {
		t.Errorf("Expected error message to contain action error, got: %q", errMsg)
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

// TestExecuteSessionStartHooks_JSONOutput tests JSON output functionality
func TestExecuteSessionStartHooks_JSONOutput(t *testing.T) {
	tests := []struct {
		name                  string
		config                *Config
		input                 *SessionStartInput
		wantContinue          bool
		wantHookEventName     string
		wantAdditionalContext string
		wantSystemMessage     string
		wantHookSpecificNil   bool
	}{
		{
			name: "Single output action with continue: true",
			config: &Config{
				SessionStart: []SessionStartHook{
					{
						Actions: []Action{
							{
								Type:    "output",
								Message: "Welcome",
							},
						},
					},
				},
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{SessionID: "test"},
				Source:    "startup",
			},
			wantContinue:          true,
			wantHookEventName:     "SessionStart",
			wantAdditionalContext: "Welcome",
			wantSystemMessage:     "",
			wantHookSpecificNil:   false,
		},
		{
			name: "Command action with valid JSON",
			config: &Config{
				SessionStart: []SessionStartHook{
					{
						Actions: []Action{
							{
								Type:    "command",
								Command: `echo eyJjb250aW51ZSI6IHRydWUsICJob29rU3BlY2lmaWNPdXRwdXQiOiB7Imhvb2tFdmVudE5hbWUiOiAiU2Vzc2lvblN0YXJ0IiwgImFkZGl0aW9uYWxDb250ZXh0IjogIlRlc3QgY29udGV4dCJ9fQ== | base64 -d`,
							},
						},
					},
				},
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{SessionID: "test"},
				Source:    "startup",
			},
			wantContinue:          true,
			wantHookEventName:     "SessionStart",
			wantAdditionalContext: "Test context",
			wantSystemMessage:     "",
			wantHookSpecificNil:   false,
		},
		{
			name: "No actions - empty output",
			config: &Config{
				SessionStart: []SessionStartHook{
					{
						Actions: []Action{},
					},
				},
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{SessionID: "test"},
				Source:    "startup",
			},
			wantContinue:        true,
			wantHookSpecificNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawJSON := map[string]interface{}{
				"session_id": tt.input.SessionID,
				"source":     tt.input.Source,
			}

			output, err := executeSessionStartHooks(tt.config, tt.input, rawJSON)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if output == nil {
				t.Fatal("Output is nil")
			}

			if output.Continue != tt.wantContinue {
				t.Errorf("Continue: want %v, got %v", tt.wantContinue, output.Continue)
			}

			if tt.wantHookSpecificNil {
				if output.HookSpecificOutput != nil {
					t.Errorf("HookSpecificOutput: want nil, got %+v", output.HookSpecificOutput)
				}
			} else {
				if output.HookSpecificOutput == nil {
					t.Fatal("HookSpecificOutput is nil")
				}
				if output.HookSpecificOutput.HookEventName != tt.wantHookEventName {
					t.Errorf("HookEventName: want %s, got %s", tt.wantHookEventName, output.HookSpecificOutput.HookEventName)
				}
				if output.HookSpecificOutput.AdditionalContext != tt.wantAdditionalContext {
					t.Errorf("AdditionalContext: want %s, got %s", tt.wantAdditionalContext, output.HookSpecificOutput.AdditionalContext)
				}
			}

			if output.SystemMessage != tt.wantSystemMessage {
				t.Errorf("SystemMessage: want %s, got %s", tt.wantSystemMessage, output.SystemMessage)
			}
		})
	}
}

// TestExecuteSessionStartHooks_FieldMerging tests field merging logic
func TestExecuteSessionStartHooks_FieldMerging(t *testing.T) {
	falseVal := false
	config := &Config{
		SessionStart: []SessionStartHook{
			{
				Actions: []Action{
					{
						Type:    "output",
						Message: "First",
					},
					{
						Type:    "output",
						Message: "Second",
					},
					{
						Type:     "output",
						Message:  "Third with false",
						Continue: &falseVal,
					},
				},
			},
		},
	}

	input := &SessionStartInput{
		BaseInput: BaseInput{SessionID: "test"},
		Source:    "startup",
	}
	rawJSON := map[string]interface{}{
		"session_id": input.SessionID,
		"source":     input.Source,
	}

	output, err := executeSessionStartHooks(config, input, rawJSON)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Continue should be false (overwritten by third action)
	if output.Continue != false {
		t.Errorf("Continue: want false, got %v", output.Continue)
	}

	// HookEventName should be set once (by first action)
	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}
	if output.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Errorf("HookEventName: want SessionStart, got %s", output.HookSpecificOutput.HookEventName)
	}

	// AdditionalContext should be concatenated with "\n"
	want := "First\nSecond\nThird with false"
	if output.HookSpecificOutput.AdditionalContext != want {
		t.Errorf("AdditionalContext: want %s, got %s", want, output.HookSpecificOutput.AdditionalContext)
	}
}

// TestExecuteSessionStartHooks_EarlyReturn tests early return on continue: false
func TestExecuteSessionStartHooks_EarlyReturn(t *testing.T) {
	falseVal := false
	config := &Config{
		SessionStart: []SessionStartHook{
			{
				Actions: []Action{
					{
						Type:    "output",
						Message: "First",
					},
					{
						Type:     "output",
						Message:  "Second with false",
						Continue: &falseVal,
					},
					{
						Type:    "output",
						Message: "Third (should not execute)",
					},
				},
			},
		},
	}

	input := &SessionStartInput{
		BaseInput: BaseInput{SessionID: "test"},
		Source:    "startup",
	}
	rawJSON := map[string]interface{}{
		"session_id": input.SessionID,
		"source":     input.Source,
	}

	output, err := executeSessionStartHooks(config, input, rawJSON)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Continue should be false
	if output.Continue != false {
		t.Errorf("Continue: want false, got %v", output.Continue)
	}

	// AdditionalContext should NOT include "Third"
	want := "First\nSecond with false"
	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}
	if output.HookSpecificOutput.AdditionalContext != want {
		t.Errorf("AdditionalContext: want %s, got %s", want, output.HookSpecificOutput.AdditionalContext)
	}
}

// TestExecuteSessionStartHooks_SystemMessageMerging tests systemMessage concatenation
func TestExecuteSessionStartHooks_SystemMessageMerging(t *testing.T) {
	config := &Config{
		SessionStart: []SessionStartHook{
			{
				Actions: []Action{
					{
						Type:    "command",
						Command: `echo eyJjb250aW51ZSI6IHRydWUsICJob29rU3BlY2lmaWNPdXRwdXQiOiB7Imhvb2tFdmVudE5hbWUiOiAiU2Vzc2lvblN0YXJ0In0sICJzeXN0ZW1NZXNzYWdlIjogIkZpcnN0IHdhcm5pbmcifQ== | base64 -d`,
					},
					{
						Type:    "command",
						Command: `echo eyJjb250aW51ZSI6IHRydWUsICJob29rU3BlY2lmaWNPdXRwdXQiOiB7Imhvb2tFdmVudE5hbWUiOiAiU2Vzc2lvblN0YXJ0In0sICJzeXN0ZW1NZXNzYWdlIjogIlNlY29uZCB3YXJuaW5nIn0= | base64 -d`,
					},
				},
			},
		},
	}

	input := &SessionStartInput{
		BaseInput: BaseInput{SessionID: "test"},
		Source:    "startup",
	}
	rawJSON := map[string]interface{}{
		"session_id": input.SessionID,
		"source":     input.Source,
	}

	output, err := executeSessionStartHooks(config, input, rawJSON)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// SystemMessage should be concatenated with "\n"
	want := "First warning\nSecond warning"
	if output.SystemMessage != want {
		t.Errorf("SystemMessage: want %s, got %s", want, output.SystemMessage)
	}
}

// TestRunHooks_SessionStartJSONOutput tests runHooks JSON output to stdout
func TestRunHooks_SessionStartJSONOutput(t *testing.T) {
	// Create temporary config file
	tmpFile, err := os.CreateTemp("", "cchook-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `
SessionStart:
  - actions:
      - type: output
        message: "Test message"
`
	if _, err := tmpFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	config, err := loadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Prepare stdin with SessionStart JSON input
	input := `{"session_id":"test123","transcript_path":"/path/to/transcript","hook_event_name":"SessionStart","source":"startup"}`
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		w.Write([]byte(input))
		w.Close()
	}()
	defer func() { os.Stdin = oldStdin }()

	// Capture stdout
	oldStdout := os.Stdout
	stdoutR, stdoutW, _ := os.Pipe()
	os.Stdout = stdoutW

	// Execute runHooks
	err = runHooks(config, SessionStart)

	// Restore stdout and read captured output
	stdoutW.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, stdoutR)
	output := strings.TrimSpace(buf.String())

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Parse output as JSON
	var result SessionStartOutput
	if parseErr := json.Unmarshal([]byte(output), &result); parseErr != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", parseErr, output)
	}

	// Verify JSON structure
	if !result.Continue {
		t.Errorf("Continue: want true, got %v", result.Continue)
	}
	if result.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}
	if result.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Errorf("HookEventName: want SessionStart, got %s", result.HookSpecificOutput.HookEventName)
	}
	if result.HookSpecificOutput.AdditionalContext != "Test message" {
		t.Errorf("AdditionalContext: want 'Test message', got %s", result.HookSpecificOutput.AdditionalContext)
	}
}

// TestRunHooks_SessionStartMultipleActions tests multiple actions with JSON output
func TestRunHooks_SessionStartMultipleActions(t *testing.T) {
	// Create temporary config file
	tmpFile, err := os.CreateTemp("", "cchook-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `
SessionStart:
  - actions:
      - type: output
        message: "First"
      - type: output
        message: "Second"
      - type: command
        command: "echo eyJjb250aW51ZSI6IHRydWUsICJob29rU3BlY2lmaWNPdXRwdXQiOiB7Imhvb2tFdmVudE5hbWUiOiAiU2Vzc2lvblN0YXJ0In0sICJzeXN0ZW1NZXNzYWdlIjogIldhcm5pbmcifQ== | base64 -d"
`
	if _, err := tmpFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	config, err := loadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Prepare stdin
	input := `{"session_id":"test123","transcript_path":"/path/to/transcript","hook_event_name":"SessionStart","source":"startup"}`
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		w.Write([]byte(input))
		w.Close()
	}()
	defer func() { os.Stdin = oldStdin }()

	// Capture stdout
	oldStdout := os.Stdout
	stdoutR, stdoutW, _ := os.Pipe()
	os.Stdout = stdoutW

	// Execute runHooks
	err = runHooks(config, SessionStart)

	// Restore and read output
	stdoutW.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, stdoutR)
	output := strings.TrimSpace(buf.String())

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Parse JSON
	var result SessionStartOutput
	if parseErr := json.Unmarshal([]byte(output), &result); parseErr != nil {
		t.Fatalf("Failed to parse JSON: %v\nOutput: %s", parseErr, output)
	}

	// Verify merged fields
	if !result.Continue {
		t.Errorf("Continue: want true, got %v", result.Continue)
	}
	if result.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}

	// AdditionalContext should be concatenated
	wantContext := "First\nSecond"
	if result.HookSpecificOutput.AdditionalContext != wantContext {
		t.Errorf("AdditionalContext: want %s, got %s", wantContext, result.HookSpecificOutput.AdditionalContext)
	}

	// SystemMessage from command action
	if result.SystemMessage != "Warning" {
		t.Errorf("SystemMessage: want 'Warning', got %s", result.SystemMessage)
	}
}
