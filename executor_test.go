package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestExecuteSessionStartAction_TypeOutput(t *testing.T) {
	tests := []struct {
		name              string
		action            Action
		wantContinue      bool
		wantHookEventName string
		wantAdditionalCtx string
		wantSystemMessage string
		wantErr           bool
	}{
		{
			name: "Message with continue unspecified defaults to true",
			action: Action{
				Type:     "output",
				Message:  "Test message",
				Continue: nil,
			},
			wantContinue:      true,
			wantHookEventName: "SessionStart",
			wantAdditionalCtx: "Test message",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Message with continue: false",
			action: Action{
				Type:     "output",
				Message:  "Stop message",
				Continue: boolPtr(false),
			},
			wantContinue:      false,
			wantHookEventName: "SessionStart",
			wantAdditionalCtx: "Stop message",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Message with continue: true explicitly",
			action: Action{
				Type:     "output",
				Message:  "Continue message",
				Continue: boolPtr(true),
			},
			wantContinue:      true,
			wantHookEventName: "SessionStart",
			wantAdditionalCtx: "Continue message",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Message with template variables",
			action: Action{
				Type:     "output",
				Message:  "Session ID: {.session_id}",
				Continue: nil,
			},
			wantContinue:      true,
			wantHookEventName: "SessionStart",
			wantAdditionalCtx: "Session ID: test-session-123",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Empty message returns continue: false with systemMessage",
			action: Action{
				Type:     "output",
				Message:  "",
				Continue: nil,
			},
			wantContinue:      false,
			wantHookEventName: "",
			wantAdditionalCtx: "",
			wantSystemMessage: "Action output has no message",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewActionExecutor(nil)
			input := &SessionStartInput{
				BaseInput: BaseInput{
					SessionID: "test-session-123",
				},
			}
			rawJSON := map[string]any{
				"session_id": "test-session-123",
			}

			output, err := executor.ExecuteSessionStartAction(tt.action, input, rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteSessionStartAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if output == nil {
				t.Fatal("ExecuteSessionStartAction() returned nil output")
			}

			if output.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
			}

			if output.HookEventName != tt.wantHookEventName {
				t.Errorf("HookEventName = %q, want %q", output.HookEventName, tt.wantHookEventName)
			}

			if output.AdditionalContext != tt.wantAdditionalCtx {
				t.Errorf("AdditionalContext = %q, want %q", output.AdditionalContext, tt.wantAdditionalCtx)
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

func TestExecuteSessionStartAction_TypeCommand(t *testing.T) {
	tests := []struct {
		name              string
		action            Action
		stdout            string
		stderr            string
		exitCode          int
		stubErr           error
		wantContinue      bool
		wantHookEventName string
		wantAdditionalCtx string
		wantSystemMessage string
		wantErr           bool
	}{
		{
			name: "Command success with valid JSON and all fields",
			action: Action{
				Type:    "command",
				Command: "get-session-info.sh",
			},
			stdout: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "SessionStart",
					"additionalContext": "Session initialized successfully"
				},
				"systemMessage": "Debug: initialization complete"
			}`,
			stderr:            "",
			exitCode:          0,
			wantContinue:      true,
			wantHookEventName: "SessionStart",
			wantAdditionalCtx: "Session initialized successfully",
			wantSystemMessage: "Debug: initialization complete",
			wantErr:           false,
		},
		{
			name: "Command with hookEventName SessionStart",
			action: Action{
				Type:    "command",
				Command: "echo-session-start.sh",
			},
			stdout: `{
				"hookSpecificOutput": {
					"hookEventName": "SessionStart"
				}
			}`,
			stderr:            "",
			exitCode:          0,
			wantContinue:      false, // continue unspecified defaults to false
			wantHookEventName: "SessionStart",
			wantAdditionalCtx: "",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Command with continue unspecified defaults to false",
			action: Action{
				Type:    "command",
				Command: "minimal-output.sh",
			},
			stdout: `{
				"hookSpecificOutput": {
					"hookEventName": "SessionStart"
				}
			}`,
			stderr:            "",
			exitCode:          0,
			wantContinue:      false,
			wantHookEventName: "SessionStart",
			wantAdditionalCtx: "",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Command failure with exit != 0",
			action: Action{
				Type:    "command",
				Command: "failing-command.sh",
			},
			stdout:            "",
			stderr:            "Permission denied",
			exitCode:          1,
			wantContinue:      false,
			wantHookEventName: "",
			wantAdditionalCtx: "",
			wantSystemMessage: "Command failed with exit code 1: Permission denied",
			wantErr:           false,
		},
		{
			name: "Empty stdout - validation tool success (requirement 1.6, 3.7)",
			action: Action{
				Type:    "command",
				Command: "empty-output.sh",
			},
			stdout:            "",
			stderr:            "",
			exitCode:          0,
			wantContinue:      true, // Allow for validation-type CLI tools
			wantHookEventName: "SessionStart",
			wantAdditionalCtx: "", // No context provided to Claude
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Invalid JSON output",
			action: Action{
				Type:    "command",
				Command: "invalid-json.sh",
			},
			stdout:            `{"invalid": json}`,
			stderr:            "",
			exitCode:          0,
			wantContinue:      false,
			wantHookEventName: "",
			wantAdditionalCtx: "",
			wantSystemMessage: "Command output is not valid JSON: {\"invalid\": json}",
			wantErr:           false,
		},
		{
			name: "Missing hookEventName",
			action: Action{
				Type:    "command",
				Command: "missing-hook-event.sh",
			},
			stdout: `{
				"continue": true,
				"hookSpecificOutput": {
					"additionalContext": "Some context"
				}
			}`,
			stderr:            "",
			exitCode:          0,
			wantContinue:      false,
			wantHookEventName: "",
			wantAdditionalCtx: "",
			wantSystemMessage: "Command output is missing required field: hookSpecificOutput.hookEventName",
			wantErr:           false,
		},
		{
			name: "Missing hookSpecificOutput entirely",
			action: Action{
				Type:    "command",
				Command: "missing-hook-specific.sh",
			},
			stdout: `{
				"continue": true,
				"systemMessage": "Test"
			}`,
			stderr:            "",
			exitCode:          0,
			wantContinue:      false,
			wantHookEventName: "",
			wantAdditionalCtx: "",
			wantSystemMessage: "Command output is missing required field: hookSpecificOutput.hookEventName",
			wantErr:           false,
		},
		{
			name: "Command failure with err variable when stderr is empty",
			action: Action{
				Type:    "command",
				Command: "failing-command.sh",
			},
			stdout:            "",
			stderr:            "",
			exitCode:          1,
			stubErr:           fmt.Errorf("failed to marshal JSON for stdin: unsupported type"),
			wantContinue:      false,
			wantHookEventName: "",
			wantAdditionalCtx: "",
			wantSystemMessage: "Command failed with exit code 1: failed to marshal JSON for stdin: unsupported type",
			wantErr:           false,
		},
		{
			name: "Command failure with stderr takes precedence over err",
			action: Action{
				Type:    "command",
				Command: "failing-command.sh",
			},
			stdout:            "",
			stderr:            "explicit error from stderr",
			exitCode:          1,
			stubErr:           fmt.Errorf("exit status 1"),
			wantContinue:      false,
			wantHookEventName: "",
			wantAdditionalCtx: "",
			wantSystemMessage: "Command failed with exit code 1: explicit error from stderr",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunnerWithOutput{
				stdout:   tt.stdout,
				stderr:   tt.stderr,
				exitCode: tt.exitCode,
				err:      tt.stubErr,
			}
			executor := NewActionExecutor(runner)
			input := &SessionStartInput{
				BaseInput: BaseInput{
					SessionID: "test-session-123",
				},
			}
			rawJSON := map[string]any{
				"session_id": "test-session-123",
			}

			output, err := executor.ExecuteSessionStartAction(tt.action, input, rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteSessionStartAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if output == nil {
				t.Fatal("ExecuteSessionStartAction() returned nil output")
			}

			if output.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
			}

			if output.HookEventName != tt.wantHookEventName {
				t.Errorf("HookEventName = %q, want %q", output.HookEventName, tt.wantHookEventName)
			}

			if output.AdditionalContext != tt.wantAdditionalCtx {
				t.Errorf("AdditionalContext = %q, want %q", output.AdditionalContext, tt.wantAdditionalCtx)
			}

			if output.SystemMessage != tt.wantSystemMessage {
				t.Errorf("SystemMessage = %q, want %q", output.SystemMessage, tt.wantSystemMessage)
			}
		})
	}
}

func TestExecuteUserPromptSubmitAction_TypeOutput(t *testing.T) {
	tests := []struct {
		name              string
		action            Action
		wantContinue      bool
		wantDecision      string
		wantHookEventName string
		wantAdditionalCtx string
		wantSystemMessage string
		wantErr           bool
	}{
		{
			name: "Message with decision unspecified defaults to allow",
			action: Action{
				Type:     "output",
				Message:  "Test message",
				Decision: nil,
			},
			wantContinue:      true,
			wantDecision:      "",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "Test message",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Message with decision: block",
			action: Action{
				Type:     "output",
				Message:  "Blocked message",
				Decision: stringPtr("block"),
			},
			wantContinue:      true,
			wantDecision:      "block",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "Blocked message",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Message with invalid decision value",
			action: Action{
				Type:     "output",
				Message:  "Invalid decision",
				Decision: stringPtr("invalid"),
			},
			wantContinue:      true,
			wantDecision:      "block",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "",
			wantSystemMessage: "Invalid decision value in action config: must be 'block' or field must be omitted",
			wantErr:           false,
		},
		{
			name: "Message with template variables",
			action: Action{
				Type:     "output",
				Message:  "User prompt: {.prompt}",
				Decision: nil,
			},
			wantContinue:      true,
			wantDecision:      "",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "User prompt: test prompt",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Empty message returns block with error",
			action: Action{
				Type:     "output",
				Message:  "",
				Decision: nil,
			},
			wantContinue:      true,
			wantDecision:      "block",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "",
			wantSystemMessage: "Action output has no message",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &ActionExecutor{
				runner: &stubRunnerWithOutput{},
			}

			input := &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session-123",
					TranscriptPath: "/path/to/transcript",
					Cwd:            "/test/cwd",
					PermissionMode: "test",
					HookEventName:  UserPromptSubmit,
				},
				Prompt: "test prompt",
			}

			rawJSON := map[string]any{
				"session_id":      "test-session-123",
				"transcript_path": "/path/to/transcript",
				"cwd":             "/test/cwd",
				"permission_mode": "test",
				"hook_event_name": "UserPromptSubmit",
				"prompt":          "test prompt",
			}

			got, err := executor.ExecuteUserPromptSubmitAction(tt.action, input, rawJSON)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteUserPromptSubmitAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got.Continue != tt.wantContinue {
				t.Errorf("ExecuteUserPromptSubmitAction() Continue = %v, want %v", got.Continue, tt.wantContinue)
			}
			if got.Decision != tt.wantDecision {
				t.Errorf("ExecuteUserPromptSubmitAction() Decision = %v, want %v", got.Decision, tt.wantDecision)
			}
			if got.HookEventName != tt.wantHookEventName {
				t.Errorf("ExecuteUserPromptSubmitAction() HookEventName = %v, want %v", got.HookEventName, tt.wantHookEventName)
			}
			if got.AdditionalContext != tt.wantAdditionalCtx {
				t.Errorf("ExecuteUserPromptSubmitAction() AdditionalContext = %v, want %v", got.AdditionalContext, tt.wantAdditionalCtx)
			}
			if got.SystemMessage != tt.wantSystemMessage {
				t.Errorf("ExecuteUserPromptSubmitAction() SystemMessage = %v, want %v", got.SystemMessage, tt.wantSystemMessage)
			}
		})
	}
}

func TestExecuteUserPromptSubmitAction_TypeCommand(t *testing.T) {
	tests := []struct {
		name              string
		action            Action
		stubStdout        string
		stubStderr        string
		stubExitCode      int
		stubErr           error
		wantContinue      bool
		wantDecision      string
		wantHookEventName string
		wantAdditionalCtx string
		wantSystemMessage string
		wantErr           bool
	}{
		{
			name: "Command success with valid JSON",
			action: Action{
				Type:    "command",
				Command: "echo valid json",
			},
			stubStdout: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "UserPromptSubmit",
					"additionalContext": "Valid output"
				}
			}`,
			stubStderr:        "",
			stubExitCode:      0,
			stubErr:           nil,
			wantContinue:      true,
			wantDecision:      "",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "Valid output",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Command with hookEventName UserPromptSubmit",
			action: Action{
				Type:    "command",
				Command: "echo hook event name",
			},
			stubStdout: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "UserPromptSubmit",
					"additionalContext": "Hook event test"
				}
			}`,
			stubStderr:        "",
			stubExitCode:      0,
			stubErr:           nil,
			wantContinue:      true,
			wantDecision:      "",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "Hook event test",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Missing decision allows prompt (empty string)",
			action: Action{
				Type:    "command",
				Command: "echo decision unspecified",
			},
			stubStdout: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "UserPromptSubmit"
				}
			}`,
			stubStderr:        "",
			stubExitCode:      0,
			stubErr:           nil,
			wantContinue:      true,
			wantDecision:      "",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Command with decision: block",
			action: Action{
				Type:    "command",
				Command: "echo decision block",
			},
			stubStdout: `{
				"continue": true,
				"decision": "block",
				"hookSpecificOutput": {
					"hookEventName": "UserPromptSubmit",
					"additionalContext": "Blocked by command"
				}
			}`,
			stubStderr:        "",
			stubExitCode:      0,
			stubErr:           nil,
			wantContinue:      true,
			wantDecision:      "block",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "Blocked by command",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Command failure with non-zero exit code",
			action: Action{
				Type:    "command",
				Command: "exit 1",
			},
			stubStdout:        "",
			stubStderr:        "command failed",
			stubExitCode:      1,
			stubErr:           nil,
			wantContinue:      true,
			wantDecision:      "block",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "",
			wantSystemMessage: "Command failed with exit code 1: command failed",
			wantErr:           false,
		},
		{
			name: "Empty stdout returns allow with hookEventName set",
			action: Action{
				Type:    "command",
				Command: "echo empty",
			},
			stubStdout:        "",
			stubStderr:        "",
			stubExitCode:      0,
			stubErr:           nil,
			wantContinue:      true,
			wantDecision:      "",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Invalid JSON output returns block",
			action: Action{
				Type:    "command",
				Command: "echo invalid json",
			},
			stubStdout:        "not json",
			stubStderr:        "",
			stubExitCode:      0,
			stubErr:           nil,
			wantContinue:      true,
			wantDecision:      "block",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "",
			wantSystemMessage: "Command output is not valid JSON: not json",
			wantErr:           false,
		},
		{
			name: "Missing hookEventName returns block",
			action: Action{
				Type:    "command",
				Command: "echo missing hook event",
			},
			stubStdout: `{
				"continue": true,
				"hookSpecificOutput": {
					"additionalContext": "Missing hookEventName"
				}
			}`,
			stubStderr:        "",
			stubExitCode:      0,
			stubErr:           nil,
			wantContinue:      true,
			wantDecision:      "block",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "",
			wantSystemMessage: "Command output is missing required field: hookSpecificOutput.hookEventName",
			wantErr:           false,
		},
		{
			name: "Invalid hookEventName value returns block",
			action: Action{
				Type:    "command",
				Command: "echo invalid hook event",
			},
			stubStdout: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "WrongEvent",
					"additionalContext": "Invalid hook"
				}
			}`,
			stubStderr:        "",
			stubExitCode:      0,
			stubErr:           nil,
			wantContinue:      true,
			wantDecision:      "block",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "",
			wantSystemMessage: "Invalid hookEventName: expected 'UserPromptSubmit', got 'WrongEvent'",
			wantErr:           false,
		},
		{
			name: "Invalid decision value returns block",
			action: Action{
				Type:    "command",
				Command: "echo invalid decision",
			},
			stubStdout: `{
				"continue": true,
				"decision": "invalid",
				"hookSpecificOutput": {
					"hookEventName": "UserPromptSubmit"
				}
			}`,
			stubStderr:        "",
			stubExitCode:      0,
			stubErr:           nil,
			wantContinue:      true,
			wantDecision:      "block",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "",
			wantSystemMessage: "Invalid decision value: must be 'block' or field must be omitted entirely",
			wantErr:           false,
		},
		{
			name: "Empty string decision is invalid (schema validation fails)",
			action: Action{
				Type:    "command",
				Command: "echo empty string decision",
			},
			stubStdout: `{
				"continue": true,
				"decision": "",
				"hookSpecificOutput": {
					"hookEventName": "UserPromptSubmit",
					"additionalContext": "Test"
				}
			}`,
			stubStderr:        "",
			stubExitCode:      0,
			stubErr:           nil,
			wantContinue:      true,
			wantDecision:      "block",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "",
			wantSystemMessage: "Command output validation failed: schema validation failed: decision: decision must be one of the following: \"block\"",
			wantErr:           false,
		},
		{
			name: "Command failure with err variable when stderr is empty",
			action: Action{
				Type:    "command",
				Command: "exit 1",
			},
			stubStdout:        "",
			stubStderr:        "",
			stubExitCode:      1,
			stubErr:           fmt.Errorf("exit status 1"),
			wantContinue:      true,
			wantDecision:      "block",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "",
			wantSystemMessage: "Command failed with exit code 1: exit status 1",
			wantErr:           false,
		},
		{
			name: "Command failure with stderr takes precedence over err",
			action: Action{
				Type:    "command",
				Command: "exit 1",
			},
			stubStdout:        "",
			stubStderr:        "explicit error from stderr",
			stubExitCode:      1,
			stubErr:           fmt.Errorf("exit status 1"),
			wantContinue:      true,
			wantDecision:      "block",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "",
			wantSystemMessage: "Command failed with exit code 1: explicit error from stderr",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &ActionExecutor{
				runner: &stubRunnerWithOutput{
					stdout:   tt.stubStdout,
					stderr:   tt.stubStderr,
					exitCode: tt.stubExitCode,
					err:      tt.stubErr,
				},
			}

			input := &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session-123",
					TranscriptPath: "/path/to/transcript",
					Cwd:            "/test/cwd",
					PermissionMode: "test",
					HookEventName:  UserPromptSubmit,
				},
				Prompt: "test prompt",
			}

			rawJSON := map[string]any{
				"session_id":      "test-session-123",
				"transcript_path": "/path/to/transcript",
				"cwd":             "/test/cwd",
				"permission_mode": "test",
				"hook_event_name": "UserPromptSubmit",
				"prompt":          "test prompt",
			}

			got, err := executor.ExecuteUserPromptSubmitAction(tt.action, input, rawJSON)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteUserPromptSubmitAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got.Continue != tt.wantContinue {
				t.Errorf("ExecuteUserPromptSubmitAction() Continue = %v, want %v", got.Continue, tt.wantContinue)
			}
			if got.Decision != tt.wantDecision {
				t.Errorf("ExecuteUserPromptSubmitAction() Decision = %v, want %v", got.Decision, tt.wantDecision)
			}
			if got.HookEventName != tt.wantHookEventName {
				t.Errorf("ExecuteUserPromptSubmitAction() HookEventName = %v, want %v", got.HookEventName, tt.wantHookEventName)
			}
			if got.AdditionalContext != tt.wantAdditionalCtx {
				t.Errorf("ExecuteUserPromptSubmitAction() AdditionalContext = %v, want %v", got.AdditionalContext, tt.wantAdditionalCtx)
			}
			if got.SystemMessage != tt.wantSystemMessage {
				t.Errorf("ExecuteUserPromptSubmitAction() SystemMessage = %v, want %v", got.SystemMessage, tt.wantSystemMessage)
			}
		})
	}
}

// TestExecutePreToolUseAction_TypeOutput tests ExecutePreToolUseAction with type: output (Phase 3)

func TestCheckUnsupportedFieldsSessionStart(t *testing.T) {
	tests := []struct {
		name           string
		stdout         string
		wantStderr     string
		wantStderrNone bool
	}{
		{
			name:           "Valid JSON with supported fields only",
			stdout:         `{"continue": true, "systemMessage": "test"}`,
			wantStderrNone: true,
		},
		{
			name:       "Valid JSON with unsupported field",
			stdout:     `{"continue": true, "unsupportedField": "value"}`,
			wantStderr: "Warning: Field 'unsupportedField' is not supported for SessionStart hooks\n",
		},
		{
			name:       "Valid JSON with multiple unsupported fields",
			stdout:     `{"continue": true, "field1": "value1", "field2": "value2"}`,
			wantStderr: "Warning: Field 'field", // Should contain warnings for both fields
		},
		{
			name:           "Invalid JSON (should not panic)",
			stdout:         `{invalid json}`,
			wantStderrNone: true,
		},
		{
			name:           "Empty string",
			stdout:         "",
			wantStderrNone: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			checkUnsupportedFieldsSessionStart(tt.stdout)

			_ = w.Close()
			os.Stderr = oldStderr

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			stderr := buf.String()

			if tt.wantStderrNone {
				if stderr != "" {
					t.Errorf("Expected no stderr output, got: %s", stderr)
				}
			} else {
				if !strings.Contains(stderr, tt.wantStderr) {
					t.Errorf("Expected stderr to contain %q, got: %s", tt.wantStderr, stderr)
				}
			}
		})
	}
}

func TestCheckUnsupportedFieldsUserPromptSubmit(t *testing.T) {
	tests := []struct {
		name           string
		stdout         string
		wantStderr     string
		wantStderrNone bool
	}{
		{
			name:           "Valid JSON with supported fields only",
			stdout:         `{"continue": true, "systemMessage": "test"}`,
			wantStderrNone: true,
		},
		{
			name:       "Valid JSON with unsupported field",
			stdout:     `{"continue": true, "unsupportedField": "value"}`,
			wantStderr: "Warning: Field 'unsupportedField' is not supported for UserPromptSubmit hooks\n",
		},
		{
			name:       "Valid JSON with multiple unsupported fields",
			stdout:     `{"decision": "block", "field1": "value1", "field2": "value2"}`,
			wantStderr: "Warning: Field 'field", // Should contain warnings for both fields
		},
		{
			name:           "Invalid JSON (should not panic)",
			stdout:         `{invalid json}`,
			wantStderrNone: true,
		},
		{
			name:           "Empty string",
			stdout:         "",
			wantStderrNone: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			checkUnsupportedFieldsUserPromptSubmit(tt.stdout)

			_ = w.Close()
			os.Stderr = oldStderr

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			stderr := buf.String()

			if tt.wantStderrNone {
				if stderr != "" {
					t.Errorf("Expected no stderr output, got: %s", stderr)
				}
			} else {
				if !strings.Contains(stderr, tt.wantStderr) {
					t.Errorf("Expected stderr to contain %q, got: %s", tt.wantStderr, stderr)
				}
			}
		})
	}
}

// TestExecuteSessionStartAction_StderrWarnings tests that command failures and JSON parse errors log to stderr
func TestExecuteSessionStartAction_StderrWarnings(t *testing.T) {
	tests := []struct {
		name                   string
		stdout                 string
		stderr                 string
		exitCode               int
		wantContinue           bool
		wantSystemMessageMatch string
		wantStderrMatch        string
	}{
		{
			name:                   "Command failure logs to stderr",
			stdout:                 "",
			stderr:                 "Permission denied",
			exitCode:               1,
			wantContinue:           false,
			wantSystemMessageMatch: "Command failed with exit code 1",
			wantStderrMatch:        "Warning:",
		},
		{
			name:                   "JSON parse error logs to stderr",
			stdout:                 `{"invalid": json}`,
			stderr:                 "",
			exitCode:               0,
			wantContinue:           false,
			wantSystemMessageMatch: "Command output is not valid JSON",
			wantStderrMatch:        "Warning:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunnerWithOutput{
				stdout:   tt.stdout,
				stderr:   tt.stderr,
				exitCode: tt.exitCode,
			}
			executor := NewActionExecutor(runner)

			action := Action{
				Type:    "command",
				Command: "test-command.sh",
			}

			input := &SessionStartInput{
				BaseInput: BaseInput{
					SessionID: "test-session-123",
				},
			}

			rawJSON := map[string]any{
				"session_id": "test-session-123",
			}

			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			output, err := executor.ExecuteSessionStartAction(action, input, rawJSON)

			_ = w.Close()
			os.Stderr = oldStderr

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			stderrOutput := buf.String()

			// Verify no error returned (fail-safe design)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			// Verify output is valid
			if output == nil {
				t.Fatal("Expected valid output, got nil")
			}
			if output.Continue != tt.wantContinue {
				t.Errorf("Expected Continue=%v, got: %v", tt.wantContinue, output.Continue)
			}
			if !strings.Contains(output.SystemMessage, tt.wantSystemMessageMatch) {
				t.Errorf("Expected SystemMessage to contain %q, got: %s", tt.wantSystemMessageMatch, output.SystemMessage)
			}

			// Verify stderr warning was logged
			if !strings.Contains(stderrOutput, tt.wantStderrMatch) {
				t.Errorf("Expected stderr to contain %q, got: %s", tt.wantStderrMatch, stderrOutput)
			}
			if !strings.Contains(stderrOutput, tt.wantSystemMessageMatch) {
				t.Errorf("Expected stderr to contain %q, got: %s", tt.wantSystemMessageMatch, stderrOutput)
			}
		})
	}
}

// TestExecuteUserPromptSubmitAction_StderrWarnings tests that command failures and JSON parse errors log to stderr
func TestExecuteUserPromptSubmitAction_StderrWarnings(t *testing.T) {
	tests := []struct {
		name                   string
		stdout                 string
		stderr                 string
		exitCode               int
		wantDecision           string
		wantSystemMessageMatch string
		wantStderrMatch        string
	}{
		{
			name:                   "Command failure logs to stderr",
			stdout:                 "",
			stderr:                 "command failed",
			exitCode:               1,
			wantDecision:           "block",
			wantSystemMessageMatch: "Command failed with exit code 1",
			wantStderrMatch:        "Warning:",
		},
		{
			name:                   "JSON parse error logs to stderr",
			stdout:                 "not json",
			stderr:                 "",
			exitCode:               0,
			wantDecision:           "block",
			wantSystemMessageMatch: "Command output is not valid JSON",
			wantStderrMatch:        "Warning:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunnerWithOutput{
				stdout:   tt.stdout,
				stderr:   tt.stderr,
				exitCode: tt.exitCode,
			}
			executor := NewActionExecutor(runner)

			action := Action{
				Type:    "command",
				Command: "test-command.sh",
			}

			input := &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session-123",
					TranscriptPath: "/path/to/transcript",
					Cwd:            "/test/cwd",
					PermissionMode: "test",
					HookEventName:  UserPromptSubmit,
				},
				Prompt: "test prompt",
			}

			rawJSON := map[string]any{
				"session_id":      "test-session-123",
				"transcript_path": "/path/to/transcript",
				"cwd":             "/test/cwd",
				"permission_mode": "test",
				"hook_event_name": "UserPromptSubmit",
				"prompt":          "test prompt",
			}

			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			output, err := executor.ExecuteUserPromptSubmitAction(action, input, rawJSON)

			_ = w.Close()
			os.Stderr = oldStderr

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			stderrOutput := buf.String()

			// Verify no error returned (fail-safe design)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			// Verify output is valid
			if output == nil {
				t.Fatal("Expected valid output, got nil")
			}
			if output.Decision != tt.wantDecision {
				t.Errorf("Expected Decision=%v, got: %v", tt.wantDecision, output.Decision)
			}
			if !strings.Contains(output.SystemMessage, tt.wantSystemMessageMatch) {
				t.Errorf("Expected SystemMessage to contain %q, got: %s", tt.wantSystemMessageMatch, output.SystemMessage)
			}

			// Verify stderr warning was logged
			if !strings.Contains(stderrOutput, tt.wantStderrMatch) {
				t.Errorf("Expected stderr to contain %q, got: %s", tt.wantStderrMatch, stderrOutput)
			}
			if !strings.Contains(stderrOutput, tt.wantSystemMessageMatch) {
				t.Errorf("Expected stderr to contain %q, got: %s", tt.wantSystemMessageMatch, stderrOutput)
			}
		})
	}
}

func TestExecuteStopAndSubagentStopAction_TypeOutput(t *testing.T) {
	tests := []struct {
		name              string
		eventType         string // "Stop" or "SubagentStop"
		action            Action
		wantDecision      string
		wantReason        string
		wantSystemMessage string
		wantErr           bool
	}{
		{
			name:      "Stop: Message only (decision unspecified) -> decision empty (allow stop)",
			eventType: "Stop",
			action: Action{
				Type:    "output",
				Message: "Please continue working",
			},
			wantDecision:      "",
			wantReason:        "",
			wantSystemMessage: "Please continue working",
			wantErr:           false,
		},
		{
			name:      "Stop: decision: block + reason specified -> decision=block, reason=specified value",
			eventType: "Stop",
			action: Action{
				Type:     "output",
				Message:  "Stop blocked by hook",
				Decision: stringPtr("block"),
				Reason:   stringPtr("Tests are still running"),
			},
			wantDecision:      "block",
			wantReason:        "Tests are still running",
			wantSystemMessage: "Stop blocked by hook",
			wantErr:           false,
		},
		{
			name:      "Stop: decision: block + reason unspecified -> decision=block, reason=processedMessage",
			eventType: "Stop",
			action: Action{
				Type:     "output",
				Message:  "Blocking stop",
				Decision: stringPtr("block"),
			},
			wantDecision:      "block",
			wantReason:        "Blocking stop",
			wantSystemMessage: "Blocking stop",
			wantErr:           false,
		},
		{
			name:      "Stop: decision: empty string (explicit allow) -> decision empty (allow stop)",
			eventType: "Stop",
			action: Action{
				Type:     "output",
				Message:  "Stop is allowed",
				Decision: stringPtr(""),
			},
			wantDecision:      "",
			wantReason:        "",
			wantSystemMessage: "Stop is allowed",
			wantErr:           false,
		},
		{
			name:      "Stop: Invalid decision value -> fail-safe (decision: block)",
			eventType: "Stop",
			action: Action{
				Type:     "output",
				Message:  "Invalid decision test",
				Decision: stringPtr("invalid"),
			},
			wantDecision:      "block",
			wantReason:        "Invalid decision test",
			wantSystemMessage: "Invalid decision value in action config: must be 'block' or field must be omitted",
			wantErr:           false,
		},
		{
			name:      "Stop: Empty message -> fail-safe (decision: block, reason=fixed message)",
			eventType: "Stop",
			action: Action{
				Type:    "output",
				Message: "",
			},
			wantDecision:      "block",
			wantReason:        "Empty message in Stop action",
			wantSystemMessage: "Empty message in Stop action",
			wantErr:           false,
		},
		{
			name:      "Stop: decision: block + empty reason -> fallback to processedMessage",
			eventType: "Stop",
			action: Action{
				Type:     "output",
				Message:  "Cannot stop now",
				Decision: stringPtr("block"),
				Reason:   stringPtr(""),
			},
			wantDecision:      "block",
			wantReason:        "Cannot stop now",
			wantSystemMessage: "Cannot stop now",
			wantErr:           false,
		},
		{
			name:      "Stop: decision: block + whitespace-only reason -> fallback to processedMessage",
			eventType: "Stop",
			action: Action{
				Type:     "output",
				Message:  "Must complete task",
				Decision: stringPtr("block"),
				Reason:   stringPtr("   "),
			},
			wantDecision:      "block",
			wantReason:        "Must complete task",
			wantSystemMessage: "Must complete task",
			wantErr:           false,
		},
		{
			name:      "Stop: exit_status set (deprecated) -> should warn but process normally",
			eventType: "Stop",
			action: Action{
				Type:       "output",
				Message:    "Stop blocked",
				Decision:   stringPtr("block"),
				ExitStatus: intPtr(2),
			},
			wantDecision:      "block",
			wantReason:        "Stop blocked",
			wantSystemMessage: "Stop blocked",
			wantErr:           false,
		},
		{
			name:      "SubagentStop: message only (decision unspecified) - allow",
			eventType: "SubagentStop",
			action: Action{
				Type:    "output",
				Message: "SubagentStop allowed",
			},
			wantDecision:      "",
			wantReason:        "",
			wantSystemMessage: "SubagentStop allowed",
		},
		{
			name:      "SubagentStop: decision: block with reason specified",
			eventType: "SubagentStop",
			action: Action{
				Type:     "output",
				Message:  "Blocking subagent stop",
				Decision: stringPtr("block"),
				Reason:   stringPtr("Subagent should continue"),
			},
			wantDecision:      "block",
			wantReason:        "Subagent should continue",
			wantSystemMessage: "Blocking subagent stop",
		},
		{
			name:      "SubagentStop: decision: block with reason unspecified (use processedMessage)",
			eventType: "SubagentStop",
			action: Action{
				Type:     "output",
				Message:  "Blocking subagent stop",
				Decision: stringPtr("block"),
			},
			wantDecision:      "block",
			wantReason:        "Blocking subagent stop",
			wantSystemMessage: "Blocking subagent stop",
		},
		{
			name:      "SubagentStop: decision: empty string (explicit allow)",
			eventType: "SubagentStop",
			action: Action{
				Type:     "output",
				Message:  "Allowing subagent stop",
				Decision: stringPtr(""),
			},
			wantDecision:      "",
			wantReason:        "",
			wantSystemMessage: "Allowing subagent stop",
		},
		{
			name:      "SubagentStop: invalid decision value - fail-safe block",
			eventType: "SubagentStop",
			action: Action{
				Type:     "output",
				Message:  "Invalid decision",
				Decision: stringPtr("invalid"),
			},
			wantDecision:      "block",
			wantReason:        "Invalid decision",
			wantSystemMessage: "Invalid decision value in action config: must be 'block' or field must be omitted",
		},
		{
			name:      "SubagentStop: empty message - fail-safe block",
			eventType: "SubagentStop",
			action: Action{
				Type:    "output",
				Message: "",
			},
			wantDecision:      "block",
			wantReason:        "Empty message in SubagentStop action",
			wantSystemMessage: "Empty message in SubagentStop action",
		},
		{
			name:      "SubagentStop: decision: block with empty reason (use processedMessage)",
			eventType: "SubagentStop",
			action: Action{
				Type:     "output",
				Message:  "Blocking with empty reason",
				Decision: stringPtr("block"),
				Reason:   stringPtr(""),
			},
			wantDecision:      "block",
			wantReason:        "Blocking with empty reason",
			wantSystemMessage: "Blocking with empty reason",
		},
		{
			name:      "SubagentStop: decision: block with whitespace-only reason (use processedMessage)",
			eventType: "SubagentStop",
			action: Action{
				Type:     "output",
				Message:  "Blocking with whitespace reason",
				Decision: stringPtr("block"),
				Reason:   stringPtr("   "),
			},
			wantDecision:      "block",
			wantReason:        "Blocking with whitespace reason",
			wantSystemMessage: "Blocking with whitespace reason",
		},
		{
			name:      "SubagentStop: exit_status set (deprecated, should warn)",
			eventType: "SubagentStop",
			action: Action{
				Type:       "output",
				Message:    "Using deprecated exit_status",
				ExitStatus: intPtr(2),
			},
			wantDecision:      "",
			wantReason:        "",
			wantSystemMessage: "Using deprecated exit_status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewActionExecutor(nil)

			var err error
			var decision, reason, systemMessage string
			var continueVal bool

			if tt.eventType == "Stop" {
				input := &StopInput{
					BaseInput: BaseInput{
						SessionID: "test-session-123",
					},
					StopHookActive: false,
				}
				rawJSON := map[string]any{
					"session_id":       "test-session-123",
					"stop_hook_active": false,
				}
				output, err := executor.ExecuteStopAction(tt.action, input, rawJSON)
				if err == nil && output != nil {
					decision = output.Decision
					reason = output.Reason
					systemMessage = output.SystemMessage
					continueVal = output.Continue
				}
			} else {
				input := &SubagentStopInput{
					BaseInput: BaseInput{
						SessionID:     "test-session-123",
						HookEventName: "SubagentStop",
					},
					StopHookActive: true,
				}
				rawJSON := map[string]any{
					"session_id":       "test-session-123",
					"hook_event_name":  "SubagentStop",
					"stop_hook_active": true,
				}
				output, err := executor.ExecuteSubagentStopAction(tt.action, input, rawJSON)
				if err == nil && output != nil {
					decision = output.Decision
					reason = output.Reason
					systemMessage = output.SystemMessage
					continueVal = output.Continue
				}
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if decision != tt.wantDecision {
				t.Errorf("Decision = %q, want %q", decision, tt.wantDecision)
			}

			if reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", reason, tt.wantReason)
			}

			if systemMessage != tt.wantSystemMessage {
				t.Errorf("SystemMessage = %q, want %q", systemMessage, tt.wantSystemMessage)
			}

			// Continue should always be true for Stop and SubagentStop
			if continueVal != true {
				t.Errorf("Continue should always be true, got: %v", continueVal)
			}
		})
	}
}

// TestExecuteStopAndSubagentStopAction_TypeCommand tests ExecuteStopAction and ExecuteSubagentStopAction with type: command

func TestExecuteStopAndSubagentStopAction_TypeCommand(t *testing.T) {
	tests := []struct {
		name              string
		eventType         string // "Stop" or "SubagentStop"
		action            Action
		stubStdout        string
		stubStderr        string
		stubExitCode      int
		stubErr           error
		wantDecision      string
		wantReason        string
		wantSystemMessage string
		wantStopReason    string
		wantSuppressOut   bool
		wantErr           bool
	}{
		{
			name:      "Stop: Valid JSON with decision: block + reason -> all fields parsed correctly",
			eventType: "Stop",
			action: Action{
				Type:    "command",
				Command: "check-stop.sh",
			},
			stubStdout: `{
				"continue": true,
				"decision": "block",
				"reason": "Tests are still running",
				"systemMessage": "Stop blocked by hook"
			}`,
			stubExitCode:      0,
			wantDecision:      "block",
			wantReason:        "Tests are still running",
			wantSystemMessage: "Stop blocked by hook",
			wantErr:           false,
		},
		{
			name:      "Stop: Valid JSON with decision omitted (allow stop) -> decision empty",
			eventType: "Stop",
			action: Action{
				Type:    "command",
				Command: "allow-stop.sh",
			},
			stubStdout: `{
				"continue": true,
				"systemMessage": "Stop is allowed"
			}`,
			stubExitCode:      0,
			wantDecision:      "",
			wantReason:        "",
			wantSystemMessage: "Stop is allowed",
			wantErr:           false,
		},
		{
			name:      "Stop: Command failure (exit != 0) -> fail-safe block",
			eventType: "Stop",
			action: Action{
				Type:    "command",
				Command: "failing-stop.sh",
			},
			stubStdout:        "",
			stubStderr:        "Permission denied",
			stubExitCode:      1,
			wantDecision:      "block",
			wantReason:        "Command failed with exit code 1: Permission denied",
			wantSystemMessage: "Command failed with exit code 1: Permission denied",
			wantErr:           false,
		},
		{
			name:      "Stop: Empty stdout -> allow stop (decision empty)",
			eventType: "Stop",
			action: Action{
				Type:    "command",
				Command: "silent-check.sh",
			},
			stubStdout:   "",
			stubExitCode: 0,
			wantDecision: "",
			wantReason:   "",
			wantErr:      false,
		},
		{
			name:      "Stop: Invalid JSON -> fail-safe block",
			eventType: "Stop",
			action: Action{
				Type:    "command",
				Command: "invalid-json.sh",
			},
			stubStdout:        `{invalid json}`,
			stubExitCode:      0,
			wantDecision:      "block",
			wantReason:        "Command output is not valid JSON: {invalid json}",
			wantSystemMessage: "Command output is not valid JSON: {invalid json}",
			wantErr:           false,
		},
		{
			name:      "Stop: decision: block + reason missing -> fail-safe with reason warning",
			eventType: "Stop",
			action: Action{
				Type:    "command",
				Command: "missing-reason.sh",
			},
			stubStdout: `{
				"continue": true,
				"decision": "block"
			}`,
			stubExitCode:      0,
			wantDecision:      "block",
			wantReason:        "Missing required field 'reason' when decision is 'block'",
			wantSystemMessage: "Missing required field 'reason' when decision is 'block'",
			wantErr:           false,
		},
		{
			name:      "Stop: Invalid decision value -> fail-safe block",
			eventType: "Stop",
			action: Action{
				Type:    "command",
				Command: "invalid-decision.sh",
			},
			stubStdout: `{
				"continue": true,
				"decision": "invalid",
				"reason": "should not matter"
			}`,
			stubExitCode:      0,
			wantDecision:      "block",
			wantReason:        "Invalid decision value: must be 'block' or field must be omitted entirely",
			wantSystemMessage: "Invalid decision value: must be 'block' or field must be omitted entirely",
			wantErr:           false,
		},
		{
			name:      "Stop: Unsupported field -> stderr warning (but still processes valid fields)",
			eventType: "Stop",
			action: Action{
				Type:    "command",
				Command: "unsupported-field.sh",
			},
			stubStdout: `{
				"continue": true,
				"decision": "block",
				"reason": "Blocking stop",
				"hookSpecificOutput": {"hookEventName": "Stop"}
			}`,
			stubExitCode:      0,
			wantDecision:      "block",
			wantReason:        "Blocking stop",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name:      "Stop: stopReason/suppressOutput included -> fields reflected correctly",
			eventType: "Stop",
			action: Action{
				Type:    "command",
				Command: "full-output.sh",
			},
			stubStdout: `{
				"continue": true,
				"decision": "block",
				"reason": "Custom block reason",
				"stopReason": "hook_blocked",
				"suppressOutput": true,
				"systemMessage": "Full output test"
			}`,
			stubExitCode:      0,
			wantDecision:      "block",
			wantReason:        "Custom block reason",
			wantStopReason:    "hook_blocked",
			wantSuppressOut:   true,
			wantSystemMessage: "Full output test",
			wantErr:           false,
		},
		{
			name:      "SubagentStop: Valid JSON with decision: block + reason -> all fields parsed correctly",
			eventType: "SubagentStop",
			action: Action{
				Type:    "command",
				Command: "check-subagent-stop.sh",
			},
			stubStdout: `{
				"continue": true,
				"decision": "block",
				"reason": "Subagent should continue working",
				"systemMessage": "SubagentStop blocked by hook"
			}`,
			stubExitCode:      0,
			wantDecision:      "block",
			wantReason:        "Subagent should continue working",
			wantSystemMessage: "SubagentStop blocked by hook",
			wantErr:           false,
		},
		{
			name:      "SubagentStop: Valid JSON with decision omitted (allow subagent stop) -> decision empty",
			eventType: "SubagentStop",
			action: Action{
				Type:    "command",
				Command: "allow-subagent-stop.sh",
			},
			stubStdout: `{
				"continue": true,
				"systemMessage": "SubagentStop is allowed"
			}`,
			stubExitCode:      0,
			wantDecision:      "",
			wantReason:        "",
			wantSystemMessage: "SubagentStop is allowed",
			wantErr:           false,
		},
		{
			name:      "SubagentStop: Command failure (exit != 0) -> fail-safe block",
			eventType: "SubagentStop",
			action: Action{
				Type:    "command",
				Command: "failing-subagent-stop.sh",
			},
			stubStdout:        "",
			stubStderr:        "Permission denied",
			stubExitCode:      1,
			wantDecision:      "block",
			wantReason:        "Command failed with exit code 1: Permission denied",
			wantSystemMessage: "Command failed with exit code 1: Permission denied",
			wantErr:           false,
		},
		{
			name:      "SubagentStop: Empty stdout -> allow subagent stop (decision empty)",
			eventType: "SubagentStop",
			action: Action{
				Type:    "command",
				Command: "silent-check.sh",
			},
			stubStdout:   "",
			stubExitCode: 0,
			wantDecision: "",
			wantReason:   "",
			wantErr:      false,
		},
		{
			name:      "SubagentStop: Invalid JSON -> fail-safe block",
			eventType: "SubagentStop",
			action: Action{
				Type:    "command",
				Command: "invalid-json.sh",
			},
			stubStdout:        `{invalid json}`,
			stubExitCode:      0,
			wantDecision:      "block",
			wantReason:        "Command output is not valid JSON: {invalid json}",
			wantSystemMessage: "Command output is not valid JSON: {invalid json}",
			wantErr:           false,
		},
		{
			name:      "SubagentStop: decision: block + reason missing -> fail-safe with reason warning",
			eventType: "SubagentStop",
			action: Action{
				Type:    "command",
				Command: "missing-reason.sh",
			},
			stubStdout: `{
				"continue": true,
				"decision": "block"
			}`,
			stubExitCode:      0,
			wantDecision:      "block",
			wantReason:        "Missing required field 'reason' when decision is 'block'",
			wantSystemMessage: "Missing required field 'reason' when decision is 'block'",
			wantErr:           false,
		},
		{
			name:      "SubagentStop: Invalid decision value -> fail-safe block",
			eventType: "SubagentStop",
			action: Action{
				Type:    "command",
				Command: "invalid-decision.sh",
			},
			stubStdout: `{
				"continue": true,
				"decision": "invalid",
				"reason": "should not matter"
			}`,
			stubExitCode:      0,
			wantDecision:      "block",
			wantReason:        "Invalid decision value: must be 'block' or field must be omitted entirely",
			wantSystemMessage: "Invalid decision value: must be 'block' or field must be omitted entirely",
			wantErr:           false,
		},
		{
			name:      "SubagentStop: Unsupported field -> stderr warning (but still processes valid fields)",
			eventType: "SubagentStop",
			action: Action{
				Type:    "command",
				Command: "unsupported-field.sh",
			},
			stubStdout: `{
				"continue": true,
				"decision": "block",
				"reason": "Blocking subagent stop",
				"hookSpecificOutput": {"hookEventName": "SubagentStop"}
			}`,
			stubExitCode:      0,
			wantDecision:      "block",
			wantReason:        "Blocking subagent stop",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name:      "SubagentStop: stopReason/suppressOutput included -> fields reflected correctly",
			eventType: "SubagentStop",
			action: Action{
				Type:    "command",
				Command: "full-output.sh",
			},
			stubStdout: `{
				"continue": true,
				"decision": "block",
				"reason": "Custom block reason",
				"stopReason": "hook_blocked",
				"suppressOutput": true,
				"systemMessage": "Full output test"
			}`,
			stubExitCode:      0,
			wantDecision:      "block",
			wantReason:        "Custom block reason",
			wantStopReason:    "hook_blocked",
			wantSuppressOut:   true,
			wantSystemMessage: "Full output test",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunnerWithOutput{
				stdout:   tt.stubStdout,
				stderr:   tt.stubStderr,
				exitCode: tt.stubExitCode,
				err:      tt.stubErr,
			}
			executor := NewActionExecutor(runner)

			var err error
			var decision, reason, systemMessage, stopReason string
			var continueVal, suppressOutput bool

			if tt.eventType == "Stop" {
				input := &StopInput{
					BaseInput: BaseInput{
						SessionID: "test-session-123",
					},
					StopHookActive: false,
				}
				rawJSON := map[string]any{
					"session_id":       "test-session-123",
					"stop_hook_active": false,
				}
				output, err := executor.ExecuteStopAction(tt.action, input, rawJSON)
				if err == nil && output != nil {
					decision = output.Decision
					reason = output.Reason
					systemMessage = output.SystemMessage
					stopReason = output.StopReason
					suppressOutput = output.SuppressOutput
					continueVal = output.Continue
				}
			} else {
				input := &SubagentStopInput{
					BaseInput: BaseInput{
						SessionID:     "test-session-123",
						HookEventName: "SubagentStop",
					},
					StopHookActive: true,
				}
				rawJSON := map[string]any{
					"session_id":       "test-session-123",
					"hook_event_name":  "SubagentStop",
					"stop_hook_active": true,
				}
				output, err := executor.ExecuteSubagentStopAction(tt.action, input, rawJSON)
				if err == nil && output != nil {
					decision = output.Decision
					reason = output.Reason
					systemMessage = output.SystemMessage
					stopReason = output.StopReason
					suppressOutput = output.SuppressOutput
					continueVal = output.Continue
				}
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Continue should always be true for Stop and SubagentStop
			if continueVal != true {
				t.Errorf("Continue should always be true, got: %v", continueVal)
			}

			if decision != tt.wantDecision {
				t.Errorf("Decision = %q, want %q", decision, tt.wantDecision)
			}

			if reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", reason, tt.wantReason)
			}

			if tt.wantSystemMessage != "" && systemMessage != tt.wantSystemMessage {
				t.Errorf("SystemMessage = %q, want %q", systemMessage, tt.wantSystemMessage)
			}

			if stopReason != tt.wantStopReason {
				t.Errorf("StopReason = %q, want %q", stopReason, tt.wantStopReason)
			}

			if suppressOutput != tt.wantSuppressOut {
				t.Errorf("SuppressOutput = %v, want %v", suppressOutput, tt.wantSuppressOut)
			}
		})
	}
}

// TestExecuteStopAction_TypeCommand_StderrWarnings tests that stderr warnings are logged correctly

func TestExecuteStopAction_TypeCommand_StderrWarnings(t *testing.T) {
	tests := []struct {
		name       string
		stdout     string
		wantStderr string
	}{
		{
			name: "Unsupported field logs warning to stderr",
			stdout: `{
				"continue": true,
				"decision": "block",
				"reason": "test",
				"hookSpecificOutput": {"hookEventName": "Stop"}
			}`,
			wantStderr: "Warning: Field 'hookSpecificOutput' is not supported for Stop hooks",
		},
		{
			name: "Missing reason with decision block logs warning",
			stdout: `{
				"continue": true,
				"decision": "block"
			}`,
			wantStderr: "Warning: Missing required field 'reason' when decision is 'block'",
		},
		{
			name: "Invalid decision value logs warning",
			stdout: `{
				"continue": true,
				"decision": "invalid"
			}`,
			wantStderr: "Warning: Invalid decision value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunnerWithOutput{
				stdout:   tt.stdout,
				exitCode: 0,
			}
			executor := NewActionExecutor(runner)

			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			_, _ = executor.ExecuteStopAction(
				Action{Type: "command", Command: "test.sh"},
				&StopInput{},
				map[string]any{},
			)

			_ = w.Close()
			os.Stderr = oldStderr

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			stderr := buf.String()

			if !strings.Contains(stderr, tt.wantStderr) {
				t.Errorf("Expected stderr to contain %q, got: %s", tt.wantStderr, stderr)
			}
		})
	}
}

func TestExecuteNotificationAction_TypeOutput(t *testing.T) {
	tests := []struct {
		name               string
		action             Action
		wantContinue       bool
		wantHookEventName  string
		wantAdditionalCtx  string
		wantSystemMessage  string
		wantStopReason     string
		wantSuppressOutput bool
		wantErr            bool
	}{
		{
			name: "Message with continue unspecified defaults to true",
			action: Action{
				Type:     "output",
				Message:  "Test notification message",
				Continue: nil,
			},
			wantContinue:      true,
			wantHookEventName: "Notification",
			wantAdditionalCtx: "Test notification message",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Message with continue: false",
			action: Action{
				Type:     "output",
				Message:  "Stop notification",
				Continue: boolPtr(false),
			},
			wantContinue:      false,
			wantHookEventName: "Notification",
			wantAdditionalCtx: "Stop notification",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Message with continue: true explicitly",
			action: Action{
				Type:     "output",
				Message:  "Continue notification",
				Continue: boolPtr(true),
			},
			wantContinue:      true,
			wantHookEventName: "Notification",
			wantAdditionalCtx: "Continue notification",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Message with template variables",
			action: Action{
				Type:     "output",
				Message:  "Notification: {.message}",
				Continue: boolPtr(true),
			},
			wantContinue:      true,
			wantHookEventName: "Notification",
			wantAdditionalCtx: "Notification: Test notification from Claude",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Empty message triggers warning",
			action: Action{
				Type:     "output",
				Message:  "",
				Continue: nil,
			},
			wantContinue:      false,
			wantHookEventName: "",
			wantAdditionalCtx: "",
			wantSystemMessage: "Action output has no message",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewActionExecutor(DefaultCommandRunner)
			input := &NotificationInput{
				BaseInput: BaseInput{
					SessionID:     "test-session-123",
					HookEventName: "Notification",
				},
				Message: "Test notification from Claude",
			}
			rawJSON := map[string]any{
				"session_id":      "test-session-123",
				"hook_event_name": "Notification",
				"message":         "Test notification from Claude",
			}

			output, err := executor.ExecuteNotificationAction(tt.action, input, rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteNotificationAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if output == nil {
				t.Fatal("ExecuteNotificationAction() returned nil output")
			}

			if output.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
			}

			if output.HookEventName != tt.wantHookEventName {
				t.Errorf("HookEventName = %q, want %q", output.HookEventName, tt.wantHookEventName)
			}

			if output.AdditionalContext != tt.wantAdditionalCtx {
				t.Errorf("AdditionalContext = %q, want %q", output.AdditionalContext, tt.wantAdditionalCtx)
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
		})
	}
}

func TestExecuteNotificationAction_TypeCommand(t *testing.T) {
	tests := []struct {
		name               string
		action             Action
		stubStdout         string
		stubStderr         string
		stubExitCode       int
		stubErr            error
		wantContinue       bool
		wantHookEventName  string
		wantAdditionalCtx  string
		wantSystemMessage  string
		wantStopReason     string
		wantSuppressOutput bool
		wantErr            bool
	}{
		{
			name: "Command success with valid JSON and all fields",
			action: Action{
				Type:    "command",
				Command: "get-notification-info.sh",
			},
			stubStdout: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "Notification",
					"additionalContext": "Notification processed successfully"
				},
				"systemMessage": "Debug: notification complete"
			}`,
			stubStderr:        "",
			stubExitCode:      0,
			wantContinue:      true,
			wantHookEventName: "Notification",
			wantAdditionalCtx: "Notification processed successfully",
			wantSystemMessage: "Debug: notification complete",
			wantErr:           false,
		},
		{
			name: "Command with hookEventName Notification",
			action: Action{
				Type:    "command",
				Command: "echo-notification.sh",
			},
			stubStdout: `{
				"hookSpecificOutput": {
					"hookEventName": "Notification"
				}
			}`,
			stubStderr:        "",
			stubExitCode:      0,
			wantContinue:      false,
			wantHookEventName: "Notification",
			wantAdditionalCtx: "",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Command failure with exit != 0",
			action: Action{
				Type:    "command",
				Command: "failing-command.sh",
			},
			stubStdout:        "",
			stubStderr:        "Permission denied",
			stubExitCode:      1,
			wantContinue:      false,
			wantHookEventName: "",
			wantAdditionalCtx: "",
			wantSystemMessage: "Command failed with exit code 1: Permission denied",
			wantErr:           false,
		},
		{
			name: "Empty stdout - validation tool success",
			action: Action{
				Type:    "command",
				Command: "empty-output.sh",
			},
			stubStdout:        "",
			stubStderr:        "",
			stubExitCode:      0,
			wantContinue:      true,
			wantHookEventName: "Notification",
			wantAdditionalCtx: "",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Command with only common fields - hookSpecificOutput auto-complemented",
			action: Action{
				Type:    "command",
				Command: "get-notification-common-only.sh",
			},
			stubStdout: `{
				"continue": true,
				"systemMessage": "Common fields only",
				"stopReason": "auto_complement_test"
			}`,
			stubStderr:         "",
			stubExitCode:       0,
			wantContinue:       true,
			wantHookEventName:  "Notification",
			wantAdditionalCtx:  "",
			wantSystemMessage:  "Common fields only",
			wantStopReason:     "auto_complement_test",
			wantSuppressOutput: false,
			wantErr:            false,
		},
		{
			name: "Command with hookSpecificOutput but missing hookEventName - fail-safe",
			action: Action{
				Type:    "command",
				Command: "invalid-output.sh",
			},
			stubStdout: `{
				"continue": true,
				"hookSpecificOutput": {
					"additionalContext": "Some context"
				}
			}`,
			stubStderr:        "",
			stubExitCode:      0,
			wantContinue:      false,
			wantHookEventName: "",
			wantAdditionalCtx: "",
			wantSystemMessage: "Command output has hookSpecificOutput but missing hookEventName",
			wantErr:           false,
		},
		{
			name: "Command with stopReason and suppressOutput",
			action: Action{
				Type:    "command",
				Command: "get-notification-with-flags.sh",
			},
			stubStdout: `{
				"continue": true,
				"stopReason": "test_stop_reason",
				"suppressOutput": true,
				"hookSpecificOutput": {
					"hookEventName": "Notification",
					"additionalContext": "Notification with flags"
				}
			}`,
			stubStderr:         "",
			stubExitCode:       0,
			wantContinue:       true,
			wantHookEventName:  "Notification",
			wantAdditionalCtx:  "Notification with flags",
			wantSystemMessage:  "",
			wantStopReason:     "test_stop_reason",
			wantSuppressOutput: true,
			wantErr:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunnerWithOutput{
				stdout:   tt.stubStdout,
				stderr:   tt.stubStderr,
				exitCode: tt.stubExitCode,
				err:      tt.stubErr,
			}
			executor := NewActionExecutor(runner)
			input := &NotificationInput{
				BaseInput: BaseInput{
					SessionID:     "test-session-123",
					HookEventName: "Notification",
				},
				Message: "Test notification from Claude",
			}
			rawJSON := map[string]any{
				"session_id":      "test-session-123",
				"hook_event_name": "Notification",
				"message":         "Test notification from Claude",
			}

			output, err := executor.ExecuteNotificationAction(tt.action, input, rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteNotificationAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if output == nil {
				t.Fatal("ExecuteNotificationAction() returned nil output")
			}

			if output.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
			}

			if output.HookEventName != tt.wantHookEventName {
				t.Errorf("HookEventName = %q, want %q", output.HookEventName, tt.wantHookEventName)
			}

			if output.AdditionalContext != tt.wantAdditionalCtx {
				t.Errorf("AdditionalContext = %q, want %q", output.AdditionalContext, tt.wantAdditionalCtx)
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
		})
	}
}

