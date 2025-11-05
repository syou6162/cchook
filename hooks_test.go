package main

import (
	"bytes"
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
		wantAdditionalContext string
		wantContinue          bool
		wantHookEventName     string
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
			wantAdditionalContext: "Session started: test-session-123",
			wantContinue:          true,
			wantHookEventName:     "SessionStart",
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
			wantAdditionalContext: "Session resumed",
			wantContinue:          true,
			wantHookEventName:     "SessionStart",
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
			wantAdditionalContext: "",
			wantContinue:          true,
			wantHookEventName:     "",
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

			// Continue チェック
			if output.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
			}

			// HookEventName チェック
			if tt.wantHookEventName != "" {
				if output.HookSpecificOutput == nil {
					t.Errorf("HookSpecificOutput is nil but expected hookEventName")
				} else if output.HookSpecificOutput.HookEventName != tt.wantHookEventName {
					t.Errorf("HookEventName = %v, want %v", output.HookSpecificOutput.HookEventName, tt.wantHookEventName)
				}
			}

			// AdditionalContext チェック
			if tt.wantAdditionalContext != "" {
				if output.HookSpecificOutput == nil {
					t.Errorf("HookSpecificOutput is nil but expected additionalContext")
				} else if output.HookSpecificOutput.AdditionalContext != tt.wantAdditionalContext {
					t.Errorf("AdditionalContext = %v, want %v", output.HookSpecificOutput.AdditionalContext, tt.wantAdditionalContext)
				}
			} else {
				if output.HookSpecificOutput != nil && output.HookSpecificOutput.AdditionalContext != "" {
					t.Errorf("AdditionalContext = %v, want empty", output.HookSpecificOutput.AdditionalContext)
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
		name                       string
		input                      *SessionStartInput
		wantAdditionalContexts     []string
		wantNotInAdditionalContext string
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
			wantAdditionalContexts:     []string{"Go project detected", "Test file found recursively"},
			wantNotInAdditionalContext: "This should not appear",
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

			// AdditionalContext チェック
			if output.HookSpecificOutput == nil {
				t.Fatalf("HookSpecificOutput is nil")
			}

			additionalContext := output.HookSpecificOutput.AdditionalContext
			for _, expected := range tt.wantAdditionalContexts {
				if !strings.Contains(additionalContext, expected) {
					t.Errorf("Expected additionalContext to contain '%s', got '%s'", expected, additionalContext)
				}
			}

			// "This should not appear"が含まれていないことを確認
			if strings.Contains(additionalContext, tt.wantNotInAdditionalContext) {
				t.Errorf("AdditionalContext should not contain '%s', got '%s'", tt.wantNotInAdditionalContext, additionalContext)
			}
		})
	}
}

