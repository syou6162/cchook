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
		action := PreToolUseAction{
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
		action := PostToolUseAction{
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
		action := NotificationAction{
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
		action := StopAction{
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
		action := SubagentStopAction{
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
		action := PreCompactAction{
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
