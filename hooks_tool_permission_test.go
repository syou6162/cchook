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
		{
			"Process substitution error returns true with error",
			PreToolUseHook{
				Matcher: "Bash",
				Conditions: []Condition{
					{Type: ConditionGitTrackedFileOperation, Value: "diff"},
				},
			},
			&PreToolUseInput{
				ToolName:  "Bash",
				ToolInput: ToolInput{Command: "diff -u file1 <(head -48 file2)"},
			},
			true, // プロセス置換検出時は条件マッチとして扱う
			true, // エラーを返す
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := shouldExecutePreToolUseHook(tt.hook, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("shouldExecutePreToolUseHook() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Phase 3: プロセス置換エラーの種類を確認
			if err != nil && strings.Contains(tt.name, "Process substitution") {
				if !errors.Is(err, ErrProcessSubstitutionDetected) {
					t.Errorf("shouldExecutePreToolUseHook() error type = %T, want ErrProcessSubstitutionDetected", err)
				}
			}
			if got != tt.want {
				t.Errorf("shouldExecutePreToolUseHook() = %v, want %v", got, tt.want)
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
		{
			"Process substitution error returns true with error",
			PostToolUseHook{
				Matcher: "Bash",
				Conditions: []Condition{
					{Type: ConditionGitTrackedFileOperation, Value: "diff"},
				},
			},
			&PostToolUseInput{
				ToolName:  "Bash",
				ToolInput: ToolInput{Command: "diff -u file1 <(head -48 file2)"},
			},
			true, // プロセス置換検出時は条件マッチとして扱う
			true, // エラーを返す
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := shouldExecutePostToolUseHook(tt.hook, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("shouldExecutePostToolUseHook() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Phase 4: プロセス置換エラーの種類を確認
			if err != nil && strings.Contains(tt.name, "Process substitution") {
				if !errors.Is(err, ErrProcessSubstitutionDetected) {
					t.Errorf("shouldExecutePostToolUseHook() error type = %T, want ErrProcessSubstitutionDetected", err)
				}
			}
			if got != tt.want {
				t.Errorf("shouldExecutePostToolUseHook() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TODO: Task 7 - Implement UserPromptSubmit integration tests with JSON output


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

	output, err := executePostToolUseHooksJSON(config, input, nil)

	// エラーが返されることを確認
	if err == nil {
		t.Fatal("Expected error for invalid condition types, got nil")
	}

	// fail-safeでdecision="block"になることを確認
	if output == nil {
		t.Fatal("Expected output, got nil")
	}
	if output.Decision != "block" {
		t.Errorf("Expected decision=block on condition errors, got: %q", output.Decision)
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

// TestExecuteNotification_ConditionErrorAggregation removed - old implementation used executeNotification,
// new implementation uses executeNotificationHooksJSON with JSON output pattern.

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

	output, err := executePostToolUseHooksJSON(config, input, nil)
	if err != nil {
		if output == nil {
			t.Fatal("Expected output, got nil")
		}
		t.Errorf("executePostToolUseHooks() error = %v", err)
	}
}

// ============================================================================
// Task 7: New tests for executePreToolUseHook (JSON output)
// ============================================================================

func TestExecutePreToolUseHook_NewSignature(t *testing.T) {
	tests := []struct {
		name                         string
		config                       *Config
		input                        *PreToolUseInput
		wantPermissionDecision       string
		wantPermissionDecisionReason string
		wantHookEventName            string
		wantUpdatedInput             map[string]interface{}
		wantSystemMessage            string
		wantNilOutput                bool
		useStubRunner                bool
		wantErr                      bool
	}{
		{
			name: "Single output action with permissionDecision: allow",
			config: &Config{
				PreToolUse: []PreToolUseHook{
					{
						Matcher: "Write",
						Actions: []Action{
							{
								Type:               "output",
								Message:            "Allowing write operation",
								PermissionDecision: stringPtr("allow"),
							},
						},
					},
				},
			},
			input: &PreToolUseInput{
				BaseInput: BaseInput{
					SessionID:      "test-123",
					HookEventName:  PreToolUse,
					TranscriptPath: "/tmp/transcript",
				},
				ToolName: "Write",
				ToolInput: ToolInput{
					FilePath: "test.txt",
				},
			},
			wantPermissionDecision:       "allow",
			wantPermissionDecisionReason: "Allowing write operation",
			wantHookEventName:            "PreToolUse",
			wantSystemMessage:            "",
			wantNilOutput:                false,
			useStubRunner:                false,
			wantErr:                      false,
		},
		{
			name: "Single output action with permissionDecision: deny",
			config: &Config{
				PreToolUse: []PreToolUseHook{
					{
						Matcher: "Write",
						Actions: []Action{
							{
								Type:               "output",
								Message:            "Blocking dangerous operation",
								PermissionDecision: stringPtr("deny"),
							},
						},
					},
				},
			},
			input: &PreToolUseInput{
				BaseInput: BaseInput{
					SessionID:      "test-123",
					HookEventName:  PreToolUse,
					TranscriptPath: "/tmp/transcript",
				},
				ToolName: "Write",
				ToolInput: ToolInput{
					FilePath: "dangerous.sh",
				},
			},
			wantPermissionDecision:       "deny",
			wantPermissionDecisionReason: "Blocking dangerous operation",
			wantHookEventName:            "PreToolUse",
			wantSystemMessage:            "",
			wantNilOutput:                false,
			useStubRunner:                false,
			wantErr:                      false,
		},
		{
			name: "Command action with empty stdout -> output is nil (delegation)",
			config: &Config{
				PreToolUse: []PreToolUseHook{
					{
						Matcher: "Write",
						Actions: []Action{
							{
								Type:    "command",
								Command: "validator.sh",
							},
						},
					},
				},
			},
			input: &PreToolUseInput{
				BaseInput: BaseInput{
					SessionID:      "test-123",
					HookEventName:  PreToolUse,
					TranscriptPath: "/tmp/transcript",
				},
				ToolName: "Write",
				ToolInput: ToolInput{
					FilePath: "test.txt",
				},
			},
			wantNilOutput: true,
			useStubRunner: true,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For command action tests, use stubRunner
			var executor *ActionExecutor
			if tt.useStubRunner {
				runner := &stubRunnerWithOutput{
					stdout:   "",
					exitCode: 0,
				}
				executor = NewActionExecutor(runner)
			} else {
				executor = NewActionExecutor(nil)
			}

			output, err := executePreToolUseHook(executor, tt.config.PreToolUse[0], tt.input, tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("executePreToolUseHook() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantNilOutput {
				if output != nil {
					t.Fatalf("Expected nil output (delegation), got: %+v", output)
				}
				return
			}

			if output == nil {
				t.Fatal("output is nil")
			}

			// Continue is ALWAYS true for PreToolUse
			if !output.Continue {
				t.Errorf("Continue = %v, want true", output.Continue)
			}

			if output.PermissionDecision != tt.wantPermissionDecision {
				t.Errorf("PermissionDecision = %q, want %q", output.PermissionDecision, tt.wantPermissionDecision)
			}

			if output.PermissionDecisionReason != tt.wantPermissionDecisionReason {
				t.Errorf("PermissionDecisionReason = %q, want %q", output.PermissionDecisionReason, tt.wantPermissionDecisionReason)
			}

			if output.HookEventName != tt.wantHookEventName {
				t.Errorf("HookEventName = %q, want %q", output.HookEventName, tt.wantHookEventName)
			}

			if output.SystemMessage != tt.wantSystemMessage {
				t.Errorf("SystemMessage = %q, want %q", output.SystemMessage, tt.wantSystemMessage)
			}
		})
	}
}

// TestExecutePreToolUseHooksJSON_HookSpecificOutput tests hookSpecificOutput generation
//
// Note on error handling tests:
// - conditionErrors: Testable via ConditionType{v: "unknown"} (same package access)
// - actionErrors: Difficult to trigger (ExecutePreToolUseAction rarely returns errors)
func TestExecutePreToolUseHooksJSON_HookSpecificOutput(t *testing.T) {
	tests := []struct {
		name                   string
		config                 *Config
		input                  *PreToolUseInput
		wantHookSpecificOutput bool
		wantPermissionDecision string
		wantContinue           bool
		useStubRunner          bool
	}{
		{
			name: "No match - hookSpecificOutput should be nil",
			config: &Config{
				PreToolUse: []PreToolUseHook{
					{
						Matcher: "Edit",
						Actions: []Action{
							{Type: "output", Message: "test", PermissionDecision: stringPtr("deny")},
						},
					},
				},
			},
			input: &PreToolUseInput{
				ToolName: "Write",
			},
			wantHookSpecificOutput: false,
			wantContinue:           true,
			useStubRunner:          false,
		},
		{
			name: "Match + output action - hookSpecificOutput should exist",
			config: &Config{
				PreToolUse: []PreToolUseHook{
					{
						Matcher: "Write",
						Actions: []Action{
							{Type: "output", Message: "validation failed", PermissionDecision: stringPtr("deny")},
						},
					},
				},
			},
			input: &PreToolUseInput{
				ToolName: "Write",
			},
			wantHookSpecificOutput: true,
			wantPermissionDecision: "deny",
			wantContinue:           true,
			useStubRunner:          false,
		},
		{
			name: "Match + empty stdout - hookSpecificOutput should be nil (delegation)",
			config: &Config{
				PreToolUse: []PreToolUseHook{
					{
						Matcher: "Write",
						Actions: []Action{
							{Type: "command", Command: "validator.sh"},
						},
					},
				},
			},
			input: &PreToolUseInput{
				ToolName: "Write",
			},
			wantHookSpecificOutput: false,
			wantContinue:           true,
			useStubRunner:          true,
		},
		{
			name: "Condition error - hookSpecificOutput with permissionDecision: deny",
			config: &Config{
				PreToolUse: []PreToolUseHook{
					{
						Matcher: "Write",
						Conditions: []Condition{
							{Type: ConditionType{v: "unknown"}, Value: "test"},
						},
						Actions: []Action{
							{Type: "output", Message: "test"},
						},
					},
				},
			},
			input: &PreToolUseInput{
				ToolName: "Write",
			},
			wantHookSpecificOutput: true,
			wantPermissionDecision: "deny",
			wantContinue:           true,
			useStubRunner:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For command action tests with empty stdout, use stubRunner
			// Note: executePreToolUseHooksJSON creates its own executor internally,
			// so we test via executePreToolUseHook directly for stub runner cases
			var output *PreToolUseOutput
			var err error

			if tt.useStubRunner {
				runner := &stubRunnerWithOutput{
					stdout:   "",
					exitCode: 0,
				}
				executor := NewActionExecutor(runner)

				// Test the single hook execution path
				actionOutput, hookErr := executePreToolUseHook(executor, tt.config.PreToolUse[0], tt.input, map[string]interface{}{
					"tool_name":  tt.input.ToolName,
					"tool_input": tt.input.ToolInput,
				})

				// Build PreToolUseOutput structure manually to match executePreToolUseHooksJSON behavior
				output = &PreToolUseOutput{
					Continue: true,
				}
				if actionOutput != nil && actionOutput.PermissionDecision != "" {
					output.HookSpecificOutput = &PreToolUseHookSpecificOutput{
						HookEventName:            "PreToolUse",
						PermissionDecision:       actionOutput.PermissionDecision,
						PermissionDecisionReason: actionOutput.PermissionDecisionReason,
					}
				}
				err = hookErr
			} else {
				output, err = executePreToolUseHooksJSON(tt.config, tt.input, map[string]interface{}{
					"tool_name":  tt.input.ToolName,
					"tool_input": tt.input.ToolInput,
				})
			}

			// For "Condition error" test, error is expected
			if tt.name == "Condition error - hookSpecificOutput with permissionDecision: deny" {
				if err == nil {
					t.Fatal("executePreToolUseHooksJSON() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("executePreToolUseHooksJSON() error = %v", err)
				}
			}

			if output.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
			}

			if tt.wantHookSpecificOutput {
				if output.HookSpecificOutput == nil {
					t.Fatal("HookSpecificOutput is nil, want non-nil")
				}
				if output.HookSpecificOutput.PermissionDecision != tt.wantPermissionDecision {
					t.Errorf("PermissionDecision = %q, want %q", output.HookSpecificOutput.PermissionDecision, tt.wantPermissionDecision)
				}
			} else {
				if output.HookSpecificOutput != nil {
					t.Errorf("HookSpecificOutput = %+v, want nil", output.HookSpecificOutput)
				}
			}

			// Test JSON serialization - verify omitempty behavior
			jsonBytes, err := json.Marshal(output)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			var jsonMap map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &jsonMap); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			_, hasHookSpecificOutput := jsonMap["hookSpecificOutput"]
			if tt.wantHookSpecificOutput && !hasHookSpecificOutput {
				t.Error("JSON missing hookSpecificOutput field, want it present")
			}
			if !tt.wantHookSpecificOutput && hasHookSpecificOutput {
				t.Errorf("JSON has hookSpecificOutput field, want it omitted. JSON: %s", string(jsonBytes))
			}
		})
	}
}

func TestExecutePreToolUseHooksJSON_ProcessSubstitution(t *testing.T) {
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

	output, err := executePreToolUseHooksJSON(config, input, rawJSON)

	// エラーがあってもJSONは返される（常に成功）
	if err != nil {
		t.Errorf("executePreToolUseHooksJSON() error = %v, want nil", err)
	}

	// permissionDecision が "deny" であることを確認
	if output.HookSpecificOutput.PermissionDecision != "deny" {
		t.Errorf("PermissionDecision = %v, want deny", output.HookSpecificOutput.PermissionDecision)
	}

	// 理由メッセージにプロセス置換の警告が含まれることを確認
	if !strings.Contains(output.HookSpecificOutput.PermissionDecisionReason, "プロセス置換") {
		t.Errorf("PermissionDecisionReason = %v, want to contain プロセス置換", output.HookSpecificOutput.PermissionDecisionReason)
	}

	// hookEventName が設定されていることを確認
	if output.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Errorf("HookEventName = %v, want PreToolUse", output.HookSpecificOutput.HookEventName)
	}
}

func TestExecutePreToolUseHooksJSON_AdditionalContext(t *testing.T) {
	tests := []struct {
		name                   string
		config                 *Config
		input                  *PreToolUseInput
		wantAdditionalContext  string
		wantPermissionDecision string
	}{
		{
			name: "Single action with additionalContext",
			config: &Config{
				PreToolUse: []PreToolUseHook{
					{
						Matcher: "Bash",
						Actions: []Action{
							{
								Type:               "output",
								Message:            "Operation allowed",
								PermissionDecision: stringPtr("allow"),
								AdditionalContext:  stringPtr("Current environment: production"),
							},
						},
					},
				},
			},
			input: &PreToolUseInput{
				ToolName: "Bash",
			},
			wantAdditionalContext:  "Current environment: production",
			wantPermissionDecision: "allow",
		},
		{
			name: "Multiple actions - additionalContext concatenated with newline",
			config: &Config{
				PreToolUse: []PreToolUseHook{
					{
						Matcher: "Write",
						Actions: []Action{
							{
								Type:               "output",
								Message:            "First check",
								PermissionDecision: stringPtr("allow"),
								AdditionalContext:  stringPtr("Environment: staging"),
							},
						},
					},
					{
						Matcher: "Write",
						Actions: []Action{
							{
								Type:               "output",
								Message:            "Second check",
								PermissionDecision: stringPtr("allow"),
								AdditionalContext:  stringPtr("User: admin"),
							},
						},
					},
				},
			},
			input: &PreToolUseInput{
				ToolName: "Write",
			},
			wantAdditionalContext:  "Environment: staging\nUser: admin",
			wantPermissionDecision: "allow",
		},
		{
			name: "additionalContext empty - omitted from output",
			config: &Config{
				PreToolUse: []PreToolUseHook{
					{
						Matcher: "Edit",
						Actions: []Action{
							{
								Type:               "output",
								Message:            "Check passed",
								PermissionDecision: stringPtr("allow"),
							},
						},
					},
				},
			},
			input: &PreToolUseInput{
				ToolName: "Edit",
			},
			wantAdditionalContext:  "",
			wantPermissionDecision: "allow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawJSON := map[string]interface{}{
				"tool_name": tt.input.ToolName,
			}

			output, err := executePreToolUseHooksJSON(tt.config, tt.input, rawJSON)

			if err != nil {
				t.Fatalf("executePreToolUseHooksJSON() error = %v, want nil", err)
			}

			if output.HookSpecificOutput == nil {
				t.Fatal("HookSpecificOutput should not be nil")
			}

			if output.HookSpecificOutput.AdditionalContext != tt.wantAdditionalContext {
				t.Errorf("AdditionalContext = %q, want %q", output.HookSpecificOutput.AdditionalContext, tt.wantAdditionalContext)
			}

			if output.HookSpecificOutput.PermissionDecision != tt.wantPermissionDecision {
				t.Errorf("PermissionDecision = %q, want %q", output.HookSpecificOutput.PermissionDecision, tt.wantPermissionDecision)
			}
		})
	}
}

func TestExecutePostToolUseHooks_ProcessSubstitution(t *testing.T) {
	// 標準エラー出力をキャプチャ
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

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

	output, err := executePostToolUseHooksJSON(config, input, rawJSON)

	// 標準エラー出力を元に戻す
	_ = w.Close()
	os.Stderr = oldStderr

	// 出力を読み取る
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	stderrOutput := buf.String()

	if output == nil {
		t.Fatal("Expected output, got nil")
	}
	// エラーがないこと（警告のみ）
	if err != nil {
		t.Errorf("executePostToolUseHooks() error = %v, want nil", err)
	}

	// 標準エラーにプロセス置換の警告が出力されていることを確認
	if !strings.Contains(stderrOutput, "プロセス置換") {
		t.Errorf("stderr output = %v, want to contain プロセス置換", stderrOutput)
	}
}

func TestExecutePostToolUseHooksJSON(t *testing.T) {
	tests := []struct {
		name                  string
		config                *Config
		input                 *PostToolUseInput
		rawJSON               interface{}
		wantContinue          bool
		wantDecision          string
		wantReason            string
		wantAdditionalContext string
		wantSystemMessage     string
		wantErr               bool
	}{
		{
			name:         "1. No hooks configured - allow tool result",
			config:       &Config{},
			input:        &PostToolUseInput{BaseInput: BaseInput{SessionID: "test", HookEventName: PostToolUse}, ToolName: "Write"},
			rawJSON:      map[string]interface{}{},
			wantContinue: true,
			wantDecision: "",
			wantErr:      false,
		},
		{
			name: "2. Output action with decision block",
			config: &Config{
				PostToolUse: []PostToolUseHook{
					{
						Actions: []Action{
							{
								Type:     "output",
								Message:  "Tool output blocked",
								Decision: stringPtr("block"),
								Reason:   stringPtr("Contains sensitive data"),
							},
						},
					},
				},
			},
			input:                 &PostToolUseInput{BaseInput: BaseInput{SessionID: "test", HookEventName: PostToolUse}, ToolName: "Write"},
			rawJSON:               map[string]interface{}{},
			wantContinue:          true,
			wantDecision:          "block",
			wantReason:            "Contains sensitive data",
			wantAdditionalContext: "Tool output blocked",
			wantErr:               false,
		},
		{
			name: "3. Output action with decision omitted (allow)",
			config: &Config{
				PostToolUse: []PostToolUseHook{
					{
						Actions: []Action{
							{
								Type:    "output",
								Message: "Tool result valid",
							},
						},
					},
				},
			},
			input:                 &PostToolUseInput{BaseInput: BaseInput{SessionID: "test", HookEventName: PostToolUse}, ToolName: "Write"},
			rawJSON:               map[string]interface{}{},
			wantContinue:          true,
			wantDecision:          "",
			wantReason:            "",
			wantAdditionalContext: "Tool result valid",
			wantErr:               false,
		},
		{
			name: "4. Multiple actions - decision last wins",
			config: &Config{
				PostToolUse: []PostToolUseHook{
					{
						Actions: []Action{
							{
								Type:     "output",
								Message:  "First allows",
								Decision: stringPtr(""),
							},
							{
								Type:     "output",
								Message:  "Then blocks",
								Decision: stringPtr("block"),
								Reason:   stringPtr("Final decision"),
							},
						},
					},
				},
			},
			input:                 &PostToolUseInput{BaseInput: BaseInput{SessionID: "test", HookEventName: PostToolUse}, ToolName: "Write"},
			rawJSON:               map[string]interface{}{},
			wantContinue:          true,
			wantDecision:          "block",
			wantReason:            "Final decision",
			wantAdditionalContext: "First allows\nThen blocks",
			wantErr:               false,
		},
		{
			name: "5. Multiple actions - reason reset on decision change",
			config: &Config{
				PostToolUse: []PostToolUseHook{
					{
						Actions: []Action{
							{
								Type:     "output",
								Message:  "Block first",
								Decision: stringPtr("block"),
								Reason:   stringPtr("Reason 1"),
							},
							{
								Type:     "output",
								Message:  "Allow second",
								Decision: stringPtr(""),
							},
						},
					},
				},
			},
			input:                 &PostToolUseInput{BaseInput: BaseInput{SessionID: "test", HookEventName: PostToolUse}, ToolName: "Write"},
			rawJSON:               map[string]interface{}{},
			wantContinue:          true,
			wantDecision:          "",
			wantReason:            "",
			wantAdditionalContext: "Block first\nAllow second",
			wantErr:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executePostToolUseHooksJSON(tt.config, tt.input, tt.rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("executePostToolUseHooksJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if output == nil {
				t.Fatal("output is nil")
			}

			if output.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
			}

			if output.Decision != tt.wantDecision {
				t.Errorf("Decision = %q, want %q", output.Decision, tt.wantDecision)
			}

			if output.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", output.Reason, tt.wantReason)
			}

			// AdditionalContextはHookSpecificOutput経由でアクセス
			gotAdditionalContext := ""
			if output.HookSpecificOutput != nil {
				gotAdditionalContext = output.HookSpecificOutput.AdditionalContext
			}
			if gotAdditionalContext != tt.wantAdditionalContext {
				t.Errorf("HookSpecificOutput.AdditionalContext = %q, want %q", gotAdditionalContext, tt.wantAdditionalContext)
			}

			if tt.wantSystemMessage != "" && output.SystemMessage != tt.wantSystemMessage {
				t.Errorf("SystemMessage = %q, want %q", output.SystemMessage, tt.wantSystemMessage)
			}
		})
	}
}

func TestExecutePermissionRequestHooks(t *testing.T) {
	config := &Config{
		PermissionRequest: []PermissionRequestHook{
			{
				Matcher: "Bash",
				Actions: []Action{
					{
						Type:    "output",
						Message: "Command: {.tool_input.command}",
						Behavior: func() *string {
							s := "allow"
							return &s
						}(),
					},
				},
			},
			{
				Matcher: "Write",
				Actions: []Action{
					{
						Type:    "output",
						Message: "Dangerous operation",
						Behavior: func() *string {
							s := "deny"
							return &s
						}(),
						Interrupt: func() *bool {
							b := true
							return &b
						}(),
					},
				},
			},
		},
	}

	tests := []struct {
		name              string
		input             *PermissionRequestInput
		wantBehavior      string
		wantMessage       string
		wantInterrupt     bool
		wantContinue      bool
		wantHookEventName string
	}{
		{
			name: "Bash matcher matches - allow",
			input: &PermissionRequestInput{
				BaseInput: BaseInput{
					SessionID:      "test-session-123",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  PermissionRequest,
				},
				ToolName: "Bash",
				ToolInput: ToolInput{
					Command: "ls -la",
				},
			},
			wantBehavior:      "allow",
			wantMessage:       "", // 公式仕様: allow時はmessageは空
			wantInterrupt:     false,
			wantContinue:      true,
			wantHookEventName: "PermissionRequest",
		},
		{
			name: "Write matcher matches - deny with interrupt",
			input: &PermissionRequestInput{
				BaseInput: BaseInput{
					SessionID:      "test-session-456",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  PermissionRequest,
				},
				ToolName: "Write",
				ToolInput: ToolInput{
					FilePath: "test.go",
				},
			},
			wantBehavior:      "deny",
			wantMessage:       "Dangerous operation",
			wantInterrupt:     true,
			wantContinue:      true,
			wantHookEventName: "PermissionRequest",
		},
		{
			name: "No matcher matches - default allow",
			input: &PermissionRequestInput{
				BaseInput: BaseInput{
					SessionID:      "test-session-789",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  PermissionRequest,
				},
				ToolName: "Read",
				ToolInput: ToolInput{
					FilePath: "test.go",
				},
			},
			wantBehavior:      "allow",
			wantMessage:       "",
			wantInterrupt:     false,
			wantContinue:      true,
			wantHookEventName: "PermissionRequest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// rawJSON作成
			rawJSON := map[string]interface{}{
				"session_id":      tt.input.SessionID,
				"transcript_path": tt.input.TranscriptPath,
				"hook_event_name": string(tt.input.HookEventName),
				"tool_name":       tt.input.ToolName,
				"tool_input": map[string]interface{}{
					"command":   tt.input.ToolInput.Command,
					"file_path": tt.input.ToolInput.FilePath,
				},
			}

			// フック実行
			output, err := executePermissionRequestHooksJSON(config, tt.input, rawJSON)

			// エラーチェック
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Continue チェック
			if output.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
			}

			// HookEventName チェック
			if output.HookSpecificOutput.HookEventName != tt.wantHookEventName {
				t.Errorf("HookEventName = %q, want %q", output.HookSpecificOutput.HookEventName, tt.wantHookEventName)
			}

			// Behavior チェック
			if output.HookSpecificOutput.Decision.Behavior != tt.wantBehavior {
				t.Errorf("Behavior = %q, want %q", output.HookSpecificOutput.Decision.Behavior, tt.wantBehavior)
			}

			// Message チェック
			if output.HookSpecificOutput.Decision.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", output.HookSpecificOutput.Decision.Message, tt.wantMessage)
			}

			// Interrupt チェック
			if output.HookSpecificOutput.Decision.Interrupt != tt.wantInterrupt {
				t.Errorf("Interrupt = %v, want %v", output.HookSpecificOutput.Decision.Interrupt, tt.wantInterrupt)
			}
		})
	}
}

