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

// ExecuteSubagentStopAction executes an action for the SubagentStop event.
// Command failures result in exit status 2 to block the subagent stop operation.
func (e *ActionExecutor) ExecuteSubagentStopAction(action Action, input *SubagentStopInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := e.runner.RunCommand(cmd, action.UseStdin, rawJSON); err != nil {
			// SubagentStopでコマンドが失敗した場合はexit 2でサブエージェント停止をブロック
			return NewExitError(2, fmt.Sprintf("Command failed: %v", err), true)
		}
	case "output":
		return handleOutput(action.Message, action.ExitStatus, rawJSON)
	}
	return nil
}

// ExecutePreCompactAction executes an action for the PreCompact event.
// Supports command execution and output actions.
func (e *ActionExecutor) ExecutePreCompactAction(action Action, input *PreCompactInput, rawJSON interface{}) error {
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

// ExecuteSessionStartAction executes an action for the SessionStart event.
// Errors are logged but do not block session startup.
func (e *ActionExecutor) ExecuteSessionStartAction(action Action, input *SessionStartInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := e.runner.RunCommand(cmd, action.UseStdin, rawJSON); err != nil {
			return err
		}
	case "output":
		// SessionStartはブロッキング不要なので、exitStatusが指定されていない場合は通常出力
		processedMessage := unifiedTemplateReplace(action.Message, rawJSON)
		if action.ExitStatus != nil && *action.ExitStatus != 0 {
			stderr := *action.ExitStatus == 2
			return NewExitError(*action.ExitStatus, processedMessage, stderr)
		}
		fmt.Println(processedMessage)
	}
	return nil
}

// ExecuteUserPromptSubmitAction executes an action for the UserPromptSubmit event.
// Command failures result in exit status 2 to block prompt processing.
func (e *ActionExecutor) ExecuteUserPromptSubmitAction(action Action, input *UserPromptSubmitInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := e.runner.RunCommand(cmd, action.UseStdin, rawJSON); err != nil {
			// UserPromptSubmitでコマンドが失敗した場合はexit 2でプロンプト処理をブロック
			return NewExitError(2, fmt.Sprintf("Command failed: %v", err), true)
		}
	case "output":
		// UserPromptSubmitはデフォルトでブロックする必要がないので、exitStatusが指定されていない場合は通常出力
		processedMessage := unifiedTemplateReplace(action.Message, rawJSON)
		if action.ExitStatus != nil && *action.ExitStatus != 0 {
			stderr := *action.ExitStatus == 2
			return NewExitError(*action.ExitStatus, processedMessage, stderr)
		}
		fmt.Println(processedMessage)
	}
	return nil
}
