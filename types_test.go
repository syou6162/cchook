package main

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestHookEventType_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		eventType HookEventType
		want      bool
	}{
		{"PreToolUse", PreToolUse, true},
		{"PostToolUse", PostToolUse, true},
		{"PermissionRequest", PermissionRequest, true},
		{"Notification", Notification, true},
		{"Stop", Stop, true},
		{"SubagentStop", SubagentStop, true},
		{"SubagentStart", SubagentStart, true},
		{"PreCompact", PreCompact, true},
		{"SessionStart", SessionStart, true},
		{"UserPromptSubmit", UserPromptSubmit, true},
		{"SessionEnd", SessionEnd, true},
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

func TestSubagentStartInput_GetToolName(t *testing.T) {
	input := &SubagentStartInput{}

	if got := input.GetToolName(); got != "" {
		t.Errorf("SubagentStartInput.GetToolName() = %v, want empty string", got)
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
		{
			name: "With agent_type and model",
			jsonInput: `{
				"session_id": "agent123",
				"transcript_path": "/tmp/transcript5.json",
				"hook_event_name": "SessionStart",
				"source": "startup",
				"agent_type": "Explore",
				"model": "claude-sonnet-4-5-20250929"
			}`,
			want: SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "agent123",
					TranscriptPath: "/tmp/transcript5.json",
					HookEventName:  SessionStart,
				},
				Source:    "startup",
				AgentType: "Explore",
				Model:     "claude-sonnet-4-5-20250929",
			},
		},
		{
			name: "agent_type omitted",
			jsonInput: `{
				"session_id": "agent456",
				"transcript_path": "/tmp/transcript6.json",
				"hook_event_name": "SessionStart",
				"source": "startup",
				"model": "claude-opus-4-6"
			}`,
			want: SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "agent456",
					TranscriptPath: "/tmp/transcript6.json",
					HookEventName:  SessionStart,
				},
				Source:    "startup",
				AgentType: "",
				Model:     "claude-opus-4-6",
			},
		},
		{
			name: "model omitted",
			jsonInput: `{
				"session_id": "agent789",
				"transcript_path": "/tmp/transcript7.json",
				"hook_event_name": "SessionStart",
				"source": "startup",
				"agent_type": "Plan"
			}`,
			want: SessionStartInput{
				BaseInput: BaseInput{
					SessionID:      "agent789",
					TranscriptPath: "/tmp/transcript7.json",
					HookEventName:  SessionStart,
				},
				Source:    "startup",
				AgentType: "Plan",
				Model:     "",
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

func TestNotificationInputParsing(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		want      NotificationInput
	}{
		{
			name: "With title and notification_type",
			jsonInput: `{
				"session_id": "test-123",
				"transcript_path": "/tmp/transcript.json",
				"hook_event_name": "Notification",
				"message": "Task completed",
				"title": "Success",
				"notification_type": "idle_prompt"
			}`,
			want: NotificationInput{
				BaseInput: BaseInput{
					SessionID:      "test-123",
					TranscriptPath: "/tmp/transcript.json",
					HookEventName:  Notification,
				},
				Message:          "Task completed",
				Title:            "Success",
				NotificationType: "idle_prompt",
			},
		},
		{
			name: "title omitted",
			jsonInput: `{
				"session_id": "test-456",
				"transcript_path": "/tmp/transcript2.json",
				"hook_event_name": "Notification",
				"message": "Permission required",
				"notification_type": "permission_prompt"
			}`,
			want: NotificationInput{
				BaseInput: BaseInput{
					SessionID:      "test-456",
					TranscriptPath: "/tmp/transcript2.json",
					HookEventName:  Notification,
				},
				Message:          "Permission required",
				Title:            "",
				NotificationType: "permission_prompt",
			},
		},
		{
			name: "notification_type omitted",
			jsonInput: `{
				"session_id": "test-789",
				"transcript_path": "/tmp/transcript3.json",
				"hook_event_name": "Notification",
				"message": "Generic notification",
				"title": "Info"
			}`,
			want: NotificationInput{
				BaseInput: BaseInput{
					SessionID:      "test-789",
					TranscriptPath: "/tmp/transcript3.json",
					HookEventName:  Notification,
				},
				Message:          "Generic notification",
				Title:            "Info",
				NotificationType: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input NotificationInput
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
			if input.Message != tt.want.Message {
				t.Errorf("Message: expected %s, got %s", tt.want.Message, input.Message)
			}
			if input.Title != tt.want.Title {
				t.Errorf("Title: expected %s, got %s", tt.want.Title, input.Title)
			}
			if input.NotificationType != tt.want.NotificationType {
				t.Errorf("NotificationType: expected %s, got %s", tt.want.NotificationType, input.NotificationType)
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

func TestSubagentStartParsing(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		want      SubagentStartInput
	}{
		{
			name: "Explore agent",
			jsonInput: `{
				"session_id": "abc123",
				"transcript_path": "/tmp/transcript.json",
				"hook_event_name": "SubagentStart",
				"agent_id": "agent-explore-001",
				"agent_type": "Explore"
			}`,
			want: SubagentStartInput{
				BaseInput: BaseInput{
					SessionID:      "abc123",
					TranscriptPath: "/tmp/transcript.json",
					HookEventName:  SubagentStart,
				},
				AgentID:   "agent-explore-001",
				AgentType: "Explore",
			},
		},
		{
			name: "Bash agent",
			jsonInput: `{
				"session_id": "def456",
				"transcript_path": "/tmp/transcript2.json",
				"hook_event_name": "SubagentStart",
				"agent_id": "agent-bash-002",
				"agent_type": "Bash"
			}`,
			want: SubagentStartInput{
				BaseInput: BaseInput{
					SessionID:      "def456",
					TranscriptPath: "/tmp/transcript2.json",
					HookEventName:  SubagentStart,
				},
				AgentID:   "agent-bash-002",
				AgentType: "Bash",
			},
		},
		{
			name: "Custom agent",
			jsonInput: `{
				"session_id": "ghi789",
				"transcript_path": "/tmp/transcript3.json",
				"hook_event_name": "SubagentStart",
				"agent_id": "agent-custom-003",
				"agent_type": "my-custom-agent"
			}`,
			want: SubagentStartInput{
				BaseInput: BaseInput{
					SessionID:      "ghi789",
					TranscriptPath: "/tmp/transcript3.json",
					HookEventName:  SubagentStart,
				},
				AgentID:   "agent-custom-003",
				AgentType: "my-custom-agent",
			},
		},
		{
			name: "With permission_mode",
			jsonInput: `{
				"session_id": "jkl012",
				"transcript_path": "/tmp/transcript4.json",
				"hook_event_name": "SubagentStart",
				"permission_mode": "plan",
				"agent_id": "agent-plan-004",
				"agent_type": "Plan"
			}`,
			want: SubagentStartInput{
				BaseInput: BaseInput{
					SessionID:      "jkl012",
					TranscriptPath: "/tmp/transcript4.json",
					HookEventName:  SubagentStart,
					PermissionMode: "plan",
				},
				AgentID:   "agent-plan-004",
				AgentType: "Plan",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input SubagentStartInput
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
			if input.AgentID != tt.want.AgentID {
				t.Errorf("AgentID: expected %s, got %s", tt.want.AgentID, input.AgentID)
			}
			if input.AgentType != tt.want.AgentType {
				t.Errorf("AgentType: expected %s, got %s", tt.want.AgentType, input.AgentType)
			}
		})
	}
}

func TestAction_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name         string
		yamlContent  string
		wantUseStdin bool
		wantType     string
		wantCommand  string
	}{
		{
			name: "use_stdin: true",
			yamlContent: `
type: command
command: "python process.py"
use_stdin: true
`,
			wantUseStdin: true,
			wantType:     "command",
			wantCommand:  "python process.py",
		},
		{
			name: "use_stdin: false",
			yamlContent: `
type: command
command: "echo 'test'"
use_stdin: false
`,
			wantUseStdin: false,
			wantType:     "command",
			wantCommand:  "echo 'test'",
		},
		{
			name: "use_stdin omitted (default false)",
			yamlContent: `
type: command
command: "ls -la"
`,
			wantUseStdin: false,
			wantType:     "command",
			wantCommand:  "ls -la",
		},
		{
			name: "use_stdin: true with type: output",
			yamlContent: `
type: output
message: "Warning message"
use_stdin: true
`,
			wantUseStdin: true,
			wantType:     "output",
			wantCommand:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var action Action
			err := yaml.Unmarshal([]byte(tt.yamlContent), &action)
			if err != nil {
				t.Fatalf("Failed to unmarshal YAML: %v", err)
			}

			if action.UseStdin != tt.wantUseStdin {
				t.Errorf("UseStdin: expected %v, got %v", tt.wantUseStdin, action.UseStdin)
			}
			if action.Type != tt.wantType {
				t.Errorf("Type: expected %s, got %s", tt.wantType, action.Type)
			}
			if action.Command != tt.wantCommand {
				t.Errorf("Command: expected %s, got %s", tt.wantCommand, action.Command)
			}
		})
	}
}

func TestAction_AdditionalContext_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name                  string
		yamlContent           string
		wantAdditionalContext *string
	}{
		{
			name: "additional_context set",
			yamlContent: `
type: output
message: "Test message"
additional_context: "Production environment"
`,
			wantAdditionalContext: stringPtr("Production environment"),
		},
		{
			name: "additional_context omitted",
			yamlContent: `
type: output
message: "Test message"
`,
			wantAdditionalContext: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var action Action
			err := yaml.Unmarshal([]byte(tt.yamlContent), &action)
			if err != nil {
				t.Fatalf("Failed to unmarshal YAML: %v", err)
			}

			if tt.wantAdditionalContext == nil {
				if action.AdditionalContext != nil {
					t.Errorf("AdditionalContext: expected nil, got %v", *action.AdditionalContext)
				}
			} else {
				if action.AdditionalContext == nil {
					t.Errorf("AdditionalContext: expected %q, got nil", *tt.wantAdditionalContext)
				} else if *action.AdditionalContext != *tt.wantAdditionalContext {
					t.Errorf("AdditionalContext: expected %q, got %q", *tt.wantAdditionalContext, *action.AdditionalContext)
				}
			}
		})
	}
}

