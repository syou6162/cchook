package main

import (
	"strings"
	"testing"
)

func TestValidateSessionStartOutput(t *testing.T) {
	tests := []struct {
		name      string
		jsonData  string
		wantError bool
		errorMsg  string
	}{
		{
			name: "Valid output with all fields",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "SessionStart",
					"additionalContext": "test context"
				},
				"systemMessage": "test message"
			}`,
			wantError: false,
		},
		{
			name: "Valid output with minimal fields",
			jsonData: `{
				"continue": false,
				"hookSpecificOutput": {
					"hookEventName": "SessionStart"
				}
			}`,
			wantError: false,
		},
		{
			name: "Missing hookSpecificOutput (required field)",
			jsonData: `{
				"continue": true
			}`,
			wantError: true,
			errorMsg:  "hookSpecificOutput",
		},
		{
			name: "Missing hookEventName (required field)",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"additionalContext": "test"
				}
			}`,
			wantError: true,
			errorMsg:  "hookEventName",
		},
		{
			name: "Invalid hookEventName value (not SessionStart)",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "PreToolUse"
				}
			}`,
			wantError: true,
			errorMsg:  "hookEventName",
		},
		{
			name: "Invalid continue type (string instead of boolean)",
			jsonData: `{
				"continue": "true",
				"hookSpecificOutput": {
					"hookEventName": "SessionStart"
				}
			}`,
			wantError: true,
			errorMsg:  "continue",
		},
		{
			name: "Invalid hookSpecificOutput type (array instead of object)",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": []
			}`,
			wantError: true,
			errorMsg:  "hookSpecificOutput",
		},
		{
			name: "Invalid additionalContext type (number instead of string)",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "SessionStart",
					"additionalContext": 123
				}
			}`,
			wantError: true,
			errorMsg:  "additionalContext",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSessionStartOutput([]byte(tt.jsonData))

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateStopAndSubagentStopOutput(t *testing.T) {
	tests := []struct {
		name      string
		eventType string // "Stop" or "SubagentStop"
		jsonData  string
		wantError bool
		errorMsg  string
	}{
		// Stop tests
		{
			name:      "Stop: Valid output with decision: block + reason",
			eventType: "Stop",
			jsonData: `{
				"continue": true,
				"decision": "block",
				"reason": "Tests are still running",
				"systemMessage": "Stop blocked"
			}`,
			wantError: false,
		},
		{
			name:      "Stop: Valid output with decision omitted (allow stop)",
			eventType: "Stop",
			jsonData: `{
				"continue": true,
				"systemMessage": "Stop allowed"
			}`,
			wantError: false,
		},
		{
			name:      "Stop: Invalid decision value",
			eventType: "Stop",
			jsonData: `{
				"continue": true,
				"decision": "invalid",
				"reason": "test"
			}`,
			wantError: true,
			errorMsg:  "decision",
		},
		{
			name:      "Stop: decision: block + reason missing -> semantic validation failure",
			eventType: "Stop",
			jsonData: `{
				"continue": true,
				"decision": "block"
			}`,
			wantError: true,
			errorMsg:  "reason",
		},
		{
			name:      "Stop: decision omitted + reason present -> valid (reason is optional)",
			eventType: "Stop",
			jsonData: `{
				"continue": true,
				"reason": "informational reason"
			}`,
			wantError: false,
		},
		// SubagentStop tests
		{
			name:      "SubagentStop: Valid output with decision: block + reason",
			eventType: "SubagentStop",
			jsonData: `{
				"continue": true,
				"decision": "block",
				"reason": "Tests are still running",
				"systemMessage": "SubagentStop blocked"
			}`,
			wantError: false,
		},
		{
			name:      "SubagentStop: Valid output with decision omitted (allow stop)",
			eventType: "SubagentStop",
			jsonData: `{
				"continue": true,
				"systemMessage": "SubagentStop allowed"
			}`,
			wantError: false,
		},
		{
			name:      "SubagentStop: Invalid decision value",
			eventType: "SubagentStop",
			jsonData: `{
				"continue": true,
				"decision": "invalid",
				"reason": "test"
			}`,
			wantError: true,
			errorMsg:  "decision",
		},
		{
			name:      "SubagentStop: decision: block + reason missing -> semantic validation failure",
			eventType: "SubagentStop",
			jsonData: `{
				"continue": true,
				"decision": "block"
			}`,
			wantError: true,
			errorMsg:  "reason",
		},
		{
			name:      "SubagentStop: decision omitted + reason present -> valid (reason is optional)",
			eventType: "SubagentStop",
			jsonData: `{
				"continue": true,
				"reason": "informational reason"
			}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.eventType == "Stop" {
				err = validateStopOutput([]byte(tt.jsonData))
			} else {
				err = validateSubagentStopOutput([]byte(tt.jsonData))
			}

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected validation error containing %q, got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %s", err.Error())
				}
			}
		})
	}
}

