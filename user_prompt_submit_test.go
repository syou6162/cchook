package main

import (
	"encoding/json"
	"testing"
)

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

func TestUserPromptSubmitParsing(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		want      UserPromptSubmitInput
	}{
		{
			name: "Basic prompt",
			jsonInput: `{
				"session_id": "abc123",
				"transcript_path": "/tmp/transcript.json",
				"cwd": "/home/user",
				"hook_event_name": "UserPromptSubmit",
				"prompt": "Write a hello world program"
			}`,
			want: UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "abc123",
					TranscriptPath: "/tmp/transcript.json",
					Cwd:            "/home/user",
					HookEventName:  UserPromptSubmit,
				},
				Prompt: "Write a hello world program",
			},
		},
		{
			name: "Empty prompt",
			jsonInput: `{
				"session_id": "def456",
				"transcript_path": "/tmp/transcript2.json",
				"hook_event_name": "UserPromptSubmit",
				"prompt": ""
			}`,
			want: UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:      "def456",
					TranscriptPath: "/tmp/transcript2.json",
					HookEventName:  UserPromptSubmit,
				},
				Prompt: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input UserPromptSubmitInput
			err := json.Unmarshal([]byte(tt.jsonInput), &input)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			if input.SessionID != tt.want.SessionID {
				t.Errorf("SessionID: expected %s, got %s", tt.want.SessionID, input.SessionID)
			}
			if input.TranscriptPath != tt.want.TranscriptPath {
				t.Errorf("TranscriptPath: expected %s, got %s", tt.want.TranscriptPath, input.TranscriptPath)
			}
			if input.Cwd != tt.want.Cwd {
				t.Errorf("Cwd: expected %s, got %s", tt.want.Cwd, input.Cwd)
			}
			if input.HookEventName != tt.want.HookEventName {
				t.Errorf("HookEventName: expected %s, got %s", tt.want.HookEventName, input.HookEventName)
			}
			if input.Prompt != tt.want.Prompt {
				t.Errorf("Prompt: expected %s, got %s", tt.want.Prompt, input.Prompt)
			}
		})
	}
}

func TestExecuteUserPromptSubmitHooks(t *testing.T) {
	config := &Config{
		UserPromptSubmit: []UserPromptSubmitHook{
			{
				Conditions: []UserPromptSubmitCondition{
					{Type: "prompt_contains", Value: "block"},
				},
				Actions: []UserPromptSubmitAction{
					{
						Type:       "output",
						Message:    "Blocked prompt",
						ExitStatus: intPtr(2),
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		input       *UserPromptSubmitInput
		shouldError bool
	}{
		{
			name: "Blocked prompt",
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test123",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "This contains block keyword",
			},
			shouldError: true,
		},
		{
			name: "Allowed prompt",
			input: &UserPromptSubmitInput{
				BaseInput: BaseInput{
					SessionID:     "test456",
					HookEventName: UserPromptSubmit,
				},
				Prompt: "This is allowed",
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawJSON := map[string]interface{}{
				"session_id":      tt.input.SessionID,
				"hook_event_name": string(tt.input.HookEventName),
				"prompt":          tt.input.Prompt,
			}

			err := executeUserPromptSubmitHooks(config, tt.input, rawJSON)

			if tt.shouldError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if tt.shouldError && err != nil {
				exitErr, ok := err.(*ExitError)
				if !ok {
					t.Errorf("Expected ExitError, got %T", err)
				} else if exitErr.Code != 2 {
					t.Errorf("Expected exit code 2, got %d", exitErr.Code)
				}
			}
		})
	}
}
