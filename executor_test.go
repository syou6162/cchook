package main

import (
	"testing"
)

// Helper function to create *bool
func boolPtr(b bool) *bool {
	return &b
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
			name: "Empty stdout",
			action: Action{
				Type:    "command",
				Command: "empty-output.sh",
			},
			stdout:            "",
			stderr:            "",
			exitCode:          0,
			wantContinue:      false,
			wantHookEventName: "",
			wantAdditionalCtx: "",
			wantSystemMessage: "Command produced no output",
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
			// Skip command tests for now - they will be enabled once we add
			// RunCommandWithOutput to CommandRunner interface
			t.Skip("Command type tests pending CommandRunner interface extension")

			// TODO: Once CommandRunner has RunCommandWithOutput method, use this pattern:
			// runner := &stubRunnerWithOutput{
			// 	stdout:   tt.stdout,
			// 	stderr:   tt.stderr,
			// 	exitCode: tt.exitCode,
			// }
			// executor := NewActionExecutor(runner)
			// ... rest of test
		})
	}
}