func TestExecuteSessionEndAndPreCompactAction_TypeOutput(t *testing.T) {
	tests := []struct {
		name              string
		eventType         string // "SessionEnd" or "PreCompact"
		action            Action
		wantContinue      bool
		wantSystemMessage string
		wantErr           bool
	}{
		{
			name:      "SessionEnd: Message only -> systemMessage set, continue=true",
			eventType: "SessionEnd",
			action: Action{
				Type:    "output",
				Message: "Session cleanup completed",
			},
			wantContinue:      true,
			wantSystemMessage: "Session cleanup completed",
			wantErr:           false,
		},
		{
			name:      "SessionEnd: Empty message -> fail-safe (systemMessage=fixed message, continue=true)",
			eventType: "SessionEnd",
			action: Action{
				Type:    "output",
				Message: "",
			},
			wantContinue:      true,
			wantSystemMessage: "Empty message in SessionEnd action",
			wantErr:           false,
		},
		{
			name:      "SessionEnd: exit_status specified -> ignore exit_status, emit warning",
			eventType: "SessionEnd",
			action: Action{
				Type:       "output",
				Message:    "Test message",
				ExitStatus: intPtr(2),
			},
			wantContinue:      true,
			wantSystemMessage: "Test message",
			wantErr:           false,
		},
		{
			name:      "PreCompact: Message only -> systemMessage set, continue=true",
			eventType: "PreCompact",
			action: Action{
				Type:    "output",
				Message: "Pre-compaction processing completed",
			},
			wantContinue:      true,
			wantSystemMessage: "Pre-compaction processing completed",
			wantErr:           false,
		},
		{
			name:      "PreCompact: Empty message -> fail-safe (systemMessage=fixed message, continue=true)",
			eventType: "PreCompact",
			action: Action{
				Type:    "output",
				Message: "",
			},
			wantContinue:      true,
			wantSystemMessage: "Empty message in PreCompact action",
			wantErr:           false,
		},
		{
			name:      "PreCompact: exit_status specified -> ignore exit_status, emit warning",
			eventType: "PreCompact",
			action: Action{
				Type:       "output",
				Message:    "Test message",
				ExitStatus: intPtr(2),
			},
			wantContinue:      true,
			wantSystemMessage: "Test message",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunnerWithOutput{}
			executor := &ActionExecutor{runner: runner}

			var result *ActionOutput
			var err error

			if tt.eventType == "SessionEnd" {
				input := &SessionEndInput{}
				result, err = executor.ExecuteSessionEndAction(tt.action, input, map[string]any{})
			} else {
				input := &PreCompactInput{}
				result, err = executor.ExecutePreCompactAction(tt.action, input, map[string]any{})
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("%s error = %v, wantErr %v", tt.eventType, err, tt.wantErr)
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil ActionOutput, got nil")
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

func TestExecuteSessionEndAndPreCompactAction_TypeCommand(t *testing.T) {
	tests := []struct {
		name              string
		eventType         string // "SessionEnd" or "PreCompact"
		stdout            string
		stderr            string
		exitCode          int
		cmdErr            error
		wantContinue      bool
		wantSystemMessage string
		wantErr           bool
	}{
		{
			name:         "SessionEnd: Valid JSON output with all fields",
			eventType:    "SessionEnd",
			stdout:       `{"continue": true, "stopReason": "cleanup done", "suppressOutput": false, "systemMessage": "Session ended"}`,
			exitCode:     0,
			wantContinue: true,
			wantErr:      false,
		},
		{
			name:         "SessionEnd: Valid JSON output with minimal fields",
			eventType:    "SessionEnd",
			stdout:       `{"continue": true}`,
			exitCode:     0,
			wantContinue: true,
			wantErr:      false,
		},
		{
			name:              "SessionEnd: Empty stdout -> continue=true, no error",
			eventType:         "SessionEnd",
			stdout:            "",
			exitCode:          0,
			wantContinue:      true,
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name:              "SessionEnd: Command failed (exit code 1) -> fail-safe (continue=true, systemMessage=error)",
			eventType:         "SessionEnd",
			stdout:            "",
			stderr:            "command error",
			exitCode:          1,
			wantContinue:      true,
			wantSystemMessage: "Command failed with exit code 1: command error",
			wantErr:           false,
		},
		{
			name:              "SessionEnd: Invalid JSON -> fail-safe (continue=true, systemMessage=error)",
			eventType:         "SessionEnd",
			stdout:            `{"continue": "invalid"}`,
			exitCode:          0,
			wantContinue:      true,
			wantSystemMessage: "Command output is not valid JSON: {\"continue\": \"invalid\"}",
			wantErr:           false,
		},
		{
			name:         "SessionEnd: Unsupported field in JSON -> warning to stderr, continue processing",
			eventType:    "SessionEnd",
			stdout:       `{"continue": true, "decision": "block"}`,
			exitCode:     0,
			wantContinue: true,
			wantErr:      false,
		},
		{
			name:         "PreCompact: Valid JSON output with all fields",
			eventType:    "PreCompact",
			stdout:       `{"continue": true, "stopReason": "compaction preparation done", "suppressOutput": false, "systemMessage": "Ready for compaction"}`,
			exitCode:     0,
			wantContinue: true,
			wantErr:      false,
		},
		{
			name:         "PreCompact: Valid JSON output with minimal fields",
			eventType:    "PreCompact",
			stdout:       `{"continue": true}`,
			exitCode:     0,
			wantContinue: true,
			wantErr:      false,
		},
		{
			name:              "PreCompact: Empty stdout -> continue=true, no error",
			eventType:         "PreCompact",
			stdout:            "",
			exitCode:          0,
			wantContinue:      true,
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name:              "PreCompact: Command failed (exit code 1) -> fail-safe (continue=true, systemMessage=error)",
			eventType:         "PreCompact",
			stdout:            "",
			stderr:            "command error",
			exitCode:          1,
			wantContinue:      true,
			wantSystemMessage: "Command failed with exit code 1: command error",
			wantErr:           false,
		},
		{
			name:              "PreCompact: Invalid JSON -> fail-safe (continue=true, systemMessage=error)",
			eventType:         "PreCompact",
			stdout:            `{"continue": "invalid"}`,
			exitCode:          0,
			wantContinue:      true,
			wantSystemMessage: "Command output is not valid JSON: {\"continue\": \"invalid\"}",
			wantErr:           false,
		},
		{
			name:         "PreCompact: Unsupported field in JSON -> warning to stderr, continue processing",
			eventType:    "PreCompact",
			stdout:       `{"continue": true, "decision": "block"}`,
			exitCode:     0,
			wantContinue: true,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunnerWithOutput{
				stdout:   tt.stdout,
				stderr:   tt.stderr,
				exitCode: tt.exitCode,
				err:      tt.cmdErr,
			}
			executor := &ActionExecutor{runner: runner}
			action := Action{
				Type:    "command",
				Command: "test-command",
			}

			// Capture stderr to check for warnings
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			var result *ActionOutput
			var err error

			if tt.eventType == "SessionEnd" {
				input := &SessionEndInput{}
				result, err = executor.ExecuteSessionEndAction(action, input, map[string]any{})
			} else {
				input := &PreCompactInput{}
				result, err = executor.ExecutePreCompactAction(action, input, map[string]any{})
			}

			_ = w.Close()
			os.Stderr = oldStderr
			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			stderrOutput := buf.String()

			if (err != nil) != tt.wantErr {
				t.Errorf("%s error = %v, wantErr %v", tt.eventType, err, tt.wantErr)
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil ActionOutput, got nil")
			}

			if result.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", result.Continue, tt.wantContinue)
			}

			if tt.wantSystemMessage != "" && result.SystemMessage != tt.wantSystemMessage {
				t.Errorf("SystemMessage = %q, want %q", result.SystemMessage, tt.wantSystemMessage)
			}

			// Check for unsupported field warnings
			if strings.Contains(tt.name, "Unsupported field in JSON") {
				if !strings.Contains(stderrOutput, "Warning") || !strings.Contains(stderrOutput, "decision") {
					t.Errorf("Expected warning about unsupported field 'decision' in stderr, got: %s", stderrOutput)
				}
			}
		})
	}
}

func TestExecuteSubagentStartAction_TypeOutput(t *testing.T) {
	tests := []struct {
		name               string
		action             Action
		wantContinue       bool
		wantHookEventName  string
		wantAdditionalCtx  string
		wantSystemMessage  string
		wantStopReason     string
		wantSuppressOutput bool
		wantErr            bool
	}{
		{
			name: "Message with continue unspecified defaults to true",
			action: Action{
				Type:     "output",
				Message:  "Explore agent started",
				Continue: nil,
			},
			wantContinue:      true,
			wantHookEventName: "SubagentStart",
			wantAdditionalCtx: "Explore agent started",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Message with continue: false",
			action: Action{
				Type:     "output",
				Message:  "Agent startup failed",
				Continue: boolPtr(false),
			},
			wantContinue:      false,
			wantHookEventName: "SubagentStart",
			wantAdditionalCtx: "Agent startup failed",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Message with template variables",
			action: Action{
				Type:     "output",
				Message:  "Agent {.agent_type} ({.agent_id}) started",
				Continue: boolPtr(true),
			},
			wantContinue:      true,
			wantHookEventName: "SubagentStart",
			wantAdditionalCtx: "Agent Explore (agent-explore-001) started",
			wantSystemMessage: "",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewActionExecutor(DefaultCommandRunner)
			input := &SubagentStartInput{
				BaseInput: BaseInput{
					SessionID:     "test-session-123",
					HookEventName: "SubagentStart",
				},
				AgentID:   "agent-explore-001",
				AgentType: "Explore",
			}
			rawJSON := map[string]any{
				"session_id":      "test-session-123",
				"hook_event_name": "SubagentStart",
				"agent_id":        "agent-explore-001",
				"agent_type":      "Explore",
			}

			output, err := executor.ExecuteSubagentStartAction(tt.action, input, rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteSubagentStartAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if output == nil {
				t.Fatal("Expected non-nil ActionOutput")
			}

			if output.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
			}

			if output.HookEventName != tt.wantHookEventName {
				t.Errorf("HookEventName = %v, want %v", output.HookEventName, tt.wantHookEventName)
			}

			if output.AdditionalContext != tt.wantAdditionalCtx {
				t.Errorf("AdditionalContext = %v, want %v", output.AdditionalContext, tt.wantAdditionalCtx)
			}
		})
	}
}

func TestExecuteSubagentStartAction_TypeCommand(t *testing.T) {
	tests := []struct {
		name              string
		action            Action
		stubStdout        string
		stubStderr        string
		stubExitCode      int
		wantContinue      bool
		wantHookEventName string
		wantAdditionalCtx string
		wantSystemMessage string
		wantErr           bool
	}{
		{
			name: "Valid JSON with hookSpecificOutput",
			action: Action{
				Type:    "command",
				Command: "echo valid-json",
			},
			stubStdout: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "SubagentStart",
					"additionalContext": "Agent Explore started successfully"
				}
			}`,
			stubStderr:        "",
			stubExitCode:      0,
			wantContinue:      true,
			wantHookEventName: "SubagentStart",
			wantAdditionalCtx: "Agent Explore started successfully",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Command returns empty stdout -> defaults",
			action: Action{
				Type:    "command",
				Command: "echo empty",
			},
			stubStdout:        "",
			stubStderr:        "",
			stubExitCode:      0,
			wantContinue:      true,
			wantHookEventName: "SubagentStart",
			wantAdditionalCtx: "",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Command fails -> warning to stderr, continue false",
			action: Action{
				Type:    "command",
				Command: "fail-command",
			},
			stubStdout:        "",
			stubStderr:        "Command failed",
			stubExitCode:      1,
			wantContinue:      false,
			wantHookEventName: "",
			wantAdditionalCtx: "",
			wantSystemMessage: "Command failed with exit code 1: Command failed",
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubRunner := &stubRunnerWithOutput{
				stdout:   tt.stubStdout,
				stderr:   tt.stubStderr,
				exitCode: tt.stubExitCode,
			}
			executor := NewActionExecutor(stubRunner)

			input := &SubagentStartInput{
				BaseInput: BaseInput{
					SessionID:     "test-session-123",
					HookEventName: "SubagentStart",
				},
				AgentID:   "agent-explore-001",
				AgentType: "Explore",
			}
			rawJSON := map[string]any{
				"session_id":      "test-session-123",
				"hook_event_name": "SubagentStart",
				"agent_id":        "agent-explore-001",
				"agent_type":      "Explore",
			}

			result, err := executor.ExecuteSubagentStartAction(tt.action, input, rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteSubagentStartAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil ActionOutput, got nil")
			}

			if result.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", result.Continue, tt.wantContinue)
			}

			if result.HookEventName != tt.wantHookEventName {
				t.Errorf("HookEventName = %v, want %v", result.HookEventName, tt.wantHookEventName)
			}

			if result.AdditionalContext != tt.wantAdditionalCtx {
				t.Errorf("AdditionalContext = %v, want %v", result.AdditionalContext, tt.wantAdditionalCtx)
			}
		})
	}
}
