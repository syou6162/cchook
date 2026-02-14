package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestCheckMatcher(t *testing.T) {
	tests := []struct {
		name     string
		matcher  string
		toolName string
		want     bool
	}{
		{"Empty matcher matches all", "", "Write", true},
		{"Exact match", "Write", "Write", true},
		{"Partial match", "Write", "WriteFile", true},
		{"No match", "Edit", "Write", false},
		{"Multiple patterns - first match", "Write|Edit", "Write", true},
		{"Multiple patterns - second match", "Write|Edit", "Edit", true},
		{"Multiple patterns - no match", "Write|Edit", "Read", false},
		{"Whitespace handling", " Write | Edit ", "Write", true},
		{"Case sensitive", "write", "Write", false},
		{"Complex tool name", "Multi", "MultiEdit", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkMatcher(tt.matcher, tt.toolName); got != tt.want {
				t.Errorf("checkMatcher(%q, %q) = %v, want %v", tt.matcher, tt.toolName, got, tt.want)
			}
		})
	}
}

func TestCheckNotificationMatcher(t *testing.T) {
	tests := []struct {
		name             string
		matcher          string
		notificationType string
		want             bool
	}{
		{"Empty matcher matches all", "", "idle_prompt", true},
		{"Exact match", "idle_prompt", "idle_prompt", true},
		{"Partial string does NOT match (idle vs idle_prompt)", "idle", "idle_prompt", false},
		{"No match", "permission_prompt", "idle_prompt", false},
		{"Multiple patterns - first match", "idle_prompt|permission_prompt", "idle_prompt", true},
		{"Multiple patterns - second match", "idle_prompt|permission_prompt", "permission_prompt", true},
		{"Multiple patterns - no match", "idle_prompt|permission_prompt", "auth_success", false},
		{"Whitespace handling", " idle_prompt | permission_prompt ", "idle_prompt", true},
		{"Case sensitive", "Idle_prompt", "idle_prompt", false},
		{"Auth success match", "auth_success", "auth_success", true},
		{"Elicitation dialog match", "elicitation_dialog", "elicitation_dialog", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkNotificationMatcher(tt.matcher, tt.notificationType); got != tt.want {
				t.Errorf("checkNotificationMatcher(%q, %q) = %v, want %v", tt.matcher, tt.notificationType, got, tt.want)
			}
		})
	}
}

func TestParseInput_Success(t *testing.T) {
	// JSONを標準入力にセット
	jsonInput := `{
		"session_id": "test-session",
		"transcript_path": "/tmp/transcript",
		"hook_event_name": "PreToolUse",
		"tool_name": "Write",
		"tool_input": {"file_path": "test.go"}
	}`

	// 標準入力をバックアップして復元
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// パイプを作成して標準入力として設定
	r, w, _ := os.Pipe()
	os.Stdin = r

	// JSONデータを書き込み
	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(jsonInput))
	}()

	// parseInputをテスト
	result, _, err := parseInput[*PreToolUseInput](PreToolUse)
	if err != nil {
		t.Fatalf("parseInput() error = %v", err)
	}

	if result.SessionID != "test-session" {
		t.Errorf("Expected SessionID 'test-session', got '%s'", result.SessionID)
	}

	if result.ToolName != "Write" {
		t.Errorf("Expected ToolName 'Write', got '%s'", result.ToolName)
	}
}

func TestParseInput_InvalidJSON(t *testing.T) {
	// 不正なJSONを標準入力にセット
	invalidJSON := `{"invalid": json}`

	// 標準入力をバックアップして復元
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// パイプを作成
	r, w, _ := os.Pipe()
	os.Stdin = r

	// 不正なJSONを書き込み
	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(invalidJSON))
	}()

	// parseInputをテスト(エラーが期待される)
	_, _, err := parseInput[*PreToolUseInput](PreToolUse)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	if !strings.Contains(err.Error(), "failed to decode JSON input") {
		t.Errorf("Expected decode error message, got: %v", err)
	}
}

func TestRunCommand_Success(t *testing.T) {
	// 成功するコマンドをテスト
	err := runCommand("echo test", false, nil)
	if err != nil {
		t.Errorf("runCommand() error = %v, expected nil", err)
	}
}

func TestRunCommand_EmptyCommand(t *testing.T) {
	err := runCommand("", false, nil)
	if err == nil {
		t.Error("Expected error for empty command, got nil")
	}

	if !strings.Contains(err.Error(), "empty command") {
		t.Errorf("Expected 'empty command' error, got: %v", err)
	}
}

