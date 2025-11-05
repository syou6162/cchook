//go:build integration

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
)

func TestExecutePreToolUseAction_WithUseStdin(t *testing.T) {
	tests := []struct {
		name     string
		action   Action
		input    *PreToolUseInput
		rawJSON  interface{}
		validate func(t *testing.T, output []byte, err error)
	}{
		{
			name: "use_stdin=true passes rawJSON to command stdin",
			action: Action{
				Type:     "command",
				Command:  "cat",
				UseStdin: true,
			},
			input: &PreToolUseInput{
				BaseInput: BaseInput{
					SessionID:     "test-session",
					HookEventName: PreToolUse,
				},
				ToolName: "Write",
				ToolInput: ToolInput{
					FilePath: "test.go",
				},
			},
			rawJSON: map[string]interface{}{
				"session_id":      "test-session",
				"hook_event_name": "PreToolUse",
				"tool_name":       "Write",
				"tool_input": map[string]interface{}{
					"file_path": "test.go",
				},
			},
			validate: func(t *testing.T, output []byte, err error) {
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}
				// 出力がrawJSONのJSON形式と一致することを確認
				var gotJSON map[string]interface{}
				if err := json.Unmarshal(output, &gotJSON); err != nil {
					t.Fatalf("Failed to parse output as JSON: %v", err)
				}
				if gotJSON["tool_name"] != "Write" {
					t.Errorf("Expected tool_name 'Write', got %v", gotJSON["tool_name"])
				}
			},
		},
		{
			name: "use_stdin=false does not pass rawJSON to stdin",
			action: Action{
				Type:     "command",
				Command:  "echo 'no stdin'",
				UseStdin: false,
			},
			input: &PreToolUseInput{
				BaseInput: BaseInput{
					SessionID:     "test-session",
					HookEventName: PreToolUse,
				},
				ToolName: "Write",
				ToolInput: ToolInput{
					FilePath: "test.go",
				},
			},
			rawJSON: map[string]interface{}{
				"session_id":      "test-session",
				"hook_event_name": "PreToolUse",
			},
			validate: func(t *testing.T, output []byte, err error) {
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}
				// 出力に"no stdin"が含まれることを確認
				if !bytes.Contains(output, []byte("no stdin")) {
					t.Errorf("Expected output to contain 'no stdin', got %s", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 標準出力をキャプチャ
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			executor := NewActionExecutor(nil)
			_, err := executor.ExecutePreToolUseAction(tt.action, tt.input, tt.rawJSON)

			// 標準出力を復元
			_ = w.Close()
			os.Stdout = oldStdout

			// キャプチャした出力を読み取り
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)

			tt.validate(t, buf.Bytes(), err)
		})
	}
}

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
