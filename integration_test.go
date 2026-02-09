//go:build integration

package main

import (
	"encoding/json"
	"testing"
)

// TestPostToolUseIntegration tests PostToolUse hook with real YAML config
func TestPostToolUseIntegration(t *testing.T) {
	config, err := loadConfig("testdata/integration_test_config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	tests := []struct {
		name      string
		jsonInput string
		wantErr   bool
	}{
		{
			name: "Markdown file modification triggers notification",
			jsonInput: `{
				"session_id": "test-session",
				"hook_event_name": "PostToolUse",
				"tool_name": "Write",
				"tool_input": {
					"file_path": "README.md",
					"content": "# Test"
				},
				"tool_output": "File written"
			}`,
			wantErr: false,
		},
		{
			name: "Non-markdown file does not trigger notification",
			jsonInput: `{
				"session_id": "test-session",
				"hook_event_name": "PostToolUse",
				"tool_name": "Write",
				"tool_input": {
					"file_path": "test.go",
					"content": "package main"
				},
				"tool_output": "File written"
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBytes := []byte(tt.jsonInput)

			var rawJSON map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &rawJSON); err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			input, err := parsePostToolUseInput(json.RawMessage(jsonBytes))
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

		output, err := executePostToolUseHooksJSON(config, input, rawJSON)

		if output == nil {
			t.Fatal("Expected output, got nil")
		}

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

// TestNotificationIntegration tests Notification hook with real YAML config
func TestNotificationIntegration(t *testing.T) {
	config, err := loadConfig("testdata/integration_test_config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	jsonInput := `{
		"session_id": "test-session",
		"hook_event_name": "Notification",
		"title": "Test Notification",
		"body": "This is a test notification"
	}`

	var rawJSON map[string]interface{}
	jsonBytes := []byte(jsonInput)
	if err := json.Unmarshal(jsonBytes, &rawJSON); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	var input NotificationInput
	if err := json.Unmarshal(jsonBytes, &input); err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	err = executeNotificationHooks(config, &input, rawJSON)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

// TestComplexJSONTemplateProcessing tests jq template processing with complex JSON
func TestComplexJSONTemplateProcessing(t *testing.T) {
	tests := []struct {
		name     string
		template string
		jsonData map[string]interface{}
		want     string
	}{
		{
			name:     "Nested object access",
			template: "File: {.tool_input.file_path}",
			jsonData: map[string]interface{}{
				"tool_input": map[string]interface{}{
					"file_path": "/path/to/file.go",
				},
			},
			want: "File: /path/to/file.go",
		},
		{
			name:     "jq transformation",
			template: "Uppercase: {.tool_name | ascii_upcase}",
			jsonData: map[string]interface{}{
				"tool_name": "write",
			},
			want: "Uppercase: WRITE",
		},
		{
			name:     "Array access",
			template: "First item: {.items[0]}",
			jsonData: map[string]interface{}{
				"items": []interface{}{"first", "second", "third"},
			},
			want: "First item: first",
		},
		{
			name:     "Multiple field access",
			template: "{.tool_name}: {.tool_input.file_path}",
			jsonData: map[string]interface{}{
				"tool_name": "Write",
				"tool_input": map[string]interface{}{
					"file_path": "test.go",
				},
			},
			want: "Write: test.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unifiedTemplateReplace(tt.template, tt.jsonData)
			if got != tt.want {
				t.Errorf("unifiedTemplateReplace() = %q, want %q", got, tt.want)
			}
		})
	}
}