func TestRunCommand_CommandNotFound(t *testing.T) {
	// 存在しないコマンドをテスト
	err := runCommand("nonexistent-command-12345", false, nil)
	if err == nil {
		t.Error("Expected error for non-existent command, got nil")
	}
}

func TestRunCommand_CommandFails(t *testing.T) {
	// 失敗するコマンドをテスト（falseコマンドは常に終了コード1を返す）
	err := runCommand("false", false, nil)
	if err == nil {
		t.Error("Expected error for failing command, got nil")
	}
}

// createTestTranscript creates a temporary transcript file for testing
func createTestTranscript(t *testing.T, sessionID string, userPromptCount int) string {
	t.Helper()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "transcript-*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Write test data
	for i := 0; i < userPromptCount; i++ {
		entry := map[string]interface{}{
			"type":      "user",
			"sessionId": sessionID,
			"message": map[string]interface{}{
				"content": fmt.Sprintf("Test prompt %d", i+1),
			},
		}

		// Add one entry with isMeta: true (but don't skip counting - it's still a user prompt)
		// Note: In the test, we're testing that isMeta entries are excluded from count,
		// but here we're generating userPromptCount regular user prompts
		// We won't add isMeta to keep the count predictable

		data, err := json.Marshal(entry)
		if err != nil {
			t.Fatalf("Failed to marshal JSON: %v", err)
		}

		if _, err := tmpFile.Write(data); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		if _, err := tmpFile.WriteString("\n"); err != nil {
			t.Fatalf("Failed to write newline: %v", err)
		}
	}

	// Add some assistant messages
	for i := 0; i < 3; i++ {
		entry := map[string]interface{}{
			"type":      "assistant",
			"sessionId": sessionID,
			"message": map[string]interface{}{
				"content": fmt.Sprintf("Response %d", i+1),
			},
		}

		data, err := json.Marshal(entry)
		if err != nil {
			t.Fatalf("Failed to marshal JSON: %v", err)
		}

		if _, err := tmpFile.Write(data); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		if _, err := tmpFile.WriteString("\n"); err != nil {
			t.Fatalf("Failed to write newline: %v", err)
		}
	}

	_ = tmpFile.Close()
	return tmpFile.Name()
}

func TestRunCommand_WithStdin(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		useStdin    bool
		data        interface{}
		wantErr     bool
		wantErrMsg  string
		validateOut func(t *testing.T) // Optional validation function
	}{
		{
			name:     "useStdin=false, simple command",
			command:  "echo 'test'",
			useStdin: false,
			data:     nil,
			wantErr:  false,
		},
		{
			name:     "useStdin=true, simple data",
			command:  "cat",
			useStdin: true,
			data: map[string]interface{}{
				"tool_name": "Write",
				"file_path": "test.go",
			},
			wantErr: false,
		},
		{
			name:     "useStdin=true, complex nested data",
			command:  "cat > /dev/null",
			useStdin: true,
			data: map[string]interface{}{
				"session_id": "test123",
				"tool_input": map[string]interface{}{
					"file_path": "main.go",
					"content":   "package main\n\nfunc main() {}",
				},
			},
			wantErr: false,
		},
		{
			name:       "useStdin=true, unmarshalable data (channel)",
			command:    "cat",
			useStdin:   true,
			data:       make(chan int), // channels cannot be marshaled to JSON
			wantErr:    true,
			wantErrMsg: "failed to marshal JSON for stdin",
		},
		{
			name:       "empty command",
			command:    "",
			useStdin:   false,
			data:       nil,
			wantErr:    true,
			wantErrMsg: "empty command",
		},
		{
			name:       "empty command with useStdin",
			command:    "   ",
			useStdin:   true,
			data:       map[string]string{"key": "value"},
			wantErr:    true,
			wantErrMsg: "empty command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runCommand(tt.command, tt.useStdin, tt.data)

			if tt.wantErr {
				if err == nil {
					t.Errorf("runCommand() expected error containing %q, got nil", tt.wantErrMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("runCommand() error = %v, want error containing %q", err, tt.wantErrMsg)
				}
			} else {
				if err != nil {
					t.Errorf("runCommand() unexpected error = %v", err)
				}
			}

			if tt.validateOut != nil {
				tt.validateOut(t)
			}
		})
	}
}

