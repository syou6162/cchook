package main

import (
	"encoding/json"
	"testing"
)

func TestJSONValidation(t *testing.T) {
	// PreToolUse用の有効なJSONデータ
	preToolUseJSON := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "allow",
			"permissionDecisionReason": "Testing",
		},
		"continue":       true,
		"suppressOutput": false,
	}

	// バリデーションテスト
	if err := validateJSONStructure(preToolUseJSON, PreToolUse); err != nil {
		t.Errorf("PreToolUse validation failed: %v", err)
	}

	// 構造化出力テスト
	output := createPreToolUseOutput(preToolUseJSON)
	if output.HookSpecificOutput == nil {
		t.Error("HookSpecificOutput should not be nil")
	}
	if output.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Error("HookEventName should be PreToolUse")
	}
	if output.HookSpecificOutput.PermissionDecision == nil || *output.HookSpecificOutput.PermissionDecision != "allow" {
		t.Error("PermissionDecision should be 'allow'")
	}

	// 無効なデータのテスト
	invalidJSON := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"permissionDecision": 123, // 無効な型
		},
	}

	if err := validateJSONStructure(invalidJSON, PreToolUse); err == nil {
		t.Error("Validation should have failed for invalid data")
	}
}

func TestPostToolUseValidation(t *testing.T) {
	// PostToolUse用の有効なJSONデータ
	postToolUseJSON := map[string]interface{}{
		"decision": "block",
		"reason":   "Testing PostToolUse",
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName": "PostToolUse",
		},
	}

	// バリデーションテスト
	if err := validateJSONStructure(postToolUseJSON, PostToolUse); err != nil {
		t.Errorf("PostToolUse validation failed: %v", err)
	}

	// 構造化出力テスト
	output := createPostToolUseOutput(postToolUseJSON)
	if output.Decision == nil || *output.Decision != "block" {
		t.Error("Decision should be 'block'")
	}
	if output.Reason == nil || *output.Reason != "Testing PostToolUse" {
		t.Error("Reason should be set correctly")
	}
}

func TestJSONProcessing(t *testing.T) {
	// JSON文字列の処理テスト
	jsonString := `{"continue":true,"suppressOutput":false}`

	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonString), &jsonData); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// バリデーション
	if err := validateJSONStructure(jsonData, Notification); err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	// 構造化出力
	output := createNotificationOutput(jsonData)
	if output.Continue == nil || *output.Continue != true {
		t.Error("Continue should be true")
	}
}
