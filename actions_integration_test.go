//go:build integration

package main

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
)

func TestExecutePostToolUseAction_WithUseStdin(t *testing.T) {
	action := Action{
		Type:     "command",
		Command:  "jq -r .tool_name",
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

	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	executor := NewActionExecutor(nil)
	err := executor.ExecutePostToolUseAction(action, input, rawJSON)

	// 標準出力を復元
	_ = w.Close()
	os.Stdout = oldStdout

	// キャプチャした出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		// jqがインストールされていない場合はスキップ
		if _, err := exec.LookPath("jq"); err != nil {
			t.Skip("jq not installed, skipping test")
		}
		t.Fatalf("Expected no error, got %v", err)
	}

	// 出力に"Edit"が含まれることを確認
	if !bytes.Contains([]byte(output), []byte("Edit")) {
		t.Errorf("Expected output to contain 'Edit', got %s", output)
	}
}