func TestValidatePostToolUseOutput(t *testing.T) {
	tests := []struct {
		name      string
		jsonData  string
		wantError bool
		errorMsg  string
	}{
		{
			name: "Valid output with decision: block + reason",
			jsonData: `{
				"continue": true,
				"decision": "block",
				"reason": "Tool output contains sensitive data",
				"systemMessage": "Tool result blocked"
			}`,
			wantError: false,
		},
		{
			name: "Valid output with decision omitted (allow tool result)",
			jsonData: `{
				"continue": true,
				"systemMessage": "Tool result allowed"
			}`,
			wantError: false,
		},
		{
			name: "Valid output with decision and reason only (hookSpecificOutput omitted)",
			jsonData: `{
				"continue": true,
				"decision": "block",
				"reason": "Tool output validation failed"
			}`,
			wantError: false,
		},
		{
			name: "Valid output with hookSpecificOutput",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "PostToolUse",
					"additionalContext": "Tool executed successfully"
				}
			}`,
			wantError: false,
		},
		{
			name: "Invalid decision value",
			jsonData: `{
				"continue": true,
				"decision": "invalid",
				"reason": "test"
			}`,
			wantError: true,
			errorMsg:  "decision",
		},
		{
			name: "decision: block + reason missing -> semantic validation failure",
			jsonData: `{
				"continue": true,
				"decision": "block"
			}`,
			wantError: true,
			errorMsg:  "reason",
		},
		{
			name: "hookEventName mismatch",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "PreToolUse",
					"additionalContext": "Wrong event name"
				}
			}`,
			wantError: true,
			errorMsg:  "hookEventName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePostToolUseOutput([]byte(tt.jsonData))

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected validation error containing %q, got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %s", err.Error())
				}
			}
		})
	}
}

func TestValidateNotificationOutput(t *testing.T) {
	tests := []struct {
		name      string
		jsonData  string
		wantError bool
		errorMsg  string
	}{
		{
			name: "Valid output with only continue",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "Notification"
				}
			}`,
			wantError: false,
		},
		{
			name: "Valid output with hookSpecificOutput and additionalContext",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "Notification",
					"additionalContext": "Test notification message"
				}
			}`,
			wantError: false,
		},
		{
			name: "Valid output with all fields",
			jsonData: `{
				"continue": true,
				"systemMessage": "System notification",
				"stopReason": "test reason",
				"suppressOutput": true,
				"hookSpecificOutput": {
					"hookEventName": "Notification",
					"additionalContext": "Additional context"
				}
			}`,
			wantError: false,
		},
		{
			name: "Invalid continue type (string instead of bool)",
			jsonData: `{
				"continue": "true",
				"hookSpecificOutput": {
					"hookEventName": "Notification"
				}
			}`,
			wantError: true,
			errorMsg:  "continue",
		},
		{
			name: "hookSpecificOutput missing",
			jsonData: `{
				"continue": true
			}`,
			wantError: true,
			errorMsg:  "hookSpecificOutput",
		},
		{
			name: "hookEventName missing in hookSpecificOutput",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"additionalContext": "Test message"
				}
			}`,
			wantError: true,
			errorMsg:  "hookEventName",
		},
		{
			name: "hookEventName mismatch",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "SessionStart",
					"additionalContext": "Wrong event name"
				}
			}`,
			wantError: true,
			errorMsg:  "hookEventName",
		},
		{
			name: "Unsupported field: decision",
			jsonData: `{
				"continue": true,
				"decision": "block",
				"hookSpecificOutput": {
					"hookEventName": "Notification"
				}
			}`,
			wantError: true,
			errorMsg:  "decision",
		},
		{
			name: "Unsupported field: reason",
			jsonData: `{
				"continue": true,
				"reason": "test reason",
				"hookSpecificOutput": {
					"hookEventName": "Notification"
				}
			}`,
			wantError: true,
			errorMsg:  "reason",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNotificationOutput([]byte(tt.jsonData))

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected validation error containing %q, got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %s", err.Error())
				}
			}
		})
	}
}

