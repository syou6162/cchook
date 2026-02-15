package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
			"permission_mode_is match - plan",
			Condition{Type: ConditionPermissionModeIs, Value: "plan"},
			&PreToolUseInput{BaseInput: BaseInput{PermissionMode: "plan"}},
			true,
			false,
		},
		{
			"permission_mode_is no match",
			Condition{Type: ConditionPermissionModeIs, Value: "plan"},
			&PreToolUseInput{BaseInput: BaseInput{PermissionMode: "default"}},
			false,
			false,
		},
		{
			"permission_mode_is match - default",
			Condition{Type: ConditionPermissionModeIs, Value: "default"},
			&PreToolUseInput{BaseInput: BaseInput{PermissionMode: "default"}},
			true,
			false,
		},
		{
			"permission_mode_is match - acceptEdits",
			Condition{Type: ConditionPermissionModeIs, Value: "acceptEdits"},
			&PreToolUseInput{BaseInput: BaseInput{PermissionMode: "acceptEdits"}},
			true,
			false,
		},
		{
			"permission_mode_is empty permission_mode",
			Condition{Type: ConditionPermissionModeIs, Value: "plan"},
			&PreToolUseInput{BaseInput: BaseInput{PermissionMode: ""}},
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
			"permission_mode_is match in PostToolUse",
			Condition{Type: ConditionPermissionModeIs, Value: "plan"},
			&PostToolUseInput{BaseInput: BaseInput{PermissionMode: "plan"}},
			true,
			false,
		},
		{
			"permission_mode_is no match in PostToolUse",
			Condition{Type: ConditionPermissionModeIs, Value: "plan"},
			&PostToolUseInput{BaseInput: BaseInput{PermissionMode: "default"}},
			false,
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
			name: "permission_mode_is match in UserPromptSubmit",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "plan",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-pm",
					HookEventName:  UserPromptSubmit,
					PermissionMode: "plan",
				},
				Prompt: "test prompt",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "permission_mode_is no match in UserPromptSubmit",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "plan",
			},
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "test-pm2",
					HookEventName:  UserPromptSubmit,
					PermissionMode: "default",
				},
				Prompt: "test prompt",
			},
			want:    false,
			wantErr: false,
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

	// Git設定(コミット用)
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
	defer func() { _ = os.Chdir(originalDir) }()
	_ = os.Chdir(tmpDir)

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
		{
			name: "process substitution in command returns ErrProcessSubstitutionDetected",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "diff",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "diff -u file1 <(head -48 file2)",
				},
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "compound command with && (no process substitution)",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "rm",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "rm tracked.txt && echo ok",
				},
			},
			want:    false, // shell.Fieldsがパースできないためfalse
			wantErr: false,
		},
		{
			name: "pipe with process substitution returns error",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "rm",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "cat <(cmd) | rm tracked.txt",
				},
			},
			want:    false,
			wantErr: true, // プロセス置換検出
		},
		{
			name: "sudo rm with tracked file",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "rm",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "sudo rm tracked.txt",
				},
			},
			want:    false,
			wantErr: false, // cmdNameがsudoのためrmとマッチしない
		},
		{
			name: "rm with -- and option-like filename",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "rm",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "rm -- -tracked.txt",
				},
			},
			want:    false,
			wantErr: false, // -tracked.txtは存在しないため
		},
		{
			name: "mv with multiple sources",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "mv",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "mv tracked.txt untracked.txt /tmp",
				},
			},
			want:    true,
			wantErr: false, // tracked.txtが含まれる
		},
		{
			name: "glob pattern with tracked file",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "rm",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: "rm *.txt",
				},
			},
			want:    false,
			wantErr: false, // globは展開されないため*.txtというファイルは存在しない
		},
		{
			name: "rm with space in filename (quoted)",
			condition: Condition{
				Type:  ConditionGitTrackedFileOperation,
				Value: "rm",
			},
			input: &PreToolUseInput{
				ToolInput: ToolInput{
					Command: `rm "file with spaces.txt"`,
				},
			},
			want:    false,
			wantErr: false, // ファイルが存在しないため
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkPreToolUseCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkPreToolUseCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Phase 2: プロセス置換エラーの種類を確認
			if err != nil && strings.Contains(tt.name, "process substitution") {
				if !errors.Is(err, ErrProcessSubstitutionDetected) {
					t.Errorf("checkPreToolUseCondition() error type = %T, want ErrProcessSubstitutionDetected", err)
				}
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
			name: "permission_mode_is match in SessionEnd",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "dontAsk",
			},
			input: &SessionEndInput{
				BaseInput: BaseInput{
					SessionID:      "test-pm",
					HookEventName:  SessionEnd,
					PermissionMode: "dontAsk",
				},
				Reason: "clear",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "permission_mode_is no match in SessionEnd",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "plan",
			},
			input: &SessionEndInput{
				BaseInput: BaseInput{
					SessionID:      "test-pm2",
					HookEventName:  SessionEnd,
					PermissionMode: "default",
				},
				Reason: "clear",
			},
			want:    false,
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

func TestCheckSessionStartCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		input     *SessionStartInput
		want      bool
		wantErr   bool
	}{
		{
			name: "permission_mode_is match - plan",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "plan",
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test123",
					HookEventName:  SessionStart,
					PermissionMode: "plan",
				},
				Source: "startup",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "permission_mode_is no match - default vs plan",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "plan",
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test456",
					HookEventName:  SessionStart,
					PermissionMode: "default",
				},
				Source: "startup",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "permission_mode_is match - default",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "default",
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test789",
					HookEventName:  SessionStart,
					PermissionMode: "default",
				},
				Source: "startup",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "permission_mode_is match - acceptEdits",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "acceptEdits",
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-ae",
					HookEventName:  SessionStart,
					PermissionMode: "acceptEdits",
				},
				Source: "startup",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "permission_mode_is empty permission_mode",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "plan",
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-empty",
					HookEventName:  SessionStart,
					PermissionMode: "",
				},
				Source: "startup",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "file_exists common condition still works",
			condition: Condition{
				Type:  ConditionFileExists,
				Value: "go.mod",
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:     "test-common",
					HookEventName: SessionStart,
				},
				Source: "startup",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "unsupported condition type for SessionStart",
			condition: Condition{
				Type:  ConditionFileExtension,
				Value: ".go",
			},
			input: &SessionStartInput{
				BaseInput: BaseInput{
					SessionID:     "test-unsupported",
					HookEventName: SessionStart,
				},
				Source: "startup",
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkSessionStartCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkSessionStartCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkSessionStartCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckPermissionRequestCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		input     *PermissionRequestInput
		want      bool
		wantErr   bool
	}{
		{
			name: "permission_mode_is match in PermissionRequest",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "plan",
			},
			input: &PermissionRequestInput{
				BaseInput: BaseInput{
					SessionID:      "test-pr",
					HookEventName:  PermissionRequest,
					PermissionMode: "plan",
				},
				ToolName: "Write",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "permission_mode_is no match in PermissionRequest",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "plan",
			},
			input: &PermissionRequestInput{
				BaseInput: BaseInput{
					SessionID:      "test-pr2",
					HookEventName:  PermissionRequest,
					PermissionMode: "bypassPermissions",
				},
				ToolName: "Bash",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "file_extension tool condition still works",
			condition: Condition{
				Type:  ConditionFileExtension,
				Value: ".go",
			},
			input: &PermissionRequestInput{
				BaseInput: BaseInput{
					SessionID:     "test-pr3",
					HookEventName: PermissionRequest,
				},
				ToolName:  "Write",
				ToolInput: ToolInput{FilePath: "main.go"},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "unknown condition type - error",
			condition: Condition{
				Type:  ConditionType{"invalid_type"},
				Value: "test",
			},
			input: &PermissionRequestInput{
				BaseInput: BaseInput{
					SessionID:     "test-pr4",
					HookEventName: PermissionRequest,
				},
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkPermissionRequestCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkPermissionRequestCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkPermissionRequestCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckPreCompactCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		input     *PreCompactInput
		want      bool
		wantErr   bool
	}{
		{
			name: "permission_mode_is match in PreCompact",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "default",
			},
			input: &PreCompactInput{
				BaseInput: BaseInput{
					SessionID:      "test-pc",
					HookEventName:  PreCompact,
					PermissionMode: "default",
				},
				Trigger: "auto",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "permission_mode_is no match in PreCompact",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "plan",
			},
			input: &PreCompactInput{
				BaseInput: BaseInput{
					SessionID:      "test-pc2",
					HookEventName:  PreCompact,
					PermissionMode: "default",
				},
				Trigger: "manual",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "file_exists common condition still works",
			condition: Condition{
				Type:  ConditionFileExists,
				Value: "go.mod",
			},
			input: &PreCompactInput{
				BaseInput: BaseInput{
					SessionID:     "test-pc3",
					HookEventName: PreCompact,
				},
				Trigger: "auto",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "unsupported condition type for PreCompact",
			condition: Condition{
				Type:  ConditionFileExtension,
				Value: ".go",
			},
			input: &PreCompactInput{
				BaseInput: BaseInput{
					SessionID:     "test-pc4",
					HookEventName: PreCompact,
				},
				Trigger: "auto",
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkPreCompactCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkPreCompactCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkPreCompactCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckNotificationCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		input     *NotificationInput
		want      bool
		wantErr   bool
	}{
		{
			name: "permission_mode_is match in Notification",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "default",
			},
			input: &NotificationInput{
				BaseInput: BaseInput{
					SessionID:      "test-notif",
					HookEventName:  Notification,
					PermissionMode: "default",
				},
				NotificationType: "idle_prompt",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "permission_mode_is no match in Notification",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "plan",
			},
			input: &NotificationInput{
				BaseInput: BaseInput{
					SessionID:      "test-notif2",
					HookEventName:  Notification,
					PermissionMode: "default",
				},
				NotificationType: "permission_prompt",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "file_exists common condition still works",
			condition: Condition{
				Type:  ConditionFileExists,
				Value: "go.mod",
			},
			input: &NotificationInput{
				BaseInput: BaseInput{
					SessionID:     "test-notif3",
					HookEventName: Notification,
				},
				NotificationType: "idle_prompt",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "unsupported condition type for Notification",
			condition: Condition{
				Type:  ConditionFileExtension,
				Value: ".go",
			},
			input: &NotificationInput{
				BaseInput: BaseInput{
					SessionID:     "test-notif4",
					HookEventName: Notification,
				},
				NotificationType: "idle_prompt",
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkNotificationCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkNotificationCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkNotificationCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckStopCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		input     *StopInput
		want      bool
		wantErr   bool
	}{
		{
			name: "permission_mode_is match in Stop",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "default",
			},
			input: &StopInput{
				BaseInput: BaseInput{
					SessionID:      "test-stop",
					HookEventName:  Stop,
					PermissionMode: "default",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "permission_mode_is no match in Stop",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "plan",
			},
			input: &StopInput{
				BaseInput: BaseInput{
					SessionID:      "test-stop2",
					HookEventName:  Stop,
					PermissionMode: "default",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "file_exists common condition still works",
			condition: Condition{
				Type:  ConditionFileExists,
				Value: "go.mod",
			},
			input: &StopInput{
				BaseInput: BaseInput{
					SessionID:     "test-stop3",
					HookEventName: Stop,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "unsupported condition type for Stop",
			condition: Condition{
				Type:  ConditionFileExtension,
				Value: ".go",
			},
			input: &StopInput{
				BaseInput: BaseInput{
					SessionID:     "test-stop4",
					HookEventName: Stop,
				},
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkStopCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkStopCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkStopCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckSubagentStopCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		input     *SubagentStopInput
		want      bool
		wantErr   bool
	}{
		{
			name: "permission_mode_is match in SubagentStop",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "default",
			},
			input: &SubagentStopInput{
				BaseInput: BaseInput{
					SessionID:      "test-sastop",
					HookEventName:  SubagentStop,
					PermissionMode: "default",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "permission_mode_is no match in SubagentStop",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "plan",
			},
			input: &SubagentStopInput{
				BaseInput: BaseInput{
					SessionID:      "test-sastop2",
					HookEventName:  SubagentStop,
					PermissionMode: "default",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "file_exists common condition still works",
			condition: Condition{
				Type:  ConditionFileExists,
				Value: "go.mod",
			},
			input: &SubagentStopInput{
				BaseInput: BaseInput{
					SessionID:     "test-sastop3",
					HookEventName: SubagentStop,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "unsupported condition type for SubagentStop",
			condition: Condition{
				Type:  ConditionFileExtension,
				Value: ".go",
			},
			input: &SubagentStopInput{
				BaseInput: BaseInput{
					SessionID:     "test-sastop4",
					HookEventName: SubagentStop,
				},
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkSubagentStopCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkSubagentStopCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkSubagentStopCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckSubagentStartCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition Condition
		input     *SubagentStartInput
		want      bool
		wantErr   bool
	}{
		{
			name: "permission_mode_is match in SubagentStart",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "default",
			},
			input: &SubagentStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-sastart",
					HookEventName:  SubagentStart,
					PermissionMode: "default",
				},
				AgentType: "Explore",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "permission_mode_is no match in SubagentStart",
			condition: Condition{
				Type:  ConditionPermissionModeIs,
				Value: "plan",
			},
			input: &SubagentStartInput{
				BaseInput: BaseInput{
					SessionID:      "test-sastart2",
					HookEventName:  SubagentStart,
					PermissionMode: "default",
				},
				AgentType: "Plan",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "file_exists common condition still works",
			condition: Condition{
				Type:  ConditionFileExists,
				Value: "go.mod",
			},
			input: &SubagentStartInput{
				BaseInput: BaseInput{
					SessionID:     "test-sastart3",
					HookEventName: SubagentStart,
				},
				AgentType: "Bash",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "unsupported condition type for SubagentStart",
			condition: Condition{
				Type:  ConditionFileExtension,
				Value: ".go",
			},
			input: &SubagentStartInput{
				BaseInput: BaseInput{
					SessionID:     "test-sastart4",
					HookEventName: SubagentStart,
				},
				AgentType: "Explore",
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkSubagentStartCondition(tt.condition, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkSubagentStartCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkSubagentStartCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}
