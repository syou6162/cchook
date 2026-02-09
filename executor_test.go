package main

import (
	"bytes"
	"fmt"
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

// TestExecutePreToolUseAction_TypeOutput tests ExecutePreToolUseAction with type: output (Phase 3)
func TestExecutePreToolUseAction_TypeOutput(t *testing.T) {
	tests := []struct {
		name                         string
		action                       Action
		input                        *PreToolUseInput
		wantPermissionDecision       string
		wantPermissionDecisionReason string
		wantSystemMessage            string
		wantHookEventName            string
	}{
		{
			name: "Message with permissionDecision unspecified -> permissionDecision: deny (backward compatibility)",
			action: Action{
				Type:    "output",
				Message: "Operation blocked by default",
			},
			input: &PreToolUseInput{
				ToolName: "Write",
				ToolInput: ToolInput{
					FilePath: "test.txt",
				},
			},
			wantPermissionDecision:       "deny",
			wantPermissionDecisionReason: "Operation blocked by default",
			wantHookEventName:            "PreToolUse",
		},
		{
			name: "Message with permissionDecision: deny",
			action: Action{
				Type:               "output",
				Message:            "Dangerous operation detected",
				PermissionDecision: stringPtr("deny"),
			},
			input: &PreToolUseInput{
				ToolName: "Bash",
			},
			wantPermissionDecision:       "deny",
			wantPermissionDecisionReason: "Dangerous operation detected",
			wantHookEventName:            "PreToolUse",
		},
		{
			name: "Message with permissionDecision: ask",
			action: Action{
				Type:               "output",
				Message:            "Please confirm this operation",
				PermissionDecision: stringPtr("ask"),
			},
			input: &PreToolUseInput{
				ToolName: "Edit",
			},
			wantPermissionDecision:       "ask",
			wantPermissionDecisionReason: "Please confirm this operation",
			wantHookEventName:            "PreToolUse",
		},
		{
			name: "Message with invalid permissionDecision value -> permissionDecision: deny + systemMessage",
			action: Action{
				Type:               "output",
				Message:            "Test message",
				PermissionDecision: stringPtr("invalid_value"),
			},
			input: &PreToolUseInput{
				ToolName: "Write",
			},
			wantPermissionDecision: "deny",
			wantSystemMessage:      "Invalid permission_decision value in action config: must be 'allow', 'deny', or 'ask'",
			wantHookEventName:      "PreToolUse",
		},
		{
			name: "Message with template variables -> correctly expanded (deny by default)",
			action: Action{
				Type:    "output",
				Message: "File operation on {.tool_input.file_path}",
			},
			input: &PreToolUseInput{
				ToolName: "Write",
				ToolInput: ToolInput{
					FilePath: "config.yaml",
				},
			},
			wantPermissionDecision:       "deny",
			wantPermissionDecisionReason: "File operation on config.yaml",
			wantHookEventName:            "PreToolUse",
		},
		{
			name: "Empty message -> permissionDecision: deny + systemMessage",
			action: Action{
				Type:    "output",
				Message: "",
			},
			input: &PreToolUseInput{
				ToolName: "Bash",
			},
			wantPermissionDecision: "deny",
			wantSystemMessage:      "Action output has no message",
			wantHookEventName:      "PreToolUse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewActionExecutor(nil)
			rawJSON := map[string]interface{}{
				"tool_name":  tt.input.ToolName,
				"tool_input": tt.input.ToolInput,
			}

			output, err := executor.ExecutePreToolUseAction(tt.action, tt.input, rawJSON)

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if output.Continue != true {
				t.Errorf("Continue should always be true for PreToolUse, got: %v", output.Continue)
			}

			if output.PermissionDecision != tt.wantPermissionDecision {
				t.Errorf("PermissionDecision mismatch. Got %q, want %q", output.PermissionDecision, tt.wantPermissionDecision)
			}

			if output.PermissionDecisionReason != tt.wantPermissionDecisionReason {
				t.Errorf("PermissionDecisionReason mismatch. Got %q, want %q", output.PermissionDecisionReason, tt.wantPermissionDecisionReason)
			}

			if output.SystemMessage != tt.wantSystemMessage {
				t.Errorf("SystemMessage mismatch. Got %q, want %q", output.SystemMessage, tt.wantSystemMessage)
			}

			if output.HookEventName != tt.wantHookEventName {
				t.Errorf("HookEventName mismatch. Got %q, want %q", output.HookEventName, tt.wantHookEventName)
			}

			// UpdatedInput should not be set for type: output
			if output.UpdatedInput != nil {
				t.Errorf("UpdatedInput should be nil for type: output, got: %v", output.UpdatedInput)
			}
		})
	}
}

