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
				Decision:      "allow",
				SystemMessage: "Test message",
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName:     "UserPromptSubmit",
					AdditionalContext: "Additional info",
				},
			},
			wantContains:   []string{"\"continue\":true", "\"decision\":\"allow\"", "\"systemMessage\":\"Test message\"", "\"hookEventName\":\"UserPromptSubmit\"", "\"additionalContext\":\"Additional info\""},
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
				Decision: "allow",
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName:     "UserPromptSubmit",
					AdditionalContext: "Context",
				},
			},
			wantContains: []string{"\"hookEventName\":\"UserPromptSubmit\""},
		},
		{
			name: "Empty additionalContext is omitted",
			output: UserPromptSubmitOutput{
				Continue: true,
				Decision: "allow",
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName:     "UserPromptSubmit",
					AdditionalContext: "",
				},
			},
			wantContains:   []string{"\"hookEventName\":\"UserPromptSubmit\""},
			wantNotContain: []string{"additionalContext"},
		},
		{
			name: "Decision field is required and not omitted even when empty",
			output: UserPromptSubmitOutput{
				Continue: true,
				Decision: "",
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName: "UserPromptSubmit",
				},
			},
			wantContains: []string{"\"decision\":\"\""},
		},
		{
			name: "Decision field accepts 'allow'",
			output: UserPromptSubmitOutput{
				Continue: true,
				Decision: "allow",
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName: "UserPromptSubmit",
				},
			},
			wantContains: []string{"\"decision\":\"allow\""},
		},
		{
			name: "Decision field accepts 'block'",
			output: UserPromptSubmitOutput{
				Continue: true,
				Decision: "block",
				HookSpecificOutput: &UserPromptSubmitHookSpecificOutput{
					HookEventName: "UserPromptSubmit",
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
