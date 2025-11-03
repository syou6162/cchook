//go:build integration
// +build integration

package main

import (
	"os"
	"strings"
	"testing"
)

// UserPromptSubmit integration tests
func TestUserPromptSubmitIntegration_BlockPattern(t *testing.T) {
	// Test case 1: Real config file with prompt pattern match -> block message, decision: "block"
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	configYAML := `UserPromptSubmit:
  - conditions:
      - type: prompt_regex
        value: "delete|rm -rf"
    actions:
      - type: output
        message: "Dangerous command detected"
        decision: "block"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &UserPromptSubmitInput{
		BaseInput: BaseInput{
			SessionID:      "test-session-123",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  UserPromptSubmit,
		},
		Prompt: "delete all files",
	}

	rawJSON := map[string]interface{}{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"prompt":          input.Prompt,
	}

	output, err := executeUserPromptSubmitHooks(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify JSON structure
	if !output.Continue {
		t.Errorf("Continue = false, want true (always true for UserPromptSubmit)")
	}

	if output.Decision != "block" {
		t.Errorf("Decision = %v, want block", output.Decision)
	}

	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}

	if output.HookSpecificOutput.HookEventName != "UserPromptSubmit" {
		t.Errorf("HookEventName = %v, want UserPromptSubmit", output.HookSpecificOutput.HookEventName)
	}

	if !strings.Contains(output.HookSpecificOutput.AdditionalContext, "Dangerous command detected") {
		t.Errorf("AdditionalContext does not contain 'Dangerous command detected', got: %s", output.HookSpecificOutput.AdditionalContext)
	}
}

func TestUserPromptSubmitIntegration_AllowPattern(t *testing.T) {
	// Test case 2: Real config file with normal prompt -> allow message, decision: "allow"
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	configYAML := `UserPromptSubmit:
  - actions:
      - type: output
        message: "Safe command"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &UserPromptSubmitInput{
		BaseInput: BaseInput{
			SessionID:      "test-session",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  UserPromptSubmit,
		},
		Prompt: "list files",
	}

	rawJSON := map[string]interface{}{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"prompt":          input.Prompt,
	}

	output, err := executeUserPromptSubmitHooks(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify JSON structure
	if !output.Continue {
		t.Errorf("Continue = false, want true (always true for UserPromptSubmit)")
	}

	if output.Decision != "allow" {
		t.Errorf("Decision = %v, want allow", output.Decision)
	}

	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}

	if output.HookSpecificOutput.HookEventName != "UserPromptSubmit" {
		t.Errorf("HookEventName = %v, want UserPromptSubmit", output.HookSpecificOutput.HookEventName)
	}

	if !strings.Contains(output.HookSpecificOutput.AdditionalContext, "Safe command") {
		t.Errorf("AdditionalContext does not contain 'Safe command', got: %s", output.HookSpecificOutput.AdditionalContext)
	}
}

func TestUserPromptSubmitIntegration_MultipleActions(t *testing.T) {
	// Test case 3: Multiple actions -> messages concatenated with "\n"
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	configYAML := `UserPromptSubmit:
  - actions:
      - type: output
        message: "First message"
      - type: output
        message: "Second message"
      - type: output
        message: "Third message"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &UserPromptSubmitInput{
		BaseInput: BaseInput{
			SessionID:      "test-session",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  UserPromptSubmit,
		},
		Prompt: "test prompt",
	}

	rawJSON := map[string]interface{}{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"prompt":          input.Prompt,
	}

	output, err := executeUserPromptSubmitHooks(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify messages are concatenated with newlines
	expected := "First message\nSecond message\nThird message"
	if output.HookSpecificOutput.AdditionalContext != expected {
		t.Errorf("AdditionalContext = %q, want %q", output.HookSpecificOutput.AdditionalContext, expected)
	}

	// Verify Continue is always true
	if !output.Continue {
		t.Errorf("Continue = false, want true (always true for UserPromptSubmit)")
	}

	// Verify Decision is "allow" (last value)
	if output.Decision != "allow" {
		t.Errorf("Decision = %v, want allow", output.Decision)
	}
}

func TestUserPromptSubmitIntegration_EarlyReturn(t *testing.T) {
	// Test case 4: First action decision: "block" -> early return
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	configYAML := `UserPromptSubmit:
  - actions:
      - type: output
        message: "Blocked"
        decision: "block"
      - type: output
        message: "Should not execute"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &UserPromptSubmitInput{
		BaseInput: BaseInput{
			SessionID:      "test-session",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  UserPromptSubmit,
		},
		Prompt: "test prompt",
	}

	rawJSON := map[string]interface{}{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"prompt":          input.Prompt,
	}

	output, err := executeUserPromptSubmitHooks(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify early return - second action should not execute
	if output.HookSpecificOutput.AdditionalContext != "Blocked" {
		t.Errorf("AdditionalContext = %q, want 'Blocked' (second action should not execute)", output.HookSpecificOutput.AdditionalContext)
	}

	// Verify Decision is "block"
	if output.Decision != "block" {
		t.Errorf("Decision = %v, want block", output.Decision)
	}

	// Verify Continue is always true
	if !output.Continue {
		t.Errorf("Continue = false, want true (always true for UserPromptSubmit)")
	}
}