func TestExecuteUserPromptSubmitHooks(t *testing.T) {
	tests := []struct {
		name              string
		config            *Config
		input             *UserPromptSubmitInput
		rawJSON           interface{}
		wantContinue      bool
		wantDecision      string
		wantHookEventName string
		wantAdditionalCtx string
		wantSystemMessage string
		wantErr           bool
	}{
		{
			name: "Single type: output action",
			config: &Config{
				UserPromptSubmit: []UserPromptSubmitHook{
					{
						Actions: []Action{
							{
								Type:    "output",
								Message: "Output message",
							},
						},
					},
				},
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					HookEventName:  UserPromptSubmit,
					TranscriptPath: "/path/to/transcript",
					Cwd:            "/test/cwd",
				},
				Prompt: "test prompt",
			},
			rawJSON: map[string]interface{}{
				"session_id":      "test-session",
				"hook_event_name": "UserPromptSubmit",
				"prompt":          "test prompt",
			},
			wantContinue:      true,
			wantDecision:      "approve",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "Output message",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Multiple actions - additionalContext concatenated",
			config: &Config{
				UserPromptSubmit: []UserPromptSubmitHook{
					{
						Actions: []Action{
							{
								Type:    "output",
								Message: "First message",
							},
							{
								Type:    "output",
								Message: "Second message",
							},
						},
					},
				},
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test-session",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "test prompt",
			},
			rawJSON: map[string]interface{}{
				"session_id":      "test-session",
				"hook_event_name": "UserPromptSubmit",
				"prompt":          "test prompt",
			},
			wantContinue:      true,
			wantDecision:      "approve",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "First message\nSecond message",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "First action decision: block - early return",
			config: &Config{
				UserPromptSubmit: []UserPromptSubmitHook{
					{
						Actions: []Action{
							{
								Type:     "output",
								Message:  "Blocked",
								Decision: stringPtr("block"),
							},
							{
								Type:    "output",
								Message: "Should not execute",
							},
						},
					},
				},
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test-session",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "test prompt",
			},
			rawJSON: map[string]interface{}{
				"session_id":      "test-session",
				"hook_event_name": "UserPromptSubmit",
				"prompt":          "test prompt",
			},
			wantContinue:      true,
			wantDecision:      "block",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "Blocked",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Second action decision: block - first results preserved",
			config: &Config{
				UserPromptSubmit: []UserPromptSubmitHook{
					{
						Actions: []Action{
							{
								Type:    "output",
								Message: "First message",
							},
							{
								Type:     "output",
								Message:  "Blocked",
								Decision: stringPtr("block"),
							},
						},
					},
				},
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test-session",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "test prompt",
			},
			rawJSON: map[string]interface{}{
				"session_id":      "test-session",
				"hook_event_name": "UserPromptSubmit",
				"prompt":          "test prompt",
			},
			wantContinue:      true,
			wantDecision:      "block",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "First message\nBlocked",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Continue always true",
			config: &Config{
				UserPromptSubmit: []UserPromptSubmitHook{
					{
						Actions: []Action{
							{
								Type:    "output",
								Message: "Test message",
							},
						},
					},
				},
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test-session",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "test prompt",
			},
			rawJSON: map[string]interface{}{
				"session_id":      "test-session",
				"hook_event_name": "UserPromptSubmit",
				"prompt":          "test prompt",
			},
			wantContinue:      true,
			wantDecision:      "approve",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "Test message",
			wantSystemMessage: "",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executeUserPromptSubmitHooks(tt.config, tt.input, tt.rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("executeUserPromptSubmitHooks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got == nil {
				t.Fatal("executeUserPromptSubmitHooks() returned nil output")
			}

			if got.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", got.Continue, tt.wantContinue)
			}

			if got.Decision != tt.wantDecision {
				t.Errorf("Decision = %v, want %v", got.Decision, tt.wantDecision)
			}

			if got.HookSpecificOutput == nil {
				t.Fatal("HookSpecificOutput is nil")
			}

			if got.HookSpecificOutput.HookEventName != tt.wantHookEventName {
				t.Errorf("HookEventName = %v, want %v", got.HookSpecificOutput.HookEventName, tt.wantHookEventName)
			}

			if got.HookSpecificOutput.AdditionalContext != tt.wantAdditionalCtx {
				t.Errorf("AdditionalContext = %v, want %v", got.HookSpecificOutput.AdditionalContext, tt.wantAdditionalCtx)
			}

			if got.SystemMessage != tt.wantSystemMessage {
				t.Errorf("SystemMessage = %v, want %v", got.SystemMessage, tt.wantSystemMessage)
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

func SkipTestExecutePreToolUseHook_OutputAction(t *testing.T) {
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

	executor := NewActionExecutor(nil)
	err := executePreToolUseHook(executor, hook, input, nil)

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

func SkipTestExecutePreToolUseHook_CommandAction(t *testing.T) {
	hook := PreToolUseHook{
		Actions: []Action{
			{Type: "command", Command: "echo test"}},
	}

	input := &PreToolUseInput{ToolName: "Write"}

	executor := NewActionExecutor(nil)
	err := executePreToolUseHook(executor, hook, input, nil)
	if err != nil {
		t.Errorf("executePreToolUseHook() error = %v", err)
	}
}

func SkipTestExecutePreToolUseHook_CommandWithVariables(t *testing.T) {
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

	executor := NewActionExecutor(nil)
	err := executePreToolUseHook(executor, hook, input, rawJSON)
	if err != nil {
		t.Errorf("executePreToolUseHook() error = %v", err)
	}
}

func SkipTestExecutePreToolUseHook_FailingCommand(t *testing.T) {
	hook := PreToolUseHook{
		Actions: []Action{
			{Type: "command", Command: "false"}, // 常に失敗するコマンド
		},
	}

	input := &PreToolUseInput{ToolName: "Write"}

	executor := NewActionExecutor(nil)
	err := executePreToolUseHook(executor, hook, input, nil)
	if err == nil {
		t.Error("Expected error for failing command, got nil")
	}
}

func SkipTestExecutePreToolUseHook_FailingCommandReturnsExit2(t *testing.T) {
	hook := PreToolUseHook{
		Actions: []Action{
			{Type: "command", Command: "false"}, // 失敗するコマンド
		},
	}

	input := &PreToolUseInput{ToolName: "Write"}

	executor := NewActionExecutor(nil)
	err := executePreToolUseHook(executor, hook, input, nil)

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

// TODO: Task 7 - Implement UserPromptSubmit integration tests with JSON output

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

	executor := NewActionExecutor(nil)
	err := executePostToolUseHook(executor, hook, input, nil)
	if err != nil {
		t.Errorf("executePostToolUseHook() error = %v", err)
	}
}

func SkipTestExecutePreToolUseHooks_Integration(t *testing.T) {
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

func SkipTestExecutePreToolUseHooks_ConditionErrorAggregation(t *testing.T) {
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

func SkipTestExecutePreToolUseHooks_ConditionErrorAndExitError(t *testing.T) {
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

// TestExecuteSessionStartHooks_NewSignature tests the new executeSessionStartHooks
// that returns (*SessionStartOutput, error) for JSON output
func TestExecuteSessionStartHooks_NewSignature(t *testing.T) {
	tests := []struct {
		name                  string
		config                *Config
		input                 *SessionStartInput
		wantContinue          bool
		wantHookEventName     string
		wantAdditionalContext string
		wantSystemMessage     string
		wantErr               bool
	}{
		{
			name: "Single output action with continue true",
			config: &Config{
				SessionStart: []SessionStartHook{
					{
						Matcher: "startup",
						Actions: []Action{
							{
								Type:     "output",
								Message:  "Session initialized",
								Continue: boolPtr(true),
							},
						},
					},
				},
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-123",
					HookEventName:  SessionStart,
					TranscriptPath: "/tmp/transcript",
				},
				Source: "startup",
			},
			wantContinue:          true,
			wantHookEventName:     "SessionStart",
			wantAdditionalContext: "Session initialized",
			wantSystemMessage:     "",
			wantErr:               false,
		},
		{
			name: "Single output action with continue false",
			config: &Config{
				SessionStart: []SessionStartHook{
					{
						Matcher: "startup",
						Actions: []Action{
							{
								Type:     "output",
								Message:  "Blocked",
								Continue: boolPtr(false),
							},
						},
					},
				},
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-123",
					HookEventName:  SessionStart,
					TranscriptPath: "/tmp/transcript",
				},
				Source: "startup",
			},
			wantContinue:          false,
			wantHookEventName:     "SessionStart",
			wantAdditionalContext: "Blocked",
			wantSystemMessage:     "",
			wantErr:               false,
		},
		{
			name: "Multiple actions both succeed - additionalContext concatenated",
			config: &Config{
				SessionStart: []SessionStartHook{
					{
						Matcher: "startup",
						Actions: []Action{
							{
								Type:     "output",
								Message:  "First message",
								Continue: boolPtr(true),
							},
							{
								Type:     "output",
								Message:  "Second message",
								Continue: boolPtr(true),
							},
						},
					},
				},
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-123",
					HookEventName:  SessionStart,
					TranscriptPath: "/tmp/transcript",
				},
				Source: "startup",
			},
			wantContinue:          true,
			wantHookEventName:     "SessionStart",
			wantAdditionalContext: "First message\nSecond message",
			wantSystemMessage:     "",
			wantErr:               false,
		},
		{
			name: "First action continue false - early return",
			config: &Config{
				SessionStart: []SessionStartHook{
					{
						Matcher: "startup",
						Actions: []Action{
							{
								Type:     "output",
								Message:  "First blocks",
								Continue: boolPtr(false),
							},
							{
								Type:     "output",
								Message:  "Second never runs",
								Continue: boolPtr(true),
							},
						},
					},
				},
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-123",
					HookEventName:  SessionStart,
					TranscriptPath: "/tmp/transcript",
				},
				Source: "startup",
			},
			wantContinue:          false,
			wantHookEventName:     "SessionStart",
			wantAdditionalContext: "First blocks", // Only first action's message
			wantSystemMessage:     "",
			wantErr:               false,
		},
		{
			name: "Second action continue false - first preserved",
			config: &Config{
				SessionStart: []SessionStartHook{
					{
						Matcher: "startup",
						Actions: []Action{
							{
								Type:     "output",
								Message:  "First succeeds",
								Continue: boolPtr(true),
							},
							{
								Type:     "output",
								Message:  "Second blocks",
								Continue: boolPtr(false),
							},
						},
					},
				},
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-123",
					HookEventName:  SessionStart,
					TranscriptPath: "/tmp/transcript",
				},
				Source: "startup",
			},
			wantContinue:          false, // Last continue value wins
			wantHookEventName:     "SessionStart",
			wantAdditionalContext: "First succeeds\nSecond blocks",
			wantSystemMessage:     "",
			wantErr:               false,
		},
		{
			name: "HookEventName set by first action and preserved",
			config: &Config{
				SessionStart: []SessionStartHook{
					{
						Matcher: "startup",
						Actions: []Action{
							{
								Type:     "output",
								Message:  "First",
								Continue: boolPtr(true),
							},
							{
								Type:     "output",
								Message:  "Second",
								Continue: boolPtr(true),
							},
						},
					},
				},
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-123",
					HookEventName:  SessionStart,
					TranscriptPath: "/tmp/transcript",
				},
				Source: "startup",
			},
			wantContinue:          true,
			wantHookEventName:     "SessionStart", // Set once, preserved
			wantAdditionalContext: "First\nSecond",
			wantSystemMessage:     "",
			wantErr:               false,
		},
		{
			name: "Matcher not matching - hook skipped",
			config: &Config{
				SessionStart: []SessionStartHook{
					{
						Matcher: "resume",
						Actions: []Action{
							{
								Type:     "output",
								Message:  "Should not run",
								Continue: boolPtr(true),
							},
						},
					},
				},
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-123",
					HookEventName:  SessionStart,
					TranscriptPath: "/tmp/transcript",
				},
				Source: "startup", // Doesn't match "resume"
			},
			wantContinue:          true,           // Default Continue: true when no actions run
			wantHookEventName:     "SessionStart", // Always set to "SessionStart" (requirement 4.1)
			wantAdditionalContext: "",             // No messages
			wantSystemMessage:     "",
			wantErr:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawJSON := map[string]interface{}{
				"session_id":      tt.input.SessionID,
				"hook_event_name": string(tt.input.HookEventName),
				"source":          tt.input.Source,
			}

			output, err := executeSessionStartHooks(tt.config, tt.input, rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("executeSessionStartHooks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if output == nil {
				t.Fatal("executeSessionStartHooks() returned nil output")
			}

			if output.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
			}

			if output.HookSpecificOutput == nil && tt.wantHookEventName != "" {
				t.Fatal("HookSpecificOutput is nil but expected hookEventName")
			}

			if output.HookSpecificOutput != nil {
				if output.HookSpecificOutput.HookEventName != tt.wantHookEventName {
					t.Errorf("HookEventName = %q, want %q", output.HookSpecificOutput.HookEventName, tt.wantHookEventName)
				}

				if output.HookSpecificOutput.AdditionalContext != tt.wantAdditionalContext {
					t.Errorf("AdditionalContext = %q, want %q", output.HookSpecificOutput.AdditionalContext, tt.wantAdditionalContext)
				}
			} else if tt.wantHookEventName == "" && tt.wantAdditionalContext == "" {
				// Expected nil HookSpecificOutput
			} else {
				t.Errorf("Expected HookSpecificOutput with HookEventName=%q, AdditionalContext=%q, but got nil",
					tt.wantHookEventName, tt.wantAdditionalContext)
			}

			if output.SystemMessage != tt.wantSystemMessage {
				t.Errorf("SystemMessage = %q, want %q", output.SystemMessage, tt.wantSystemMessage)
			}

			// Phase 1: StopReason and SuppressOutput should remain zero values
			if output.StopReason != "" {
				t.Errorf("StopReason should be empty, got %q", output.StopReason)
			}

			if output.SuppressOutput != false {
				t.Errorf("SuppressOutput should be false, got %v", output.SuppressOutput)
			}
		})
	}
}

