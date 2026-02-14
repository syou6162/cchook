package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRunHooks_UnknownEventType(t *testing.T) {
	config := &Config{}

	err := runHooks(config, HookEventType("UnknownEvent"))
	if err == nil {
		t.Error("Expected error for unknown event type, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported event type") {
		t.Errorf("Expected 'unsupported event type' error, got: %v", err)
	}
}

func TestDryRunHooks_UnknownEventType(t *testing.T) {
	config := &Config{}

	err := dryRunHooks(config, HookEventType("UnknownEvent"))
	if err == nil {
		t.Error("Expected error for unknown event type, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported event type") {
		t.Errorf("Expected 'unsupported event type' error, got: %v", err)
	}
}

func TestDryRunHooks_Success(t *testing.T) {
	// 標準入力をバックアップして復元
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r2, w2, _ := os.Pipe()
	os.Stdout = w2

	// 正常なJSONを標準入力に設定
	r, w, _ := os.Pipe()
	os.Stdin = r

	jsonInput := `{
		"session_id": "test",
		"transcript_path": "/tmp/test",
		"hook_event_name": "PreToolUse",
		"tool_name": "Write",
		"tool_input": {"file_path": "test.go"}
	}`

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(jsonInput))
	}()

	config := &Config{
		PreToolUse: []PreToolUseHook{
			{
				Matcher: "Write",
				Actions: []Action{
					{Type: "command", Command: "echo {.tool_input.file_path}"},
				},
			},
		},
	}

	err := dryRunHooks(config, PreToolUse)

	// 標準出力を復元
	_ = w2.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r2)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunHooks() error = %v", err)
	}

	if !strings.Contains(output, "=== PreToolUse Hooks (Dry Run) ===") {
		t.Errorf("Expected dry run header in output, got: %q", output)
	}

	if !strings.Contains(output, "Command: echo test.go") {
		t.Errorf("Expected command with replaced variables, got: %q", output)
	}
	t.Logf("Full output: %q", output)
}

// エラーケース: 空の標準入力
func TestRunHooks_EmptyInput(t *testing.T) {
	// 標準入力をバックアップして復元
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// 空の標準入力を設定
	r, w, _ := os.Pipe()
	os.Stdin = r
	_ = w.Close() // すぐに閉じて EOF を発生させる

	config := &Config{}
	err := runHooks(config, PreToolUse)

	if err == nil {
		t.Error("Expected error for empty input, got nil")
	}
}

// エラーケース: 部分的なJSON
func TestRunHooks_PartialJSON(t *testing.T) {
	// 標準入力をバックアップして復元
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// 部分的なJSONを標準入力に設定
	r, w, _ := os.Pipe()
	os.Stdin = r

	partialJSON := `{"session_id": "test"` // 不完全なJSON

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(partialJSON))
	}()

	config := &Config{}
	err := runHooks(config, PreToolUse)

	if err == nil {
		t.Error("Expected error for partial JSON, got nil")
	}
}

// エラーケース: 型が一致しないJSON
func TestRunHooks_WrongJSONType(t *testing.T) {
	// 標準入力をバックアップして復元
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// PreToolUse以外の構造のJSONを標準入力に設定
	r, w, _ := os.Pipe()
	os.Stdin = r

	wrongJSON := `{
		"session_id": "test",
		"transcript_path": "/tmp/test",
		"hook_event_name": "PreToolUse",
		"message": "This is notification format"
	}` // NotificationInput の形式

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(wrongJSON))
	}()

	config := &Config{}
	// PreToolUseとして解釈しようとするが、tool_nameがない
	err := runHooks(config, PreToolUse)

	// この場合エラーにはならないが、tool_nameは空文字になる
	// 実際のアプリケーションではバリデーションを追加することを推奨
	if err != nil {
		t.Logf("Note: Got error for wrong JSON type: %v", err)
	}
}