// TestExecutePreToolUseAction_TypeCommand tests ExecutePreToolUseAction with type: command (Phase 3)
func TestExecutePreToolUseAction_TypeCommand(t *testing.T) {
	tests := []struct {
		name                         string
		action                       Action
		input                        *PreToolUseInput
		commandOutput                string
		commandStderr                string
		commandExitCode              int
		commandErr                   error
		wantPermissionDecision       string
		wantPermissionDecisionReason string
		wantUpdatedInput             map[string]interface{}
		wantSystemMessage            string
		wantHookEventName            string
	}{
		{
			name: "Command success with complete JSON format -> correctly parsed and fields propagated",
			action: Action{
				Type:    "command",
				Command: "echo test",
			},
			input: &PreToolUseInput{
				ToolName: "Write",
				ToolInput: ToolInput{
					FilePath: "test.txt",
				},
			},
			commandOutput: `{
				"continue": true,
				"systemMessage": "Command executed successfully",
				"hookSpecificOutput": {
					"hookEventName": "PreToolUse",
					"permissionDecision": "allow",
					"permissionDecisionReason": "Safe operation",
					"updatedInput": {
						"file_path": "modified.txt"
					}
				}
			}`,
			commandExitCode:              0,
			wantPermissionDecision:       "allow",
			wantPermissionDecisionReason: "Safe operation",
			wantUpdatedInput: map[string]interface{}{
				"file_path": "modified.txt",
			},
			wantSystemMessage: "Command executed successfully",
			wantHookEventName: "PreToolUse",
		},
		{
			name: "Command with permissionDecision missing -> fail-safe to deny",
			action: Action{
				Type:    "command",
				Command: "echo test",
			},
			input: &PreToolUseInput{
				ToolName: "Edit",
			},
			commandOutput: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "PreToolUse"
				}
			}`,
			commandExitCode:        0,
			wantPermissionDecision: "deny",
			wantSystemMessage:      "Missing required field 'permissionDecision' in command output",
			wantHookEventName:      "PreToolUse",
		},
		{
			name: "Command with permissionDecision: deny",
			action: Action{
				Type:    "command",
				Command: "validate.sh",
			},
			input: &PreToolUseInput{
				ToolName: "Bash",
			},
			commandOutput: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "PreToolUse",
					"permissionDecision": "deny",
					"permissionDecisionReason": "Blocked by policy"
				}
			}`,
			commandExitCode:              0,
			wantPermissionDecision:       "deny",
			wantPermissionDecisionReason: "Blocked by policy",
			wantHookEventName:            "PreToolUse",
		},
		{
			name: "Command with permissionDecision: ask",
			action: Action{
				Type:    "command",
				Command: "check.sh",
			},
			input: &PreToolUseInput{
				ToolName: "Write",
			},
			commandOutput: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "PreToolUse",
					"permissionDecision": "ask",
					"permissionDecisionReason": "Needs confirmation"
				}
			}`,
			commandExitCode:              0,
			wantPermissionDecision:       "ask",
			wantPermissionDecisionReason: "Needs confirmation",
			wantHookEventName:            "PreToolUse",
		},
		{
			name: "Command with updatedInput + permissionDecision: allow (recommended)",
			action: Action{
				Type:    "command",
				Command: "modifier.sh",
			},
			input: &PreToolUseInput{
				ToolName: "Write",
				ToolInput: ToolInput{
					FilePath: "original.txt",
				},
			},
			commandOutput: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "PreToolUse",
					"permissionDecision": "allow",
					"updatedInput": {
						"file_path": "sanitized.txt"
					}
				}
			}`,
			commandExitCode:        0,
			wantPermissionDecision: "allow",
			wantUpdatedInput: map[string]interface{}{
				"file_path": "sanitized.txt",
			},
			wantHookEventName: "PreToolUse",
		},
		{
			name: "Command with updatedInput + permissionDecision: ask (acceptable)",
			action: Action{
				Type:    "command",
				Command: "modifier.sh",
			},
			input: &PreToolUseInput{
				ToolName: "Edit",
			},
			commandOutput: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "PreToolUse",
					"permissionDecision": "ask",
					"updatedInput": {
						"content": "modified content"
					}
				}
			}`,
			commandExitCode:        0,
			wantPermissionDecision: "ask",
			wantUpdatedInput: map[string]interface{}{
				"content": "modified content",
			},
			wantHookEventName: "PreToolUse",
		},
		{
			name: "Command with updatedInput + permissionDecision: deny (acceptable)",
			action: Action{
				Type:    "command",
				Command: "modifier.sh",
			},
			input: &PreToolUseInput{
				ToolName: "Bash",
			},
			commandOutput: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "PreToolUse",
					"permissionDecision": "deny",
					"updatedInput": {
						"command": "safe command"
					}
				}
			}`,
			commandExitCode:        0,
			wantPermissionDecision: "deny",
			wantUpdatedInput: map[string]interface{}{
				"command": "safe command",
			},
			wantHookEventName: "PreToolUse",
		},
		{
			name: "Command failure (exit != 0) -> permissionDecision: deny + systemMessage",
			action: Action{
				Type:    "command",
				Command: "failing.sh",
			},
			input: &PreToolUseInput{
				ToolName: "Write",
			},
			commandOutput:          "",
			commandExitCode:        1,
			wantPermissionDecision: "deny",
			wantSystemMessage:      "Command failed with exit code 1",
			wantHookEventName:      "PreToolUse",
		},
		// Removed: Empty stdout now returns nil to delegate to Claude Code's permission system
		// This test is now covered by TestExecutePreToolUseAction_EmptyStdout below
		{
			name: "Invalid JSON output -> permissionDecision: deny + systemMessage",
			action: Action{
				Type:    "command",
				Command: "echo invalid",
			},
			input: &PreToolUseInput{
				ToolName: "Bash",
			},
			commandOutput:          "not json",
			commandExitCode:        0,
			wantPermissionDecision: "deny",
			wantSystemMessage:      "Command output is not valid JSON",
			wantHookEventName:      "PreToolUse",
		},
		{
			name: "Missing hookEventName -> permissionDecision: deny + systemMessage",
			action: Action{
				Type:    "command",
				Command: "echo test",
			},
			input: &PreToolUseInput{
				ToolName: "Write",
			},
			commandOutput: `{
				"continue": true,
				"hookSpecificOutput": {
					"permissionDecision": "allow"
				}
			}`,
			commandExitCode:        0,
			wantPermissionDecision: "deny",
			wantSystemMessage:      "Command output is missing required field: hookSpecificOutput.hookEventName",
			wantHookEventName:      "PreToolUse",
		},
		{
			name: "Invalid hookEventName value -> permissionDecision: deny + systemMessage",
			action: Action{
				Type:    "command",
				Command: "echo test",
			},
			input: &PreToolUseInput{
				ToolName: "Edit",
			},
			commandOutput: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "WrongEvent",
					"permissionDecision": "allow"
				}
			}`,
			commandExitCode:        0,
			wantPermissionDecision: "deny",
			wantSystemMessage:      "Invalid hookEventName: expected 'PreToolUse', got 'WrongEvent'",
			wantHookEventName:      "PreToolUse",
		},
		{
			name: "Invalid permissionDecision value -> permissionDecision: deny + systemMessage",
			action: Action{
				Type:    "command",
				Command: "echo test",
			},
			input: &PreToolUseInput{
				ToolName: "Bash",
			},
			commandOutput: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "PreToolUse",
					"permissionDecision": "invalid"
				}
			}`,
			commandExitCode:        0,
			wantPermissionDecision: "deny",
			wantSystemMessage:      "Invalid permissionDecision value: must be 'allow', 'deny', or 'ask'",
			wantHookEventName:      "PreToolUse",
		},
		{
			name: "Command failure with err variable (JSON marshal failure simulation)",
			action: Action{
				Type:    "command",
				Command: "failing.sh",
			},
			input: &PreToolUseInput{
				ToolName: "Bash",
			},
			commandOutput:          "",
			commandStderr:          "",
			commandExitCode:        1,
			commandErr:             fmt.Errorf("failed to marshal JSON for stdin: json: unsupported type: chan int"),
			wantPermissionDecision: "deny",
			wantSystemMessage:      "Command failed with exit code 1: failed to marshal JSON for stdin: json: unsupported type: chan int",
			wantHookEventName:      "PreToolUse",
		},
		{
			name: "Command failure with stderr takes precedence over err",
			action: Action{
				Type:    "command",
				Command: "failing.sh",
			},
			input: &PreToolUseInput{
				ToolName: "Bash",
			},
			commandOutput:          "",
			commandStderr:          "explicit error message",
			commandExitCode:        1,
			commandErr:             fmt.Errorf("exit status 1"),
			wantPermissionDecision: "deny",
			wantSystemMessage:      "Command failed with exit code 1: explicit error message",
			wantHookEventName:      "PreToolUse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunnerWithOutput{
				stdout:   tt.commandOutput,
				stderr:   tt.commandStderr,
				exitCode: tt.commandExitCode,
				err:      tt.commandErr,
			}
			executor := NewActionExecutor(runner)
			rawJSON := map[string]interface{}{
				"tool_name":  tt.input.ToolName,
				"tool_input": tt.input.ToolInput,
			}

			output, err := executor.ExecutePreToolUseAction(tt.action, tt.input, rawJSON)

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if output.Continue != true {
				t.Errorf("Continue should always be true for PreToolUse, got: %v", output.Continue)
			}

			if output.PermissionDecision != tt.wantPermissionDecision {
				t.Errorf("PermissionDecision mismatch. Got %q, want %q", output.PermissionDecision, tt.wantPermissionDecision)
			}

			if output.PermissionDecisionReason != tt.wantPermissionDecisionReason {
				t.Errorf("PermissionDecisionReason mismatch. Got %q, want %q", output.PermissionDecisionReason, tt.wantPermissionDecisionReason)
			}

			if tt.wantSystemMessage != "" && !stringContains2(output.SystemMessage, tt.wantSystemMessage) {
				t.Errorf("SystemMessage should contain %q, got %q", tt.wantSystemMessage, output.SystemMessage)
			}

			if output.HookEventName != tt.wantHookEventName {
				t.Errorf("HookEventName mismatch. Got %q, want %q", output.HookEventName, tt.wantHookEventName)
			}

			// Check UpdatedInput
			if tt.wantUpdatedInput != nil {
				if output.UpdatedInput == nil {
					t.Errorf("UpdatedInput should not be nil")
				} else {
					for key, wantVal := range tt.wantUpdatedInput {
						gotVal, ok := output.UpdatedInput[key]
						if !ok {
							t.Errorf("UpdatedInput missing key %q", key)
						} else if gotVal != wantVal {
							t.Errorf("UpdatedInput[%q] mismatch. Got %v, want %v", key, gotVal, wantVal)
						}
					}
				}
			}
		})
	}
}