func TestPreToolUseAction_UnmarshalYAML_WithAction(t *testing.T) {
	tests := []struct {
		name         string
		yamlContent  string
		wantUseStdin bool
		wantType     string
		wantCommand  string
	}{
		{
			name: "PreToolUseAction with use_stdin: true",
			yamlContent: `
type: command
command: "python validate.py"
use_stdin: true
`,
			wantUseStdin: true,
			wantType:     "command",
			wantCommand:  "python validate.py",
		},
		{
			name: "PreToolUseAction with use_stdin omitted",
			yamlContent: `
type: output
message: "Test message"
`,
			wantUseStdin: false,
			wantType:     "output",
			wantCommand:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var action Action
			err := yaml.Unmarshal([]byte(tt.yamlContent), &action)
			if err != nil {
				t.Fatalf("Failed to unmarshal YAML: %v", err)
			}

			if action.UseStdin != tt.wantUseStdin {
				t.Errorf("UseStdin: expected %v, got %v", tt.wantUseStdin, action.UseStdin)
			}
			if action.Type != tt.wantType {
				t.Errorf("Type: expected %s, got %s", tt.wantType, action.Type)
			}
			if action.Command != tt.wantCommand {
				t.Errorf("Command: expected %s, got %s", tt.wantCommand, action.Command)
			}
		})
	}
}