// SessionStart integration tests
func TestSessionStartIntegration_RealConfigWithGoMod(t *testing.T) {
	// Test case 1: Real config file (go.mod exists) -> serena recommendation message in additionalContext
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	configYAML := `SessionStart:
  - matcher: "startup"
    conditions:
      - type: file_exists
        value: "go.mod"
    actions:
      - type: output
        message: "Golangファイルを検索や修正する際は、serena mcpを活用しましょう"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &SessionStartInput{
		BaseInput: BaseInput{
			SessionID:      "test-session-123",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  SessionStart,
		},
		Source: "startup",
	}

	rawJSON := map[string]any{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"source":          input.Source,
	}

	output, err := executeSessionStartHooks(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify JSON structure
	if !output.Continue {
		t.Errorf("Continue = false, want true")
	}

	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}

	if output.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Errorf("HookEventName = %v, want SessionStart", output.HookSpecificOutput.HookEventName)
	}

	if !strings.Contains(output.HookSpecificOutput.AdditionalContext, "serena mcp") {
		t.Errorf("AdditionalContext does not contain 'serena mcp', got: %s", output.HookSpecificOutput.AdditionalContext)
	}
}

func TestSessionStartIntegration_MultipleActions(t *testing.T) {
	// Test case 3: Multiple actions -> messages concatenated with "\n"
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	configYAML := `SessionStart:
  - matcher: "startup"
    actions:
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

	input := &SessionStartInput{
		BaseInput: BaseInput{
			SessionID:      "test-session",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  SessionStart,
		},
		Source: "startup",
	}

	rawJSON := map[string]any{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"source":          input.Source,
	}

	output, err := executeSessionStartHooks(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedContext := "First message\nSecond message\nThird message"
	if output.HookSpecificOutput.AdditionalContext != expectedContext {
		t.Errorf("AdditionalContext = %q, want %q", output.HookSpecificOutput.AdditionalContext, expectedContext)
	}
}

func TestSessionStartIntegration_ContinueFalse(t *testing.T) {
	// Test case: Action with continue: false
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	configYAML := `SessionStart:
  - matcher: "startup"
    actions:
      - type: output
        message: "Session blocked"
        continue: false
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &SessionStartInput{
		BaseInput: BaseInput{
			SessionID:      "test-session",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  SessionStart,
		},
		Source: "startup",
	}

	rawJSON := map[string]any{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"source":          input.Source,
	}

	output, err := executeSessionStartHooks(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify continue is false
	if output.Continue {
		t.Errorf("Continue = true, want false")
	}

	if output.HookSpecificOutput.AdditionalContext != "Session blocked" {
		t.Errorf("AdditionalContext = %q, want 'Session blocked'", output.HookSpecificOutput.AdditionalContext)
	}
}

func TestSessionStartIntegration_JSONFieldsAlwaysPresent(t *testing.T) {
	// Test case 6: JSON output validation - continue field always present
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	// Empty config - no hooks match
	configYAML := `SessionStart: []
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &SessionStartInput{
		BaseInput: BaseInput{
			SessionID:      "test-session",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  SessionStart,
		},
		Source: "startup",
	}

	rawJSON := map[string]any{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"source":          input.Source,
	}

	output, err := executeSessionStartHooks(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify continue field is always present (defaults to true)
	if !output.Continue {
		t.Errorf("Continue = false, want true (default)")
	}

	// Verify HookSpecificOutput can be nil when no hooks match
	// This is expected behavior - no hooks executed means no output
}

func TestSessionStartIntegration_DirectoryNotExists(t *testing.T) {
	// Test case 2: directory not exists -> creation request message in additionalContext
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	// Use a subdirectory that definitely doesn't exist
	nonExistentDir := tmpDir + "/nonexistent-dir-12345"

	configYAML := `SessionStart:
  - matcher: "startup"
    conditions:
      - type: dir_not_exists
        value: "` + nonExistentDir + `"
    actions:
      - type: output
        message: "Directory does not exist. Please create it."
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &SessionStartInput{
		BaseInput: BaseInput{
			SessionID:      "test-session",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  SessionStart,
		},
		Source: "startup",
	}

	rawJSON := map[string]any{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"source":          input.Source,
	}

	output, err := executeSessionStartHooks(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}

	if !strings.Contains(output.HookSpecificOutput.AdditionalContext, "does not exist") {
		t.Errorf("AdditionalContext does not contain 'does not exist', got: %s", output.HookSpecificOutput.AdditionalContext)
	}
}

func TestSessionStartIntegration_CommandActionSuccess(t *testing.T) {
	// Test case 4: Command action success -> valid JSON output
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	// Create a script that outputs valid JSON
	scriptPath := tmpDir + "/output-json.sh"
	scriptContent := `#!/bin/sh
cat <<'EOF'
{
  "continue": true,
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "Command executed successfully"
  },
  "systemMessage": "Script output"
}
EOF
`

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to write script file: %v", err)
	}

	configYAML := `SessionStart:
  - matcher: "startup"
    actions:
      - type: command
        command: "` + scriptPath + `"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &SessionStartInput{
		BaseInput: BaseInput{
			SessionID:      "test-session",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  SessionStart,
		},
		Source: "startup",
	}

	rawJSON := map[string]any{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"source":          input.Source,
	}

	output, err := executeSessionStartHooks(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify command output was parsed correctly
	if !output.Continue {
		t.Errorf("Continue = false, want true")
	}

	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}

	if output.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Errorf("HookEventName = %v, want SessionStart", output.HookSpecificOutput.HookEventName)
	}

	if output.HookSpecificOutput.AdditionalContext != "Command executed successfully" {
		t.Errorf("AdditionalContext = %q, want 'Command executed successfully'", output.HookSpecificOutput.AdditionalContext)
	}

	if output.SystemMessage != "Script output" {
		t.Errorf("SystemMessage = %q, want 'Script output'", output.SystemMessage)
	}
}