func TestRunCommandWithOutput_Success(t *testing.T) {
	stdout, stderr, exitCode, err := runCommandWithOutput("echo 'hello'", false, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
	if stdout != "hello\n" {
		t.Errorf("Expected stdout 'hello\\n', got %q", stdout)
	}
	if stderr != "" {
		t.Errorf("Expected empty stderr, got %q", stderr)
	}
}

func TestRunCommandWithOutput_Failure(t *testing.T) {
	stdout, stderr, exitCode, err := runCommandWithOutput("sh -c 'echo error >&2; exit 42'", false, nil)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if exitCode != 42 {
		t.Errorf("Expected exit code 42, got %d", exitCode)
	}
	if stdout != "" {
		t.Errorf("Expected empty stdout, got %q", stdout)
	}
	if stderr != "error\n" {
		t.Errorf("Expected stderr 'error\\n', got %q", stderr)
	}
}

func TestRunCommandWithOutput_UseStdin(t *testing.T) {
	data := map[string]string{"key": "value"}
	stdout, stderr, exitCode, err := runCommandWithOutput("cat", true, data)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
	if stdout != "{\"key\":\"value\"}" {
		t.Errorf("Expected stdout '{\"key\":\"value\"}', got %q", stdout)
	}
	if stderr != "" {
		t.Errorf("Expected empty stderr, got %q", stderr)
	}
}

func TestRunCommandWithOutput_EmptyCommand(t *testing.T) {
	stdout, stderr, exitCode, err := runCommandWithOutput("", false, nil)

	if err == nil {
		t.Error("Expected error for empty command, got nil")
	}
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}
	if stdout != "" {
		t.Errorf("Expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Errorf("Expected empty stderr, got %q", stderr)
	}
}

func TestRunCommandWithOutput_EmptyOutput(t *testing.T) {
	stdout, stderr, exitCode, err := runCommandWithOutput("true", false, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
	if stdout != "" {
		t.Errorf("Expected empty stdout, got %q", stdout)
	}
	if stderr != "" {
		t.Errorf("Expected empty stderr, got %q", stderr)
	}
}

func TestRunCommandWithOutput_ExitCodes(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		wantExitCode int
	}{
		{"Exit 0", "exit 0", 0},
		{"Exit 1", "exit 1", 1},
		{"Exit 2", "exit 2", 2},
		{"Exit 127", "exit 127", 127},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, exitCode, _ := runCommandWithOutput(tt.command, false, nil)
			if exitCode != tt.wantExitCode {
				t.Errorf("Expected exit code %d, got %d", tt.wantExitCode, exitCode)
			}
		})
	}
}

func TestContainsProcessSubstitution(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{
			name:    "empty string returns false",
			command: "",
			want:    false,
		},
		{
			name:    "normal command returns false",
			command: "ls -la",
			want:    false,
		},
		{
			name:    "command with <() returns true",
			command: "diff -u file1 <(head -48 file2)",
			want:    true,
		},
		{
			name:    "command with >() returns true",
			command: "echo foo >(cat > output.txt)",
			want:    true,
		},
		{
			name:    "double quoted <( returns false",
			command: `echo "<(cmd)"`,
			want:    false,
		},
		{
			name:    "single quoted <( returns false",
			command: `echo '<(cmd)'`,
			want:    false,
		},
		{
			name:    "escaped <( returns true",
			command: `echo \<(cmd)`,
			want:    true,
		},
		{
			name:    "command with && and <() returns true",
			command: "cmd1 && cmd2 <(cmd)",
			want:    true,
		},
		{
			name:    "command with || and <() returns true",
			command: "cmd1 || cmd2 <(cmd)",
			want:    true,
		},
		{
			name:    "command with ; and <() returns true",
			command: "cmd1 ; cmd2 <(cmd)",
			want:    true,
		},
		{
			name:    "command with | and <() returns true",
			command: "cmd1 | cmd2 <(cmd)",
			want:    true,
		},
		{
			name:    "nested process substitution returns true",
			command: "<(diff <(a) <(b))",
			want:    true,
		},
		{
			name:    "command substitution $(cmd) returns false",
			command: "echo $(cmd)",
			want:    false,
		},
		{
			name:    "parse error with <( fallback returns true",
			command: "echo '<unclosed <(",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsProcessSubstitution(tt.command)
			if got != tt.want {
				t.Errorf("containsProcessSubstitution(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}