func TestSessionStartOutput_JSONSerialization(t *testing.T) {
	tests := []struct {
		name           string
		output         SessionStartOutput
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "Full output with all Phase 1 used fields",
			output: SessionStartOutput{
				Continue:      true,
				SystemMessage: "Test message",
				HookSpecificOutput: &SessionStartHookSpecificOutput{
					HookEventName:     "SessionStart",
					AdditionalContext: "Additional info",
				},
			},
			wantContains:   []string{"\"continue\":true", "\"systemMessage\":\"Test message\"", "\"hookEventName\":\"SessionStart\"", "\"additionalContext\":\"Additional info\""},
			wantNotContain: []string{"stopReason", "suppressOutput"},
		},
		{
			name: "Phase 1 unused fields (stopReason, suppressOutput) are omitted when zero",
			output: SessionStartOutput{
				Continue:       true,
				StopReason:     "",    // zero value
				SuppressOutput: false, // zero value
				HookSpecificOutput: &SessionStartHookSpecificOutput{
					HookEventName: "SessionStart",
				},
			},
			wantContains:   []string{"\"continue\":true", "\"hookEventName\":\"SessionStart\""},
			wantNotContain: []string{"stopReason", "suppressOutput"},
		},
		{
			name: "HookEventName is always SessionStart",
			output: SessionStartOutput{
				Continue: true,
				HookSpecificOutput: &SessionStartHookSpecificOutput{
					HookEventName:     "SessionStart",
					AdditionalContext: "Context",
				},
			},
			wantContains: []string{"\"hookEventName\":\"SessionStart\""},
		},
		{
			name: "Empty additionalContext is omitted",
			output: SessionStartOutput{
				Continue: true,
				HookSpecificOutput: &SessionStartHookSpecificOutput{
					HookEventName:     "SessionStart",
					AdditionalContext: "",
				},
			},
			wantContains:   []string{"\"hookEventName\":\"SessionStart\""},
			wantNotContain: []string{"additionalContext"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			jsonBytes, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}
			jsonStr := string(jsonBytes)

			// Check expected content
			for _, want := range tt.wantContains {
				if !stringContains(jsonStr, want) {
					t.Errorf("JSON does not contain expected string %q. JSON: %s", want, jsonStr)
				}
			}

			// Check unexpected content
			for _, notWant := range tt.wantNotContain {
				if stringContains(jsonStr, notWant) {
					t.Errorf("JSON contains unexpected string %q. JSON: %s", notWant, jsonStr)
				}
			}

			// Unmarshal (round-trip)
			var unmarshaled SessionStartOutput
			if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify round-trip preserves data
			if unmarshaled.Continue != tt.output.Continue {
				t.Errorf("Round-trip: Continue mismatch. Got %v, want %v", unmarshaled.Continue, tt.output.Continue)
			}
			if unmarshaled.SystemMessage != tt.output.SystemMessage {
				t.Errorf("Round-trip: SystemMessage mismatch. Got %q, want %q", unmarshaled.SystemMessage, tt.output.SystemMessage)
			}
			if tt.output.HookSpecificOutput != nil {
				if unmarshaled.HookSpecificOutput == nil {
					t.Errorf("Round-trip: HookSpecificOutput is nil")
				} else {
					if unmarshaled.HookSpecificOutput.HookEventName != tt.output.HookSpecificOutput.HookEventName {
						t.Errorf("Round-trip: HookEventName mismatch. Got %q, want %q",
							unmarshaled.HookSpecificOutput.HookEventName, tt.output.HookSpecificOutput.HookEventName)
					}
					if unmarshaled.HookSpecificOutput.AdditionalContext != tt.output.HookSpecificOutput.AdditionalContext {
						t.Errorf("Round-trip: AdditionalContext mismatch. Got %q, want %q",
							unmarshaled.HookSpecificOutput.AdditionalContext, tt.output.HookSpecificOutput.AdditionalContext)
					}
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestSessionStartOutputSchemaValidation(t *testing.T) {
	tests := []struct {
		name      string
		output    SessionStartOutput
		wantValid bool
		wantError string
	}{
		{
			name: "Valid full output with all fields",
			output: SessionStartOutput{
				Continue:       true,
				StopReason:     "test",
				SuppressOutput: false,
				SystemMessage:  "Test message",
				HookSpecificOutput: &SessionStartHookSpecificOutput{
					HookEventName:     "SessionStart",
					AdditionalContext: "Additional info",
				},
			},
			wantValid: true,
		},
		{
			name: "Valid minimal output with only hookEventName",
			output: SessionStartOutput{
				Continue: false,
				HookSpecificOutput: &SessionStartHookSpecificOutput{
					HookEventName: "SessionStart",
				},
			},
			wantValid: true,
		},
		{
			name: "Invalid: wrong hookEventName value",
			output: SessionStartOutput{
				Continue: true,
				HookSpecificOutput: &SessionStartHookSpecificOutput{
					HookEventName:     "WrongEvent",
					AdditionalContext: "Context",
				},
			},
			wantValid: false,
			wantError: "hookEventName",
		},
		{
			name: "Valid: Phase 1 unused fields omitted (omitempty)",
			output: SessionStartOutput{
				Continue: true,
				HookSpecificOutput: &SessionStartHookSpecificOutput{
					HookEventName: "SessionStart",
				},
			},
			wantValid: true,
		},
		{
			name: "Valid: continue field with different boolean values",
			output: SessionStartOutput{
				Continue: false,
				HookSpecificOutput: &SessionStartHookSpecificOutput{
					HookEventName: "SessionStart",
				},
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			jsonBytes, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Validate against schema
			err = validateSessionStartOutput(jsonBytes)

			if tt.wantValid {
				if err != nil {
					t.Errorf("Expected valid, but got error: %v\nJSON: %s", err, string(jsonBytes))
				}
			} else {
				if err == nil {
					t.Errorf("Expected invalid, but validation passed\nJSON: %s", string(jsonBytes))
				} else if tt.wantError != "" && !stringContains(err.Error(), tt.wantError) {
					t.Errorf("Error message should contain %q, but got: %v", tt.wantError, err)
				}
			}
		})
	}
}

func TestUserPromptSubmitOutput_JSONSerialization(t *testing.T) {
	tests := []struct {
		name           string
		output         UserPromptSubmitOutput
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "Full output with all Phase 2 used fields",
			output: UserPromptSubmitOutput{
				Continue:      true,
				Decision:      "block",
				SystemMessage: "Test message",
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName:     "UserPromptSubmit",
					AdditionalContext: "Additional info",
				},
			},
			wantContains:   []string{"\"continue\":true", "\"decision\":\"block\"", "\"systemMessage\":\"Test message\"", "\"hookEventName\":\"UserPromptSubmit\"", "\"additionalContext\":\"Additional info\""},
			wantNotContain: []string{"stopReason", "suppressOutput"},
		},
		{
			name: "Phase 2 unused fields (stopReason, suppressOutput) are omitted when zero",
			output: UserPromptSubmitOutput{
				Continue:       true,
				Decision:       "block",
				StopReason:     "",    // zero value
				SuppressOutput: false, // zero value
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName: "UserPromptSubmit",
				},
			},
			wantContains:   []string{"\"continue\":true", "\"decision\":\"block\"", "\"hookEventName\":\"UserPromptSubmit\""},
			wantNotContain: []string{"stopReason", "suppressOutput"},
		},
		{
			name: "HookEventName is always UserPromptSubmit",
			output: UserPromptSubmitOutput{
				Continue: true,
				Decision: "",
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName:     "UserPromptSubmit",
					AdditionalContext: "Context",
				},
			},
			wantContains: []string{"\"hookEventName\":\"UserPromptSubmit\""},
		},
		{
			name: "Empty additionalContext is included (required field)",
			output: UserPromptSubmitOutput{
				Continue: true,
				Decision: "",
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName:     "UserPromptSubmit",
					AdditionalContext: "",
				},
			},
			wantContains:   []string{"\"hookEventName\":\"UserPromptSubmit\"", "\"additionalContext\":\"\""},
			wantNotContain: []string{},
		},
		{
			name: "Decision field is omitted when empty (allows prompt)",
			output: UserPromptSubmitOutput{
				Continue: true,
				Decision: "",
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName: "UserPromptSubmit",
				},
			},
			wantNotContain: []string{"\"decision\""},
		},
		{
			name: "Decision field accepts 'block'",
			output: UserPromptSubmitOutput{
				Continue: true,
				Decision: "block",
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName:     "UserPromptSubmit",
					AdditionalContext: "",
				},
			},
			wantContains: []string{"\"decision\":\"block\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			jsonBytes, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}
			jsonStr := string(jsonBytes)

			// Check expected content
			for _, want := range tt.wantContains {
				if !stringContains(jsonStr, want) {
					t.Errorf("JSON does not contain expected string %q. JSON: %s", want, jsonStr)
				}
			}

			// Check unexpected content
			for _, notWant := range tt.wantNotContain {
				if stringContains(jsonStr, notWant) {
					t.Errorf("JSON contains unexpected string %q. JSON: %s", notWant, jsonStr)
				}
			}

			// Unmarshal (round-trip)
			var unmarshaled UserPromptSubmitOutput
			if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify round-trip preserves data
			if unmarshaled.Continue != tt.output.Continue {
				t.Errorf("Round-trip: Continue mismatch. Got %v, want %v", unmarshaled.Continue, tt.output.Continue)
			}
			if unmarshaled.Decision != tt.output.Decision {
				t.Errorf("Round-trip: Decision mismatch. Got %q, want %q", unmarshaled.Decision, tt.output.Decision)
			}
			if unmarshaled.SystemMessage != tt.output.SystemMessage {
				t.Errorf("Round-trip: SystemMessage mismatch. Got %q, want %q", unmarshaled.SystemMessage, tt.output.SystemMessage)
			}
			if tt.output.HookSpecificOutput != nil {
				if unmarshaled.HookSpecificOutput == nil {
					t.Errorf("Round-trip: HookSpecificOutput is nil")
				} else {
					if unmarshaled.HookSpecificOutput.HookEventName != tt.output.HookSpecificOutput.HookEventName {
						t.Errorf("Round-trip: HookEventName mismatch. Got %q, want %q",
							unmarshaled.HookSpecificOutput.HookEventName, tt.output.HookSpecificOutput.HookEventName)
					}
					if unmarshaled.HookSpecificOutput.AdditionalContext != tt.output.HookSpecificOutput.AdditionalContext {
						t.Errorf("Round-trip: AdditionalContext mismatch. Got %q, want %q",
							unmarshaled.HookSpecificOutput.AdditionalContext, tt.output.HookSpecificOutput.AdditionalContext)
					}
				}
			}
		})
	}
}