func TestSessionStartIntegration_CommandActionFailure(t *testing.T) {
	// Test case 5: Command action failure -> continue: false + systemMessage
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	// Create a script that fails
	scriptPath := tmpDir + "/fail-script.sh"
	scriptContent := `#!/bin/sh
echo "Error: command failed" >&2
exit 1
`

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to write script file: %v", err)
	}

	configYAML := `SessionStart:
  - matcher: "startup"
    actions:
      - type: command
        command: "` + scriptPath + `"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &SessionStartInput{
		BaseInput: BaseInput{
			SessionID:      "test-session",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  SessionStart,
		},
		Source: "startup",
	}

	rawJSON := map[string]any{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"source":          input.Source,
	}

	output, err := executeSessionStartHooks(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify command failure results in continue: false
	if output.Continue {
		t.Errorf("Continue = true, want false")
	}

	// Verify systemMessage contains error info
	if !strings.Contains(output.SystemMessage, "Command failed with exit code 1") {
		t.Errorf("SystemMessage does not contain error info, got: %s", output.SystemMessage)
	}

	if !strings.Contains(output.SystemMessage, "Error: command failed") {
		t.Errorf("SystemMessage does not contain stderr, got: %s", output.SystemMessage)
	}
}

func TestSessionStartIntegration_InvalidHookEventName(t *testing.T) {
	// Test schema validation: hookEventName must be "SessionStart"
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	// Create a script that outputs wrong hookEventName
	scriptPath := tmpDir + "/wrong-hook-name.sh"
	scriptContent := `#!/bin/sh
