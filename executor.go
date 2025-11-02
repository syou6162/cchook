package main

import (
	"fmt"
)

// ActionExecutor executes actions with a specified CommandRunner.
// This struct-based approach makes dependencies explicit and enables
// safe dependency injection in tests without global state.
type ActionExecutor struct {
	runner CommandRunner
}

// NewActionExecutor creates a new ActionExecutor with the given CommandRunner.
// If runner is nil, DefaultCommandRunner is used.
func NewActionExecutor(runner CommandRunner) *ActionExecutor {
	if runner == nil {
		runner = DefaultCommandRunner
	}
	return &ActionExecutor{runner: runner}
}

// ExecuteNotificationAction executes an action for the Notification event.
// Supports command execution and output actions.
func (e *ActionExecutor) ExecuteNotificationAction(action Action, input *NotificationInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := e.runner.RunCommand(cmd, action.UseStdin, rawJSON); err != nil {
			return err
		}
	case "output":
		return handleOutput(action.Message, action.ExitStatus, rawJSON)
	}
	return nil
}

// ExecuteStopAction executes an action for the Stop event.
// Command failures result in exit status 2 to block the stop operation.
func (e *ActionExecutor) ExecuteStopAction(action Action, input *StopInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := e.runner.RunCommand(cmd, action.UseStdin, rawJSON); err != nil {
			// Stopでコマンドが失敗した場合はexit 2で停止をブロック
			return NewExitError(2, fmt.Sprintf("Command failed: %v", err), true)
		}
	case "output":
		return handleOutput(action.Message, action.ExitStatus, rawJSON)
	}
	return nil
}