func TestExecutePermissionRequestHooks_BehaviorChange(t *testing.T) {
	tests := []struct {
		name             string
		actions          []Action
		commandOutputs   []string // stubRunnerで返すJSON出力（各アクション用）
		input            *PermissionRequestInput
		rawJSON          map[string]interface{}
		wantBehavior     string
		wantUpdatedInput map[string]interface{}
		wantMessage      string
		wantInterrupt    bool
	}{
		{
			name: "allow→deny: updatedInput should be cleared",
			actions: []Action{
				{Type: "command", Command: "mock_allow_updated_input.sh"},
				{Type: "command", Command: "mock_deny_message.sh"},
			},
			commandOutputs: []string{
				`{"continue":true,"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"allow","updatedInput":{"file_path":"modified.go"}}}}`,
				`{"continue":true,"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"deny","message":"Operation blocked"}}}`,
			},
			input: &PermissionRequestInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  PermissionRequest,
				},
				ToolName: "Write",
				ToolInput: ToolInput{
					FilePath: "test.go",
				},
			},
			rawJSON: map[string]interface{}{
				"tool_name": "Write",
				"tool_input": map[string]interface{}{
					"file_path": "test.go",
				},
			},
			wantBehavior:     "deny",
			wantUpdatedInput: nil, // クリアされるべき
			wantMessage:      "Operation blocked",
			wantInterrupt:    false,
		},
		{
			name: "deny→allow: message and interrupt should be cleared",
			actions: []Action{
				{Type: "command", Command: "mock_deny_message_interrupt.sh"},
				{Type: "command", Command: "mock_allow_updated_input.sh"},
			},
			commandOutputs: []string{
				`{"continue":true,"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"deny","message":"Blocked","interrupt":true}}}`,
				`{"continue":true,"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"allow","updatedInput":{"file_path":"test.go"}}}}`,
			},
			input: &PermissionRequestInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  PermissionRequest,
				},
				ToolName: "Write",
				ToolInput: ToolInput{
					FilePath: "test.go",
				},
			},
			rawJSON: map[string]interface{}{
				"tool_name": "Write",
				"tool_input": map[string]interface{}{
					"file_path": "test.go",
				},
			},
			wantBehavior:     "allow",
			wantUpdatedInput: map[string]interface{}{"file_path": "test.go"},
			wantMessage:      "",    // クリアされるべき
			wantInterrupt:    false, // クリアされるべき
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// stubRunnerWithMultipleOutputsを使用
			runner := &stubRunnerWithMultipleOutputs{
				outputs: tt.commandOutputs,
				index:   0,
			}
			executor := NewActionExecutor(runner)

			hook := PermissionRequestHook{
				Matcher: "Write",
				Actions: tt.actions,
			}

			// executePermissionRequestHookを直接呼び出し
			output, err := executePermissionRequestHook(executor, hook, tt.input, tt.rawJSON)

			// エラーチェック
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Behavior チェック
			if output.Behavior != tt.wantBehavior {
				t.Errorf("Behavior = %q, want %q", output.Behavior, tt.wantBehavior)
			}

			// UpdatedInput チェック
			if tt.wantUpdatedInput == nil {
				if output.UpdatedInput != nil {
					t.Errorf("UpdatedInput = %v, want nil", output.UpdatedInput)
				}
			} else {
				if output.UpdatedInput == nil {
					t.Errorf("UpdatedInput = nil, want %v", tt.wantUpdatedInput)
				} else {
					// map比較
					gotJSON, _ := json.Marshal(output.UpdatedInput)
					wantJSON, _ := json.Marshal(tt.wantUpdatedInput)
					if string(gotJSON) != string(wantJSON) {
						t.Errorf("UpdatedInput = %s, want %s", gotJSON, wantJSON)
					}
				}
			}

			// Message チェック
			if output.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", output.Message, tt.wantMessage)
			}

			// Interrupt チェック
			if output.Interrupt != tt.wantInterrupt {
				t.Errorf("Interrupt = %v, want %v", output.Interrupt, tt.wantInterrupt)
			}
		})
	}
}

