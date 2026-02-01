package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func TestCheckCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		input     *PreToolUseInput
		want      bool
		wantErr   bool
	}{
		{
			"file_extension match",
			Condition{Type: ConditionFileExtension, Value: ".go"},
			&PreToolUseInput{ToolInput: ToolInput{FilePath: "main.go"}},
			true,
			false,
		},
		{
			"file_extension no match",
			Condition{Type: ConditionFileExtension, Value: ".py"},
			&PreToolUseInput{ToolInput: ToolInput{FilePath: "main.go"}},
			false,
			false,
		},
		{
			"file_extension no file_path",
			Condition{Type: ConditionFileExtension, Value: ".go"},
			&PreToolUseInput{ToolInput: ToolInput{}},
			false,
			false,
		},
		{
			"command_contains match",
			Condition{Type: ConditionCommandContains, Value: "git add"},
			&PreToolUseInput{ToolInput: ToolInput{Command: "git add file.txt"}},
			true,
			false,
		},
		{
			"command_contains no match",
			Condition{Type: ConditionCommandContains, Value: "git commit"},
			&PreToolUseInput{ToolInput: ToolInput{Command: "git add file.txt"}},
			false,
			false,
		},
		{
			"command_contains no command",
			Condition{Type: ConditionCommandContains, Value: "git add"},
			&PreToolUseInput{ToolInput: ToolInput{}},
			false,
			false,
		},
		{
			"command_starts_with match",
			Condition{Type: ConditionCommandStartsWith, Value: "git"},
			&PreToolUseInput{ToolInput: ToolInput{Command: "git status"}},
			true,
			false,
		},
		{
			"command_starts_with no match",
			Condition{Type: ConditionCommandStartsWith, Value: "docker"},
			&PreToolUseInput{ToolInput: ToolInput{Command: "git status"}},
			false,
			false,
		},
		{
			"command_starts_with no command",
			Condition{Type: ConditionCommandStartsWith, Value: "git"},
			&PreToolUseInput{ToolInput: ToolInput{}},
			false,
			false,
		},
		{
			"file_exists match",
			Condition{Type: ConditionFileExists, Value: "/tmp"},
			&PreToolUseInput{ToolInput: ToolInput{}},
			true,
			false,
		},
		{
			"file_exists no match",
			Condition{Type: ConditionFileExists, Value: "/nonexistent/path"},
			&PreToolUseInput{ToolInput: ToolInput{}},
			false,
			false,
		},
		{
			"file_exists empty value",
			Condition{Type: ConditionFileExists, Value: ""},
			&PreToolUseInput{ToolInput: ToolInput{}},
			false,
			false,
		},
		{
			"url_starts_with match",
			Condition{Type: ConditionURLStartsWith, Value: "https://example.com"},
			&PreToolUseInput{ToolInput: ToolInput{URL: "https://example.com/page"}},
			true,
			false,
		},
		{
			"url_starts_with no match",
			Condition{Type: ConditionURLStartsWith, Value: "https://other.com"},
			&PreToolUseInput{ToolInput: ToolInput{URL: "https://example.com/page"}},
			false,
			false,
		},
		{
			"url_starts_with no url",
			Condition{Type: ConditionURLStartsWith, Value: "https://example.com"},
			&PreToolUseInput{ToolInput: ToolInput{}},
			false,
			false,
		},
		{
			"file_exists_recursive - file exists in current dir",
			Condition{Type: ConditionFileExistsRecursive, Value: "utils_test.go"},
			&PreToolUseInput{},
			true,
			false,
		},
		{
			"file_exists_recursive - file does not exist",
			Condition{Type: ConditionFileExistsRecursive, Value: "nonexistent.txt"},
			&PreToolUseInput{},
			false,
			false,
		},
		{
			"file_exists_recursive - go.mod exists",
			Condition{Type: ConditionFileExistsRecursive, Value: "go.mod"},
			&PreToolUseInput{},
			true,
			false,
		},
		{
			"cwd_is exact match",
			Condition{Type: ConditionCwdIs, Value: "/Users/yasuhisa.yoshida/work/cchook"},
			&PreToolUseInput{BaseInput: BaseInput{Cwd: "/Users/yasuhisa.yoshida/work/cchook"}},
			true,
			false,
		},
		{
			"cwd_is no match",
			Condition{Type: ConditionCwdIs, Value: "/Users/yasuhisa.yoshida/work/other"},
			&PreToolUseInput{BaseInput: BaseInput{Cwd: "/Users/yasuhisa.yoshida/work/cchook"}},
			false,
			false,
		},
		{
			"cwd_is_not matches when different",
			Condition{Type: ConditionCwdIsNot, Value: "/Users/yasuhisa.yoshida/work/other"},
			&PreToolUseInput{BaseInput: BaseInput{Cwd: "/Users/yasuhisa.yoshida/work/cchook"}},
			true,
			false,
		},
		{
			"cwd_is_not doesn't match when same",
			Condition{Type: ConditionCwdIsNot, Value: "/Users/yasuhisa.yoshida/work/cchook"},
			&PreToolUseInput{BaseInput: BaseInput{Cwd: "/Users/yasuhisa.yoshida/work/cchook"}},
			false,
			false,
		},
		{
			"cwd_contains matches substring",
			Condition{Type: ConditionCwdContains, Value: "cchook"},
			&PreToolUseInput{BaseInput: BaseInput{Cwd: "/Users/yasuhisa.yoshida/work/cchook"}},
			true,
			false,
		},
		{
			"cwd_contains matches path segment",
			Condition{Type: ConditionCwdContains, Value: "/work/"},
			&PreToolUseInput{BaseInput: BaseInput{Cwd: "/Users/yasuhisa.yoshida/work/cchook"}},
			true,
			false,
		},
		{
			"cwd_contains no match",
			Condition{Type: ConditionCwdContains, Value: "other-project"},
			&PreToolUseInput{BaseInput: BaseInput{Cwd: "/Users/yasuhisa.yoshida/work/cchook"}},
			false,
			false,
		},
		{
			"cwd_not_contains matches when not present",
			Condition{Type: ConditionCwdNotContains, Value: "other-project"},
			&PreToolUseInput{BaseInput: BaseInput{Cwd: "/Users/yasuhisa.yoshida/work/cchook"}},
			true,
			false,
		},
		{
			"cwd_not_contains doesn't match when present",
			Condition{Type: ConditionCwdNotContains, Value: "cchook"},
			&PreToolUseInput{BaseInput: BaseInput{Cwd: "/Users/yasuhisa.yoshida/work/cchook"}},
			false,
			false,
		},
		{
			"cwd_contains works with empty cwd",
			Condition{Type: ConditionCwdContains, Value: "test"},
			&PreToolUseInput{BaseInput: BaseInput{Cwd: ""}},
			false,
			false,
		},
		{
			"file_not_exists - file doesn't exist",
			Condition{Type: ConditionFileNotExists, Value: "/nonexistent/path/file.txt"},
			&PreToolUseInput{},
			true,
			false,
		},
		{
			"file_not_exists - file exists",
			Condition{Type: ConditionFileNotExists, Value: "utils_test.go"},
			&PreToolUseInput{},
			false,
			false,
		},
		{
			"file_not_exists_recursive - file doesn't exist",
			Condition{Type: ConditionFileNotExistsRecursive, Value: "nonexistent.txt"},
			&PreToolUseInput{},
			true,
			false,
		},
		{
			"file_not_exists_recursive - file exists",
			Condition{Type: ConditionFileNotExistsRecursive, Value: "go.mod"},
			&PreToolUseInput{},
			false,
			false,
		},
		{
			"dir_exists - directory exists",
			Condition{Type: ConditionDirExists, Value: "."},
			&PreToolUseInput{},
			true,
			false,
		},
		{
			"dir_exists - directory doesn't exist",
			Condition{Type: ConditionDirExists, Value: "/nonexistent/directory"},
			&PreToolUseInput{},
			false,
			false,
		},
		{
			"dir_exists - file is not a directory",
			Condition{Type: ConditionDirExists, Value: "utils_test.go"},
			&PreToolUseInput{},
			false,
			false,
		},
		{
			"dir_exists_recursive - directory exists",
			Condition{Type: ConditionDirExistsRecursive, Value: ".github"},
			&PreToolUseInput{},
			true,
			false,
		},
		{
			"dir_exists_recursive - directory doesn't exist",
			Condition{Type: ConditionDirExistsRecursive, Value: "nonexistent_dir"},
			&PreToolUseInput{},
			false,
			false,
		},
		{
			"dir_not_exists - directory doesn't exist",
			Condition{Type: ConditionDirNotExists, Value: "/nonexistent/directory"},
			&PreToolUseInput{},
			true,
			false,
		},
		{
			"dir_not_exists - directory exists",
			Condition{Type: ConditionDirNotExists, Value: "."},
			&PreToolUseInput{},
			false,
			false,
		},
		{
			"dir_not_exists_recursive - directory doesn't exist",
			Condition{Type: ConditionDirNotExistsRecursive, Value: "nonexistent_dir"},
			&PreToolUseInput{},
			true,
			false,
		},
		{
			"dir_not_exists_recursive - directory exists",
			Condition{Type: ConditionDirNotExistsRecursive, Value: ".github"},
			&PreToolUseInput{},
			false,
			false,
		},
		{
			"unknown condition type - error",
			Condition{Type: ConditionType{"unknown_type"}, Value: "test"},
			&PreToolUseInput{},
			false,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkPreToolUseCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkPreToolUseCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkPreToolUseCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckPostToolUseCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		input     *PostToolUseInput
		want      bool
		wantErr   bool
	}{
		{
			"file_extension match",
			Condition{Type: ConditionFileExtension, Value: ".go"},
			&PostToolUseInput{ToolInput: ToolInput{FilePath: "main.go"}},
			true,
			false,
		},
		{
			"command_contains match",
			Condition{Type: ConditionCommandContains, Value: "build"},
			&PostToolUseInput{ToolInput: ToolInput{Command: "go build main.go"}},
			true,
			false,
		},
		{
			"command_starts_with match",
			Condition{Type: ConditionCommandStartsWith, Value: "npm"},
			&PostToolUseInput{ToolInput: ToolInput{Command: "npm install"}},
			true,
			false,
		},
		{
			"command_starts_with no match",
			Condition{Type: ConditionCommandStartsWith, Value: "yarn"},
			&PostToolUseInput{ToolInput: ToolInput{Command: "npm install"}},
			false,
			false,
		},
		{
			"file_exists match",
			Condition{Type: ConditionFileExists, Value: "/tmp"},
			&PostToolUseInput{ToolInput: ToolInput{}},
			true,
			false,
		},
		{
			"file_exists no match",
			Condition{Type: ConditionFileExists, Value: "/nonexistent/path"},
			&PostToolUseInput{ToolInput: ToolInput{}},
			false,
			false,
		},
		{
			"url_starts_with match",
			Condition{Type: ConditionURLStartsWith, Value: "https://api.example.com"},
			&PostToolUseInput{ToolInput: ToolInput{URL: "https://api.example.com/v1/data"}},
			true,
			false,
		},
		{
			"url_starts_with no match",
			Condition{Type: ConditionURLStartsWith, Value: "https://api.other.com"},
			&PostToolUseInput{ToolInput: ToolInput{URL: "https://api.example.com/v1/data"}},
			false,
			false,
		},
		{
			"no match",
			Condition{Type: ConditionFileExtension, Value: ".py"},
			&PostToolUseInput{ToolInput: ToolInput{FilePath: "main.go"}},
			false,
			false,
		},
		{
			"cwd_contains in PostToolUse",
			Condition{Type: ConditionCwdContains, Value: "project"},
			&PostToolUseInput{BaseInput: BaseInput{Cwd: "/home/user/project/src"}},
			true,
			false,
		},
		{
			"cwd_is_not in PostToolUse",
			Condition{Type: ConditionCwdIsNot, Value: "/tmp"},
			&PostToolUseInput{BaseInput: BaseInput{Cwd: "/home/user/project"}},
			true,
			false,
		},
		{
			"unknown condition type - error",
			Condition{Type: ConditionType{"invalid_condition"}, Value: "test"},
			&PostToolUseInput{},
			false,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkPostToolUseCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkPostToolUseCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkPostToolUseCondition() = %v, want %v", got, tt.want)
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

	// parseInputをテスト（エラーが期待される）
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

	tmpFile.Close()
	return tmpFile.Name()
}

func TestCheckUserPromptSubmitCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		input     *UserPromptSubmitInput
		want      bool
		wantErr   bool
	}{
		{
			name: "prompt_regex contains pattern",
			condition: Condition{
				Type:  ConditionPromptRegex,
				Value: "secret",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "This contains a secret keyword",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "prompt_regex doesn't match",
			condition: Condition{
				Type:  ConditionPromptRegex,
				Value: "password",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "This is a normal prompt",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "prompt_regex starts with pattern",
			condition: Condition{
				Type:  ConditionPromptRegex,
				Value: "^DEBUG:",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "DEBUG: Show me the logs",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "prompt_regex starts with doesn't match",
			condition: Condition{
				Type:  ConditionPromptRegex,
				Value: "^ERROR:",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "Show me the error logs",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "prompt_regex ends with pattern",
			condition: Condition{
				Type:  ConditionPromptRegex,
				Value: "\\?$",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "What is this?",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "prompt_regex ends with doesn't match",
			condition: Condition{
				Type:  ConditionPromptRegex,
				Value: "!$",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "This is a statement",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "prompt_regex with OR pattern matches first",
			condition: Condition{
				Type:  ConditionPromptRegex,
				Value: "help|助けて|サポート",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "I need help with this",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "prompt_regex with OR pattern matches second",
			condition: Condition{
				Type:  ConditionPromptRegex,
				Value: "error|エラー|問題",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "エラーが発生しています",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "prompt_regex with complex regex pattern",
			condition: Condition{
				Type:  ConditionPromptRegex,
				Value: "^(DEBUG|INFO|WARN|ERROR):",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "ERROR: Connection failed",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "prompt_regex doesn't match",
			condition: Condition{
				Type:  ConditionPromptRegex,
				Value: "^(fix|修正|修理)",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "Show me the current status",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "prompt_regex with invalid regex pattern",
			condition: Condition{
				Type:  ConditionPromptRegex,
				Value: "[invalid(regex",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "test prompt",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "unknown condition type - error",
			condition: Condition{
				Type:  ConditionType{"not_supported"},
				Value: "test",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "test",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "every_n_prompts - 5th prompt should match",
			condition: Condition{
				Type:  ConditionEveryNPrompts,
				Value: "5",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: createTestTranscript(t, "test-session", 4), // 4 previous prompts
					HookEventName:  UserPromptSubmit,
				},
				Prompt: "This is the 5th prompt",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "every_n_prompts - 4th prompt should not match",
			condition: Condition{
				Type:  ConditionEveryNPrompts,
				Value: "5",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: createTestTranscript(t, "test-session", 3), // 3 previous prompts
					HookEventName:  UserPromptSubmit,
				},
				Prompt: "This is the 4th prompt",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "every_n_prompts - 10th prompt should match",
			condition: Condition{
				Type:  ConditionEveryNPrompts,
				Value: "10",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: createTestTranscript(t, "test-session", 9), // 9 previous prompts
					HookEventName:  UserPromptSubmit,
				},
				Prompt: "This is the 10th prompt",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "every_n_prompts - 15th prompt should match (every 5)",
			condition: Condition{
				Type:  ConditionEveryNPrompts,
				Value: "5",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: createTestTranscript(t, "test-session", 14), // 14 previous prompts
					HookEventName:  UserPromptSubmit,
				},
				Prompt: "This is the 15th prompt",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "every_n_prompts - different session ID should not count",
			condition: Condition{
				Type:  ConditionEveryNPrompts,
				Value: "5",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "different-session",
					TranscriptPath: createTestTranscript(t, "test-session", 10), // Different session in transcript
					HookEventName:  UserPromptSubmit,
				},
				Prompt: "First prompt in different session",
			},
			want:    false, // Should be 1st prompt for this session
			wantErr: false,
		},
		{
			name: "every_n_prompts - invalid value",
			condition: Condition{
				Type:  ConditionEveryNPrompts,
				Value: "invalid",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: createTestTranscript(t, "test-session", 5),
					HookEventName:  UserPromptSubmit,
				},
				Prompt: "Test prompt",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "every_n_prompts - negative value",
			condition: Condition{
				Type:  ConditionEveryNPrompts,
				Value: "-5",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: createTestTranscript(t, "test-session", 5),
					HookEventName:  UserPromptSubmit,
				},
				Prompt: "Test prompt",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "every_n_prompts - zero value",
			condition: Condition{
				Type:  ConditionEveryNPrompts,
				Value: "0",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: createTestTranscript(t, "test-session", 5),
					HookEventName:  UserPromptSubmit,
				},
				Prompt: "Test prompt",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "every_n_prompts - nonexistent transcript file",
			condition: Condition{
				Type:  ConditionEveryNPrompts,
				Value: "5",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-session",
					TranscriptPath: "/nonexistent/transcript.jsonl",
					HookEventName:  UserPromptSubmit,
				},
				Prompt: "Test prompt",
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkUserPromptSubmitCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkUserPromptSubmitCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkUserPromptSubmitCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckGitTrackedFileOperation(t *testing.T) {
	// テスト用の一時的なGitリポジトリを作成
	tmpDir := t.TempDir()

	// Git リポジトリを初期化
	if err := runCommand("cd "+tmpDir+" && git init", false, nil); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Git設定（コミット用）
	if err := runCommand("cd "+tmpDir+" && git config user.email 'test@example.com' && git config user.name 'Test User'", false, nil); err != nil {
		t.Fatalf("Failed to configure git: %v", err)
	}

	// テスト用のファイルを作成してGitに追加
	testFile := filepath.Join(tmpDir, "tracked.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := runCommand("cd "+tmpDir+" && git add tracked.txt && git commit -m 'test'", false, nil); err != nil {
		t.Fatalf("Failed to add file to git: %v", err)
	}

	// Git管理外のファイルも作成
	untrackedFile := filepath.Join(tmpDir, "untracked.txt")
	if err := os.WriteFile(untrackedFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create untracked file: %v", err)
	}

	// 元のディレクトリを保存して、テスト後に戻す
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	tests := []struct {
		name      string
		condition Condition
		input     *PreToolUseInput
		want      bool
		wantErr   bool
	}{
		{
			name: "rm command with git tracked file",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "rm",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "rm tracked.txt",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "rm command with untracked file",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "rm",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "rm untracked.txt",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "mv command with git tracked file",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "mv",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "mv tracked.txt tracked_backup.txt",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "rm with multiple files including git tracked",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "rm|mv",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "rm -rf untracked.txt tracked.txt",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "rm with options and quoted file",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "rm",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: `rm -f "tracked.txt"`,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "mv with target directory option",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "mv",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "mv -t /tmp tracked.txt",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "ls command should not match",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "rm|mv",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "ls -la tracked.txt",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "rm with environment variable expansion",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "rm",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "rm $HOME/nonexistent.txt",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "empty command",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "rm",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "git rm should NOT be blocked",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "rm|mv",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "git rm tracked.txt",
				},
			},
			want:    false, // git rmはブロック対象ではない
			wantErr: false,
		},
		{
			name: "git mv should NOT be blocked",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "rm|mv",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "git mv tracked.txt renamed.txt",
				},
			},
			want:    false, // git mvはブロック対象ではない
			wantErr: false,
		},
		{
			name: "git rm with options should NOT be blocked",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "rm",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "git rm --cached tracked.txt",
				},
			},
			want:    false, // git rmはブロック対象ではない
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkPreToolUseCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkPreToolUseCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkPreToolUseCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckSessionEndCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		input     *SessionEndInput
		want      bool
		wantErr   bool
	}{
		{
			name: "file_exists match",
			condition: Condition{
				Type:  ConditionFileExists,
				Value: "go.mod",
			},
			input: &SessionEndInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: SessionEnd,
				},
				Reason: "clear",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "file_not_exists match",
			condition: Condition{
				Type:  ConditionFileNotExists,
				Value: "nonexistent.file",
			},
			input: &SessionEndInput{
				BaseInput: BaseInput{
					SessionID:     "test456",
					HookEventName: SessionEnd,
				},
				Reason: "logout",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "reason_is match",
			condition: Condition{
				Type:  ConditionReasonIs,
				Value: "clear",
			},
			input: &SessionEndInput{
				BaseInput: BaseInput{
					SessionID:     "test101",
					HookEventName: SessionEnd,
				},
				Reason: "clear",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "reason_is not match",
			condition: Condition{
				Type:  ConditionReasonIs,
				Value: "logout",
			},
			input: &SessionEndInput{
				BaseInput: BaseInput{
					SessionID:     "test102",
					HookEventName: SessionEnd,
				},
				Reason: "clear",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "reason_is match - prompt_input_exit",
			condition: Condition{
				Type:  ConditionReasonIs,
				Value: "prompt_input_exit",
			},
			input: &SessionEndInput{
				BaseInput: BaseInput{
					SessionID:     "test103",
					HookEventName: SessionEnd,
				},
				Reason: "prompt_input_exit",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "unsupported condition type",
			condition: Condition{
				Type:  ConditionFileExtension,
				Value: ".go",
			},
			input: &SessionEndInput{
				BaseInput: BaseInput{
					SessionID:     "test789",
					HookEventName: SessionEnd,
				},
				Reason: "other",
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkSessionEndCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkSessionEndCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkSessionEndCondition() = %v, want %v", got, tt.want)
			}
		})
	}
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

func TestValidateSessionStartOutput(t *testing.T) {
	tests := []struct {
		name      string
		jsonData  string
		wantError bool
		errorMsg  string
	}{
		{
			name: "Valid output with all fields",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "SessionStart",
					"additionalContext": "test context"
				},
				"systemMessage": "test message"
			}`,
			wantError: false,
		},
		{
			name: "Valid output with minimal fields",
			jsonData: `{
				"continue": false,
				"hookSpecificOutput": {
					"hookEventName": "SessionStart"
				}
			}`,
			wantError: false,
		},
		{
			name: "Missing hookSpecificOutput (required field)",
			jsonData: `{
				"continue": true
			}`,
			wantError: true,
			errorMsg:  "hookSpecificOutput",
		},
		{
			name: "Missing hookEventName (required field)",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"additionalContext": "test"
				}
			}`,
			wantError: true,
			errorMsg:  "hookEventName",
		},
		{
			name: "Invalid hookEventName value (not SessionStart)",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "PreToolUse"
				}
			}`,
			wantError: true,
			errorMsg:  "hookEventName",
		},
		{
			name: "Invalid continue type (string instead of boolean)",
			jsonData: `{
				"continue": "true",
				"hookSpecificOutput": {
					"hookEventName": "SessionStart"
				}
			}`,
			wantError: true,
			errorMsg:  "continue",
		},
		{
			name: "Invalid hookSpecificOutput type (array instead of object)",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": []
			}`,
			wantError: true,
			errorMsg:  "hookSpecificOutput",
		},
		{
			name: "Invalid additionalContext type (number instead of string)",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "SessionStart",
					"additionalContext": 123
				}
			}`,
			wantError: true,
			errorMsg:  "additionalContext",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSessionStartOutput([]byte(tt.jsonData))

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
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
			name:    "command with parse error and <( returns true",
			command: "echo '<unclosed",
			want:    false,
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
