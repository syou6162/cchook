package main

import (
	"strings"
	"testing"
)

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
					{Type: ConditionFileExistsRecursive, Value: "hooks_execute_test.go"},
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
			wantDecision:      "",
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
			wantDecision:      "",
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
			wantDecision:      "",
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

// TestExecuteStopHook_FailingCommandReturnsExit2 tests that failing commands
// result in decision="block" (fail-safe) from executeStopHooks.
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

	output, err := executeStopHooks(config, input, nil)

	// JSON出力パターン: コマンド失敗時はdecision="block"（fail-safe）
	// エラーは返さない（JSON出力で制御）
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if output == nil {
		t.Fatal("Expected output, got nil")
	}

	if output.Decision != "block" {
		t.Errorf("Expected decision 'block' for failing command, got %q", output.Decision)
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

	output, err := executeSubagentStopHooks(config, input, nil)

	// JSON対応後は、コマンド失敗時もerrorではなくdecision="block"のOutputを返す
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if output == nil {
		t.Fatal("Expected output, got nil")
	}

	// Fail-safe: decision should be "block"
	if output.Decision != "block" {
		t.Errorf("Expected decision 'block', got %q", output.Decision)
	}

	// Continue should always be true for SubagentStop
	if output.Continue != true {
		t.Errorf("Expected continue true, got %v", output.Continue)
	}
}

// TestExecuteNotification removed - old implementation used executeNotification,
// new implementation uses executeNotificationHooksJSON. See TestExecuteNotificationHooksJSON in hooks_execute_test.go.

func TestExecuteStopHooks(t *testing.T) {
	config := &Config{}
	input := &StopInput{}

	output, err := executeStopHooks(config, input, nil)
	if err != nil {
		t.Errorf("executeStopHooks() error = %v, expected nil", err)
	}
	if output == nil {
		t.Fatal("Expected output, got nil")
	}
	if !output.Continue {
		t.Error("Expected Continue=true")
	}
	if output.Decision != "" {
		t.Errorf("Expected empty decision, got %q", output.Decision)
	}
}

func TestExecuteSubagentStopHooks(t *testing.T) {
	config := &Config{}
	input := &SubagentStopInput{}

	output, err := executeSubagentStopHooks(config, input, nil)
	if err != nil {
		t.Errorf("executeSubagentStopHooks() error = %v, expected nil", err)
	}

	// JSON対応後は、空のconfigでもOutputを返す（decision=""でallow）
	if output == nil {
		t.Fatal("Expected output, got nil")
	}

	if output.Decision != "" {
		t.Errorf("Expected decision empty (allow), got %q", output.Decision)
	}

	if output.Continue != true {
		t.Errorf("Expected continue true, got %v", output.Continue)
	}
}

func TestExecutePreCompactHooks(t *testing.T) {
	config := &Config{}
	input := &PreCompactInput{}

	output, err := executePreCompactHooksJSON(config, input, nil)
	if err != nil {
		t.Errorf("executePreCompactHooksJSON() error = %v, expected nil", err)
	}
	if output == nil {
		t.Fatal("executePreCompactHooksJSON() returned nil output")
	}
	if !output.Continue {
		t.Error("executePreCompactHooksJSON() Continue = false, expected true")
	}
}

