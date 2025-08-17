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
	err := runCommand("echo test")
	if err != nil {
		t.Errorf("runCommand() error = %v, expected nil", err)
	}
}

func TestRunCommand_EmptyCommand(t *testing.T) {
	err := runCommand("")
	if err == nil {
		t.Error("Expected error for empty command, got nil")
	}

	if !strings.Contains(err.Error(), "empty command") {
		t.Errorf("Expected 'empty command' error, got: %v", err)
	}
}

func TestRunCommand_CommandNotFound(t *testing.T) {
	// 存在しないコマンドをテスト
	err := runCommand("nonexistent-command-12345")
	if err == nil {
		t.Error("Expected error for non-existent command, got nil")
	}
}

func TestRunCommand_CommandFails(t *testing.T) {
	// 失敗するコマンドをテスト（falseコマンドは常に終了コード1を返す）
	err := runCommand("false")
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
	if err := runCommand("cd " + tmpDir + " && git init"); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Git設定（コミット用）
	if err := runCommand("cd " + tmpDir + " && git config user.email 'test@example.com' && git config user.name 'Test User'"); err != nil {
		t.Fatalf("Failed to configure git: %v", err)
	}

	// テスト用のファイルを作成してGitに追加
	testFile := filepath.Join(tmpDir, "tracked.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := runCommand("cd " + tmpDir + " && git add tracked.txt && git commit -m 'test'"); err != nil {
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
