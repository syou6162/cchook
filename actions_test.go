package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
)

func TestGetExitStatus(t *testing.T) {
	tests := []struct {
		name       string
		exitStatus *int
		actionType string
		want       int
	}{
		{
			name:       "nil exitStatus with output action",
			exitStatus: nil,
			actionType: "output",
			want:       2,
		},
		{
			name:       "nil exitStatus with command action",
			exitStatus: nil,
			actionType: "command",
			want:       0,
		},
		{
			name:       "explicit exitStatus 1",
			exitStatus: intPtr(1),
			actionType: "output",
			want:       1,
		},
		{
			name:       "explicit exitStatus 0 for output",
			exitStatus: intPtr(0),
			actionType: "output",
			want:       0,
		},
		{
			name:       "explicit exitStatus 2 for command",
			exitStatus: intPtr(2),
			actionType: "command",
			want:       2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getExitStatus(tt.exitStatus, tt.actionType); got != tt.want {
				t.Errorf("getExitStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActionStructsWithExitStatus(t *testing.T) {
	t.Run("PreToolUseAction with ExitStatus", func(t *testing.T) {
		action := Action{
			Type:       "output",
			Message:    "test message",
			ExitStatus: intPtr(1),
		}

		if action.ExitStatus == nil {
			t.Error("ExitStatus should not be nil")
		}
		if *action.ExitStatus != 1 {
			t.Errorf("ExitStatus = %v, want 1", *action.ExitStatus)
		}
	})

	t.Run("PostToolUseAction with ExitStatus", func(t *testing.T) {
		action := Action{
			Type:       "output",
			Message:    "test message",
			ExitStatus: intPtr(2),
		}

		if action.ExitStatus == nil {
			t.Error("ExitStatus should not be nil")
		}
		if *action.ExitStatus != 2 {
			t.Errorf("ExitStatus = %v, want 2", *action.ExitStatus)
		}
	})

	t.Run("NotificationAction with ExitStatus", func(t *testing.T) {
		action := Action{
			Type:       "output",
			Message:    "test message",
			ExitStatus: intPtr(0),
		}

		if action.ExitStatus == nil {
			t.Error("ExitStatus should not be nil")
		}
		if *action.ExitStatus != 0 {
			t.Errorf("ExitStatus = %v, want 0", *action.ExitStatus)
		}
	})

	t.Run("StopAction with ExitStatus", func(t *testing.T) {
		action := Action{
			Type:       "output",
			Message:    "test message",
			ExitStatus: intPtr(2),
		}

		if action.ExitStatus == nil {
			t.Error("ExitStatus should not be nil")
		}
		if *action.ExitStatus != 2 {
			t.Errorf("ExitStatus = %v, want 2", *action.ExitStatus)
		}
	})

	t.Run("SubagentStopAction with ExitStatus", func(t *testing.T) {
		action := Action{
			Type:       "output",
			Message:    "test message",
			ExitStatus: intPtr(1),
		}

		if action.ExitStatus == nil {
			t.Error("ExitStatus should not be nil")
		}
		if *action.ExitStatus != 1 {
			t.Errorf("ExitStatus = %v, want 1", *action.ExitStatus)
		}
	})

	t.Run("PreCompactAction with ExitStatus", func(t *testing.T) {
		action := Action{
			Type:       "output",
			Message:    "test message",
			ExitStatus: intPtr(2),
		}

		if action.ExitStatus == nil {
			t.Error("ExitStatus should not be nil")
		}
		if *action.ExitStatus != 2 {
			t.Errorf("ExitStatus = %v, want 2", *action.ExitStatus)
		}
	})
}

// Helper function to create *int
func intPtr(i int) *int {
	return &i
}

func TestHandleOutput(t *testing.T) {
	tests := []struct {
		name       string
		message    string
		exitStatus *int
		wantErr    bool
		wantCode   int
		wantStderr bool
	}{
		{
			name:       "ExitStatus 2 returns ExitError",
			message:    "Test error message",
			exitStatus: intPtr(2),
			wantErr:    true,
			wantCode:   2,
			wantStderr: true,
		},
		{
			name:       "ExitStatus 0 prints and returns nil",
			message:    "Test info message",
			exitStatus: intPtr(0),
			wantErr:    false,
		},
		{
			name:       "nil ExitStatus defaults to 2 for output",
			message:    "Default exit status message",
			exitStatus: nil,
			wantErr:    true,
			wantCode:   2,
			wantStderr: true,
		},
		{
			name:       "ExitStatus 1 returns ExitError",
			message:    "Custom exit code",
			exitStatus: intPtr(1),
			wantErr:    true,
			wantCode:   1,
			wantStderr: false, // 1の場合はstdout（stderrはfalse）
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawJSON := map[string]interface{}{}
			err := handleOutput(tt.message, tt.exitStatus, rawJSON)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				exitErr, ok := err.(*ExitError)
				if !ok {
					t.Fatalf("Expected *ExitError, got %T", err)
				}
				if exitErr.Code != tt.wantCode {
					t.Errorf("Expected exit code %d, got %d", tt.wantCode, exitErr.Code)
				}
				if exitErr.Stderr != tt.wantStderr {
					t.Errorf("Expected stderr %v, got %v", tt.wantStderr, exitErr.Stderr)
				}
				if exitErr.Message != tt.message {
					t.Errorf("Expected message '%s', got '%s'", tt.message, exitErr.Message)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestExecuteNotificationAction_WithExitError(t *testing.T) {
	action := Action{
		Type:       "output",
		Message:    "Notification error message",
		ExitStatus: intPtr(2),
	}

	err := executeNotificationAction(action, &NotificationInput{}, map[string]interface{}{})

	if err == nil {
		t.Fatal("Expected ExitError, got nil")
	}

	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("Expected *ExitError, got %T", err)
	}

	if exitErr.Code != 2 {
		t.Errorf("Expected exit code 2, got %d", exitErr.Code)
	}

	if !exitErr.Stderr {
		t.Error("Expected stderr output")
	}
}

func TestNewExitError(t *testing.T) {
	err := NewExitError(2, "test message", true)

	if err.Code != 2 {
		t.Errorf("Expected code 2, got %d", err.Code)
	}

	if err.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", err.Message)
	}

	if !err.Stderr {
		t.Error("Expected stderr true")
	}

	if err.Error() != "test message" {
		t.Errorf("Expected Error() to return 'test message', got '%s'", err.Error())
	}
}

func TestExecuteSessionEndAction_WithExitError(t *testing.T) {
	action := Action{
		Type:       "output",
		Message:    "SessionEnd error message",
		ExitStatus: intPtr(2),
	}

	err := executeSessionEndAction(action, &SessionEndInput{}, map[string]interface{}{})

	if err == nil {
		t.Fatal("Expected ExitError, got nil")
	}

	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("Expected *ExitError, got %T", err)
	}

	if exitErr.Code != 2 {
		t.Errorf("Expected exit code 2, got %d", exitErr.Code)
	}

	if !exitErr.Stderr {
		t.Error("Expected stderr output")
	}
}

func TestExecuteSessionEndAction_OutputWithDefaultExitStatus(t *testing.T) {
	tests := []struct {
		name       string
		exitStatus *int
		wantErr    bool
	}{
		{
			name:       "nil ExitStatus should print without error",
			exitStatus: nil,
			wantErr:    false,
		},
		{
			name:       "ExitStatus 0 should print without error",
			exitStatus: intPtr(0),
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := Action{
				Type:       "output",
				Message:    "SessionEnd message",
				ExitStatus: tt.exitStatus,
			}

			err := executeSessionEndAction(action, &SessionEndInput{}, map[string]interface{}{})

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
			},
			rawJSON: map[string]interface{}{
				"tool_name": "Write",
			},
			validate: func(t *testing.T, output []byte, err error) {
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}
				// 出力に"no stdin"が含まれることを確認
				if !bytes.Contains(output, []byte("no stdin")) {
					t.Errorf("Expected output to contain 'no stdin', got %s", string(output))
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

			err := executePreToolUseAction(tt.action, tt.input, tt.rawJSON)

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

	err := executePostToolUseAction(action, input, rawJSON)

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

func TestExecuteSessionStartAction_TypeOutput_Success(t *testing.T) {
	tests := []struct {
		name              string
		action            Action
		wantContinue      bool
		wantHookEventName string
		wantAddContext    string
		wantSysMessage    string
	}{
		{
			name: "Message with continue unspecified (default true)",
			action: Action{
				Type:    "output",
				Message: "Test message",
			},
			wantContinue:      true,
			wantHookEventName: "SessionStart",
			wantAddContext:    "Test message",
			wantSysMessage:    "",
		},
		{
			name: "Message with continue: false",
			action: Action{
				Type:     "output",
				Message:  "Test message",
				Continue: func() *bool { b := false; return &b }(),
			},
			wantContinue:      false,
			wantHookEventName: "SessionStart",
			wantAddContext:    "Test message",
			wantSysMessage:    "",
		},
		{
			name: "Message with template variables",
			action: Action{
				Type:    "output",
				Message: "Session {.source}",
			},
			wantContinue:      true,
			wantHookEventName: "SessionStart",
			wantAddContext:    "Session startup",
			wantSysMessage:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &SessionStartInput{
				BaseInput: BaseInput{SessionID: "test123"},
				Source:    "startup",
			}
			rawJSON := map[string]interface{}{
				"source": "startup",
			}

			result, err := executeSessionStartAction(tt.action, input, rawJSON)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("Result is nil")
			}

			if result.Continue != tt.wantContinue {
				t.Errorf("Continue: want %v, got %v", tt.wantContinue, result.Continue)
			}
			if result.HookEventName != tt.wantHookEventName {
				t.Errorf("HookEventName: want %s, got %s", tt.wantHookEventName, result.HookEventName)
			}
			if result.AdditionalContext != tt.wantAddContext {
				t.Errorf("AdditionalContext: want %s, got %s", tt.wantAddContext, result.AdditionalContext)
			}
			if result.SystemMessage != tt.wantSysMessage {
				t.Errorf("SystemMessage: want %s, got %s", tt.wantSysMessage, result.SystemMessage)
			}
		})
	}
}

func TestExecuteSessionStartAction_TypeOutput_EmptyMessage(t *testing.T) {
	action := Action{
		Type:    "output",
		Message: "",
	}
	input := &SessionStartInput{
		BaseInput: BaseInput{SessionID: "test123"},
		Source:    "startup",
	}

	result, err := executeSessionStartAction(action, input, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Result is nil")
	}

	if result.Continue != false {
		t.Errorf("Continue: want false, got %v", result.Continue)
	}
	if result.SystemMessage != "Action output has no message" {
		t.Errorf("SystemMessage: want 'Action output has no message', got %s", result.SystemMessage)
	}
}

func TestExecuteSessionStartAction_TypeCommand_Success(t *testing.T) {
	tests := []struct {
		name              string
		command           string
		wantContinue      bool
		wantHookEventName string
		wantAddContext    string
		wantSysMessage    string
	}{
		{
			name:              "Command with valid JSON and continue: true",
			command:           `echo eyJjb250aW51ZSI6IHRydWUsICJob29rU3BlY2lmaWNPdXRwdXQiOiB7Imhvb2tFdmVudE5hbWUiOiAiU2Vzc2lvblN0YXJ0IiwgImFkZGl0aW9uYWxDb250ZXh0IjogIlRlc3QgY29udGV4dCJ9fQ== | base64 -d`,
			wantContinue:      true,
			wantHookEventName: "SessionStart",
			wantAddContext:    "Test context",
			wantSysMessage:    "",
		},
		{
			name:              "Command with continue unspecified (default false)",
			command:           `echo eyJob29rU3BlY2lmaWNPdXRwdXQiOiB7Imhvb2tFdmVudE5hbWUiOiAiU2Vzc2lvblN0YXJ0In19 | base64 -d`,
			wantContinue:      false, // Default to false
			wantHookEventName: "SessionStart",
			wantAddContext:    "",
			wantSysMessage:    "",
		},
		{
			name:              "Command with systemMessage",
			command:           `echo eyJjb250aW51ZSI6IHRydWUsICJob29rU3BlY2lmaWNPdXRwdXQiOiB7Imhvb2tFdmVudE5hbWUiOiAiU2Vzc2lvblN0YXJ0In0sICJzeXN0ZW1NZXNzYWdlIjogIldhcm5pbmcgbWVzc2FnZSJ9 | base64 -d`,
			wantContinue:      true,
			wantHookEventName: "SessionStart",
			wantAddContext:    "",
			wantSysMessage:    "Warning message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := Action{
				Type:    "command",
				Command: tt.command,
			}
			input := &SessionStartInput{
				BaseInput: BaseInput{SessionID: "test123"},
				Source:    "startup",
			}

			result, err := executeSessionStartAction(action, input, nil)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("Result is nil")
			}

			if result.Continue != tt.wantContinue {
				t.Errorf("Continue: want %v, got %v", tt.wantContinue, result.Continue)
			}
			if result.HookEventName != tt.wantHookEventName {
				t.Errorf("HookEventName: want %s, got %s", tt.wantHookEventName, result.HookEventName)
			}
			if result.AdditionalContext != tt.wantAddContext {
				t.Errorf("AdditionalContext: want %s, got %s", tt.wantAddContext, result.AdditionalContext)
			}
			if result.SystemMessage != tt.wantSysMessage {
				t.Errorf("SystemMessage: want %s, got %s", tt.wantSysMessage, result.SystemMessage)
			}
		})
	}
}

func TestExecuteSessionStartAction_TypeCommand_Errors(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		wantContinue   bool
		wantSysMessage string
	}{
		{
			name:           "Command failed (exit code non-zero)",
			command:        `echo 'error' >&2 && exit 1`,
			wantContinue:   false,
			wantSysMessage: "Command failed with exit code 1: error",
		},
		{
			name:           "Empty stdout",
			command:        `true`,
			wantContinue:   false,
			wantSysMessage: "Command produced no output",
		},
		{
			name:           "Invalid JSON output",
			command:        `echo 'not json'`,
			wantContinue:   false,
			wantSysMessage: "Command output is not valid JSON: not json",
		},
		{
			name:           "Missing hookEventName",
			command:        `echo eyJjb250aW51ZSI6IHRydWV9 | base64 -d`,
			wantContinue:   false,
			wantSysMessage: "Command output is missing required field: hookSpecificOutput.hookEventName",
		},
		{
			name:           "Missing hookSpecificOutput",
			command:        `echo eyJjb250aW51ZSI6IHRydWUsICJzeXN0ZW1NZXNzYWdlIjogInRlc3QifQ== | base64 -d`,
			wantContinue:   false,
			wantSysMessage: "Command output is missing required field: hookSpecificOutput.hookEventName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := Action{
				Type:    "command",
				Command: tt.command,
			}
			input := &SessionStartInput{
				BaseInput: BaseInput{SessionID: "test123"},
				Source:    "startup",
			}

			result, _ := executeSessionStartAction(action, input, nil)
			if result == nil {
				t.Fatal("Result is nil")
			}

			if result.Continue != tt.wantContinue {
				t.Errorf("Continue: want %v, got %v", tt.wantContinue, result.Continue)
			}
			if !bytes.Contains([]byte(result.SystemMessage), []byte(tt.wantSysMessage)) {
				t.Errorf("SystemMessage: want to contain %q, got %q", tt.wantSysMessage, result.SystemMessage)
			}
		})
	}
}
