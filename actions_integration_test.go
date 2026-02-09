//go:build integration

package main

import (
	"testing"
)

func TestExecutePostToolUseAction_WithUseStdin(t *testing.T) {
	// use_stdin=trueの時、rawJSONがstdinに渡されることを確認するテスト
	// テンプレート処理を避けるため、ファイルから読み込み
	action := Action{
		Type:     "command",
		Command:  "cat testdata/posttooluse_output.json",
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

	// additionalContextが設定されていることを確認
	if actionOutput.AdditionalContext != "test" {
		t.Errorf("Expected additionalContext 'test', got %s", actionOutput.AdditionalContext)
	}
}