func TestUserPromptSubmitOutputSchemaValidation(t *testing.T) {
	tests := []struct {
		name      string
		output    UserPromptSubmitOutput
		wantValid bool
		wantError string
	}{
		{
			name: "Valid full output with all fields",
			output: UserPromptSubmitOutput{
				Continue:       true,
				Decision:       "",
				StopReason:     "test",
				SuppressOutput: false,
				SystemMessage:  "Test message",
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName:     "UserPromptSubmit",
					AdditionalContext: "Additional info",
				},
			},
			wantValid: true,
		},
		{
			name: "Valid minimal output with only required fields",
			output: UserPromptSubmitOutput{
				Continue: true,
				Decision: "block",
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName: "UserPromptSubmit",
				},
			},
			wantValid: true,
		},
		{
			name: "Valid: missing decision field (allows prompt)",
			output: UserPromptSubmitOutput{
				Continue: true,
				Decision: "", // empty decision allows prompt
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName: "UserPromptSubmit",
				},
			},
			wantValid: true,
		},
		{
			name: "Invalid: wrong hookEventName value",
			output: UserPromptSubmitOutput{
				Continue: true,
				Decision: "",
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName:     "WrongEvent",
					AdditionalContext: "Context",
				},
			},
			wantValid: false,
			wantError: "hookEventName",
		},
		{
			name: "Invalid: wrong decision value",
			output: UserPromptSubmitOutput{
				Continue: true,
				Decision: "invalid_decision",
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName: "UserPromptSubmit",
				},
			},
			wantValid: false,
			wantError: "decision",
		},
		{
			name: "Valid: decision 'block'",
			output: UserPromptSubmitOutput{
				Continue: false,
				Decision: "block",
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName: "UserPromptSubmit",
				},
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			jsonBytes, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Validate against schema
			err = validateUserPromptSubmitOutput(jsonBytes)

			if tt.wantValid {
				if err != nil {
					t.Errorf("Expected valid, but got error: %v\nJSON: %s", err, string(jsonBytes))
				}
			} else {
				if err == nil {
					t.Errorf("Expected invalid, but validation passed\nJSON: %s", string(jsonBytes))
				} else if tt.wantError != "" && !stringContains(err.Error(), tt.wantError) {
					t.Errorf("Error message should contain %q, but got: %v", tt.wantError, err)
				}
			}
		})
	}
}