func TestExecuteSessionStartHooks_ErrorHandling(t *testing.T) {
	tests := []struct {
		name              string
		config            *Config
		input             *SessionStartInput
		wantContinue      bool
		wantHookEventName string
		wantSystemMessage string // Expected substring in SystemMessage
		wantErr           bool
	}{
		{
			name: "Condition error sets continue to false and includes error in SystemMessage",
			config: &Config{
				SessionStart: []SessionStartHook{
					{
						Matcher: "startup",
						Conditions: []Condition{
							{
								Type:  ConditionPromptRegex,
								Value: "test", // Invalid for SessionStart
							},
						},
						Actions: []Action{
							{
								Type:    "output",
								Message: "This should not execute",
							},
						},
					},
				},
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  SessionStart,
				},
				Source: "startup",
			},
			wantContinue:      false,                                     // Safe side default: error sets continue to false
			wantHookEventName: "SessionStart",                            // Always set for SessionStart
			wantSystemMessage: "unknown condition type for SessionStart", // Error message should be in SystemMessage (graceful degradation)
			wantErr:           true,                                      // Error is returned
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawJSON := map[string]interface{}{
				"session_id":      tt.input.SessionID,
				"transcript_path": tt.input.TranscriptPath,
				"hook_event_name": string(tt.input.HookEventName),
				"source":          tt.input.Source,
			}

			output, err := executeSessionStartHooks(tt.config, tt.input, rawJSON)

			// Error check
			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Continue check
			if output.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
			}

			// HookEventName check
			if output.HookSpecificOutput != nil {
				if output.HookSpecificOutput.HookEventName != tt.wantHookEventName {
					t.Errorf("HookEventName = %v, want %v", output.HookSpecificOutput.HookEventName, tt.wantHookEventName)
				}
			}

			// SystemMessage check (graceful degradation: error should be included in JSON output)
			if tt.wantSystemMessage != "" {
				if output.SystemMessage == "" {
					t.Errorf("Expected SystemMessage to contain error, but got empty string")
				} else if !strings.Contains(output.SystemMessage, tt.wantSystemMessage) {
					t.Errorf("SystemMessage = %q, want to contain %q", output.SystemMessage, tt.wantSystemMessage)
				}
			}
		})
	}
}

