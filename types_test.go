package main

import (
	"testing"
)

func TestHookEventType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		eventType HookEventType
		want     bool
	}{
		{"PreToolUse", PreToolUse, true},
		{"PostToolUse", PostToolUse, true},
		{"Notification", Notification, true},
		{"Stop", Stop, true},
		{"SubagentStop", SubagentStop, true},
		{"PreCompact", PreCompact, true},
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