package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

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
// Supports command execution and output actions.
func executeNotificationAction(action Action, input *NotificationInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd, action.UseStdin, rawJSON); err != nil {
			return err
		}
	case "output":
		return handleOutput(action.Message, action.ExitStatus, rawJSON)
	}
	return nil
}

// executeStopAction executes an action for the Stop event.
// Command failures result in exit status 2 to block the stop operation.
func executeStopAction(action Action, input *StopInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd, action.UseStdin, rawJSON); err != nil {
			// Stopでコマンドが失敗した場合はexit 2で停止をブロック
			return NewExitError(2, fmt.Sprintf("Command failed: %v", err), true)
		}
	case "output":
		return handleOutput(action.Message, action.ExitStatus, rawJSON)
	}
	return nil
}

// executeSubagentStopAction executes an action for the SubagentStop event.
// Command failures result in exit status 2 to block the subagent stop operation.
func executeSubagentStopAction(action Action, input *SubagentStopInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd, action.UseStdin, rawJSON); err != nil {
			// SubagentStopでコマンドが失敗した場合はexit 2でサブエージェント停止をブロック
			return NewExitError(2, fmt.Sprintf("Command failed: %v", err), true)
		}
	case "output":
		return handleOutput(action.Message, action.ExitStatus, rawJSON)
	}
	return nil
}

// executePreCompactAction executes an action for the PreCompact event.
// Supports command execution and output actions.
func executePreCompactAction(action Action, input *PreCompactInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd, action.UseStdin, rawJSON); err != nil {
			return err
		}
	case "output":
		return handleOutput(action.Message, action.ExitStatus, rawJSON)
	}
	return nil
}

// executeSessionStartAction executes an action for the SessionStart event.
// Errors are logged but do not block session startup.
func executeSessionStartAction(action Action, input *SessionStartInput, rawJSON interface{}) (*ActionOutput, error) {
	switch action.Type {
	case "output":
		// Process message with template
		processedMessage := unifiedTemplateReplace(action.Message, rawJSON)

		// If message is empty, return error
		if processedMessage == "" {
			return &ActionOutput{
				Continue:      false,
				SystemMessage: "Action output has no message",
			}, nil
		}

		// Set continue based on action.Continue (default true if unspecified)
		continueValue := true
		if action.Continue != nil {
			continueValue = *action.Continue
		}

		return &ActionOutput{
			Continue:          continueValue,
			HookEventName:     "SessionStart",
			AdditionalContext: processedMessage,
		}, nil

	case "command":
		// Process command with template
		cmd := unifiedTemplateReplace(action.Command, rawJSON)

		// Execute command and capture output
		stdout, stderr, exitCode, err := runCommandWithOutput(cmd, action.UseStdin, rawJSON)

		// If exit code != 0, return error
		if exitCode != 0 {
			return &ActionOutput{
				Continue:      false,
				SystemMessage: fmt.Sprintf("Command failed with exit code %d: %s", exitCode, stderr),
			}, err
		}

		// If stdout is empty, return error
		if strings.TrimSpace(stdout) == "" {
			return &ActionOutput{
				Continue:      false,
				SystemMessage: "Command produced no output",
			}, nil
		}

		// Parse stdout as JSON
		var cmdOutput map[string]interface{}
		if parseErr := json.Unmarshal([]byte(stdout), &cmdOutput); parseErr != nil {
			return &ActionOutput{
				Continue:      false,
				SystemMessage: fmt.Sprintf("Command output is not valid JSON: %s", stdout),
			}, nil
		}

		// Check for hookSpecificOutput.hookEventName
		hookSpecific, hasHookSpecific := cmdOutput["hookSpecificOutput"].(map[string]interface{})
		if !hasHookSpecific {
			return &ActionOutput{
				Continue:      false,
				SystemMessage: "Command output is missing required field: hookSpecificOutput.hookEventName",
			}, nil
		}

		hookEventName, hasEventName := hookSpecific["hookEventName"].(string)
		if !hasEventName || hookEventName == "" {
			return &ActionOutput{
				Continue:      false,
				SystemMessage: "Command output is missing required field: hookSpecificOutput.hookEventName",
			}, nil
		}

		// Extract fields from JSON output
		continueValue := false // Default to false if not specified
		if continueVal, ok := cmdOutput["continue"].(bool); ok {
			continueValue = continueVal
		}

		additionalContext := ""
		if additionalCtx, ok := hookSpecific["additionalContext"].(string); ok {
			additionalContext = additionalCtx
		}

		systemMessage := ""
		if sysMsg, ok := cmdOutput["systemMessage"].(string); ok {
			systemMessage = sysMsg
		}

		return &ActionOutput{
			Continue:          continueValue,
			HookEventName:     hookEventName,
			AdditionalContext: additionalContext,
			SystemMessage:     systemMessage,
		}, nil
	}

	return nil, fmt.Errorf("unknown action type: %s", action.Type)
}

// executeUserPromptSubmitAction executes an action for the UserPromptSubmit event.
// Command failures result in exit status 2 to block prompt processing.
func executeUserPromptSubmitAction(action Action, input *UserPromptSubmitInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd, action.UseStdin, rawJSON); err != nil {
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

// executePreToolUseAction executes an action for the PreToolUse event.
// Command failures result in exit status 2 to block tool execution.
func executePreToolUseAction(action Action, input *PreToolUseInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd, action.UseStdin, rawJSON); err != nil {
			// PreToolUseでコマンドが失敗した場合はexit 2でツール実行をブロック
			return NewExitError(2, fmt.Sprintf("Command failed: %v", err), true)
		}
	case "output":
		return handleOutput(action.Message, action.ExitStatus, rawJSON)
	}
	return nil
}

// executePostToolUseAction executes an action for the PostToolUse event.
// Supports command execution and output actions.
func executePostToolUseAction(action Action, input *PostToolUseInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd, action.UseStdin, rawJSON); err != nil {
			return err
		}
	case "output":
		return handleOutput(action.Message, action.ExitStatus, rawJSON)
	}
	return nil
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
// Errors are logged but do not block session end.
func executeSessionEndAction(action Action, input *SessionEndInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd, action.UseStdin, rawJSON); err != nil {
			return err
		}
	case "output":
		// SessionEndはブロッキング不要なので、exitStatusが指定されていない場合は通常出力
		processedMessage := unifiedTemplateReplace(action.Message, rawJSON)
		if action.ExitStatus != nil && *action.ExitStatus != 0 {
			stderr := *action.ExitStatus == 2
			return NewExitError(*action.ExitStatus, processedMessage, stderr)
		}
		fmt.Println(processedMessage)
	}
	return nil
}
