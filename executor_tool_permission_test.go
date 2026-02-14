package main

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestExecutePreToolUseAction_TypeOutput(t *testing.T) {
	tests := []struct {
		name                         string
		action                       Action
		input                        *PreToolUseInput
		wantPermissionDecision       string
		wantPermissionDecisionReason string
		wantAdditionalContext        string
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
		{
			name: "additional_context set -> reflected in AdditionalContext",
			action: Action{
				Type:               "output",
				Message:            "Operation allowed",
				PermissionDecision: stringPtr("allow"),
				AdditionalContext:  stringPtr("Current environment: production"),
			},
			input: &PreToolUseInput{
				ToolName: "Bash",
			},
			wantPermissionDecision:       "allow",
			wantPermissionDecisionReason: "Operation allowed",
			wantAdditionalContext:        "Current environment: production",
			wantHookEventName:            "PreToolUse",
		},
		{
			name: "additional_context omitted -> AdditionalContext empty",
			action: Action{
				Type:               "output",
				Message:            "Operation allowed",
				PermissionDecision: stringPtr("allow"),
			},
			input: &PreToolUseInput{
				ToolName: "Write",
			},
			wantPermissionDecision:       "allow",
			wantPermissionDecisionReason: "Operation allowed",
			wantAdditionalContext:        "",
			wantHookEventName:            "PreToolUse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewActionExecutor(nil)
			rawJSON := map[string]any{
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

			if output.AdditionalContext != tt.wantAdditionalContext {
				t.Errorf("AdditionalContext mismatch. Got %q, want %q", output.AdditionalContext, tt.wantAdditionalContext)
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
		wantAdditionalContext        string
		wantUpdatedInput             map[string]any
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
			wantUpdatedInput: map[string]any{
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
			wantUpdatedInput: map[string]any{
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
			wantUpdatedInput: map[string]any{
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
			wantUpdatedInput: map[string]any{
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
		{
			name: "Command JSON output with additionalContext -> reflected in ActionOutput",
			action: Action{
				Type:    "command",
				Command: "check-env.sh",
			},
			input: &PreToolUseInput{
				ToolName: "Bash",
			},
			commandOutput: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "PreToolUse",
					"permissionDecision": "allow",
					"additionalContext": "Current environment: production"
				}
			}`,
			commandExitCode:        0,
			wantPermissionDecision: "allow",
			wantAdditionalContext:  "Current environment: production",
			wantHookEventName:      "PreToolUse",
		},
		{
			name: "Command JSON output without additionalContext -> AdditionalContext empty",
			action: Action{
				Type:    "command",
				Command: "validator.sh",
			},
			input: &PreToolUseInput{
				ToolName: "Write",
			},
			commandOutput: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "PreToolUse",
					"permissionDecision": "allow"
				}
			}`,
			commandExitCode:        0,
			wantPermissionDecision: "allow",
			wantAdditionalContext:  "",
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
			rawJSON := map[string]any{
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

			if output.AdditionalContext != tt.wantAdditionalContext {
				t.Errorf("AdditionalContext mismatch. Got %q, want %q", output.AdditionalContext, tt.wantAdditionalContext)
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
	rawJSON := map[string]any{
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
			rawJSON := map[string]any{
				"session_id":      "test-session",
				"hook_event_name": "PostToolUse",
				"tool_name":       "Write",
				"tool_input":      map[string]any{},
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

func TestExecutePostToolUseAction_CommandWithUpdatedMCPToolOutput(t *testing.T) {
	tests := []struct {
		name                     string
		stubStdout               string
		stubStderr               string
		stubExitCode             int
		action                   Action
		wantContinue             bool
		wantDecision             string
		wantReason               string
		wantAdditionalCtx        string
		wantUpdatedMCPToolOutput any
		wantErr                  bool
	}{
		{
			name: "updatedMCPToolOutput with string value",
			stubStdout: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "PostToolUse",
					"additionalContext": "Tool output replaced"
				},
				"updatedMCPToolOutput": "replaced output value"
			}`,
			stubStderr:   "",
			stubExitCode: 0,
			action: Action{
				Type:    "command",
				Command: "echo test",
			},
			wantContinue:             true,
			wantDecision:             "",
			wantReason:               "",
			wantAdditionalCtx:        "Tool output replaced",
			wantUpdatedMCPToolOutput: "replaced output value",
			wantErr:                  false,
		},
		{
			name: "updatedMCPToolOutput with object value",
			stubStdout: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "PostToolUse"
				},
				"updatedMCPToolOutput": {
					"result": "success",
					"data": {"key": "value"}
				}
			}`,
			stubStderr:   "",
			stubExitCode: 0,
			action: Action{
				Type:    "command",
				Command: "echo test",
			},
			wantContinue:      true,
			wantDecision:      "",
			wantReason:        "",
			wantAdditionalCtx: "",
			wantUpdatedMCPToolOutput: map[string]any{
				"result": "success",
				"data":   map[string]any{"key": "value"},
			},
			wantErr: false,
		},
		{
			name: "updatedMCPToolOutput omitted (nil)",
			stubStdout: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "PostToolUse",
					"additionalContext": "No tool output replacement"
				}
			}`,
			stubStderr:   "",
			stubExitCode: 0,
			action: Action{
				Type:    "command",
				Command: "echo test",
			},
			wantContinue:             true,
			wantDecision:             "",
			wantReason:               "",
			wantAdditionalCtx:        "No tool output replacement",
			wantUpdatedMCPToolOutput: nil,
			wantErr:                  false,
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

			input := &PostToolUseInput{
				BaseInput: BaseInput{
					SessionID:     "test-session-123",
					HookEventName: "PostToolUse",
				},
				ToolName: "Write",
			}
			rawJSON := map[string]any{
				"session_id":      "test-session-123",
				"hook_event_name": "PostToolUse",
				"tool_name":       "Write",
			}

			result, err := executor.ExecutePostToolUseAction(tt.action, input, rawJSON)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecutePostToolUseAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil ActionOutput, got nil")
			}

			if result.Continue != tt.wantContinue {
				t.Errorf("Continue = %v, want %v", result.Continue, tt.wantContinue)
			}

			if result.Decision != tt.wantDecision {
				t.Errorf("Decision = %v, want %v", result.Decision, tt.wantDecision)
			}

			if result.Reason != tt.wantReason {
				t.Errorf("Reason = %v, want %v", result.Reason, tt.wantReason)
			}

			if result.AdditionalContext != tt.wantAdditionalCtx {
				t.Errorf("AdditionalContext = %v, want %v", result.AdditionalContext, tt.wantAdditionalCtx)
			}

			// Compare UpdatedMCPToolOutput
			if !reflect.DeepEqual(result.UpdatedMCPToolOutput, tt.wantUpdatedMCPToolOutput) {
				t.Errorf("UpdatedMCPToolOutput = %v, want %v", result.UpdatedMCPToolOutput, tt.wantUpdatedMCPToolOutput)
			}
		})
	}
}