func TestValidateSubagentStartOutput(t *testing.T) {
	tests := []struct {
		name      string
		jsonData  string
		wantError bool
		errorMsg  string
	}{
		{
			name: "Valid output with only continue",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "SubagentStart"
				}
			}`,
			wantError: false,
		},
		{
			name: "Valid output with hookSpecificOutput and additionalContext",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "SubagentStart",
					"additionalContext": "Explore agent started"
				}
			}`,
			wantError: false,
		},
		{
			name: "Valid output with all fields",
			jsonData: `{
				"continue": true,
				"systemMessage": "System message",
				"stopReason": "test reason",
				"suppressOutput": true,
				"hookSpecificOutput": {
					"hookEventName": "SubagentStart",
					"additionalContext": "Additional context"
				}
			}`,
			wantError: false,
		},
		{
			name: "Invalid continue type (string instead of bool)",
			jsonData: `{
				"continue": "true",
				"hookSpecificOutput": {
					"hookEventName": "SubagentStart"
				}
			}`,
			wantError: true,
			errorMsg:  "continue",
		},
		{
			name: "hookSpecificOutput missing",
			jsonData: `{
				"continue": true
			}`,
			wantError: true,
			errorMsg:  "hookSpecificOutput",
		},
		{
			name: "hookEventName missing in hookSpecificOutput",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"additionalContext": "Test message"
				}
			}`,
			wantError: true,
			errorMsg:  "hookEventName",
		},
		{
			name: "hookEventName mismatch",
			jsonData: `{
				"continue": true,
				"hookSpecificOutput": {
					"hookEventName": "Notification",
					"additionalContext": "Wrong event name"
				}
			}`,
			wantError: true,
			errorMsg:  "hookEventName",
		},
		{
			name: "Unsupported field: decision",
			jsonData: `{
				"continue": true,
				"decision": "block",
				"hookSpecificOutput": {
					"hookEventName": "SubagentStart"
				}
			}`,
			wantError: true,
			errorMsg:  "decision",
		},
		{
			name: "Unsupported field: reason",
			jsonData: `{
				"continue": true,
				"reason": "test reason",
				"hookSpecificOutput": {
					"hookEventName": "SubagentStart"
				}
			}`,
			wantError: true,
			errorMsg:  "reason",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSubagentStartOutput([]byte(tt.jsonData))

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected validation error containing %q, got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %s", err.Error())
				}
			}
		})
	}
}

func TestValidatePreCompactOutput(t *testing.T) {
	tests := []struct {
		name      string
		jsonData  string
		wantError bool
		errorMsg  string
	}{
		{
			name: "Valid output with all fields",
			jsonData: `{
				"continue": true,
				"stopReason": "test reason",
				"suppressOutput": true,
				"systemMessage": "test message"
			}`,
			wantError: false,
		},
		{
			name: "Valid output with only continue",
			jsonData: `{
				"continue": true
			}`,
			wantError: false,
		},
		{
			name: "Valid output with empty optional fields",
			jsonData: `{
				"continue": false,
				"stopReason": "",
				"suppressOutput": false,
				"systemMessage": ""
			}`,
			wantError: false,
		},
		{
			name: "Invalid continue type (string instead of bool)",
			jsonData: `{
				"continue": "true"
			}`,
			wantError: true,
			errorMsg:  "continue",
		},
		{
			name: "Invalid stopReason type (number instead of string)",
			jsonData: `{
				"continue": true,
				"stopReason": 123
			}`,
			wantError: true,
			errorMsg:  "stopReason",
		},
		{
			name: "Invalid suppressOutput type (string instead of bool)",
			jsonData: `{
				"continue": true,
				"suppressOutput": "false"
			}`,
			wantError: true,
			errorMsg:  "suppressOutput",
		},
		{
			name: "Invalid systemMessage type (number instead of string)",
			jsonData: `{
				"continue": true,
				"systemMessage": 456
			}`,
			wantError: true,
			errorMsg:  "systemMessage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePreCompactOutput([]byte(tt.jsonData))

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected validation error containing %q, got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %s", err.Error())
				}
			}
		})
	}
}