func TestExecuteUserPromptSubmitHooks_ConditionErrorAggregation(t *testing.T) {
	// Test that condition errors are aggregated and other hooks continue to execute
	config := &Config{
		UserPromptSubmit: []UserPromptSubmitHook{
			{
				// This hook will fail at condition check
				Conditions: []Condition{
					{
						Type:  ConditionReasonIs, // Invalid for UserPromptSubmit
						Value: "test",
					},
				},
				Actions: []Action{
					{
						Type:    "output",
						Message: "Should not execute due to condition error",
					},
				},
			},
			{
				// This hook should still execute after the first one's condition fails
				Actions: []Action{
					{
						Type:    "output",
						Message: "Second hook executed",
					},
				},
			},
		},
	}

	input := &UserPromptSubmitInput{
		BaseInput: BaseInput{
			SessionID:     "test-session",
			HookEventName: UserPromptSubmit,
		},
		Prompt: "test prompt",
	}

	rawJSON := map[string]interface{}{
		"session_id":      "test-session",
		"hook_event_name": "UserPromptSubmit",
		"prompt":          "test prompt",
	}

	output, err := executeUserPromptSubmitHooks(config, input, rawJSON)

	// Should return an error (from first hook's condition)
	if err == nil {
		t.Error("Expected error from condition check, got nil")
	}

	// Error message should contain condition error
	if !strings.Contains(err.Error(), "unknown condition type") {
		t.Errorf("Expected 'unknown condition type' in error message, got: %s", err.Error())
	}

	// Should still have output (from second hook)
	if output == nil {
		t.Fatal("Expected output despite error, got nil")
	}

	// Second hook should have executed
	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil - second hook did not execute")
	}

	if !strings.Contains(output.HookSpecificOutput.AdditionalContext, "Second hook executed") {
		t.Errorf("Expected 'Second hook executed' in AdditionalContext, got: %s", output.HookSpecificOutput.AdditionalContext)
	}
}

