package main

import (
	"encoding/json"
	"testing"
)

func TestHookEventType_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		eventType HookEventType
		want      bool
	}{
		{"PreToolUse", PreToolUse, true},
		{"PostToolUse", PostToolUse, true},
		{"Notification", Notification, true},
		{"Stop", Stop, true},
		{"SubagentStop", SubagentStop, true},
		{"PreCompact", PreCompact, true},
		{"SessionStart", SessionStart, true},
		{"UserPromptSubmit", UserPromptSubmit, true},
		{"Invalid empty", HookEventType(""), false},
		{"Invalid unknown", HookEventType("Unknown"), false},
		{"Invalid case", HookEventType("pretooluse"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.eventType.IsValid(); got != tt.want {
				t.Errorf("HookEventType.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseInput_GetEventType(t *testing.T) {
	tests := []struct {
		name  string
		input BaseInput
		want  HookEventType
	}{
		{
			"PreToolUse",
			BaseInput{HookEventName: PreToolUse},
			PreToolUse,
		},
		{
			"PostToolUse",
			BaseInput{HookEventName: PostToolUse},
			PostToolUse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.input.GetEventType(); got != tt.want {
				t.Errorf("BaseInput.GetEventType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPreToolUseInput_GetToolName(t *testing.T) {
	input := &PreToolUseInput{
		ToolName: "Write",
	}

	if got := input.GetToolName(); got != "Write" {
		t.Errorf("PreToolUseInput.GetToolName() = %v, want %v", got, "Write")
	}
}

func TestPostToolUseInput_GetToolName(t *testing.T) {
	input := &PostToolUseInput{
		ToolName: "Edit",
	}

	if got := input.GetToolName(); got != "Edit" {
		t.Errorf("PostToolUseInput.GetToolName() = %v, want %v", got, "Edit")
	}
}

func TestNotificationInput_GetToolName(t *testing.T) {
	input := &NotificationInput{}

	if got := input.GetToolName(); got != "" {
		t.Errorf("NotificationInput.GetToolName() = %v, want empty string", got)
	}
}

func TestStopInput_GetToolName(t *testing.T) {
	input := &StopInput{}

	if got := input.GetToolName(); got != "" {
		t.Errorf("StopInput.GetToolName() = %v, want empty string", got)
	}
}

func TestSubagentStopInput_GetToolName(t *testing.T) {
	input := &SubagentStopInput{}

	if got := input.GetToolName(); got != "" {
		t.Errorf("SubagentStopInput.GetToolName() = %v, want empty string", got)
	}
}

func TestPreCompactInput_GetToolName(t *testing.T) {
	input := &PreCompactInput{}

	if got := input.GetToolName(); got != "" {
		t.Errorf("PreCompactInput.GetToolName() = %v, want empty string", got)
	}
}

func TestSessionStartInput_GetToolName(t *testing.T) {
	input := &SessionStartInput{}

	if got := input.GetToolName(); got != "" {
		t.Errorf("SessionStartInput.GetToolName() = %v, want empty string", got)
	}
}

func TestUserPromptSubmitInput_GetToolName(t *testing.T) {
	input := &UserPromptSubmitInput{}

	if got := input.GetToolName(); got != "" {
		t.Errorf("UserPromptSubmitInput.GetToolName() = %v, want empty string", got)
	}
}

func TestSessionStartParsing(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		want      SessionStartInput
	}{
		{
			name: "Startup event",
			jsonInput: `{
				"session_id": "abc123",
				"transcript_path": "/tmp/transcript.json",
				"hook_event_name": "SessionStart",
				"source": "startup"
			}`,
			want: SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "abc123",
					TranscriptPath: "/tmp/transcript.json",
					HookEventName:  SessionStart,
				},
				Source: "startup",
			},
		},
		{
			name: "Resume event",
			jsonInput: `{
				"session_id": "def456",
				"transcript_path": "/tmp/transcript2.json",
				"hook_event_name": "SessionStart",
				"source": "resume"
			}`,
			want: SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "def456",
					TranscriptPath: "/tmp/transcript2.json",
					HookEventName:  SessionStart,
				},
				Source: "resume",
			},
		},
		{
			name: "With permission_mode",
			jsonInput: `{
				"session_id": "xyz789",
				"transcript_path": "/tmp/transcript3.json",
				"hook_event_name": "SessionStart",
				"permission_mode": "plan",
				"source": "startup"
			}`,
			want: SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "xyz789",
					TranscriptPath: "/tmp/transcript3.json",
					HookEventName:  SessionStart,
					PermissionMode: "plan",
				},
				Source: "startup",
			},
		},
		{
			name: "Compact event",
			jsonInput: `{
				"session_id": "compact123",
				"transcript_path": "/tmp/transcript4.json",
				"hook_event_name": "SessionStart",
				"source": "compact"
			}`,
			want: SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "compact123",
					TranscriptPath: "/tmp/transcript4.json",
					HookEventName:  SessionStart,
				},
				Source: "compact",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input SessionStartInput
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
			if input.HookEventName != tt.want.HookEventName {
				t.Errorf("HookEventName: expected %s, got %s", tt.want.HookEventName, input.HookEventName)
			}
			if input.PermissionMode != tt.want.PermissionMode {
				t.Errorf("PermissionMode: expected %s, got %s", tt.want.PermissionMode, input.PermissionMode)
			}
			if input.Source != tt.want.Source {
				t.Errorf("Source: expected %s, got %s", tt.want.Source, input.Source)
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

func TestSessionEndParsing(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		want      SessionEndInput
	}{
		{
			name: "Clear event",
			jsonInput: `{
				"session_id": "abc123",
				"transcript_path": "/tmp/transcript.json",
				"hook_event_name": "SessionEnd",
				"reason": "clear"
			}`,
			want: SessionEndInput{
				BaseInput: BaseInput{
					SessionID:      "abc123",
					TranscriptPath: "/tmp/transcript.json",
					HookEventName:  SessionEnd,
				},
				Reason: "clear",
			},
		},
		{
			name: "Logout event",
			jsonInput: `{
				"session_id": "def456",
				"transcript_path": "/tmp/transcript2.json",
				"hook_event_name": "SessionEnd",
				"reason": "logout"
			}`,
			want: SessionEndInput{
				BaseInput: BaseInput{
					SessionID:      "def456",
					TranscriptPath: "/tmp/transcript2.json",
					HookEventName:  SessionEnd,
				},
				Reason: "logout",
			},
		},
		{
			name: "Prompt input exit event",
			jsonInput: `{
				"session_id": "ghi789",
				"transcript_path": "/tmp/transcript3.json",
				"hook_event_name": "SessionEnd",
				"reason": "prompt_input_exit"
			}`,
			want: SessionEndInput{
				BaseInput: BaseInput{
					SessionID:      "ghi789",
					TranscriptPath: "/tmp/transcript3.json",
					HookEventName:  SessionEnd,
				},
				Reason: "prompt_input_exit",
			},
		},
		{
			name: "Other event",
			jsonInput: `{
				"session_id": "jkl012",
				"transcript_path": "/tmp/transcript4.json",
				"hook_event_name": "SessionEnd",
				"reason": "other"
			}`,
			want: SessionEndInput{
				BaseInput: BaseInput{
					SessionID:      "jkl012",
					TranscriptPath: "/tmp/transcript4.json",
					HookEventName:  SessionEnd,
				},
				Reason: "other",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input SessionEndInput
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
			if input.HookEventName != tt.want.HookEventName {
				t.Errorf("HookEventName: expected %s, got %s", tt.want.HookEventName, input.HookEventName)
			}
			if input.Reason != tt.want.Reason {
				t.Errorf("Reason: expected %s, got %s", tt.want.Reason, input.Reason)
			}
		})
	}
}
