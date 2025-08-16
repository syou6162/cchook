package main

import (
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

func TestCheckPreToolUseCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition PreToolUseCondition
		input     *PreToolUseInput
		want      bool
	}{
		{
			"file_extension match",
			PreToolUseCondition{Type: "file_extension", Value: ".go"},
			&PreToolUseInput{ToolInput: ToolInput{FilePath: "main.go"}},
			true,
		},
		{
			"file_extension no match",
			PreToolUseCondition{Type: "file_extension", Value: ".py"},
			&PreToolUseInput{ToolInput: ToolInput{FilePath: "main.go"}},
			false,
		},
		{
			"file_extension no file_path",
			PreToolUseCondition{Type: "file_extension", Value: ".go"},
			&PreToolUseInput{ToolInput: ToolInput{}},
			false,
		},
		{
			"command_contains match",
			PreToolUseCondition{Type: "command_contains", Value: "git add"},
			&PreToolUseInput{ToolInput: ToolInput{Command: "git add file.txt"}},
			true,
		},
		{
			"command_contains no match",
			PreToolUseCondition{Type: "command_contains", Value: "git commit"},
			&PreToolUseInput{ToolInput: ToolInput{Command: "git add file.txt"}},
			false,
		},
		{
			"command_contains no command",
			PreToolUseCondition{Type: "command_contains", Value: "git add"},
			&PreToolUseInput{ToolInput: ToolInput{}},
			false,
		},
		{
			"command_starts_with match",
			PreToolUseCondition{Type: "command_starts_with", Value: "git"},
			&PreToolUseInput{ToolInput: ToolInput{Command: "git status"}},
			true,
		},
		{
			"command_starts_with no match",
			PreToolUseCondition{Type: "command_starts_with", Value: "docker"},
			&PreToolUseInput{ToolInput: ToolInput{Command: "git status"}},
			false,
		},
		{
			"command_starts_with no command",
			PreToolUseCondition{Type: "command_starts_with", Value: "git"},
			&PreToolUseInput{ToolInput: ToolInput{}},
			false,
		},
		{
			"file_exists match",
			PreToolUseCondition{Type: "file_exists", Value: "/tmp"},
			&PreToolUseInput{ToolInput: ToolInput{}},
			true,
		},
		{
			"file_exists no match",
			PreToolUseCondition{Type: "file_exists", Value: "/nonexistent/path"},
			&PreToolUseInput{ToolInput: ToolInput{}},
			false,
		},
		{
			"file_exists empty value",
			PreToolUseCondition{Type: "file_exists", Value: ""},
			&PreToolUseInput{ToolInput: ToolInput{}},
			false,
		},
		{
			"url_starts_with match",
			PreToolUseCondition{Type: "url_starts_with", Value: "https://example.com"},
			&PreToolUseInput{ToolInput: ToolInput{URL: "https://example.com/page"}},
			true,
		},
		{
			"url_starts_with no match",
			PreToolUseCondition{Type: "url_starts_with", Value: "https://other.com"},
			&PreToolUseInput{ToolInput: ToolInput{URL: "https://example.com/page"}},
			false,
		},
		{
			"url_starts_with no url",
			PreToolUseCondition{Type: "url_starts_with", Value: "https://example.com"},
			&PreToolUseInput{ToolInput: ToolInput{}},
			false,
		},
		{
			"unknown condition type",
			PreToolUseCondition{Type: "unknown", Value: "value"},
			&PreToolUseInput{ToolInput: ToolInput{FilePath: "main.go"}},
			false,
		},
		{
			"file_exists_recursive - file exists in current dir",
			PreToolUseCondition{Type: "file_exists_recursive", Value: "utils_test.go"},
			&PreToolUseInput{},
			true,
		},
		{
			"file_exists_recursive - file does not exist",
			PreToolUseCondition{Type: "file_exists_recursive", Value: "nonexistent.txt"},
			&PreToolUseInput{},
			false,
		},
		{
			"file_exists_recursive - go.mod exists",
			PreToolUseCondition{Type: "file_exists_recursive", Value: "go.mod"},
			&PreToolUseInput{},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkPreToolUseCondition(tt.condition, tt.input); got != tt.want {
				t.Errorf("checkPreToolUseCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckPostToolUseCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition PostToolUseCondition
		input     *PostToolUseInput
		want      bool
	}{
		{
			"file_extension match",
			PostToolUseCondition{Type: "file_extension", Value: ".go"},
			&PostToolUseInput{ToolInput: ToolInput{FilePath: "main.go"}},
			true,
		},
		{
			"command_contains match",
			PostToolUseCondition{Type: "command_contains", Value: "build"},
			&PostToolUseInput{ToolInput: ToolInput{Command: "go build main.go"}},
			true,
		},
		{
			"command_starts_with match",
			PostToolUseCondition{Type: "command_starts_with", Value: "npm"},
			&PostToolUseInput{ToolInput: ToolInput{Command: "npm install"}},
			true,
		},
		{
			"command_starts_with no match",
			PostToolUseCondition{Type: "command_starts_with", Value: "yarn"},
			&PostToolUseInput{ToolInput: ToolInput{Command: "npm install"}},
			false,
		},
		{
			"file_exists match",
			PostToolUseCondition{Type: "file_exists", Value: "/tmp"},
			&PostToolUseInput{ToolInput: ToolInput{}},
			true,
		},
		{
			"file_exists no match",
			PostToolUseCondition{Type: "file_exists", Value: "/nonexistent/path"},
			&PostToolUseInput{ToolInput: ToolInput{}},
			false,
		},
		{
			"url_starts_with match",
			PostToolUseCondition{Type: "url_starts_with", Value: "https://api.example.com"},
			&PostToolUseInput{ToolInput: ToolInput{URL: "https://api.example.com/v1/data"}},
			true,
		},
		{
			"url_starts_with no match",
			PostToolUseCondition{Type: "url_starts_with", Value: "https://api.other.com"},
			&PostToolUseInput{ToolInput: ToolInput{URL: "https://api.example.com/v1/data"}},
			false,
		},
		{
			"no match",
			PostToolUseCondition{Type: "file_extension", Value: ".py"},
			&PostToolUseInput{ToolInput: ToolInput{FilePath: "main.go"}},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkPostToolUseCondition(tt.condition, tt.input); got != tt.want {
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

func TestCheckUserPromptSubmitCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition UserPromptSubmitCondition
		input     *UserPromptSubmitInput
		want      bool
	}{
		{
			name: "prompt_contains matches",
			condition: UserPromptSubmitCondition{
				Type:  "prompt_contains",
				Value: "secret",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "This contains a secret keyword",
			},
			want: true,
		},
		{
			name: "prompt_contains doesn't match",
			condition: UserPromptSubmitCondition{
				Type:  "prompt_contains",
				Value: "password",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "This is a normal prompt",
			},
			want: false,
		},
		{
			name: "prompt_starts_with matches",
			condition: UserPromptSubmitCondition{
				Type:  "prompt_starts_with",
				Value: "DEBUG:",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "DEBUG: Show me the logs",
			},
			want: true,
		},
		{
			name: "prompt_starts_with doesn't match",
			condition: UserPromptSubmitCondition{
				Type:  "prompt_starts_with",
				Value: "ERROR:",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "Show me the error logs",
			},
			want: false,
		},
		{
			name: "prompt_ends_with matches",
			condition: UserPromptSubmitCondition{
				Type:  "prompt_ends_with",
				Value: "?",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "What is this?",
			},
			want: true,
		},
		{
			name: "prompt_ends_with doesn't match",
			condition: UserPromptSubmitCondition{
				Type:  "prompt_ends_with",
				Value: "!",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "This is a statement",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkUserPromptSubmitCondition(tt.condition, tt.input)
			if got != tt.want {
				t.Errorf("checkUserPromptSubmitCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}
