package main

import (
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
			rawJSON := map[string]any{}
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

// TestExecuteNotificationAction_WithExitError removed - old implementation used ExitError,
// new implementation uses ActionOutput. See TestExecuteNotificationAction_TypeOutput in executor_test.go.

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

// TestExecuteSessionEndAction tests removed - SessionEnd is now JSON-based with (*ActionOutput, error) return.
// See executor_test.go for SessionEnd JSON output tests.

// TestExecuteStopAction_CommandWithStubRunner tests Stop command actions using stubRunnerWithOutput.
// Updated for JSON output: uses RunCommandWithOutput instead of RunCommand.
func TestExecuteStopAction_CommandWithStubRunner(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		stdout       string
		stderr       string
		exitCode     int
		wantDecision string
	}{
		{
			name:     "command success allows stop (empty stdout)",
			command:  "exit 0",
			stdout:   "",
			exitCode: 0,
			// Empty stdout → allow stop
			wantDecision: "",
		},
		{
			name:     "command failure blocks stop with decision: block",
			command:  "exit 1",
			stderr:   "stop command failed",
			exitCode: 1,
			// Non-zero exit → fail-safe block
			wantDecision: "block",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunnerWithOutput{
				stdout:   tt.stdout,
				stderr:   tt.stderr,
				exitCode: tt.exitCode,
			}
			executor := NewActionExecutor(runner)

			action := Action{
				Type:    "command",
				Command: tt.command,
			}

			output, err := executor.ExecuteStopAction(action, &StopInput{}, map[string]any{})

			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if output == nil {
				t.Fatal("Expected output, got nil")
			}

			if output.Decision != tt.wantDecision {
				t.Errorf("Expected decision=%q, got %q", tt.wantDecision, output.Decision)
			}
		})
	}
}