// stringContains2 checks if a string contains a substring (helper for PreToolUse tests)
func stringContains2(s, substr string) bool {
	return len(s) >= len(substr) && contains2(s, substr)
}

func contains2(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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

// TestExecuteSessionStartAction_CommandFailure_StderrWarning tests that command failure logs to stderr
func TestExecuteSessionStartAction_CommandFailure_StderrWarning(t *testing.T) {
	runner := &stubRunnerWithOutput{
		stdout:   "",
		stderr:   "Permission denied",
		exitCode: 1,
	}
	executor := NewActionExecutor(runner)

	action := Action{
		Type:    "command",
		Command: "failing-command.sh",
	}

	input := &SessionStartInput{
		BaseInput: BaseInput{
			SessionID: "test-session-123",
		},
	}

	rawJSON := map[string]interface{}{
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
	stderr := buf.String()

	// Verify no error returned (fail-safe design)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify output is valid with Continue=false and SystemMessage set
	if output == nil {
		t.Fatal("Expected valid output, got nil")
	}
	if output.Continue != false {
		t.Errorf("Expected Continue=false, got: %v", output.Continue)
	}
	if !strings.Contains(output.SystemMessage, "Command failed with exit code 1") {
		t.Errorf("Expected SystemMessage to contain error, got: %s", output.SystemMessage)
	}

	// Verify stderr warning was logged
	if !strings.Contains(stderr, "Warning:") {
		t.Errorf("Expected warning in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "Command failed with exit code 1") {
		t.Errorf("Expected error message in stderr, got: %s", stderr)
	}
}

// TestExecuteSessionStartAction_JSONParseError_StderrWarning tests that JSON parse error logs to stderr
func TestExecuteSessionStartAction_JSONParseError_StderrWarning(t *testing.T) {
	runner := &stubRunnerWithOutput{
		stdout:   `{"invalid": json}`,
		stderr:   "",
		exitCode: 0,
	}
	executor := NewActionExecutor(runner)

	action := Action{
		Type:    "command",
		Command: "invalid-json.sh",
	}

	input := &SessionStartInput{
		BaseInput: BaseInput{
			SessionID: "test-session-123",
		},
	}

	rawJSON := map[string]interface{}{
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
	stderr := buf.String()

	// Verify no error returned (fail-safe design)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify output is valid with Continue=false and SystemMessage set
	if output == nil {
		t.Fatal("Expected valid output, got nil")
	}
	if output.Continue != false {
		t.Errorf("Expected Continue=false, got: %v", output.Continue)
	}
	if !strings.Contains(output.SystemMessage, "Command output is not valid JSON") {
		t.Errorf("Expected SystemMessage to contain JSON error, got: %s", output.SystemMessage)
	}

	// Verify stderr warning was logged
	if !strings.Contains(stderr, "Warning:") {
		t.Errorf("Expected warning in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "Command output is not valid JSON") {
		t.Errorf("Expected JSON error message in stderr, got: %s", stderr)
	}
}

// TestExecuteUserPromptSubmitAction_CommandFailure_StderrWarning tests that command failure logs to stderr
func TestExecuteUserPromptSubmitAction_CommandFailure_StderrWarning(t *testing.T) {
	runner := &stubRunnerWithOutput{
		stdout:   "",
		stderr:   "command failed",
		exitCode: 1,
	}
	executor := NewActionExecutor(runner)

	action := Action{
		Type:    "command",
		Command: "exit 1",
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

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	output, err := executor.ExecuteUserPromptSubmitAction(action, input, rawJSON)

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	stderr := buf.String()

	// Verify no error returned (fail-safe design)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify output is valid with Decision=block and SystemMessage set
	if output == nil {
		t.Fatal("Expected valid output, got nil")
	}
	if output.Decision != "block" {
		t.Errorf("Expected Decision=block, got: %v", output.Decision)
	}
	if !strings.Contains(output.SystemMessage, "Command failed with exit code 1") {
		t.Errorf("Expected SystemMessage to contain error, got: %s", output.SystemMessage)
	}

	// Verify stderr warning was logged
	if !strings.Contains(stderr, "Warning:") {
		t.Errorf("Expected warning in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "Command failed with exit code 1") {
		t.Errorf("Expected error message in stderr, got: %s", stderr)
	}
}

// TestExecuteUserPromptSubmitAction_JSONParseError_StderrWarning tests that JSON parse error logs to stderr
func TestExecuteUserPromptSubmitAction_JSONParseError_StderrWarning(t *testing.T) {
	runner := &stubRunnerWithOutput{
		stdout:   "not json",
		stderr:   "",
		exitCode: 0,
	}
	executor := NewActionExecutor(runner)

	action := Action{
		Type:    "command",
		Command: "echo invalid json",
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

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	output, err := executor.ExecuteUserPromptSubmitAction(action, input, rawJSON)

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	stderr := buf.String()

	// Verify no error returned (fail-safe design)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify output is valid with Decision=block and SystemMessage set
	if output == nil {
		t.Fatal("Expected valid output, got nil")
	}
	if output.Decision != "block" {
		t.Errorf("Expected Decision=block, got: %v", output.Decision)
	}
	if !strings.Contains(output.SystemMessage, "Command output is not valid JSON") {
		t.Errorf("Expected SystemMessage to contain JSON error, got: %s", output.SystemMessage)
	}

	// Verify stderr warning was logged
	if !strings.Contains(stderr, "Warning:") {
		t.Errorf("Expected warning in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "Command output is not valid JSON") {
		t.Errorf("Expected JSON error message in stderr, got: %s", stderr)
	}
}

func TestExecutePermissionRequestAction_TypeOutput(t *testing.T) {
	tests := []struct {
		name              string
		action            Action
		input             *PermissionRequestInput
		wantBehavior      string
		wantMessage       string
		wantInterrupt     bool
		wantSystemMessage string
		wantHookEventName string
	}{
		{
			name: "behavior unspecified -> deny (fail-safe)",
			action: Action{
				Type:    "output",
				Message: "Operation blocked by default",
			},
			input: &PermissionRequestInput{
				ToolName: "Write",
				ToolInput: ToolInput{
					FilePath: "test.txt",
				},
			},
			wantBehavior:      "deny",
			wantMessage:       "Operation blocked by default",
			wantHookEventName: "PermissionRequest",
		},
		{
			name: "behavior: allow",
			action: Action{
				Type:     "output",
				Message:  "Operation allowed",
				Behavior: stringPtr("allow"),
			},
			input: &PermissionRequestInput{
				ToolName: "Write",
			},
			wantBehavior:      "allow",
			wantMessage:       "", // 公式仕様: allow時はmessageは空
			wantHookEventName: "PermissionRequest",
		},
		{
			name: "behavior: deny + message",
			action: Action{
				Type:     "output",
				Message:  "Dangerous operation detected",
				Behavior: stringPtr("deny"),
			},
			input: &PermissionRequestInput{
				ToolName: "Bash",
			},
			wantBehavior:      "deny",
			wantMessage:       "Dangerous operation detected",
			wantHookEventName: "PermissionRequest",
		},
		{
			name: "behavior: deny + interrupt: true",
			action: Action{
				Type:      "output",
				Message:   "Critical operation blocked",
				Behavior:  stringPtr("deny"),
				Interrupt: boolPtr(true),
			},
			input: &PermissionRequestInput{
				ToolName: "Edit",
			},
			wantBehavior:      "deny",
			wantMessage:       "Critical operation blocked",
			wantInterrupt:     true,
			wantHookEventName: "PermissionRequest",
		},
		{
			name: "behavior: ask -> error (ask is invalid for PermissionRequest)",
			action: Action{
				Type:     "output",
				Message:  "Please confirm",
				Behavior: stringPtr("ask"),
			},
			input: &PermissionRequestInput{
				ToolName: "Write",
			},
			wantBehavior:      "deny",
			wantMessage:       "Invalid behavior value in action config: must be 'allow' or 'deny'",
			wantSystemMessage: "Invalid behavior value in action config: must be 'allow' or 'deny'",
			wantHookEventName: "PermissionRequest",
		},
		{
			name: "Empty message -> behavior: deny + systemMessage",
			action: Action{
				Type:    "output",
				Message: "",
			},
			input: &PermissionRequestInput{
				ToolName: "Bash",
			},
			wantBehavior:      "deny",
			wantMessage:       "Action output has no message for deny behavior",
			wantSystemMessage: "Action output has no message for deny behavior",
			wantHookEventName: "PermissionRequest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &ActionExecutor{runner: &stubRunnerWithOutput{}}
			output, err := executor.ExecutePermissionRequestAction(tt.action, tt.input, tt.input)

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if output == nil {
				t.Fatal("Expected output, got nil")
			}

			if output.Behavior != tt.wantBehavior {
				t.Errorf("Expected Behavior=%s, got: %s", tt.wantBehavior, output.Behavior)
			}

			if output.Message != tt.wantMessage {
				t.Errorf("Expected Message=%s, got: %s", tt.wantMessage, output.Message)
			}

			if output.Interrupt != tt.wantInterrupt {
				t.Errorf("Expected Interrupt=%v, got: %v", tt.wantInterrupt, output.Interrupt)
			}

			if output.SystemMessage != tt.wantSystemMessage {
				t.Errorf("Expected SystemMessage=%s, got: %s", tt.wantSystemMessage, output.SystemMessage)
			}

			if output.HookEventName != tt.wantHookEventName {
				t.Errorf("Expected HookEventName=%s, got: %s", tt.wantHookEventName, output.HookEventName)
			}
		})
	}
}

func TestExecutePermissionRequestAction_TypeCommand(t *testing.T) {
	tests := []struct {
		name              string
		action            Action
		input             *PermissionRequestInput
		stubStdout        string
		stubStderr        string
		stubExitCode      int
		stubErr           error
		wantBehavior      string
		wantMessage       string
		wantInterrupt     bool
		wantSystemMessage string
		wantHookEventName string
	}{
		{
			name: "正常JSON出力",
			action: Action{
				Type:    "command",
				Command: "echo '{\"continue\":true,\"hookSpecificOutput\":{\"hookEventName\":\"PermissionRequest\",\"decision\":{\"behavior\":\"allow\"}}}'",
			},
			input: &PermissionRequestInput{
				ToolName: "Write",
			},
			stubStdout:        `{"continue":true,"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"allow"}}}`,
			stubExitCode:      0,
			wantBehavior:      "allow",
			wantHookEventName: "PermissionRequest",
		},
		{
			name: "コマンド失敗 -> deny",
			action: Action{
				Type:    "command",
				Command: "exit 1",
			},
			input: &PermissionRequestInput{
				ToolName: "Bash",
			},
			stubStdout:        "",
			stubStderr:        "command failed",
			stubExitCode:      1,
			wantBehavior:      "deny",
			wantMessage:       "Command failed with exit code 1: command failed",
			wantSystemMessage: "Command failed with exit code 1: command failed",
			wantHookEventName: "PermissionRequest",
		},
		{
			name: "空stdout -> deny (PermissionRequestではallowではなくdenyがfail-safe)",
			action: Action{
				Type:    "command",
				Command: "echo",
			},
			input: &PermissionRequestInput{
				ToolName: "Write",
			},
			stubStdout:        "",
			stubExitCode:      0,
			wantBehavior:      "deny",
			wantMessage:       "Command output is empty",
			wantSystemMessage: "Command output is empty",
			wantHookEventName: "PermissionRequest",
		},
		{
			name: "hookEventName欠落",
			action: Action{
				Type:    "command",
				Command: "echo '{\"continue\":true}'",
			},
			input: &PermissionRequestInput{
				ToolName: "Write",
			},
			stubStdout:        `{"continue":true}`,
			stubExitCode:      0,
			wantBehavior:      "deny",
			wantMessage:       "Command output is missing required field: hookSpecificOutput.hookEventName",
			wantSystemMessage: "Command output is missing required field: hookSpecificOutput.hookEventName",
			wantHookEventName: "PermissionRequest",
		},
		{
			name: "behavior欠落 -> deny",
			action: Action{
				Type:    "command",
				Command: "echo '{\"continue\":true,\"hookSpecificOutput\":{\"hookEventName\":\"PermissionRequest\",\"decision\":{}}}'",
			},
			input: &PermissionRequestInput{
				ToolName: "Write",
			},
			stubStdout:        `{"continue":true,"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{}}}`,
			stubExitCode:      0,
			wantBehavior:      "deny",
			wantMessage:       "Command output is missing required field: hookSpecificOutput.decision.behavior",
			wantSystemMessage: "Command output is missing required field: hookSpecificOutput.decision.behavior",
			wantHookEventName: "PermissionRequest",
		},
		{
			name: "allow時にinterruptが立っている -> deny (semantic validation)",
			action: Action{
				Type:    "command",
				Command: "echo",
			},
			input: &PermissionRequestInput{
				ToolName: "Write",
			},
			stubStdout:        `{"continue":true,"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"allow","interrupt":true}}}`,
			stubExitCode:      0,
			wantBehavior:      "deny",
			wantMessage:       "Command output validation failed: semantic validation failed: 'interrupt' should be false when behavior is 'allow'",
			wantSystemMessage: "semantic validation failed: 'interrupt' should be false when behavior is 'allow'",
			wantHookEventName: "PermissionRequest",
		},
		{
			name: "allow時にmessageが存在 -> deny (semantic validation)",
			action: Action{
				Type:    "command",
				Command: "echo",
			},
			input: &PermissionRequestInput{
				ToolName: "Write",
			},
			stubStdout:        `{"continue":true,"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"allow","message":"should not be here"}}}`,
			stubExitCode:      0,
			wantBehavior:      "deny",
			wantMessage:       "Command output validation failed: semantic validation failed: 'message' should be empty when behavior is 'allow', got: \"should not be here\"",
			wantSystemMessage: "semantic validation failed: 'message' should be empty when behavior is 'allow'",
			wantHookEventName: "PermissionRequest",
		},
		{
			name: "deny時にupdatedInputが存在 -> deny (semantic validation)",
			action: Action{
				Type:    "command",
				Command: "echo",
			},
			input: &PermissionRequestInput{
				ToolName: "Write",
			},
			stubStdout:        `{"continue":true,"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"deny","updatedInput":{"file_path":"test.txt"}}}}`,
			stubExitCode:      0,
			wantBehavior:      "deny",
			wantMessage:       "Command output validation failed: semantic validation failed: 'updatedInput' should not exist when behavior is 'deny'",
			wantSystemMessage: "semantic validation failed: 'updatedInput' should not exist when behavior is 'deny'",
			wantHookEventName: "PermissionRequest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubRunnerWithOutput{
				stdout:   tt.stubStdout,
				stderr:   tt.stubStderr,
				exitCode: tt.stubExitCode,
				err:      tt.stubErr,
			}
			executor := &ActionExecutor{runner: stub}
			output, err := executor.ExecutePermissionRequestAction(tt.action, tt.input, tt.input)

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if output == nil {
				t.Fatal("Expected output, got nil")
			}

			if output.Behavior != tt.wantBehavior {
				t.Errorf("Expected Behavior=%s, got: %s", tt.wantBehavior, output.Behavior)
			}

			if output.Message != tt.wantMessage {
				t.Errorf("Expected Message=%s, got: %s", tt.wantMessage, output.Message)
			}

			if output.Interrupt != tt.wantInterrupt {
				t.Errorf("Expected Interrupt=%v, got: %v", tt.wantInterrupt, output.Interrupt)
			}

			if tt.wantSystemMessage != "" && !strings.Contains(output.SystemMessage, tt.wantSystemMessage) {
				t.Errorf("Expected SystemMessage to contain %s, got: %s", tt.wantSystemMessage, output.SystemMessage)
			}

			if output.HookEventName != tt.wantHookEventName {
				t.Errorf("Expected HookEventName=%s, got: %s", tt.wantHookEventName, output.HookEventName)
			}
		})
	}
}

// TestExecutePreToolUseAction_EmptyStdout tests that empty stdout returns nil to delegate to Claude Code
func TestExecutePreToolUseAction_EmptyStdout(t *testing.T) {
	action := Action{
		Type:    "command",
		Command: "validator.sh",
	}
	input := &PreToolUseInput{
		ToolName: "Edit",
	}
	runner := &stubRunnerWithOutput{
		stdout:   "",
		exitCode: 0,
	}
	executor := NewActionExecutor(runner)
	rawJSON := map[string]interface{}{
		"tool_name":  input.ToolName,
		"tool_input": input.ToolInput,
	}

	output, err := executor.ExecutePreToolUseAction(action, input, rawJSON)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if output != nil {
		t.Errorf("Expected nil output (delegate to Claude Code), got: %+v", output)
	}
}

// TestExecuteStopAction_TypeOutput tests ExecuteStopAction with type: output
func TestExecuteStopAction_TypeOutput(t *testing.T) {
	tests := []struct {
		name              string
		action            Action
		wantDecision      string
		wantReason        string
		wantSystemMessage string
		wantErr           bool
	}{
		{
			name: "Message only (decision unspecified) -> decision empty (allow stop)",
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
			name: "decision: block + reason specified -> decision=block, reason=specified value",
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
			name: "decision: block + reason unspecified -> decision=block, reason=processedMessage",
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
			name: "decision: empty string (explicit allow) -> decision empty (allow stop)",
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
			name: "Invalid decision value -> fail-safe (decision: block)",
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
			name: "Empty message -> fail-safe (decision: block, reason=fixed message)",
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
			name: "decision: block + empty reason -> fallback to processedMessage",
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
			name: "decision: block + whitespace-only reason -> fallback to processedMessage",
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
			name: "exit_status set (deprecated) -> should warn but process normally",
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
			// Note: stderr warning is emitted but not checked in this test
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewActionExecutor(nil)
			input := &StopInput{
				BaseInput: BaseInput{
					SessionID: "test-session-123",
				},
				StopHookActive: false,
			}
			rawJSON := map[string]interface{}{
				"session_id":       "test-session-123",
				"stop_hook_active": false,
			}

			output, err := executor.ExecuteStopAction(tt.action, input, rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteStopAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if output == nil {
				t.Fatal("ExecuteStopAction() returned nil output")
			}

			if output.Decision != tt.wantDecision {
				t.Errorf("Decision = %q, want %q", output.Decision, tt.wantDecision)
			}

			if output.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", output.Reason, tt.wantReason)
			}

			if output.SystemMessage != tt.wantSystemMessage {
				t.Errorf("SystemMessage = %q, want %q", output.SystemMessage, tt.wantSystemMessage)
			}

			// Continue should always be true for Stop
			if output.Continue != true {
				t.Errorf("Continue should always be true for Stop, got: %v", output.Continue)
			}
		})
	}
}

// TestExecuteStopAction_TypeCommand tests ExecuteStopAction with type: command
func TestExecuteStopAction_TypeCommand(t *testing.T) {
	tests := []struct {
		name              string
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
			name: "Valid JSON with decision: block + reason -> all fields parsed correctly",
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
			name: "Valid JSON with decision omitted (allow stop) -> decision empty",
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
			name: "Command failure (exit != 0) -> fail-safe block",
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
			name: "Empty stdout -> allow stop (decision empty)",
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
			name: "Invalid JSON -> fail-safe block",
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
			name: "decision: block + reason missing -> fail-safe with reason warning",
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
			name: "Invalid decision value -> fail-safe block",
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
			name: "Unsupported field -> stderr warning (but still processes valid fields)",
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
			name: "stopReason/suppressOutput included -> fields reflected correctly",
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
			input := &StopInput{
				BaseInput: BaseInput{
					SessionID: "test-session-123",
				},
				StopHookActive: false,
			}
			rawJSON := map[string]interface{}{
				"session_id":       "test-session-123",
				"stop_hook_active": false,
			}

			output, err := executor.ExecuteStopAction(tt.action, input, rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteStopAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if output == nil {
				t.Fatal("ExecuteStopAction() returned nil output")
			}

			// Continue should always be true for Stop
			if output.Continue != true {
				t.Errorf("Continue should always be true for Stop, got: %v", output.Continue)
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

			if output.StopReason != tt.wantStopReason {
				t.Errorf("StopReason = %q, want %q", output.StopReason, tt.wantStopReason)
			}

			if output.SuppressOutput != tt.wantSuppressOut {
				t.Errorf("SuppressOutput = %v, want %v", output.SuppressOutput, tt.wantSuppressOut)
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
				map[string]interface{}{},
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

func TestExecutePostToolUseAction_TypeOutput(t *testing.T) {
	tests := []struct {
		name                  string
		action                Action
		wantDecision          string
		wantReason            string
		wantAdditionalContext string
		wantSystemMessage     string
		wantErr               bool
	}{
		{
			name: "Message only (decision unspecified) -> decision empty (allow tool result)",
			action: Action{
				Type:    "output",
				Message: "Tool executed successfully",
			},
			wantDecision:          "",
			wantReason:            "",
			wantAdditionalContext: "Tool executed successfully",
			wantSystemMessage:     "",
			wantErr:               false,
		},
		{
			name: "decision: block + reason specified -> decision=block, reason=specified value",
			action: Action{
				Type:     "output",
				Message:  "Tool result contains sensitive data",
				Decision: stringPtr("block"),
				Reason:   stringPtr("Output validation failed"),
			},
			wantDecision:          "block",
			wantReason:            "Output validation failed",
			wantAdditionalContext: "Tool result contains sensitive data",
			wantSystemMessage:     "",
			wantErr:               false,
		},
		{
			name: "decision: block + reason unspecified -> decision=block, reason=processedMessage",
			action: Action{
				Type:     "output",
				Message:  "Blocking tool output",
				Decision: stringPtr("block"),
			},
			wantDecision:          "block",
			wantReason:            "Blocking tool output",
			wantAdditionalContext: "Blocking tool output",
			wantSystemMessage:     "",
			wantErr:               false,
		},
		{
			name: "decision: empty string (explicit allow) -> decision empty (allow tool result)",
			action: Action{
				Type:     "output",
				Message:  "Tool result is valid",
				Decision: stringPtr(""),
			},
			wantDecision:          "",
			wantReason:            "",
			wantAdditionalContext: "Tool result is valid",
			wantSystemMessage:     "",
			wantErr:               false,
		},
		{
			name: "Invalid decision value -> fail-safe (decision: block)",
			action: Action{
				Type:     "output",
				Message:  "Invalid decision test",
				Decision: stringPtr("invalid"),
			},
			wantDecision:          "block",
			wantReason:            "Invalid decision test",
			wantAdditionalContext: "",
			wantSystemMessage:     "Invalid decision value in action config: must be 'block' or field must be omitted",
			wantErr:               false,
		},
		{
			name: "Empty message -> fail-safe (decision: block, reason=fixed message)",
			action: Action{
				Type:    "output",
				Message: "",
			},
			wantDecision:          "block",
			wantReason:            "Empty message in PostToolUse action",
			wantAdditionalContext: "",
			wantSystemMessage:     "Empty message in PostToolUse action",
			wantErr:               false,
		},
		{
			name: "exit_status deprecated warning -> stderr warning, decision empty (allow)",
			action: Action{
				Type:       "output",
				Message:    "Tool result allowed",
				ExitStatus: intPtr(0),
			},
			wantDecision:          "",
			wantReason:            "",
			wantAdditionalContext: "Tool result allowed",
			wantSystemMessage:     "",
			wantErr:               false,
		},
		{
			name: "exit_status=2 deprecated warning -> stderr warning, decision empty (allow)",
			action: Action{
				Type:       "output",
				Message:    "Tool result blocked (legacy)",
				ExitStatus: intPtr(2),
			},
			wantDecision:          "",
			wantReason:            "",
			wantAdditionalContext: "Tool result blocked (legacy)",
			wantSystemMessage:     "",
			wantErr:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewActionExecutor(nil)
			input := &PostToolUseInput{
				BaseInput: BaseInput{
					SessionID:     "test-session",
					HookEventName: "PostToolUse",
				},
				ToolName: "Write",
			}
			rawJSON := map[string]interface{}{
				"session_id":      "test-session",
				"hook_event_name": "PostToolUse",
				"tool_name":       "Write",
				"tool_input":      map[string]interface{}{},
				"tool_response":   "ok",
			}

			// Capture stderr for deprecation warning check
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			output, err := executor.ExecutePostToolUseAction(tt.action, input, rawJSON)

			_ = w.Close()
			os.Stderr = oldStderr

			var stderrBuf strings.Builder
			_, _ = io.Copy(&stderrBuf, r)
			stderr := stderrBuf.String()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if output == nil {
				t.Fatal("Expected output, got nil")
			}

			if output.Decision != tt.wantDecision {
				t.Errorf("Decision mismatch: want %q, got %q", tt.wantDecision, output.Decision)
			}

			if output.Reason != tt.wantReason {
				t.Errorf("Reason mismatch: want %q, got %q", tt.wantReason, output.Reason)
			}

			if output.AdditionalContext != tt.wantAdditionalContext {
				t.Errorf("AdditionalContext mismatch: want %q, got %q", tt.wantAdditionalContext, output.AdditionalContext)
			}

			if output.SystemMessage != tt.wantSystemMessage {
				t.Errorf("SystemMessage mismatch: want %q, got %q", tt.wantSystemMessage, output.SystemMessage)
			}

			if output.HookEventName != "PostToolUse" {
				t.Errorf("HookEventName mismatch: want %q, got %q", "PostToolUse", output.HookEventName)
			}

			// Check for deprecation warning in stderr when exit_status is set
			if tt.action.ExitStatus != nil {
				if !strings.Contains(stderr, "exit_status") && !strings.Contains(stderr, "deprecated") {
					t.Errorf("Expected deprecation warning in stderr when exit_status is set")
				}
			}
		})
	}
}

func TestExecuteSubagentStopAction_TypeOutput(t *testing.T) {
	tests := []struct {
		name               string
		action             Action
		wantDecision       string
		wantReason         string
		wantSystemMessage  string
		wantContinue       bool
		wantStdoutContains string
		wantStderrContains string
	}{
		{
			name: "message only (decision unspecified) - allow",
			action: Action{
				Type:    "output",
				Message: "SubagentStop allowed",
			},
			wantDecision:      "",
			wantReason:        "",
			wantSystemMessage: "SubagentStop allowed",
			wantContinue:      true,
		},
		{
			name: "decision: block with reason specified",
			action: Action{
				Type:     "output",
				Message:  "Blocking subagent stop",
				Decision: stringPtr("block"),
				Reason:   stringPtr("Subagent should continue"),
			},
			wantDecision:      "block",
			wantReason:        "Subagent should continue",
			wantSystemMessage: "Blocking subagent stop",
			wantContinue:      true,
		},
		{
			name: "decision: block with reason unspecified (use processedMessage)",
			action: Action{
				Type:     "output",
				Message:  "Blocking subagent stop",
				Decision: stringPtr("block"),
			},
			wantDecision:      "block",
			wantReason:        "Blocking subagent stop",
			wantSystemMessage: "Blocking subagent stop",
			wantContinue:      true,
		},
		{
			name: "decision: empty string (explicit allow)",
			action: Action{
				Type:     "output",
				Message:  "Allowing subagent stop",
				Decision: stringPtr(""),
			},
			wantDecision:      "",
			wantReason:        "",
			wantSystemMessage: "Allowing subagent stop",
			wantContinue:      true,
		},
		{
			name: "invalid decision value - fail-safe block",
			action: Action{
				Type:     "output",
				Message:  "Invalid decision",
				Decision: stringPtr("invalid"),
			},
			wantDecision:      "block",
			wantReason:        "Invalid decision",
			wantSystemMessage: "Invalid decision value in action config: must be 'block' or field must be omitted",
			wantContinue:      true,
		},
		{
			name: "empty message - fail-safe block",
			action: Action{
				Type:    "output",
				Message: "",
			},
			wantDecision:      "block",
			wantReason:        "Empty message in SubagentStop action",
			wantSystemMessage: "Empty message in SubagentStop action",
			wantContinue:      true,
		},
		{
			name: "decision: block with empty reason (use processedMessage)",
			action: Action{
				Type:     "output",
				Message:  "Blocking with empty reason",
				Decision: stringPtr("block"),
				Reason:   stringPtr(""),
			},
			wantDecision:      "block",
			wantReason:        "Blocking with empty reason",
			wantSystemMessage: "Blocking with empty reason",
			wantContinue:      true,
		},
		{
			name: "decision: block with whitespace-only reason (use processedMessage)",
			action: Action{
				Type:     "output",
				Message:  "Blocking with whitespace reason",
				Decision: stringPtr("block"),
				Reason:   stringPtr("   "),
			},
			wantDecision:      "block",
			wantReason:        "Blocking with whitespace reason",
			wantSystemMessage: "Blocking with whitespace reason",
			wantContinue:      true,
		},
		{
			name: "exit_status set (deprecated, should warn)",
			action: Action{
				Type:       "output",
				Message:    "Using deprecated exit_status",
				ExitStatus: intPtr(2),
			},
			wantDecision:       "",
			wantReason:         "",
			wantSystemMessage:  "Using deprecated exit_status",
			wantContinue:       true,
			wantStderrContains: "exit_status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewActionExecutor(nil)
			input := &SubagentStopInput{
				BaseInput: BaseInput{
					SessionID:     "test-session-123",
					HookEventName: "SubagentStop",
				},
				StopHookActive: true,
			}
			rawJSON := map[string]interface{}{
				"session_id":       "test-session-123",
				"hook_event_name":  "SubagentStop",
				"stop_hook_active": true,
			}

			output, err := executor.ExecuteSubagentStopAction(tt.action, input, rawJSON)

			if err != nil {
				t.Fatalf("ExecuteSubagentStopAction() error = %v", err)
			}

			if output == nil {
				t.Fatal("ExecuteSubagentStopAction() returned nil output")
			}

			if output.Decision != tt.wantDecision {
				t.Errorf("Decision = %q, want %q", output.Decision, tt.wantDecision)
			}

			if output.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", output.Reason, tt.wantReason)
			}

			if output.SystemMessage != tt.wantSystemMessage {
				t.Errorf("SystemMessage = %q, want %q", output.SystemMessage, tt.wantSystemMessage)
			}

			if output.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", output.Continue, tt.wantContinue)
			}
		})
	}
}

func TestExecuteSubagentStopAction_TypeCommand(t *testing.T) {
	tests := []struct {
		name              string
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
			name: "Valid JSON with decision: block + reason -> all fields parsed correctly",
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
			name: "Valid JSON with decision omitted (allow subagent stop) -> decision empty",
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
			name: "Command failure (exit != 0) -> fail-safe block",
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
			name: "Empty stdout -> allow subagent stop (decision empty)",
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
			name: "Invalid JSON -> fail-safe block",
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
			name: "decision: block + reason missing -> fail-safe with reason warning",
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
			name: "Invalid decision value -> fail-safe block",
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
			name: "Unsupported field -> stderr warning (but still processes valid fields)",
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
			name: "stopReason/suppressOutput included -> fields reflected correctly",
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
			input := &SubagentStopInput{
				BaseInput: BaseInput{
					SessionID:     "test-session-123",
					HookEventName: "SubagentStop",
				},
				StopHookActive: true,
			}
			rawJSON := map[string]interface{}{
				"session_id":       "test-session-123",
				"hook_event_name":  "SubagentStop",
				"stop_hook_active": true,
			}

			output, err := executor.ExecuteSubagentStopAction(tt.action, input, rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteSubagentStopAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if output == nil {
				t.Fatal("ExecuteSubagentStopAction() returned nil output")
			}

			// Continue should always be true for SubagentStop
			if output.Continue != true {
				t.Errorf("Continue should always be true for SubagentStop, got: %v", output.Continue)
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

			if output.StopReason != tt.wantStopReason {
				t.Errorf("StopReason = %q, want %q", output.StopReason, tt.wantStopReason)
			}

			if output.SuppressOutput != tt.wantSuppressOut {
				t.Errorf("SuppressOutput = %v, want %v", output.SuppressOutput, tt.wantSuppressOut)
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
			rawJSON := map[string]interface{}{
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
			rawJSON := map[string]interface{}{
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
