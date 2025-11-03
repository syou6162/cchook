package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
// Returns ActionOutput for JSON serialization.
func (e *ActionExecutor) ExecuteSessionStartAction(action Action, input *SessionStartInput, rawJSON interface{}) (*ActionOutput, error) {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		stdout, stderr, exitCode, err := e.runner.RunCommandWithOutput(cmd, action.UseStdin, rawJSON)

		// Command failed with non-zero exit code
		if exitCode != 0 {
			return &ActionOutput{
				Continue:      false,
				SystemMessage: fmt.Sprintf("Command failed with exit code %d: %s", exitCode, stderr),
			}, nil
		}

		// Empty stdout - Allow for validation-type CLI tools (requirement 1.6, 3.7)
		// Tools like fmt, linter, and pre-commit exit 0 with no output when everything is OK.
		// In this case, we return continue: true to allow the session to proceed.
		// Note: additionalContext will be empty, so no information is provided to Claude.
		if strings.TrimSpace(stdout) == "" {
			return &ActionOutput{
				Continue:      true,
				HookEventName: "SessionStart",
			}, nil
		}

		// Parse JSON output
		var cmdOutput SessionStartOutput
		if err := json.Unmarshal([]byte(stdout), &cmdOutput); err != nil {
			return &ActionOutput{
				Continue:      false,
				SystemMessage: fmt.Sprintf("Command output is not valid JSON: %s", stdout),
			}, nil
		}

		// Check for required field: hookSpecificOutput.hookEventName (requirement 3.4)
		if cmdOutput.HookSpecificOutput == nil || cmdOutput.HookSpecificOutput.HookEventName == "" {
			return &ActionOutput{
				Continue:      false,
				SystemMessage: "Command output is missing required field: hookSpecificOutput.hookEventName",
			}, nil
		}

		// Validate against JSON Schema
		// This checks:
		// - hookSpecificOutput exists (required field)
		// - hookEventName is "SessionStart" (enum validation)
		// - All field types match the schema
		if err := validateSessionStartOutput([]byte(stdout)); err != nil {
			return &ActionOutput{
				Continue:      false,
				SystemMessage: fmt.Sprintf("Command output validation failed: %s", err.Error()),
			}, nil
		}

		// Check for unsupported fields and log warnings to stderr
		checkUnsupportedFieldsSessionStart(stdout)

		// Build ActionOutput from parsed JSON
		// After schema validation, hookSpecificOutput is guaranteed to exist
		result := &ActionOutput{
			Continue:      cmdOutput.Continue,
			HookEventName: cmdOutput.HookSpecificOutput.HookEventName,
			SystemMessage: cmdOutput.SystemMessage,
		}

		// Set AdditionalContext
		result.AdditionalContext = cmdOutput.HookSpecificOutput.AdditionalContext

		return result, err

	case "output":
		processedMessage := unifiedTemplateReplace(action.Message, rawJSON)

		// Empty message check
		if strings.TrimSpace(processedMessage) == "" {
			return &ActionOutput{
				Continue:      false,
				SystemMessage: "Action output has no message",
			}, nil
		}

		// Determine continue value: default to true if unspecified
		continueValue := true
		if action.Continue != nil {
			continueValue = *action.Continue
		}

		return &ActionOutput{
			Continue:          continueValue,
			HookEventName:     "SessionStart",
			AdditionalContext: processedMessage,
		}, nil
	}

	return nil, nil
}