func TestExecutePermissionRequestHooks_MultipleActions(t *testing.T) {
	runner := &stubRunnerWithMultipleOutputs{
		outputs: []string{
			`{"continue":true,"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"deny","message":"First message"}},"systemMessage":"System message"}`,
			`{"continue":true,"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"deny","message":"Second message"}}}`,
		},
		index: 0,
	}
	executor := NewActionExecutor(runner)

	hook := PermissionRequestHook{
		Matcher: "Write",
		Actions: []Action{
			{Type: "command", Command: "mock_deny_message1.sh"},
			{Type: "command", Command: "mock_deny_message2.sh"},
		},
	}

	input := &PermissionRequestInput{
		BaseInput: BaseInput{
			SessionID:      "test-session",
			TranscriptPath: "/path/to/transcript",
			HookEventName:  PermissionRequest,
		},
		ToolName: "Write",
		ToolInput: ToolInput{
			FilePath: "test.go",
		},
	}

	rawJSON := map[string]interface{}{
		"tool_name": "Write",
		"tool_input": map[string]interface{}{
			"file_path": "test.go",
		},
	}

	output, err := executePermissionRequestHook(executor, hook, input, rawJSON)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Behavior should be deny (last value wins)
	if output.Behavior != "deny" {
		t.Errorf("Behavior = %q, want %q", output.Behavior, "deny")
	}

	// Messages should be concatenated
	expectedMessage := "First message\nSecond message"
	if output.Message != expectedMessage {
		t.Errorf("Message = %q, want %q", output.Message, expectedMessage)
	}

	// SystemMessage should be set
	if output.SystemMessage != "System message" {
		t.Errorf("SystemMessage = %q, want %q", output.SystemMessage, "System message")
	}
}

