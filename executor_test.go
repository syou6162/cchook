package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// stubRunnerWithOutput is a test stub that implements CommandRunner for testing ExecuteSessionStartAction.
type stubRunnerWithOutput struct {
	stdout   string
	stderr   string
	exitCode int
	err      error
}

func (s *stubRunnerWithOutput) RunCommand(cmd string, useStdin bool, data interface{}) error {
	return s.err
}

func (s *stubRunnerWithOutput) RunCommandWithOutput(cmd string, useStdin bool, data interface{}) (stdout, stderr string, exitCode int, err error) {
	return s.stdout, s.stderr, s.exitCode, s.err
}

// Helper function to create *bool
func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}

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
			rawJSON := map[string]interface{}{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunnerWithOutput{
				stdout:   tt.stdout,
				stderr:   tt.stderr,
				exitCode: tt.exitCode,
			}
			executor := NewActionExecutor(runner)
			input := &SessionStartInput{
				BaseInput: BaseInput{
					SessionID: "test-session-123",
				},
			}
			rawJSON := map[string]interface{}{
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
			wantDecision:      "approve",
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
			wantSystemMessage: "Invalid decision value: must be 'approve' or 'block'",
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
			wantDecision:      "approve",
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

			rawJSON := map[string]interface{}{
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
			wantDecision:      "approve",
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
			wantDecision:      "approve",
			wantHookEventName: "UserPromptSubmit",
			wantAdditionalCtx: "Hook event test",
			wantSystemMessage: "",
			wantErr:           false,
		},
		{
			name: "Command with decision unspecified defaults to allow",
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
			wantDecision:      "approve",
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
			wantDecision:      "approve",
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
			wantSystemMessage: "Invalid decision value: must be 'approve' or 'block'",
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

			rawJSON := map[string]interface{}{
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
			stdout:         `{"continue": true, "decision": "approve", "systemMessage": "test"}`,
			wantStderrNone: true,
		},
		{
			name:       "Valid JSON with unsupported field",
			stdout:     `{"continue": true, "decision": "approve", "unsupportedField": "value"}`,
			wantStderr: "Warning: Field 'unsupportedField' is not supported for UserPromptSubmit hooks\n",
		},
		{
			name:       "Valid JSON with multiple unsupported fields",
			stdout:     `{"decision": "approve", "field1": "value1", "field2": "value2"}`,
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