// ExecuteUserPromptSubmitAction executes an action for the UserPromptSubmit event and returns JSON output.
// This method implements Phase 2 JSON output functionality for UserPromptSubmit hooks.
func (e *ActionExecutor) ExecuteUserPromptSubmitAction(action Action, input *UserPromptSubmitInput, rawJSON interface{}) (*ActionOutput, error) {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		stdout, stderr, exitCode, err := e.runner.RunCommandWithOutput(cmd, action.UseStdin, rawJSON)

		// Command failed with non-zero exit code
		if exitCode != 0 {
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				HookEventName: "UserPromptSubmit",
				SystemMessage: fmt.Sprintf("Command failed with exit code %d: %s", exitCode, stderr),
			}, nil
		}

		// Empty stdout - Allow for validation-type CLI tools
		// Tools like linters exit 0 with no output when everything is OK.
		// In this case, we return continue: true with decision: allow to proceed.
		if strings.TrimSpace(stdout) == "" {
			return &ActionOutput{
				Continue:      true,
				Decision:      "allow",
				HookEventName: "UserPromptSubmit",
			}, nil
		}

		// Parse JSON output
		var cmdOutput UserPromptSubmitOutput
		if err := json.Unmarshal([]byte(stdout), &cmdOutput); err != nil {
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				HookEventName: "UserPromptSubmit",
				SystemMessage: fmt.Sprintf("Command output is not valid JSON: %s", stdout),
			}, nil
		}

		// Check for required field: hookSpecificOutput.hookEventName
		if cmdOutput.HookSpecificOutput == nil || cmdOutput.HookSpecificOutput.HookEventName == "" {
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				HookEventName: "UserPromptSubmit",
				SystemMessage: "Command output is missing required field: hookSpecificOutput.hookEventName",
			}, nil
		}

		// Validate hookEventName value
		if cmdOutput.HookSpecificOutput.HookEventName != "UserPromptSubmit" {
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				HookEventName: "UserPromptSubmit",
				SystemMessage: fmt.Sprintf("Invalid hookEventName: expected 'UserPromptSubmit', got '%s'", cmdOutput.HookSpecificOutput.HookEventName),
			}, nil
		}

		// Validate decision field
		decision := cmdOutput.Decision
		if decision == "" {
			// Default to "allow" if unspecified
			decision = "allow"
		} else if decision != "allow" && decision != "block" {
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				HookEventName: "UserPromptSubmit",
				SystemMessage: "Invalid decision value: must be 'allow' or 'block'",
			}, nil
		}

		// Build ActionOutput from parsed JSON
		// After validation, hookSpecificOutput is guaranteed to exist
		result := &ActionOutput{
			Continue:      true,
			Decision:      decision,
			HookEventName: cmdOutput.HookSpecificOutput.HookEventName,
			SystemMessage: cmdOutput.SystemMessage,
		}

		// Set AdditionalContext
		result.AdditionalContext = cmdOutput.HookSpecificOutput.AdditionalContext

		return result, err

	case "output":
		processedMessage := unifiedTemplateReplace(action.Message, rawJSON)

		// Empty message check
		if strings.TrimSpace(processedMessage) == "" {
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				HookEventName: "UserPromptSubmit",
				SystemMessage: "Action output has no message",
			}, nil
		}

		// Validate action.Decision if set
		decision := "allow" // default
		if action.Decision != nil {
			if *action.Decision != "allow" && *action.Decision != "block" {
				return &ActionOutput{
					Continue:      true,
					Decision:      "block",
					HookEventName: "UserPromptSubmit",
					SystemMessage: "Invalid decision value: must be 'allow' or 'block'",
				}, nil
			}
			decision = *action.Decision
		}

		return &ActionOutput{
			Continue:          true,
			Decision:          decision,
			HookEventName:     "UserPromptSubmit",
			AdditionalContext: processedMessage,
		}, nil
	}

	return nil, nil
}

// ExecutePreToolUseAction executes an action for the PreToolUse event.
// Command failures result in exit status 2 to block tool execution.
func (e *ActionExecutor) ExecutePreToolUseAction(action Action, input *PreToolUseInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := e.runner.RunCommand(cmd, action.UseStdin, rawJSON); err != nil {
			// PreToolUseでコマンドが失敗した場合はexit 2でツール実行をブロック
			return NewExitError(2, fmt.Sprintf("Command failed: %v", err), true)
		}
	case "output":
		return handleOutput(action.Message, action.ExitStatus, rawJSON)
	}
	return nil
}

// ExecutePostToolUseAction executes an action for the PostToolUse event.
// Supports command execution and output actions.
func (e *ActionExecutor) ExecutePostToolUseAction(action Action, input *PostToolUseInput, rawJSON interface{}) error {
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

// ExecuteSessionEndAction executes an action for the SessionEnd event.
// Errors are logged but do not block session end.
func (e *ActionExecutor) ExecuteSessionEndAction(action Action, input *SessionEndInput, rawJSON interface{}) error {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		if err := e.runner.RunCommand(cmd, action.UseStdin, rawJSON); err != nil {
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

// checkUnsupportedFieldsSessionStart checks for unsupported fields in SessionStart JSON output
// and logs warnings to stderr. Supported fields are: continue, stopReason, suppressOutput,
// systemMessage, and hookSpecificOutput.
func checkUnsupportedFieldsSessionStart(stdout string) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &data); err != nil {
		// JSON parsing failed - this will be caught by the main validation
		return
	}

	supportedFields := map[string]bool{
		"continue":           true,
		"stopReason":         true,
		"suppressOutput":     true,
		"systemMessage":      true,
		"hookSpecificOutput": true,
	}

	for field := range data {
		if !supportedFields[field] {
			fmt.Fprintf(os.Stderr, "Warning: Field '%s' is not supported for SessionStart hooks\n", field)
		}
	}
}