// TestPreToolUseOutput_JSONSerialization tests JSON serialization/deserialization for PreToolUseOutput
func TestPreToolUseOutput_JSONSerialization(t *testing.T) {
	tests := []struct {
		name           string
		output         PreToolUseOutput
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "Full output with all Phase 3 used fields",
			output: PreToolUseOutput{
				Continue:      true,
				SystemMessage: "Test message",
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:            "PreToolUse",
					PermissionDecision:       "allow",
					PermissionDecisionReason: "Safe operation",
					UpdatedInput: map[string]any{
						"file_path": "modified.txt",
						"content":   "new content",
					},
				},
			},
			wantContains:   []string{"\"continue\":true", "\"systemMessage\":\"Test message\"", "\"hookEventName\":\"PreToolUse\"", "\"permissionDecision\":\"allow\"", "\"permissionDecisionReason\":\"Safe operation\"", "\"updatedInput\"", "\"file_path\":\"modified.txt\""},
			wantNotContain: []string{"stopReason", "suppressOutput"},
		},
		{
			name: "Phase 3 unused fields (stopReason, suppressOutput) are omitted when zero",
			output: PreToolUseOutput{
				Continue:       true,
				StopReason:     "",    // zero value
				SuppressOutput: false, // zero value
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:      "PreToolUse",
					PermissionDecision: "deny",
				},
			},
			wantContains:   []string{"\"continue\":true", "\"hookEventName\":\"PreToolUse\"", "\"permissionDecision\":\"deny\""},
			wantNotContain: []string{"stopReason", "suppressOutput"},
		},
		{
			name: "HookEventName is always PreToolUse",
			output: PreToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:      "PreToolUse",
					PermissionDecision: "ask",
				},
			},
			wantContains: []string{"\"hookEventName\":\"PreToolUse\"", "\"permissionDecision\":\"ask\""},
		},
		{
			name: "PermissionDecision allow",
			output: PreToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:      "PreToolUse",
					PermissionDecision: "allow",
				},
			},
			wantContains: []string{"\"permissionDecision\":\"allow\""},
		},
		{
			name: "PermissionDecision deny",
			output: PreToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:      "PreToolUse",
					PermissionDecision: "deny",
				},
			},
			wantContains: []string{"\"permissionDecision\":\"deny\""},
		},
		{
			name: "PermissionDecision ask",
			output: PreToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:      "PreToolUse",
					PermissionDecision: "ask",
				},
			},
			wantContains: []string{"\"permissionDecision\":\"ask\""},
		},
		{
			name: "UpdatedInput with complex types",
			output: PreToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:      "PreToolUse",
					PermissionDecision: "allow",
					UpdatedInput: map[string]any{
						"string_field": "value",
						"number_field": 42,
						"bool_field":   true,
						"array_field":  []any{"a", "b", "c"},
						"object_field": map[string]any{
							"nested": "value",
						},
					},
				},
			},
			wantContains: []string{"\"updatedInput\"", "\"string_field\":\"value\"", "\"number_field\":42", "\"bool_field\":true"},
		},
		{
			name: "Empty permissionDecisionReason and updatedInput are omitted",
			output: PreToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:            "PreToolUse",
					PermissionDecision:       "allow",
					PermissionDecisionReason: "",
					UpdatedInput:             nil,
				},
			},
			wantContains:   []string{"\"hookEventName\":\"PreToolUse\"", "\"permissionDecision\":\"allow\""},
			wantNotContain: []string{"permissionDecisionReason", "updatedInput"},
		},
		{
			name: "additionalContext is serialized",
			output: PreToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:      "PreToolUse",
					PermissionDecision: "allow",
					AdditionalContext:  "Current environment: production",
				},
			},
			wantContains: []string{"\"additionalContext\":\"Current environment: production\""},
		},
		{
			name: "Empty additionalContext is omitted",
			output: PreToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:      "PreToolUse",
					PermissionDecision: "allow",
					AdditionalContext:  "",
				},
			},
			wantContains:   []string{"\"hookEventName\":\"PreToolUse\""},
			wantNotContain: []string{"additionalContext"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			jsonBytes, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}
			jsonStr := string(jsonBytes)

			// Check expected content
			for _, want := range tt.wantContains {
				if !stringContains(jsonStr, want) {
					t.Errorf("JSON does not contain expected string %q. JSON: %s", want, jsonStr)
				}
			}

			// Check unexpected content
			for _, notWant := range tt.wantNotContain {
				if stringContains(jsonStr, notWant) {
					t.Errorf("JSON contains unexpected string %q. JSON: %s", notWant, jsonStr)
				}
			}

			// Unmarshal (round-trip)
			var unmarshaled PreToolUseOutput
			if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify round-trip preserves data
			if unmarshaled.Continue != tt.output.Continue {
				t.Errorf("Round-trip: Continue mismatch. Got %v, want %v", unmarshaled.Continue, tt.output.Continue)
			}
			if unmarshaled.SystemMessage != tt.output.SystemMessage {
				t.Errorf("Round-trip: SystemMessage mismatch. Got %q, want %q", unmarshaled.SystemMessage, tt.output.SystemMessage)
			}
			if tt.output.HookSpecificOutput != nil {
				if unmarshaled.HookSpecificOutput == nil {
					t.Errorf("Round-trip: HookSpecificOutput is nil")
				} else {
					if unmarshaled.HookSpecificOutput.HookEventName != tt.output.HookSpecificOutput.HookEventName {
						t.Errorf("Round-trip: HookEventName mismatch. Got %q, want %q",
							unmarshaled.HookSpecificOutput.HookEventName, tt.output.HookSpecificOutput.HookEventName)
					}
					if unmarshaled.HookSpecificOutput.PermissionDecision != tt.output.HookSpecificOutput.PermissionDecision {
						t.Errorf("Round-trip: PermissionDecision mismatch. Got %q, want %q",
							unmarshaled.HookSpecificOutput.PermissionDecision, tt.output.HookSpecificOutput.PermissionDecision)
					}
					if unmarshaled.HookSpecificOutput.PermissionDecisionReason != tt.output.HookSpecificOutput.PermissionDecisionReason {
						t.Errorf("Round-trip: PermissionDecisionReason mismatch. Got %q, want %q",
							unmarshaled.HookSpecificOutput.PermissionDecisionReason, tt.output.HookSpecificOutput.PermissionDecisionReason)
					}
					// UpdatedInput comparison requires deep comparison
					if tt.output.HookSpecificOutput.UpdatedInput != nil {
						if unmarshaled.HookSpecificOutput.UpdatedInput == nil {
							t.Errorf("Round-trip: UpdatedInput is nil")
						}
						// Basic check - detailed comparison would require reflect.DeepEqual
					}
				}
			}
		})
	}
}