func TestExecuteUserPromptSubmitHooks_MultipleConditionErrors(t *testing.T) {
	// Test that multiple condition errors are collected and joined with errors.Join
	config := &Config{
		UserPromptSubmit: []UserPromptSubmitHook{
			{
				// This hook will fail at condition check
				Conditions: []Condition{
					{
						Type:  ConditionReasonIs, // Invalid for UserPromptSubmit
						Value: "test",
					},
				},
				Actions: []Action{
					{
						Type:    "output",
						Message: "Should not execute",
					},
				},
			},
			{
				// This hook will also fail at condition check
				Conditions: []Condition{
					{
						Type:  ConditionEveryNPrompts, // Requires transcript file
						Value: "5",
					},
				},
				Actions: []Action{
					{
						Type:    "output",
						Message: "Should also not execute",
					},
				},
			},
			{
				// This hook should execute successfully
				Actions: []Action{
					{
						Type:    "output",
						Message: "Third hook executed",
					},
				},
			},
		},
	}

	input := &UserPromptSubmitInput{
		BaseInput: BaseInput{
			SessionID:      "test-session",
			HookEventName:  UserPromptSubmit,
			TranscriptPath: "/nonexistent/transcript.jsonl", // Will cause error in every_n_prompts
		},
		Prompt: "test prompt",
	}

	rawJSON := map[string]interface{}{
		"session_id":      "test-session",
		"hook_event_name": "UserPromptSubmit",
		"transcript_path": "/nonexistent/transcript.jsonl",
		"prompt":          "test prompt",
	}

	output, err := executeUserPromptSubmitHooks(config, input, rawJSON)

	// Should return an error containing both condition errors
	if err == nil {
		t.Fatal("Expected errors from multiple conditions, got nil")
	}

	errMsg := err.Error()

	// Should contain first condition error (unknown condition type)
	if !strings.Contains(errMsg, "unknown condition type") {
		t.Errorf("Expected 'unknown condition type' in error message, got: %s", errMsg)
	}

	// Should contain second condition error (transcript file error)
	if !strings.Contains(errMsg, "transcript") || !strings.Contains(errMsg, "hook[UserPromptSubmit][1]") {
		t.Errorf("Expected second condition error about transcript in error message, got: %s", errMsg)
	}

	// Output should still be returned (graceful degradation)
	if output == nil {
		t.Fatal("Expected output despite errors, got nil")
	}

	// Third hook should have executed
	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}

	if !strings.Contains(output.HookSpecificOutput.AdditionalContext, "Third hook executed") {
		t.Errorf("Expected 'Third hook executed' in AdditionalContext, got: %s", output.HookSpecificOutput.AdditionalContext)
	}
}

