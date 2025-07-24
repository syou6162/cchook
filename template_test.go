package main

import (
	"reflect"
	"testing"
)

func TestSnakeCaseReplaceVariables(t *testing.T) {
	// WriteToolInput を使ったテストケース
	input := &PreToolUseInput{
		BaseInput: BaseInput{
			SessionID:      "session123",
			TranscriptPath: "/tmp/transcript.jsonl",
			Cwd:            "/home/user/project",
			HookEventName:  PreToolUse,
		},
		ToolName: "Write",
		ToolInput: ToolInput{
			FilePath: "main.go",
			Content:  "package main",
		},
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			"基本的なsnake_caseアクセス",
			"Processing {tool_input.file_path}",
			"Processing main.go",
		},
		{
			"ベースフィールドへのアクセス",
			"Session: {session_id}",
			"Session: session123",
		},
		{
			"複数の変数",
			"Tool {tool_name} processing {tool_input.file_path} in {session_id}",
			"Tool Write processing main.go in session123",
		},
		{
			"存在しない変数",
			"Unknown: {unknown_field}",
			"Unknown: {unknown_field}",
		},
		{
			"ネストしたアクセス",
			"Content: {tool_input.content}",
			"Content: package main",
		},
		{
			"cwdフィールド",
			"Working dir: {cwd}",
			"Working dir: /home/user/project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snakeCaseReplaceVariables(tt.template, input)
			if got != tt.want {
				t.Errorf("snakeCaseReplaceVariables() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTemplateLookup(t *testing.T) {
	input := &PreToolUseInput{
		BaseInput: BaseInput{
			SessionID:      "test-session",
			TranscriptPath: "/tmp/test.jsonl",
			HookEventName:  PreToolUse,
		},
		ToolName: "Write",
		ToolInput: ToolInput{
			FilePath: "test.go",
			Content:  "test content",
		},
	}

	tests := []struct {
		name    string
		path    string
		want    interface{}
		wantErr bool
	}{
		{
			"session_id アクセス",
			"session_id",
			"test-session",
			false,
		},
		{
			"tool_name アクセス",
			"tool_name",
			"Write",
			false,
		},
		{
			"tool_input.file_path アクセス",
			"tool_input.file_path",
			"test.go",
			false,
		},
		{
			"tool_input.content アクセス",
			"tool_input.content",
			"test content",
			false,
		},
		{
			"存在しないパス",
			"non_existent_field",
			nil,
			true,
		},
		{
			"存在しないネストパス",
			"tool_input.non_existent",
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := templateLookup(input, tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("templateLookup() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("templateLookup() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("templateLookup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPostToolUseSnakeCaseReplaceVariables(t *testing.T) {
	input := &PostToolUseInput{
		BaseInput: BaseInput{
			SessionID:      "post-session",
			TranscriptPath: "/tmp/post.jsonl",
			HookEventName:  PostToolUse,
		},
		ToolName: "Write",
		ToolInput: ToolInput{
			FilePath: "output.go",
			Content:  "package output",
		},
		ToolResponse: ToolResponse{
			FilePath: "output.go",
			Success:  true,
		},
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			"PostToolUseでのsnake_caseアクセス",
			"Completed {tool_name} on {tool_input.file_path}",
			"Completed Write on output.go",
		},
		{
			"tool_responseアクセス", 
			"Success: {tool_response.success}",
			"Success: true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snakeCaseReplaceVariables(tt.template, input)
			if got != tt.want {
				t.Errorf("snakeCaseReplaceVariables() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildPathTable(t *testing.T) {
	// buildPathTable が正しくパスを生成するかテスト
	table := buildPathTable(reflect.TypeOf(PreToolUseInput{}), "")
	
	// デバッグ: 実際に生成されたパスを出力
	t.Logf("Generated paths:")
	for path := range table {
		t.Logf("  %s", path)
	}
	
	expectedPaths := []string{
		"session_id",
		"transcript_path", 
		"cwd",
		"hook_event_name",
		"tool_name",
		"tool_input.file_path",
		"tool_input.content",
	}

	for _, path := range expectedPaths {
		if _, exists := table[path]; !exists {
			t.Errorf("Expected path %q not found in path table", path)
		}
	}
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"SessionID", "session_id"},
		{"ToolName", "tool_name"},
		{"FilePath", "file_path"},
		{"HookEventName", "hook_event_name"},
		{"TranscriptPath", "transcript_path"},
		{"ABC", "abc"},
		{"SimpleWord", "simple_word"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := camelToSnake(tt.input)
			if got != tt.want {
				t.Errorf("camelToSnake(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}