// TestPreToolUseOutputSchemaValidation tests JSON schema validation for PreToolUseOutput
func TestPreToolUseOutputSchemaValidation(t *testing.T) {
	tests := []struct {
		name      string
		output    PreToolUseOutput
		wantValid bool
		wantError string
	}{
		{
			name: "Valid full output with all fields including updatedInput",
			output: PreToolUseOutput{
				Continue:      true,
				SystemMessage: "Test message",
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:            "PreToolUse",
					PermissionDecision:       "allow",
					PermissionDecisionReason: "Safe operation",
					UpdatedInput: map[string]any{
						"file_path": "modified.txt",
					},
				},
			},
			wantValid: true,
		},
		{
			name: "Valid minimal output with only required fields",
			output: PreToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:      "PreToolUse",
					PermissionDecision: "deny",
				},
			},
			wantValid: true,
		},
		{
			name: "Valid: hookSpecificOutput omitted (delegation)",
			output: PreToolUseOutput{
				Continue:           true,
				HookSpecificOutput: nil, // Omitted to delegate to Claude Code's permission system
			},
			wantValid: true,
		},
		{
			name: "Invalid: missing hookEventName",
			output: PreToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:      "", // empty hookEventName
					PermissionDecision: "allow",
				},
			},
			wantValid: false,
			wantError: "hookEventName",
		},
		{
			name: "Invalid: wrong hookEventName value",
			output: PreToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:      "WrongEvent",
					PermissionDecision: "allow",
				},
			},
			wantValid: false,
			wantError: "hookEventName",
		},
		{
			name: "Invalid: missing permissionDecision",
			output: PreToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:      "PreToolUse",
					PermissionDecision: "", // empty permissionDecision
				},
			},
			wantValid: false,
			wantError: "permissionDecision",
		},
		{
			name: "Invalid: wrong permissionDecision value",
			output: PreToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:      "PreToolUse",
					PermissionDecision: "invalid_value",
				},
			},
			wantValid: false,
			wantError: "permissionDecision",
		},
		{
			name: "Valid: permissionDecision allow",
			output: PreToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:      "PreToolUse",
					PermissionDecision: "allow",
				},
			},
			wantValid: true,
		},
		{
			name: "Valid: permissionDecision deny",
			output: PreToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:      "PreToolUse",
					PermissionDecision: "deny",
				},
			},
			wantValid: true,
		},
		{
			name: "Valid: permissionDecision ask",
			output: PreToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:      "PreToolUse",
					PermissionDecision: "ask",
				},
			},
			wantValid: true,
		},
		{
			name: "Valid: updatedInput with complex types",
			output: PreToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PreToolUseHookSpecificOutput{
					HookEventName:      "PreToolUse",
					PermissionDecision: "allow",
					UpdatedInput: map[string]any{
						"string_field": "value",
						"number_field": 42,
						"bool_field":   true,
						"array_field":  []any{"a", "b", "c"},
					},
				},
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			jsonBytes, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Validate
			err = validatePreToolUseOutput(jsonBytes)

			if tt.wantValid {
				if err != nil {
					t.Errorf("Expected valid, but got error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected validation error, but got none")
				} else if tt.wantError != "" && !stringContains(err.Error(), tt.wantError) {
					t.Errorf("Expected error to contain %q, but got: %v", tt.wantError, err)
				}
			}
		})
	}
}

// TestValidatePreToolUseOutput_ContinueFieldOmitted tests that continue field can be omitted
// per Claude Code specification (defaults to true)
func TestValidatePreToolUseOutput_ContinueFieldOmitted(t *testing.T) {
	// 外部コマンドが返すJSON（continueフィールドなし）
	// Claude Code公式仕様: continueはオプショナル（デフォルト: true）
	// https://code.claude.com/docs/en/hooks
	jsonData := []byte(`{
		"hookSpecificOutput": {
			"hookEventName": "PreToolUse",
			"permissionDecision": "allow"
		}
	}`)

	err := validatePreToolUseOutput(jsonData)
	if err != nil {
		t.Errorf("Expected valid JSON without 'continue' field (per Claude Code spec), but got error: %v", err)
	}
}

