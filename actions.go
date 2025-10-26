package main

import (
	"fmt"
)

// handleOutput は output アクションを処理し、必要に応じて ExitError を返す
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

func executeNotificationAction(action NotificationAction, input *NotificationInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd); err != nil {
			return err
		}
	case "output":
		return handleOutput(action.Message, action.ExitStatus, rawJSON)
	}
	return nil
}

func executeStopAction(action StopAction, input *StopInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd); err != nil {
			// Stopでコマンドが失敗した場合はexit 2で停止をブロック
			return NewExitError(2, fmt.Sprintf("Command failed: %v", err), true)
		}
	case "output":
		return handleOutput(action.Message, action.ExitStatus, rawJSON)
	}
	return nil
}

func executeSubagentStopAction(action SubagentStopAction, input *SubagentStopInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd); err != nil {
			// SubagentStopでコマンドが失敗した場合はexit 2でサブエージェント停止をブロック
			return NewExitError(2, fmt.Sprintf("Command failed: %v", err), true)
		}
	case "output":
		return handleOutput(action.Message, action.ExitStatus, rawJSON)
	}
	return nil
}

func executePreCompactAction(action PreCompactAction, input *PreCompactInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd); err != nil {
			return err
		}
	case "output":
		return handleOutput(action.Message, action.ExitStatus, rawJSON)
	}
	return nil
}

func executeSessionStartAction(action SessionStartAction, input *SessionStartInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd); err != nil {
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

func executeUserPromptSubmitAction(action UserPromptSubmitAction, input *UserPromptSubmitInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd); err != nil {
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

func executePreToolUseAction(action PreToolUseAction, input *PreToolUseInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd); err != nil {
			// PreToolUseでコマンドが失敗した場合はexit 2でツール実行をブロック
			return NewExitError(2, fmt.Sprintf("Command failed: %v", err), true)
		}
	case "output":
		return handleOutput(action.Message, action.ExitStatus, rawJSON)
	}
	return nil
}

func executePostToolUseAction(action PostToolUseAction, input *PostToolUseInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd); err != nil {
			return err
		}
	case "output":
		return handleOutput(action.Message, action.ExitStatus, rawJSON)
	}
	return nil
}

// getExitStatus returns the exit status for the given action type
// Default for "output" actions is 2, others default to 0
func getExitStatus(exitStatus *int, actionType string) int {
	if exitStatus != nil {
		return *exitStatus
	}
	if actionType == "output" {
		return 2
	}
	return 0
}

func executeSessionEndAction(action SessionEndAction, input *SessionEndInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := runCommand(cmd); err != nil {
			return err
		}
	case "output":
		return handleOutput(action.Message, action.ExitStatus, rawJSON)
	}
	return nil
}