func TestExecutePermissionRequestHooks_DenyEarlyReturn(t *testing.T) {
	runner := &stubRunnerWithMultipleOutputs{
		outputs: []string{
			`{"continue":false,"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"deny","message":"Operation blocked"}}}`,
			`{"continue":true,"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"deny","message":"This script should not have been executed"}}}`,
		},
		index: 0,
	}
	executor := NewActionExecutor(runner)

	hook := PermissionRequestHook{
		Matcher: "Write",
		Actions: []Action{
			{Type: "command", Command: "mock_deny_message.sh"},
			{Type: "command", Command: "mock_should_not_execute.sh"},
		},
	}

	input := &PermissionRequestInput{
		BaseInput: BaseInput{
			SessionID:      "test-session",
			TranscriptPath: "/path/to/transcript",
			HookEventName:  PermissionRequest,
		},
		ToolName: "Write",
		ToolInput: ToolInput{
			FilePath: "test.go",
		},
	}

	rawJSON := map[string]interface{}{
		"tool_name": "Write",
		"tool_input": map[string]interface{}{
			"file_path": "test.go",
		},
	}

	output, err := executePermissionRequestHook(executor, hook, input, rawJSON)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Behavior should be deny
	if output.Behavior != "deny" {
		t.Errorf("Behavior = %q, want %q", output.Behavior, "deny")
	}

	// continue=falseで早期リターンするため、2番目のメッセージは含まれない
	if !strings.Contains(output.Message, "Operation blocked") {
		t.Errorf("Message should contain 'Operation blocked', got %q", output.Message)
	}
	if strings.Contains(output.Message, "This script should not have been executed") {
		t.Errorf("Message should NOT contain second action message (early return), got %q", output.Message)
	}
}
