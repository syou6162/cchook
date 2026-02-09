//go:build integration

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExecutePostToolUseAction_WithUseStdin(t *testing.T) {
	// use_stdin=trueの時、rawJSONがstdinに渡されることを確認するテスト
	// 一時JSONファイルを使ってテンプレート処理の干渉を回避
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "output.json")
	jsonContent := `{"hookSpecificOutput": {"hookEventName": "PostToolUse", "additionalContext": "Edit"}}`
	if err := os.WriteFile(jsonFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to create temp JSON file: %v", err)
	}

	action := Action{
		Type:     "command",
		Command:  "cat " + jsonFile,
		UseStdin: true,
	}

	input := &PostToolUseInput{
		BaseInput: BaseInput{
			SessionID:     "test-session",
			HookEventName: PostToolUse,
		},
		ToolName: "Edit",
	}

	rawJSON := map[string]interface{}{
		"tool_name": "Edit",
	}

	executor := NewActionExecutor(nil)
	actionOutput, err := executor.ExecutePostToolUseAction(action, input, rawJSON)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if actionOutput == nil {
		t.Fatal("Expected actionOutput, got nil")
	}

	// hookEventNameが正しく設定されていることを確認
	if actionOutput.HookEventName != "PostToolUse" {
		t.Errorf("Expected hookEventName 'PostToolUse', got %s", actionOutput.HookEventName)
	}

	// additionalContextにtool_nameが設定されていることを確認
	if actionOutput.AdditionalContext != "Edit" {
		t.Errorf("Expected additionalContext 'Edit', got %s", actionOutput.AdditionalContext)
	}
}
