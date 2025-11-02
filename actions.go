package main

import (
	"fmt"
)

// commandRunner is the package-level CommandRunner used by action executors.
// Can be swapped in tests for mocking.
var commandRunner CommandRunner = DefaultCommandRunner

// defaultExecutor is the default ActionExecutor instance used by package-level functions.
var defaultExecutor = NewActionExecutor(DefaultCommandRunner)

// handleOutput processes an output action and returns ExitError if a non-zero exit status is specified.
// Exit status 2 outputs to stderr, while other non-zero statuses output to stdout with error.
func handleOutput(message string, exitStatus *int, rawJSON interface{}) error {
	processedMessage := unifiedTemplateReplace(message, rawJSON)
	status := getExitStatus(exitStatus, "output")
	if status != 0 {
		// 0以外のExitStatusはすべてExitErrorとして返す
		stderr := status == 2 // 2の場合のみstderrに出力
		return NewExitError(status, processedMessage, stderr)
	}
	fmt.Println(processedMessage)
	return nil
}

// executeNotificationAction executes an action for the Notification event.
// This is a wrapper function that uses the default ActionExecutor.
func executeNotificationAction(action Action, input *NotificationInput, rawJSON interface{}) error {
	return defaultExecutor.ExecuteNotificationAction(action, input, rawJSON)
}

// executeStopAction executes an action for the Stop event.
// This is a wrapper function that uses the default ActionExecutor.
func executeStopAction(action Action, input *StopInput, rawJSON interface{}) error {
	return defaultExecutor.ExecuteStopAction(action, input, rawJSON)
}

// executeSubagentStopAction executes an action for the SubagentStop event.
// This is a wrapper function that uses the default ActionExecutor.
func executeSubagentStopAction(action Action, input *SubagentStopInput, rawJSON interface{}) error {
	return defaultExecutor.ExecuteSubagentStopAction(action, input, rawJSON)
}

// executePreCompactAction executes an action for the PreCompact event.
// This is a wrapper function that uses the default ActionExecutor.
func executePreCompactAction(action Action, input *PreCompactInput, rawJSON interface{}) error {
	return defaultExecutor.ExecutePreCompactAction(action, input, rawJSON)
}

// executeSessionStartAction executes an action for the SessionStart event.
// This is a wrapper function that uses the default ActionExecutor.
func executeSessionStartAction(action Action, input *SessionStartInput, rawJSON interface{}) error {
	return defaultExecutor.ExecuteSessionStartAction(action, input, rawJSON)
}

// executeUserPromptSubmitAction executes an action for the UserPromptSubmit event.
// This is a wrapper function that uses the default ActionExecutor.
func executeUserPromptSubmitAction(action Action, input *UserPromptSubmitInput, rawJSON interface{}) error {
	return defaultExecutor.ExecuteUserPromptSubmitAction(action, input, rawJSON)
}

// executePreToolUseAction executes an action for the PreToolUse event.
// This is a wrapper function that uses the default ActionExecutor.
func executePreToolUseAction(action Action, input *PreToolUseInput, rawJSON interface{}) error {
	return defaultExecutor.ExecutePreToolUseAction(action, input, rawJSON)
}

// executePostToolUseAction executes an action for the PostToolUse event.
// This is a wrapper function that uses the default ActionExecutor.
func executePostToolUseAction(action Action, input *PostToolUseInput, rawJSON interface{}) error {
	return defaultExecutor.ExecutePostToolUseAction(action, input, rawJSON)
}

// getExitStatus returns the exit status for the given action type.
// Default for "output" actions is 2, others default to 0.
func getExitStatus(exitStatus *int, actionType string) int {
	if exitStatus != nil {
		return *exitStatus
	}
	if actionType == "output" {
		return 2
	}
	return 0
}

// executeSessionEndAction executes an action for the SessionEnd event.
// This is a wrapper function that uses the default ActionExecutor.
func executeSessionEndAction(action Action, input *SessionEndInput, rawJSON interface{}) error {
	return defaultExecutor.ExecuteSessionEndAction(action, input, rawJSON)
}