func TestExecuteUserPromptSubmitHooks_AdditionalContextConcatenation(t *testing.T) {
	// Test that AdditionalContext from multiple actions is concatenated with newline
	// Note: SystemMessage concatenation is covered in integration tests
	config := &Config{
		UserPromptSubmit: []UserPromptSubmitHook{
			{
				Actions: []Action{
					{
						Type:    "output",
						Message: "First context",
					},
					{
						Type:    "output",
						Message: "Second context",
					},
					{
						Type:    "output",
						Message: "Third context",
					},
				},
			},
		},
	}

	input := &UserPromptSubmitInput{
		BaseInput: BaseInput{
			SessionID:     "test-session",
			HookEventName: UserPromptSubmit,
		},
		Prompt: "test prompt",
	}

	rawJSON := map[string]interface{}{
		"session_id":      "test-session",
		"hook_event_name": "UserPromptSubmit",
		"prompt":          "test prompt",
	}

	output, err := executeUserPromptSubmitHooks(config, input, rawJSON)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if output == nil {
		t.Fatal("Expected output, got nil")
	}

	// AdditionalContext should be concatenated with newline
	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}

	expectedAdditionalContext := "First context\nSecond context\nThird context"
	if output.HookSpecificOutput.AdditionalContext != expectedAdditionalContext {
		t.Errorf("AdditionalContext = %q, want %q", output.HookSpecificOutput.AdditionalContext, expectedAdditionalContext)
	}
}
