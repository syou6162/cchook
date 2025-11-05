package main

import (
	"fmt"
	"testing"
)

// stubRunner is a test implementation of CommandRunner that allows mocking command execution.
type stubRunner struct {
	runFunc func(cmd string, useStdin bool, data interface{}) error
}

func (s *stubRunner) RunCommand(cmd string, useStdin bool, data interface{}) error {
	if s.runFunc != nil {
		return s.runFunc(cmd, useStdin, data)
	}
	return nil
}

func (s *stubRunner) RunCommandWithOutput(cmd string, useStdin bool, data interface{}) (stdout, stderr string, exitCode int, err error) {
	// For actions_test.go, we don't need this method to be functional
	// as it's only used in executor_test.go with stubRunnerWithOutput
	return "", "", 0, nil
}

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

	executor := NewActionExecutor(nil)
	err := executor.ExecuteNotificationAction(action, &NotificationInput{}, map[string]interface{}{})

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

	executor := NewActionExecutor(nil)
	err := executor.ExecuteSessionEndAction(action, &SessionEndInput{}, map[string]interface{}{})

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

			executor := NewActionExecutor(nil)
			err := executor.ExecuteSessionEndAction(action, &SessionEndInput{}, map[string]interface{}{})

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

func TestExecuteNotificationAction_CommandWithStubRunner(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		useStdin  bool
		runFunc   func(cmd string, useStdin bool, data interface{}) error
		wantErr   bool
		wantCmd   string
		wantStdin bool
	}{
		{
			name:     "command executes successfully",
			command:  "echo test",
			useStdin: false,
			runFunc: func(cmd string, useStdin bool, data interface{}) error {
				return nil
			},
			wantErr:   false,
			wantCmd:   "echo test",
			wantStdin: false,
		},
		{
			name:     "command with stdin",
			command:  "cat",
			useStdin: true,
			runFunc: func(cmd string, useStdin bool, data interface{}) error {
				return nil
			},
			wantErr:   false,
			wantCmd:   "cat",
			wantStdin: true,
		},
		{
			name:     "command fails",
			command:  "false",
			useStdin: false,
			runFunc: func(cmd string, useStdin bool, data interface{}) error {
				return fmt.Errorf("command failed")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedCmd string
			var capturedStdin bool

			runner := &stubRunner{
				runFunc: func(cmd string, useStdin bool, data interface{}) error {
					capturedCmd = cmd
					capturedStdin = useStdin
					return tt.runFunc(cmd, useStdin, data)
				},
			}

			executor := NewActionExecutor(runner)
			action := Action{
				Type:     "command",
				Command:  tt.command,
				UseStdin: tt.useStdin,
			}

			err := executor.ExecuteNotificationAction(action, &NotificationInput{}, map[string]interface{}{})

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if capturedCmd != tt.wantCmd {
					t.Errorf("Expected command %q, got %q", tt.wantCmd, capturedCmd)
				}
				if capturedStdin != tt.wantStdin {
					t.Errorf("Expected useStdin %v, got %v", tt.wantStdin, capturedStdin)
				}
			}
		})
	}
}

// TODO: Task 12 - This old test will be removed when ExitError handling is removed
/*
func TestExecutePreToolUseAction_CommandWithStubRunner(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		runFunc  func(cmd string, useStdin bool, data interface{}) error
		wantErr  bool
		wantCode int
	}{
		{
			name:    "command success does not block",
			command: "echo success",
			runFunc: func(cmd string, useStdin bool, data interface{}) error {
				return nil
			},
			wantErr: false,
		},
		{
			name:    "command failure blocks with exit 2",
			command: "false",
			runFunc: func(cmd string, useStdin bool, data interface{}) error {
				return fmt.Errorf("command failed")
			},
			wantErr:  true,
			wantCode: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunner{runFunc: tt.runFunc}
			executor := NewActionExecutor(runner)

			action := Action{
				Type:    "command",
				Command: tt.command,
			}

			err := executor.ExecutePreToolUseAction(action, &PreToolUseInput{}, map[string]interface{}{})

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
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}
*/

func TestExecuteStopAction_CommandWithStubRunner(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		runFunc  func(cmd string, useStdin bool, data interface{}) error
		wantErr  bool
		wantCode int
	}{
		{
			name:    "command success allows stop",
			command: "exit 0",
			runFunc: func(cmd string, useStdin bool, data interface{}) error {
				return nil
			},
			wantErr: false,
		},
		{
			name:    "command failure blocks stop with exit 2",
			command: "exit 1",
			runFunc: func(cmd string, useStdin bool, data interface{}) error {
				return fmt.Errorf("stop command failed")
			},
			wantErr:  true,
			wantCode: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunner{runFunc: tt.runFunc}
			executor := NewActionExecutor(runner)

			action := Action{
				Type:    "command",
				Command: tt.command,
			}

			err := executor.ExecuteStopAction(action, &StopInput{}, map[string]interface{}{})

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
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}
