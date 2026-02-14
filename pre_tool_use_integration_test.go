//go:build integration
// +build integration

package main

import (
	"os"
	"strings"
	"testing"
)

// PreToolUse integration tests for Phase 3 JSON output
func TestPreToolUseIntegration_AllowOperation(t *testing.T) {
	// Test case 1: Real config file with safe operation -> permissionDecision: "allow"
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	configYAML := `PreToolUse:
  - matcher: "Write"
    actions:
      - type: output
        message: "Allowing safe write operation"
        permission_decision: "allow"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &PreToolUseInput{
		BaseInput: BaseInput{
			SessionID:      "test-session-123",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  PreToolUse,
		},
		ToolName: "Write",
		ToolInput: ToolInput{
			FilePath: "test.txt",
		},
	}

	rawJSON := map[string]any{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"tool_name":       input.ToolName,
		"tool_input": map[string]any{
			"file_path": input.ToolInput.FilePath,
		},
	}

	output, err := executePreToolUseHooksJSON(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify JSON structure
	if !output.Continue {
		t.Errorf("Continue = false, want true (always true for PreToolUse)")
	}

	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}

	if output.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Errorf("HookEventName = %v, want PreToolUse", output.HookSpecificOutput.HookEventName)
	}

	if output.HookSpecificOutput.PermissionDecision != "allow" {
		t.Errorf("PermissionDecision = %v, want allow", output.HookSpecificOutput.PermissionDecision)
	}

	if !strings.Contains(output.HookSpecificOutput.PermissionDecisionReason, "Allowing safe write operation") {
		t.Errorf("PermissionDecisionReason does not contain expected message, got: %s", output.HookSpecificOutput.PermissionDecisionReason)
	}
}

func TestPreToolUseIntegration_DenyOperation(t *testing.T) {
	// Test case 2: Real config file with dangerous operation -> permissionDecision: "deny"
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	configYAML := `PreToolUse:
  - matcher: "Bash"
    conditions:
      - type: command_contains
        value: "rm -rf"
    actions:
      - type: output
        message: "Dangerous command blocked"
        permission_decision: "deny"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &PreToolUseInput{
		BaseInput: BaseInput{
			SessionID:      "test-session-123",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  PreToolUse,
		},
		ToolName: "Bash",
		ToolInput: ToolInput{
			Command: "rm -rf /tmp/test",
		},
	}

	rawJSON := map[string]any{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"tool_name":       input.ToolName,
		"tool_input": map[string]any{
			"command": input.ToolInput.Command,
		},
	}

	output, err := executePreToolUseHooksJSON(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify JSON structure
	if !output.Continue {
		t.Errorf("Continue = false, want true (always true for PreToolUse)")
	}

	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}

	if output.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Errorf("HookEventName = %v, want PreToolUse", output.HookSpecificOutput.HookEventName)
	}

	if output.HookSpecificOutput.PermissionDecision != "deny" {
		t.Errorf("PermissionDecision = %v, want deny", output.HookSpecificOutput.PermissionDecision)
	}

	if !strings.Contains(output.HookSpecificOutput.PermissionDecisionReason, "Dangerous command blocked") {
		t.Errorf("PermissionDecisionReason does not contain expected message, got: %s", output.HookSpecificOutput.PermissionDecisionReason)
	}
}

func TestPreToolUseIntegration_AskOperation(t *testing.T) {
	// Test case 3: Real config file with ask decision -> permissionDecision: "ask"
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	configYAML := `PreToolUse:
  - matcher: "Write"
    conditions:
      - type: file_extension
        value: ".env"
    actions:
      - type: output
        message: "Modifying sensitive file - user confirmation required"
        permission_decision: "ask"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &PreToolUseInput{
		BaseInput: BaseInput{
			SessionID:      "test-session-123",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  PreToolUse,
		},
		ToolName: "Write",
		ToolInput: ToolInput{
			FilePath: ".env",
		},
	}

	rawJSON := map[string]any{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"tool_name":       input.ToolName,
		"tool_input": map[string]any{
			"file_path": input.ToolInput.FilePath,
		},
	}

	output, err := executePreToolUseHooksJSON(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify JSON structure
	if !output.Continue {
		t.Errorf("Continue = false, want true (always true for PreToolUse)")
	}

	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}

	if output.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Errorf("HookEventName = %v, want PreToolUse", output.HookSpecificOutput.HookEventName)
	}

	if output.HookSpecificOutput.PermissionDecision != "ask" {
		t.Errorf("PermissionDecision = %v, want ask", output.HookSpecificOutput.PermissionDecision)
	}

	if !strings.Contains(output.HookSpecificOutput.PermissionDecisionReason, "user confirmation required") {
		t.Errorf("PermissionDecisionReason does not contain expected message, got: %s", output.HookSpecificOutput.PermissionDecisionReason)
	}
}

func TestPreToolUseIntegration_MultipleActions(t *testing.T) {
	// Test case 4: Multiple actions with field concatenation
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	configYAML := `PreToolUse:
  - matcher: "Write"
    actions:
      - type: output
        message: "First validation passed"
        permission_decision: "allow"
      - type: output
        message: "Second validation passed"
        permission_decision: "allow"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &PreToolUseInput{
		BaseInput: BaseInput{
			SessionID:      "test-session-123",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  PreToolUse,
		},
		ToolName: "Write",
		ToolInput: ToolInput{
			FilePath: "test.txt",
		},
	}

	rawJSON := map[string]any{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"tool_name":       input.ToolName,
		"tool_input": map[string]any{
			"file_path": input.ToolInput.FilePath,
		},
	}

	output, err := executePreToolUseHooksJSON(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify field concatenation
	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}

	expectedReason := "First validation passed\nSecond validation passed"
	if output.HookSpecificOutput.PermissionDecisionReason != expectedReason {
		t.Errorf("PermissionDecisionReason = %q, want %q", output.HookSpecificOutput.PermissionDecisionReason, expectedReason)
	}

	// Last action's permissionDecision should win
	if output.HookSpecificOutput.PermissionDecision != "allow" {
		t.Errorf("PermissionDecision = %v, want allow (last action wins)", output.HookSpecificOutput.PermissionDecision)
	}
}

func TestPreToolUseIntegration_EarlyReturnOnDeny(t *testing.T) {
	// Test case 5: Early return when permissionDecision is "deny"
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	configYAML := `PreToolUse:
  - matcher: "Write"
    actions:
      - type: output
        message: "First action blocks"
        permission_decision: "deny"
      - type: output
        message: "Second action should not run"
        permission_decision: "allow"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &PreToolUseInput{
		BaseInput: BaseInput{
			SessionID:      "test-session-123",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  PreToolUse,
		},
		ToolName: "Write",
		ToolInput: ToolInput{
			FilePath: "test.txt",
		},
	}

	rawJSON := map[string]any{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"tool_name":       input.ToolName,
		"tool_input": map[string]any{
			"file_path": input.ToolInput.FilePath,
		},
	}

	output, err := executePreToolUseHooksJSON(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify early return
	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}

	// Only first action's message should be present
	if output.HookSpecificOutput.PermissionDecisionReason != "First action blocks" {
		t.Errorf("PermissionDecisionReason = %q, want 'First action blocks' (second action should not run)", output.HookSpecificOutput.PermissionDecisionReason)
	}

	if output.HookSpecificOutput.PermissionDecision != "deny" {
		t.Errorf("PermissionDecision = %v, want deny", output.HookSpecificOutput.PermissionDecision)
	}
}