func TestPostToolUseOutput_JSONSerialization(t *testing.T) {
	tests := []struct {
		name           string
		output         PostToolUseOutput
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "Full output with all fields including hookSpecificOutput",
			output: PostToolUseOutput{
				Continue:       true,
				Decision:       "block",
				Reason:         "Tool output contains sensitive data",
				SystemMessage:  "Test message",
				StopReason:     "stopped",
				SuppressOutput: true,
				HookSpecificOutput: &PostToolUseHookSpecificOutput{
					HookEventName:     "PostToolUse",
					AdditionalContext: "Additional info for Claude",
				},
			},
			wantContains:   []string{"\"continue\":true", "\"decision\":\"block\"", "\"reason\":\"Tool output contains sensitive data\"", "\"systemMessage\":\"Test message\"", "\"stopReason\":\"stopped\"", "\"suppressOutput\":true", "\"hookEventName\":\"PostToolUse\"", "\"additionalContext\":\"Additional info for Claude\""},
			wantNotContain: []string{},
		},
		{
			name: "Decision and reason only (hookSpecificOutput omitted)",
			output: PostToolUseOutput{
				Continue: true,
				Decision: "block",
				Reason:   "Tool output validation failed",
			},
			wantContains:   []string{"\"continue\":true", "\"decision\":\"block\"", "\"reason\":\"Tool output validation failed\""},
			wantNotContain: []string{"hookSpecificOutput", "additionalContext", "systemMessage", "stopReason", "suppressOutput"},
		},
		{
			name: "Allow with hookSpecificOutput (decision omitted)",
			output: PostToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PostToolUseHookSpecificOutput{
					HookEventName:     "PostToolUse",
					AdditionalContext: "Tool executed successfully",
				},
			},
			wantContains:   []string{"\"continue\":true", "\"hookEventName\":\"PostToolUse\"", "\"additionalContext\":\"Tool executed successfully\""},
			wantNotContain: []string{"\"decision\"", "\"reason\"", "systemMessage", "stopReason", "suppressOutput"},
		},
		{
			name: "hookSpecificOutput with empty additionalContext (omitempty)",
			output: PostToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PostToolUseHookSpecificOutput{
					HookEventName:     "PostToolUse",
					AdditionalContext: "",
				},
			},
			wantContains:   []string{"\"continue\":true", "\"hookEventName\":\"PostToolUse\""},
			wantNotContain: []string{"\"additionalContext\"", "\"decision\"", "\"reason\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBytes, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("Failed to marshal JSON: %v", err)
			}
			jsonStr := string(jsonBytes)

			for _, want := range tt.wantContains {
				if !stringContains(jsonStr, want) {
					t.Errorf("JSON should contain %q, got: %s", want, jsonStr)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if stringContains(jsonStr, notWant) {
					t.Errorf("JSON should not contain %q, got: %s", notWant, jsonStr)
				}
			}
		})
	}
}

func TestNotificationOutput_JSONSerialization(t *testing.T) {
	tests := []struct {
		name           string
		output         NotificationOutput
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "Full output with all fields",
			output: NotificationOutput{
				Continue:       true,
				SystemMessage:  "Test message",
				StopReason:     "test reason",
				SuppressOutput: true,
				HookSpecificOutput: &NotificationHookSpecificOutput{
					HookEventName:     "Notification",
					AdditionalContext: "Additional info",
				},
			},
			wantContains: []string{
				"\"continue\":true",
				"\"systemMessage\":\"Test message\"",
				"\"stopReason\":\"test reason\"",
				"\"suppressOutput\":true",
				"\"hookEventName\":\"Notification\"",
				"\"additionalContext\":\"Additional info\"",
			},
		},
		{
			name: "Minimal output with only continue and hookSpecificOutput",
			output: NotificationOutput{
				Continue: true,
				HookSpecificOutput: &NotificationHookSpecificOutput{
					HookEventName: "Notification",
				},
			},
			wantContains:   []string{"\"continue\":true", "\"hookEventName\":\"Notification\""},
			wantNotContain: []string{"stopReason", "suppressOutput", "systemMessage", "additionalContext"},
		},
		{
			name: "Empty additionalContext is omitted",
			output: NotificationOutput{
				Continue: true,
				HookSpecificOutput: &NotificationHookSpecificOutput{
					HookEventName:     "Notification",
					AdditionalContext: "",
				},
			},
			wantContains:   []string{"\"hookEventName\":\"Notification\""},
			wantNotContain: []string{"additionalContext"},
		},
		{
			name: "Zero values are omitted (omitempty)",
			output: NotificationOutput{
				Continue:       true,
				StopReason:     "",
				SuppressOutput: false,
				SystemMessage:  "",
				HookSpecificOutput: &NotificationHookSpecificOutput{
					HookEventName: "Notification",
				},
			},
			wantContains:   []string{"\"continue\":true", "\"hookEventName\":\"Notification\""},
			wantNotContain: []string{"stopReason", "suppressOutput", "systemMessage"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			jsonBytes, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}
			jsonStr := string(jsonBytes)

			// Check expected content
			for _, want := range tt.wantContains {
				if !stringContains(jsonStr, want) {
					t.Errorf("JSON does not contain expected string %q. JSON: %s", want, jsonStr)
				}
			}

			// Check unexpected content
			for _, notWant := range tt.wantNotContain {
				if stringContains(jsonStr, notWant) {
					t.Errorf("JSON contains unexpected string %q. JSON: %s", notWant, jsonStr)
				}
			}

			// Unmarshal (round-trip)
			var unmarshaled NotificationOutput
			if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify round-trip preserves data
			if unmarshaled.Continue != tt.output.Continue {
				t.Errorf("Round-trip: Continue mismatch. Got %v, want %v", unmarshaled.Continue, tt.output.Continue)
			}
			if unmarshaled.SystemMessage != tt.output.SystemMessage {
				t.Errorf("Round-trip: SystemMessage mismatch. Got %q, want %q", unmarshaled.SystemMessage, tt.output.SystemMessage)
			}
			if unmarshaled.StopReason != tt.output.StopReason {
				t.Errorf("Round-trip: StopReason mismatch. Got %q, want %q", unmarshaled.StopReason, tt.output.StopReason)
			}
			if unmarshaled.SuppressOutput != tt.output.SuppressOutput {
				t.Errorf("Round-trip: SuppressOutput mismatch. Got %v, want %v", unmarshaled.SuppressOutput, tt.output.SuppressOutput)
			}
			if tt.output.HookSpecificOutput != nil {
				if unmarshaled.HookSpecificOutput == nil {
					t.Errorf("Round-trip: HookSpecificOutput is nil")
				} else {
					if unmarshaled.HookSpecificOutput.HookEventName != tt.output.HookSpecificOutput.HookEventName {
						t.Errorf("Round-trip: HookEventName mismatch. Got %q, want %q",
							unmarshaled.HookSpecificOutput.HookEventName, tt.output.HookSpecificOutput.HookEventName)
					}
					if unmarshaled.HookSpecificOutput.AdditionalContext != tt.output.HookSpecificOutput.AdditionalContext {
						t.Errorf("Round-trip: AdditionalContext mismatch. Got %q, want %q",
							unmarshaled.HookSpecificOutput.AdditionalContext, tt.output.HookSpecificOutput.AdditionalContext)
					}
				}
			}
		})
	}
}

