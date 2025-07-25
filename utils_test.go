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
			"unknown condition type",
			PreToolUseCondition{Type: "unknown", Value: "value"},
			&PreToolUseInput{ToolInput: ToolInput{FilePath: "main.go"}},
			false,
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
		defer w.Close()
		w.Write([]byte(jsonInput))
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
		defer w.Close()
		w.Write([]byte(invalidJSON))
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