cat <<'EOF'
{
  "continue": true,
  "hookSpecificOutput": {
    "hookEventName": "UserPromptSubmit",
    "additionalContext": "Wrong hook type"
  }
}
EOF
`

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to write script file: %v", err)
	}

	configYAML := `SessionStart:
  - matcher: "startup"
    actions:
      - type: command
        command: "` + scriptPath + `"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &SessionStartInput{
		BaseInput: BaseInput{
			SessionID:      "test-session",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  SessionStart,
		},
		Source: "startup",
	}

	rawJSON := map[string]any{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"source":          input.Source,
	}

	output, err := executeSessionStartHooks(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify schema validation rejected wrong hookEventName
	if output.Continue {
		t.Errorf("Continue = true, want false (schema validation should reject)")
	}

	if !strings.Contains(output.SystemMessage, "validation failed") {
		t.Errorf("SystemMessage should contain 'validation failed', got: %s", output.SystemMessage)
	}
}

func TestSessionStartIntegration_UnsupportedFields(t *testing.T) {
	// Test that unsupported fields log warnings to stderr and are ignored (requirement 3.8)
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	// Create a script that outputs unsupported fields (permissionDecision, decision)
	scriptPath := tmpDir + "/unsupported-field.sh"
	scriptContent := `#!/bin/sh
cat <<'EOF'
{
  "continue": true,
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "Valid context"
  },
  "permissionDecision": "approve",
  "decision": "block"
}
EOF
`

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to write script file: %v", err)
	}

	configYAML := `SessionStart:
  - matcher: "startup"
    actions:
      - type: command
        command: "` + scriptPath + `"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &SessionStartInput{
		BaseInput: BaseInput{
			SessionID:      "test-session",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  SessionStart,
		},
		Source: "startup",
	}

	rawJSON := map[string]any{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"source":          input.Source,
	}

	// Note: This test logs warnings to stderr for unsupported fields:
	// "Warning: Field 'permissionDecision' is not supported for SessionStart hooks"
	// "Warning: Field 'decision' is not supported for SessionStart hooks"
	output, err := executeSessionStartHooks(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify processing continues despite unsupported fields
	if !output.Continue {
		t.Errorf("Continue = false, want true (processing should continue despite unsupported fields)")
	}

	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil")
	}

	if output.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Errorf("HookEventName = %q, want %q", output.HookSpecificOutput.HookEventName, "SessionStart")
	}

	if output.HookSpecificOutput.AdditionalContext != "Valid context" {
		t.Errorf("AdditionalContext = %q, want %q", output.HookSpecificOutput.AdditionalContext, "Valid context")
	}

	// Unsupported fields should be ignored (not cause errors)
	if output.SystemMessage != "" {
		t.Errorf("SystemMessage = %q, want empty (no error)", output.SystemMessage)
	}
}

func TestSessionStartIntegration_MissingHookSpecificOutput(t *testing.T) {
	// Test schema validation: hookSpecificOutput is required (prevents panic)
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	// Create a script that outputs JSON without hookSpecificOutput
	scriptPath := tmpDir + "/missing-hook-specific.sh"
	scriptContent := `#!/bin/sh
cat <<'EOF'
{
  "continue": true,
  "systemMessage": "Missing hookSpecificOutput"
}
EOF
`

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to write script file: %v", err)
	}

	configYAML := `SessionStart:
  - matcher: "startup"
    actions:
      - type: command
        command: "` + scriptPath + `"
`

	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	input := &SessionStartInput{
		BaseInput: BaseInput{
			SessionID:      "test-session",
			TranscriptPath: "/tmp/transcript",
			HookEventName:  SessionStart,
		},
		Source: "startup",
	}

	rawJSON := map[string]any{
		"session_id":      input.SessionID,
		"transcript_path": input.TranscriptPath,
		"hook_event_name": string(input.HookEventName),
		"source":          input.Source,
	}

	output, err := executeSessionStartHooks(config, input, rawJSON)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify hookEventName check rejected missing hookSpecificOutput (prevents panic)
	if output.Continue {
		t.Errorf("Continue = true, want false (should reject missing hookSpecificOutput)")
	}

	expectedMsg := "Command output is missing required field: hookSpecificOutput.hookEventName"
	if output.SystemMessage != expectedMsg {
		t.Errorf("SystemMessage = %q, want %q", output.SystemMessage, expectedMsg)
	}
}