func TestPreCompactOutput_JSONSerialization(t *testing.T) {
	tests := []struct {
		name           string
		output         PreCompactOutput
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "Full output with all fields",
			output: PreCompactOutput{
				Continue:       true,
				StopReason:     "test reason",
				SuppressOutput: true,
				SystemMessage:  "Test message",
			},
			wantContains: []string{
				"\"continue\":true",
				"\"stopReason\":\"test reason\"",
				"\"suppressOutput\":true",
				"\"systemMessage\":\"Test message\"",
			},
			wantNotContain: []string{},
		},
		{
			name: "Minimal output with only continue",
			output: PreCompactOutput{
				Continue: true,
			},
			wantContains:   []string{"\"continue\":true"},
			wantNotContain: []string{"stopReason", "suppressOutput", "systemMessage"},
		},
		{
			name: "Empty strings are omitted (omitempty)",
			output: PreCompactOutput{
				Continue:       true,
				StopReason:     "",
				SuppressOutput: false,
				SystemMessage:  "",
			},
			wantContains:   []string{"\"continue\":true"},
			wantNotContain: []string{"stopReason", "suppressOutput", "systemMessage"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			jsonBytes, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}
			jsonStr := string(jsonBytes)

			// Check expected content
			for _, want := range tt.wantContains {
				if !stringContains(jsonStr, want) {
					t.Errorf("JSON does not contain expected string %q. JSON: %s", want, jsonStr)
				}
			}

			// Check unexpected content
			for _, notWant := range tt.wantNotContain {
				if stringContains(jsonStr, notWant) {
					t.Errorf("JSON contains unexpected string %q. JSON: %s", notWant, jsonStr)
				}
			}

			// Unmarshal (round-trip)
			var unmarshaled PreCompactOutput
			if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify round-trip preserves data
			if unmarshaled.Continue != tt.output.Continue {
				t.Errorf("Round-trip: Continue mismatch. Got %v, want %v", unmarshaled.Continue, tt.output.Continue)
			}
			if unmarshaled.StopReason != tt.output.StopReason {
				t.Errorf("Round-trip: StopReason mismatch. Got %q, want %q", unmarshaled.StopReason, tt.output.StopReason)
			}
			if unmarshaled.SuppressOutput != tt.output.SuppressOutput {
				t.Errorf("Round-trip: SuppressOutput mismatch. Got %v, want %v", unmarshaled.SuppressOutput, tt.output.SuppressOutput)
			}
			if unmarshaled.SystemMessage != tt.output.SystemMessage {
				t.Errorf("Round-trip: SystemMessage mismatch. Got %q, want %q", unmarshaled.SystemMessage, tt.output.SystemMessage)
			}
		})
	}
}

func TestPostToolUseOutput_JSONSerialization_UpdatedMCPToolOutput(t *testing.T) {
	tests := []struct {
		name           string
		output         PostToolUseOutput
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "With updatedMCPToolOutput string",
			output: PostToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PostToolUseHookSpecificOutput{
					HookEventName:     "PostToolUse",
					AdditionalContext: "Context message",
				},
				UpdatedMCPToolOutput: "replaced output",
			},
			wantContains: []string{
				"\"continue\":true",
				"\"updatedMCPToolOutput\":\"replaced output\"",
				"\"hookSpecificOutput\"",
				"\"additionalContext\":\"Context message\"",
			},
			wantNotContain: []string{},
		},
		{
			name: "With updatedMCPToolOutput object",
			output: PostToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PostToolUseHookSpecificOutput{
					HookEventName: "PostToolUse",
				},
				UpdatedMCPToolOutput: map[string]any{
					"key": "value",
					"nested": map[string]any{
						"field": 123,
					},
				},
			},
			wantContains: []string{
				"\"continue\":true",
				"\"updatedMCPToolOutput\"",
				"\"key\":\"value\"",
				"\"nested\"",
			},
			wantNotContain: []string{},
		},
		{
			name: "Without updatedMCPToolOutput (omitempty)",
			output: PostToolUseOutput{
				Continue: true,
				HookSpecificOutput: &PostToolUseHookSpecificOutput{
					HookEventName:     "PostToolUse",
					AdditionalContext: "Context only",
				},
			},
			wantContains: []string{
				"\"continue\":true",
				"\"additionalContext\":\"Context only\"",
			},
			wantNotContain: []string{"updatedMCPToolOutput"},
		},
		{
			name: "With nil updatedMCPToolOutput (omitempty)",
			output: PostToolUseOutput{
				Continue:             true,
				UpdatedMCPToolOutput: nil,
			},
			wantContains:   []string{"\"continue\":true"},
			wantNotContain: []string{"updatedMCPToolOutput", "hookSpecificOutput"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			jsonBytes, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}
			jsonStr := string(jsonBytes)

			// Check expected content
			for _, want := range tt.wantContains {
				if !stringContains(jsonStr, want) {
					t.Errorf("JSON does not contain expected string %q. JSON: %s", want, jsonStr)
				}
			}

			// Check unexpected content
			for _, notWant := range tt.wantNotContain {
				if stringContains(jsonStr, notWant) {
					t.Errorf("JSON contains unexpected string %q. JSON: %s", notWant, jsonStr)
				}
			}

			// Unmarshal (round-trip)
			var unmarshaled PostToolUseOutput
			if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify round-trip preserves data
			if unmarshaled.Continue != tt.output.Continue {
				t.Errorf("Round-trip: Continue mismatch. Got %v, want %v", unmarshaled.Continue, tt.output.Continue)
			}
		})
	}
}
