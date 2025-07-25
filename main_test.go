package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRunHooks_UnknownEventType(t *testing.T) {
	config := &Config{}

	err := runHooks(config, HookEventType("UnknownEvent"))
	if err == nil {
		t.Error("Expected error for unknown event type, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported event type") {
		t.Errorf("Expected 'unsupported event type' error, got: %v", err)
	}
}

func TestDryRunHooks_UnknownEventType(t *testing.T) {
	config := &Config{}

	err := dryRunHooks(config, HookEventType("UnknownEvent"))
	if err == nil {
		t.Error("Expected error for unknown event type, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported event type") {
		t.Errorf("Expected 'unsupported event type' error, got: %v", err)
	}
}

func TestRunHooks_InvalidJSON(t *testing.T) {
	// 標準入力をバックアップして復元
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// 不正なJSONを標準入力に設定
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(`{"invalid": json`)) // 不正なJSON
	}()

	config := &Config{}
	err := runHooks(config, PreToolUse)

	if err == nil {
		t.Error("Expected error for invalid JSON input, got nil")
	}

	if !strings.Contains(err.Error(), "failed to decode JSON input") {
		t.Errorf("Expected JSON decode error, got: %v", err)
	}
}

func TestRunHooks_PreToolUse_Success(t *testing.T) {
	// 標準入力をバックアップして復元
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// 正常なJSONを標準入力に設定
	r, w, _ := os.Pipe()
	os.Stdin = r

	jsonInput := `{
		"session_id": "test",
		"transcript_path": "/tmp/test",
		"hook_event_name": "PreToolUse",
		"tool_name": "Write",
		"tool_input": {"file_path": "test.go"}
	}`

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(jsonInput))
	}()

	config := &Config{
		PreToolUse: []PreToolUseHook{
			{
				Matcher: "Write",
				Actions: []PreToolUseAction{
					{BaseAction: BaseAction{Type: "structured_output"}},
				},
			},
		},
	}

	err := runHooks(config, PreToolUse)
	if err != nil {
		t.Errorf("runHooks() error = %v", err)
	}
}

func TestRunHooks_PostToolUse_Success(t *testing.T) {
	// 標準入力をバックアップして復元
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// 正常なJSONを標準入力に設定
	r, w, _ := os.Pipe()
	os.Stdin = r

	jsonInput := `{
		"session_id": "test",
		"transcript_path": "/tmp/test",
		"hook_event_name": "PostToolUse",
		"tool_name": "Edit",
		"tool_input": {"file_path": "test.go"},
		"tool_response": {}
	}`

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(jsonInput))
	}()

	config := &Config{
		PostToolUse: []PostToolUseHook{
			{
				Matcher: "Edit",
				Actions: []PostToolUseAction{
					{BaseAction: BaseAction{Type: "command", Command: "echo test"}},
				},
			},
		},
	}

	err := runHooks(config, PostToolUse)
	if err != nil {
		t.Errorf("runHooks() error = %v", err)
	}
}

func TestDryRunHooks_Success(t *testing.T) {
	// 標準入力をバックアップして復元
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r2, w2, _ := os.Pipe()
	os.Stdout = w2

	// 正常なJSONを標準入力に設定
	r, w, _ := os.Pipe()
	os.Stdin = r

	jsonInput := `{
		"session_id": "test",
		"transcript_path": "/tmp/test",
		"hook_event_name": "PreToolUse",
		"tool_name": "Write",
		"tool_input": {"file_path": "test.go"}
	}`

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(jsonInput))
	}()

	config := &Config{
		PreToolUse: []PreToolUseHook{
			{
				Matcher: "Write",
				Actions: []PreToolUseAction{
					{BaseAction: BaseAction{Type: "command", Command: "echo {.tool_input.file_path}"}},
				},
			},
		},
	}

	err := dryRunHooks(config, PreToolUse)

	// 標準出力を復元
	_ = w2.Close()
	os.Stdout = oldStdout

	// 出力を読み取り
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r2)
	output := buf.String()

	if err != nil {
		t.Errorf("dryRunHooks() error = %v", err)
	}

	if !strings.Contains(output, "=== PreToolUse Hooks (Dry Run) ===") {
		t.Errorf("Expected dry run header in output, got: %q", output)
	}

	if !strings.Contains(output, "Command: echo test.go") {
		t.Errorf("Expected command with replaced variables, got: %q", output)
	}
	t.Logf("Full output: %q", output)
}

// エラーケース: 空の標準入力
func TestRunHooks_EmptyInput(t *testing.T) {
	// 標準入力をバックアップして復元
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// 空の標準入力を設定
	r, w, _ := os.Pipe()
	os.Stdin = r
	_ = w.Close() // すぐに閉じて EOF を発生させる

	config := &Config{}
	err := runHooks(config, PreToolUse)

	if err == nil {
		t.Error("Expected error for empty input, got nil")
	}
}

// エラーケース: 部分的なJSON
func TestRunHooks_PartialJSON(t *testing.T) {
	// 標準入力をバックアップして復元
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// 部分的なJSONを標準入力に設定
	r, w, _ := os.Pipe()
	os.Stdin = r

	partialJSON := `{"session_id": "test"` // 不完全なJSON

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(partialJSON))
	}()

	config := &Config{}
	err := runHooks(config, PreToolUse)

	if err == nil {
		t.Error("Expected error for partial JSON, got nil")
	}
}

// エラーケース: 型が一致しないJSON
func TestRunHooks_WrongJSONType(t *testing.T) {
	// 標準入力をバックアップして復元
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// PreToolUse以外の構造のJSONを標準入力に設定
	r, w, _ := os.Pipe()
	os.Stdin = r

	wrongJSON := `{
		"session_id": "test",
		"transcript_path": "/tmp/test",
		"hook_event_name": "PreToolUse",
		"message": "This is notification format"
	}` // NotificationInput の形式

	go func() {
		defer func() { _ = w.Close() }()
		_, _ = w.Write([]byte(wrongJSON))
	}()

	config := &Config{}
	// PreToolUseとして解釈しようとするが、tool_nameがない
	err := runHooks(config, PreToolUse)

	// この場合エラーにはならないが、tool_nameは空文字になる
	// 実際のアプリケーションではバリデーションを追加することを推奨
	if err != nil {
		t.Logf("Note: Got error for wrong JSON type: %v", err)
	}
}