func TestExecutePreCompactHooksJSON(t *testing.T) {
	tests := []struct {
		name              string
		config            *Config
		input             *PreCompactInput
		rawJSON           interface{}
		wantContinue      bool
		wantSystemMessage string
		wantErr           bool
	}{
		{
			name:         "1. No hooks configured",
			config:       &Config{},
			input:        &PreCompactInput{BaseInput: BaseInput{SessionID: "test", HookEventName: PreCompact}, Trigger: "manual"},
			rawJSON:      map[string]interface{}{},
			wantContinue: true,
			wantErr:      false,
		},
		{
			name: "2. Output action with message",
			config: &Config{
				PreCompact: []PreCompactHook{
					{
						Actions: []Action{
							{
								Type:    "output",
								Message: "Pre-compaction processing",
							},
						},
					},
				},
			},
			input:             &PreCompactInput{BaseInput: BaseInput{SessionID: "test", HookEventName: PreCompact}, Trigger: "manual"},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantSystemMessage: "Pre-compaction processing",
			wantErr:           false,
		},
		{
			name: "3. Multiple actions - systemMessage concatenated",
			config: &Config{
				PreCompact: []PreCompactHook{
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
			input:             &PreCompactInput{BaseInput: BaseInput{SessionID: "test", HookEventName: PreCompact}, Trigger: "auto"},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantSystemMessage: "First message\nSecond message",
			wantErr:           false,
		},
		{
			name: "4. Matcher check - manual matches",
			config: &Config{
				PreCompact: []PreCompactHook{
					{
						Matcher: "manual",
						Actions: []Action{
							{
								Type:    "output",
								Message: "Manual trigger action",
							},
						},
					},
				},
			},
			input:             &PreCompactInput{BaseInput: BaseInput{SessionID: "test", HookEventName: PreCompact}, Trigger: "manual"},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantSystemMessage: "Manual trigger action",
			wantErr:           false,
		},
		{
			name: "5. Matcher check - auto matches",
			config: &Config{
				PreCompact: []PreCompactHook{
					{
						Matcher: "auto",
						Actions: []Action{
							{
								Type:    "output",
								Message: "Auto trigger action",
							},
						},
					},
				},
			},
			input:             &PreCompactInput{BaseInput: BaseInput{SessionID: "test", HookEventName: PreCompact}, Trigger: "auto"},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantSystemMessage: "Auto trigger action",
			wantErr:           false,
		},
		{
			name: "6. Matcher check - no match (should skip)",
			config: &Config{
				PreCompact: []PreCompactHook{
					{
						Matcher: "manual",
						Actions: []Action{
							{
								Type:    "output",
								Message: "Should not appear",
							},
						},
					},
				},
			},
			input:             &PreCompactInput{BaseInput: BaseInput{SessionID: "test", HookEventName: PreCompact}, Trigger: "auto"},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "7. Matcher check - empty matcher (should execute)",
			config: &Config{
				PreCompact: []PreCompactHook{
					{
						Matcher: "",
						Actions: []Action{
							{
								Type:    "output",
								Message: "Empty matcher executes",
							},
						},
					},
				},
			},
			input:             &PreCompactInput{BaseInput: BaseInput{SessionID: "test", HookEventName: PreCompact}, Trigger: "manual"},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantSystemMessage: "Empty matcher executes",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executePreCompactHooksJSON(tt.config, tt.input, tt.rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("executePreCompactHooksJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if output == nil {
				t.Fatal("executePreCompactHooksJSON() returned nil output")
			}

			if output.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
			}

			if output.SystemMessage != tt.wantSystemMessage {
				t.Errorf("SystemMessage = %q, want %q", output.SystemMessage, tt.wantSystemMessage)
			}
		})
	}
}

func TestExecuteSessionEndHooksJSON(t *testing.T) {
	tests := []struct {
		name              string
		config            *Config
		input             *SessionEndInput
		rawJSON           interface{}
		wantContinue      bool
		wantSystemMessage string
		wantErr           bool
		wantErrContains   string
	}{
		{
			name:         "1. No hooks configured - allow session end",
			config:       &Config{},
			input:        &SessionEndInput{BaseInput: BaseInput{SessionID: "test", HookEventName: SessionEnd}, Reason: "clear"},
			rawJSON:      map[string]interface{}{},
			wantContinue: true,
			wantErr:      false,
		},
		{
			name: "2. Output action with message",
			config: &Config{
				SessionEnd: []SessionEndHook{
					{
						Actions: []Action{
							{
								Type:    "output",
								Message: "Session cleanup completed",
							},
						},
					},
				},
			},
			input:             &SessionEndInput{BaseInput: BaseInput{SessionID: "test", HookEventName: SessionEnd}, Reason: "clear"},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantSystemMessage: "Session cleanup completed",
			wantErr:           false,
		},
		{
			name: "3. Multiple actions - systemMessage concatenated",
			config: &Config{
				SessionEnd: []SessionEndHook{
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
			input:             &SessionEndInput{BaseInput: BaseInput{SessionID: "test", HookEventName: SessionEnd}, Reason: "logout"},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantSystemMessage: "First message\nSecond message",
			wantErr:           false,
		},
		{
			name: "4. Condition matches (reason_is)",
			config: &Config{
				SessionEnd: []SessionEndHook{
					{
						Conditions: []Condition{
							{Type: ConditionReasonIs, Value: "clear"},
						},
						Actions: []Action{
							{
								Type:    "output",
								Message: "Clear cleanup",
							},
						},
					},
				},
			},
			input:             &SessionEndInput{BaseInput: BaseInput{SessionID: "test", HookEventName: SessionEnd}, Reason: "clear"},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantSystemMessage: "Clear cleanup",
			wantErr:           false,
		},
		{
			name: "5. Condition doesn't match - no action executed",
			config: &Config{
				SessionEnd: []SessionEndHook{
					{
						Conditions: []Condition{
							{Type: ConditionReasonIs, Value: "logout"},
						},
						Actions: []Action{
							{
								Type:    "output",
								Message: "Should not appear",
							},
						},
					},
				},
			},
			input:             &SessionEndInput{BaseInput: BaseInput{SessionID: "test", HookEventName: SessionEnd}, Reason: "clear"},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "6. Action fails - fail-safe (continue=true, systemMessage=error)",
			config: &Config{
				SessionEnd: []SessionEndHook{
					{
						Actions: []Action{
							{
								Type:    "output",
								Message: "", // Empty message triggers error
							},
						},
					},
				},
			},
			input:             &SessionEndInput{BaseInput: BaseInput{SessionID: "test", HookEventName: SessionEnd}, Reason: "clear"},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantSystemMessage: "Empty message in SessionEnd action",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeSessionEndHooksJSON(tt.config, tt.input, tt.rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("executeSessionEndHooksJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("Error message = %q, want to contain %q", err.Error(), tt.wantErrContains)
				}
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil SessionEndOutput, got nil")
			}

			if result.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", result.Continue, tt.wantContinue)
			}

			if result.SystemMessage != tt.wantSystemMessage {
				t.Errorf("SystemMessage = %q, want %q", result.SystemMessage, tt.wantSystemMessage)
			}
		})
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

func TestExecuteUserPromptSubmitHooks_ErrorHandling(t *testing.T) {
	tests := []struct {
		name              string
		config            *Config
		input             *UserPromptSubmitInput
		wantContinue      bool
		wantDecision      string
		wantHookEventName string
		wantSystemMessage string // Expected substring in SystemMessage
		wantErr           bool
	}{
		{
			name: "Condition error does not block prompt (decision remains empty)",
			config: &Config{
				UserPromptSubmit: []UserPromptSubmitHook{
					{
						Conditions: []Condition{
							{
								Type:  ConditionReasonIs, // Invalid for UserPromptSubmit
								Value: "test",
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
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: "/path/to/transcript",
					HookEventName:  UserPromptSubmit,
				},
				Prompt: "test prompt",
			},
			wantContinue:      true,               // Continue is always true for UserPromptSubmit
			wantDecision:      "",                 // Condition error does not block (decision field omitted to allow prompt)
			wantHookEventName: "UserPromptSubmit", // Always set
			wantSystemMessage: "",                 // Condition errors are not included in SystemMessage
			wantErr:           true,               // Error is returned (but does not block)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawJSON := map[string]interface{}{
				"session_id":      tt.input.SessionID,
				"transcript_path": tt.input.TranscriptPath,
				"hook_event_name": string(tt.input.HookEventName),
				"prompt":          tt.input.Prompt,
			}

			output, err := executeUserPromptSubmitHooks(tt.config, tt.input, rawJSON)

			// Error check
			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Continue check (always true for UserPromptSubmit)
			if output.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
			}

			// Decision check
			if output.Decision != tt.wantDecision {
				t.Errorf("Decision = %v, want %v", output.Decision, tt.wantDecision)
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

// TestExecuteStopHooksJSON tests the executeStopHooks function with JSON output
// (new signature returning (*StopOutput, error)).
// Stop uses top-level decision pattern (no hookSpecificOutput).
func TestExecuteStopHooksJSON(t *testing.T) {
	tests := []struct {
		name              string
		config            *Config
		input             *StopInput
		rawJSON           interface{}
		wantContinue      bool
		wantDecision      string
		wantReason        string
		wantSystemMessage string
		wantErr           bool
		wantErrContains   string
	}{
		{
			name:         "1. No hooks configured - allow stop",
			config:       &Config{},
			input:        &StopInput{BaseInput: BaseInput{SessionID: "test", HookEventName: Stop}},
			rawJSON:      map[string]interface{}{},
			wantContinue: true,
			wantDecision: "",
			wantErr:      false,
		},
		{
			name: "2. Output action with decision block",
			config: &Config{
				Stop: []StopHook{
					{
						Actions: []Action{
							{
								Type:     "output",
								Message:  "Stop is blocked",
								Decision: stringPtr("block"),
								Reason:   stringPtr("Not safe to stop"),
							},
						},
					},
				},
			},
			input:             &StopInput{BaseInput: BaseInput{SessionID: "test", HookEventName: Stop}},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantDecision:      "block",
			wantReason:        "Not safe to stop",
			wantSystemMessage: "Stop is blocked",
			wantErr:           false,
		},
		{
			name: "3. Output action with explicit allow (decision empty string)",
			config: &Config{
				Stop: []StopHook{
					{
						Actions: []Action{
							{
								Type:     "output",
								Message:  "Stop allowed",
								Decision: stringPtr(""),
							},
						},
					},
				},
			},
			input:             &StopInput{BaseInput: BaseInput{SessionID: "test", HookEventName: Stop}},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantDecision:      "",
			wantReason:        "",
			wantSystemMessage: "Stop allowed",
			wantErr:           false,
		},
		{
			name: "4. Output action without decision (defaults to allow)",
			config: &Config{
				Stop: []StopHook{
					{
						Actions: []Action{
							{
								Type:    "output",
								Message: "Default allow message",
							},
						},
					},
				},
			},
			input:             &StopInput{BaseInput: BaseInput{SessionID: "test", HookEventName: Stop}},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantDecision:      "",
			wantReason:        "",
			wantSystemMessage: "Default allow message",
			wantErr:           false,
		},
		{
			name: "5. Multiple actions - decision last wins (allow then block)",
			config: &Config{
				Stop: []StopHook{
					{
						Actions: []Action{
							{
								Type:     "output",
								Message:  "First allows",
								Decision: stringPtr(""),
							},
							{
								Type:     "output",
								Message:  "Second blocks",
								Decision: stringPtr("block"),
								Reason:   stringPtr("Second reason"),
							},
						},
					},
				},
			},
			input:             &StopInput{BaseInput: BaseInput{SessionID: "test", HookEventName: Stop}},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantDecision:      "block",
			wantReason:        "Second reason",
			wantSystemMessage: "First allows\nSecond blocks",
			wantErr:           false,
		},
		{
			name: "6. Early return on decision block",
			config: &Config{
				Stop: []StopHook{
					{
						Actions: []Action{
							{
								Type:     "output",
								Message:  "Block stop",
								Decision: stringPtr("block"),
								Reason:   stringPtr("Blocked reason"),
							},
							{
								Type:     "output",
								Message:  "Should not execute",
								Decision: stringPtr(""),
							},
						},
					},
				},
			},
			input:             &StopInput{BaseInput: BaseInput{SessionID: "test", HookEventName: Stop}},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantDecision:      "block",
			wantReason:        "Blocked reason",
			wantSystemMessage: "Block stop",
			wantErr:           false,
		},
		{
			name: "7. SystemMessage concatenation with newline",
			config: &Config{
				Stop: []StopHook{
					{
						Actions: []Action{
							{
								Type:     "output",
								Message:  "First system msg",
								Decision: stringPtr(""),
							},
							{
								Type:     "output",
								Message:  "Second system msg",
								Decision: stringPtr(""),
							},
						},
					},
				},
			},
			input:             &StopInput{BaseInput: BaseInput{SessionID: "test", HookEventName: Stop}},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantDecision:      "",
			wantReason:        "",
			wantSystemMessage: "First system msg\nSecond system msg",
			wantErr:           false,
		},
		{
			name: "8. Action error - fail-safe block",
			config: &Config{
				Stop: []StopHook{
					{
						Actions: []Action{
							{
								Type:    "output",
								Message: "", // Empty message triggers error in ExecuteStopAction
							},
						},
					},
				},
			},
			input:        &StopInput{BaseInput: BaseInput{SessionID: "test", HookEventName: Stop}},
			rawJSON:      map[string]interface{}{},
			wantContinue: true,
			wantDecision: "block",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeStopHooks(tt.config, tt.input, tt.rawJSON)

			// Error check
			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("Expected error containing %q, got: %v", tt.wantErrContains, err)
				}
			}

			// Output should always be non-nil
			if output == nil {
				t.Fatal("Expected output, got nil")
			}

			// Continue is always true for Stop
			if output.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
			}

			// Decision check
			if output.Decision != tt.wantDecision {
				t.Errorf("Decision = %q, want %q", output.Decision, tt.wantDecision)
			}

			// Reason check
			if tt.wantReason != "" && output.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", output.Reason, tt.wantReason)
			}

			// SystemMessage check
			if tt.wantSystemMessage != "" && output.SystemMessage != tt.wantSystemMessage {
				t.Errorf("SystemMessage = %q, want %q", output.SystemMessage, tt.wantSystemMessage)
			}
		})
	}
}

func TestExecuteSubagentStopHooksJSON(t *testing.T) {
	tests := []struct {
		name              string
		config            *Config
		input             *SubagentStopInput
		rawJSON           interface{}
		wantContinue      bool
		wantDecision      string
		wantReason        string
		wantSystemMessage string
		wantErr           bool
		wantErrContains   string
	}{
		{
			name:         "1. No hooks configured - allow subagent stop",
			config:       &Config{},
			input:        &SubagentStopInput{BaseInput: BaseInput{SessionID: "test", HookEventName: SubagentStop}},
			rawJSON:      map[string]interface{}{},
			wantContinue: true,
			wantDecision: "",
			wantErr:      false,
		},
		{
			name: "2. Output action with decision block",
			config: &Config{
				SubagentStop: []SubagentStopHook{
					{
						Actions: []Action{
							{
								Type:     "output",
								Message:  "SubagentStop is blocked",
								Decision: stringPtr("block"),
								Reason:   stringPtr("Not safe to stop subagent"),
							},
						},
					},
				},
			},
			input:             &SubagentStopInput{BaseInput: BaseInput{SessionID: "test", HookEventName: SubagentStop}},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantDecision:      "block",
			wantReason:        "Not safe to stop subagent",
			wantSystemMessage: "SubagentStop is blocked",
			wantErr:           false,
		},
		{
			name: "3. Output action with explicit allow (decision empty string)",
			config: &Config{
				SubagentStop: []SubagentStopHook{
					{
						Actions: []Action{
							{
								Type:     "output",
								Message:  "SubagentStop allowed",
								Decision: stringPtr(""),
							},
						},
					},
				},
			},
			input:             &SubagentStopInput{BaseInput: BaseInput{SessionID: "test", HookEventName: SubagentStop}},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantDecision:      "",
			wantReason:        "",
			wantSystemMessage: "SubagentStop allowed",
			wantErr:           false,
		},
		{
			name: "4. Output action without decision (defaults to allow)",
			config: &Config{
				SubagentStop: []SubagentStopHook{
					{
						Actions: []Action{
							{
								Type:    "output",
								Message: "Default allow message",
							},
						},
					},
				},
			},
			input:             &SubagentStopInput{BaseInput: BaseInput{SessionID: "test", HookEventName: SubagentStop}},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantDecision:      "",
			wantReason:        "",
			wantSystemMessage: "Default allow message",
			wantErr:           false,
		},
		{
			name: "5. Multiple actions - decision last wins (allow then block)",
			config: &Config{
				SubagentStop: []SubagentStopHook{
					{
						Actions: []Action{
							{
								Type:     "output",
								Message:  "First allows",
								Decision: stringPtr(""),
							},
							{
								Type:     "output",
								Message:  "Second blocks",
								Decision: stringPtr("block"),
								Reason:   stringPtr("Second reason"),
							},
						},
					},
				},
			},
			input:             &SubagentStopInput{BaseInput: BaseInput{SessionID: "test", HookEventName: SubagentStop}},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantDecision:      "block",
			wantReason:        "Second reason",
			wantSystemMessage: "First allows\nSecond blocks",
			wantErr:           false,
		},
		{
			name: "6. Early return on decision block",
			config: &Config{
				SubagentStop: []SubagentStopHook{
					{
						Actions: []Action{
							{
								Type:     "output",
								Message:  "Block subagent stop",
								Decision: stringPtr("block"),
								Reason:   stringPtr("Blocked reason"),
							},
							{
								Type:     "output",
								Message:  "Should not execute",
								Decision: stringPtr(""),
							},
						},
					},
				},
			},
			input:             &SubagentStopInput{BaseInput: BaseInput{SessionID: "test", HookEventName: SubagentStop}},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantDecision:      "block",
			wantReason:        "Blocked reason",
			wantSystemMessage: "Block subagent stop",
			wantErr:           false,
		},
		{
			name: "7. SystemMessage concatenation with newline",
			config: &Config{
				SubagentStop: []SubagentStopHook{
					{
						Actions: []Action{
							{
								Type:     "output",
								Message:  "First system msg",
								Decision: stringPtr(""),
							},
							{
								Type:     "output",
								Message:  "Second system msg",
								Decision: stringPtr(""),
							},
						},
					},
				},
			},
			input:             &SubagentStopInput{BaseInput: BaseInput{SessionID: "test", HookEventName: SubagentStop}},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantDecision:      "",
			wantReason:        "",
			wantSystemMessage: "First system msg\nSecond system msg",
			wantErr:           false,
		},
		{
			name: "8. Action error - fail-safe block",
			config: &Config{
				SubagentStop: []SubagentStopHook{
					{
						Actions: []Action{
							{
								Type:    "output",
								Message: "", // Empty message triggers error in ExecuteSubagentStopAction
							},
						},
					},
				},
			},
			input:        &SubagentStopInput{BaseInput: BaseInput{SessionID: "test", HookEventName: SubagentStop}},
			rawJSON:      map[string]interface{}{},
			wantContinue: true,
			wantDecision: "block",
			wantReason:   "Empty message in SubagentStop action",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeSubagentStopHooks(tt.config, tt.input, tt.rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("executeSubagentStopHooks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Errorf("error message should contain %q, got %q", tt.wantErrContains, err.Error())
			}

			if output == nil && !tt.wantErr {
				t.Fatal("executeSubagentStopHooks() returned nil output")
			}

			if output != nil {
				if output.Continue != tt.wantContinue {
					t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
				}

				if output.Decision != tt.wantDecision {
					t.Errorf("Decision = %q, want %q", output.Decision, tt.wantDecision)
				}

				if output.Reason != tt.wantReason {
					t.Errorf("Reason = %q, want %q", output.Reason, tt.wantReason)
				}

				if tt.wantSystemMessage != "" && output.SystemMessage != tt.wantSystemMessage {
					t.Errorf("SystemMessage = %q, want %q", output.SystemMessage, tt.wantSystemMessage)
				}
			}
		})
	}
}

func TestExecuteNotificationHooksJSON(t *testing.T) {
	tests := []struct {
		name                  string
		config                *Config
		input                 *NotificationInput
		rawJSON               map[string]interface{}
		wantContinue          bool
		wantHookEventName     string
		wantAdditionalContext string
		wantSystemMessage     string
		wantStopReason        string
		wantSuppressOutput    bool
		wantErr               bool
		wantErrContains       string
	}{
		{
			name: "No hooks - continue true",
			config: &Config{
				Notification: []NotificationHook{},
			},
			input:             &NotificationInput{BaseInput: BaseInput{SessionID: "test", HookEventName: "Notification"}},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantHookEventName: "Notification",
			wantErr:           false,
		},
		{
			name: "Single action - output type",
			config: &Config{
				Notification: []NotificationHook{
					{
						Actions: []Action{
							{
								Type:    "output",
								Message: "Test notification",
							},
						},
					},
				},
			},
			input:                 &NotificationInput{BaseInput: BaseInput{SessionID: "test", HookEventName: "Notification"}},
			rawJSON:               map[string]interface{}{},
			wantContinue:          true,
			wantHookEventName:     "Notification",
			wantAdditionalContext: "Test notification",
			wantErr:               false,
		},
		{
			name: "Multiple actions - additionalContext concatenated",
			config: &Config{
				Notification: []NotificationHook{
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
			input:                 &NotificationInput{BaseInput: BaseInput{SessionID: "test", HookEventName: "Notification"}},
			rawJSON:               map[string]interface{}{},
			wantContinue:          true,
			wantHookEventName:     "Notification",
			wantAdditionalContext: "First message\nSecond message",
			wantErr:               false,
		},
		{
			name: "SystemMessage concatenation",
			config: &Config{
				Notification: []NotificationHook{
					{
						Actions: []Action{
							{
								Type:    "output",
								Message: "Message with system warning",
							},
						},
					},
				},
			},
			input:                 &NotificationInput{BaseInput: BaseInput{SessionID: "test", HookEventName: "Notification"}},
			rawJSON:               map[string]interface{}{},
			wantContinue:          true,
			wantHookEventName:     "Notification",
			wantAdditionalContext: "Message with system warning",
			wantErr:               false,
		},
		{
			name: "Matcher - exact match",
			config: &Config{
				Notification: []NotificationHook{
					{
						Matcher: "idle_prompt",
						Actions: []Action{
							{
								Type:    "output",
								Message: "Idle detected",
							},
						},
					},
				},
			},
			input: &NotificationInput{
				BaseInput:        BaseInput{SessionID: "test", HookEventName: "Notification"},
				NotificationType: "idle_prompt",
			},
			rawJSON:               map[string]interface{}{},
			wantContinue:          true,
			wantHookEventName:     "Notification",
			wantAdditionalContext: "Idle detected",
			wantErr:               false,
		},
		{
			name: "Matcher - no match (hook skipped)",
			config: &Config{
				Notification: []NotificationHook{
					{
						Matcher: "permission_prompt",
						Actions: []Action{
							{
								Type:    "output",
								Message: "Permission required",
							},
						},
					},
				},
			},
			input: &NotificationInput{
				BaseInput:        BaseInput{SessionID: "test", HookEventName: "Notification"},
				NotificationType: "idle_prompt",
			},
			rawJSON:               map[string]interface{}{},
			wantContinue:          true,
			wantHookEventName:     "Notification",
			wantAdditionalContext: "",
			wantErr:               false,
		},
		{
			name: "Matcher - pipe-separated OR",
			config: &Config{
				Notification: []NotificationHook{
					{
						Matcher: "idle_prompt|permission_prompt",
						Actions: []Action{
							{
								Type:    "output",
								Message: "Matched",
							},
						},
					},
				},
			},
			input: &NotificationInput{
				BaseInput:        BaseInput{SessionID: "test", HookEventName: "Notification"},
				NotificationType: "permission_prompt",
			},
			rawJSON:               map[string]interface{}{},
			wantContinue:          true,
			wantHookEventName:     "Notification",
			wantAdditionalContext: "Matched",
			wantErr:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeNotificationHooksJSON(tt.config, tt.input, tt.rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("executeNotificationHooksJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Errorf("error message should contain %q, got %q", tt.wantErrContains, err.Error())
			}

			if output == nil && !tt.wantErr {
				t.Fatal("executeNotificationHooksJSON() returned nil output")
			}

			if output != nil {
				if output.Continue != tt.wantContinue {
					t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
				}

				if output.HookSpecificOutput != nil {
					if output.HookSpecificOutput.HookEventName != tt.wantHookEventName {
						t.Errorf("HookEventName = %q, want %q", output.HookSpecificOutput.HookEventName, tt.wantHookEventName)
					}

					if output.HookSpecificOutput.AdditionalContext != tt.wantAdditionalContext {
						t.Errorf("AdditionalContext = %q, want %q", output.HookSpecificOutput.AdditionalContext, tt.wantAdditionalContext)
					}
				}

				if tt.wantSystemMessage != "" && output.SystemMessage != tt.wantSystemMessage {
					t.Errorf("SystemMessage = %q, want %q", output.SystemMessage, tt.wantSystemMessage)
				}

				if tt.wantStopReason != "" && output.StopReason != tt.wantStopReason {
					t.Errorf("StopReason = %q, want %q", output.StopReason, tt.wantStopReason)
				}

				if output.SuppressOutput != tt.wantSuppressOutput {
					t.Errorf("SuppressOutput = %v, want %v", output.SuppressOutput, tt.wantSuppressOutput)
				}
			}
		})
	}
}

func TestExecuteSubagentStartHooksJSON(t *testing.T) {
	tests := []struct {
		name                  string
		config                *Config
		input                 *SubagentStartInput
		rawJSON               map[string]interface{}
		wantContinue          bool
		wantHookEventName     string
		wantAdditionalContext string
		wantSystemMessage     string
		wantStopReason        string
		wantSuppressOutput    bool
		wantErr               bool
		wantErrContains       string
	}{
		{
			name: "No hooks - continue true",
			config: &Config{
				SubagentStart: []SubagentStartHook{},
			},
			input:             &SubagentStartInput{BaseInput: BaseInput{SessionID: "test", HookEventName: "SubagentStart"}, AgentType: "Explore"},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantHookEventName: "SubagentStart",
			wantErr:           false,
		},
		{
			name: "Single action - output type",
			config: &Config{
				SubagentStart: []SubagentStartHook{
					{
						Matcher: "Explore",
						Actions: []Action{
							{
								Type:    "output",
								Message: "Explore agent started",
							},
						},
					},
				},
			},
			input:                 &SubagentStartInput{BaseInput: BaseInput{SessionID: "test", HookEventName: "SubagentStart"}, AgentType: "Explore"},
			rawJSON:               map[string]interface{}{},
			wantContinue:          true,
			wantHookEventName:     "SubagentStart",
			wantAdditionalContext: "Explore agent started",
			wantErr:               false,
		},
		{
			name: "Matcher mismatch - hook not executed",
			config: &Config{
				SubagentStart: []SubagentStartHook{
					{
						Matcher: "Plan",
						Actions: []Action{
							{
								Type:    "output",
								Message: "Plan agent started",
							},
						},
					},
				},
			},
			input:             &SubagentStartInput{BaseInput: BaseInput{SessionID: "test", HookEventName: "SubagentStart"}, AgentType: "Explore"},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantHookEventName: "SubagentStart",
			wantErr:           false,
		},
		{
			name: "Matcher partial match - hook executed",
			config: &Config{
				SubagentStart: []SubagentStartHook{
					{
						Matcher: "Exp",
						Actions: []Action{
							{
								Type:    "output",
								Message: "Agent started",
							},
						},
					},
				},
			},
			input:                 &SubagentStartInput{BaseInput: BaseInput{SessionID: "test", HookEventName: "SubagentStart"}, AgentType: "Explore"},
			rawJSON:               map[string]interface{}{},
			wantContinue:          true,
			wantHookEventName:     "SubagentStart",
			wantAdditionalContext: "Agent started",
			wantErr:               false,
		},
		{
			name: "Matcher pipe-separated OR - first matches",
			config: &Config{
				SubagentStart: []SubagentStartHook{
					{
						Matcher: "Explore|Plan|Bash",
						Actions: []Action{
							{
								Type:    "output",
								Message: "Known agent started",
							},
						},
					},
				},
			},
			input:                 &SubagentStartInput{BaseInput: BaseInput{SessionID: "test", HookEventName: "SubagentStart"}, AgentType: "Explore"},
			rawJSON:               map[string]interface{}{},
			wantContinue:          true,
			wantHookEventName:     "SubagentStart",
			wantAdditionalContext: "Known agent started",
			wantErr:               false,
		},
		{
			name: "Multiple actions - additionalContext concatenated",
			config: &Config{
				SubagentStart: []SubagentStartHook{
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
			input:                 &SubagentStartInput{BaseInput: BaseInput{SessionID: "test", HookEventName: "SubagentStart"}, AgentType: "Explore"},
			rawJSON:               map[string]interface{}{},
			wantContinue:          true,
			wantHookEventName:     "SubagentStart",
			wantAdditionalContext: "First message\nSecond message",
			wantErr:               false,
		},
		{
			name: "Condition check - cwd_is matched",
			config: &Config{
				SubagentStart: []SubagentStartHook{
					{
						Conditions: []Condition{
							{Type: ConditionCwdIs, Value: "/test/path"},
						},
						Actions: []Action{
							{
								Type:    "output",
								Message: "In correct directory",
							},
						},
					},
				},
			},
			input:                 &SubagentStartInput{BaseInput: BaseInput{SessionID: "test", HookEventName: "SubagentStart", Cwd: "/test/path"}, AgentType: "Explore"},
			rawJSON:               map[string]interface{}{},
			wantContinue:          true,
			wantHookEventName:     "SubagentStart",
			wantAdditionalContext: "In correct directory",
			wantErr:               false,
		},
		{
			name: "Condition check - cwd_is not matched",
			config: &Config{
				SubagentStart: []SubagentStartHook{
					{
						Conditions: []Condition{
							{Type: ConditionCwdIs, Value: "/nonexistent/path"},
						},
						Actions: []Action{
							{
								Type:    "output",
								Message: "Should not execute",
							},
						},
					},
				},
			},
			input:             &SubagentStartInput{BaseInput: BaseInput{SessionID: "test", HookEventName: "SubagentStart", Cwd: "/test/path"}, AgentType: "Explore"},
			rawJSON:           map[string]interface{}{},
			wantContinue:      true,
			wantHookEventName: "SubagentStart",
			wantErr:           false,
		},
		{
			name: "Continue forced to true even if action returns false",
			config: &Config{
				SubagentStart: []SubagentStartHook{
					{
						Actions: []Action{
							{
								Type:     "output",
								Message:  "Test message",
								Continue: boolPtr(false),
							},
						},
					},
				},
			},
			input:                 &SubagentStartInput{BaseInput: BaseInput{SessionID: "test", HookEventName: "SubagentStart"}, AgentType: "Explore"},
			rawJSON:               map[string]interface{}{},
			wantContinue:          true, // Forced to true
			wantHookEventName:     "SubagentStart",
			wantAdditionalContext: "Test message",
			wantErr:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeSubagentStartHooksJSON(tt.config, tt.input, tt.rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("executeSubagentStartHooksJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Errorf("error message should contain %q, got %q", tt.wantErrContains, err.Error())
			}

			if output == nil && !tt.wantErr {
				t.Fatal("executeSubagentStartHooksJSON() returned nil output")
			}

			if output != nil {
				if output.Continue != tt.wantContinue {
					t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
				}

				if output.HookSpecificOutput == nil {
					t.Fatal("HookSpecificOutput is nil")
				}

				if output.HookSpecificOutput.HookEventName != tt.wantHookEventName {
					t.Errorf("HookEventName = %q, want %q", output.HookSpecificOutput.HookEventName, tt.wantHookEventName)
				}

				if output.HookSpecificOutput.AdditionalContext != tt.wantAdditionalContext {
					t.Errorf("AdditionalContext = %q, want %q", output.HookSpecificOutput.AdditionalContext, tt.wantAdditionalContext)
				}

				if tt.wantSystemMessage != "" && output.SystemMessage != tt.wantSystemMessage {
					t.Errorf("SystemMessage = %q, want %q", output.SystemMessage, tt.wantSystemMessage)
				}

				if tt.wantStopReason != "" && output.StopReason != tt.wantStopReason {
					t.Errorf("StopReason = %q, want %q", output.StopReason, tt.wantStopReason)
				}

				if output.SuppressOutput != tt.wantSuppressOutput {
					t.Errorf("SuppressOutput = %v, want %v", output.SuppressOutput, tt.wantSuppressOutput)
				}
			}
		})
	}
}
