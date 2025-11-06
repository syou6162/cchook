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
				Decision:      "approve",
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
			// Default to "approve" if unspecified
			decision = "approve"
		} else if decision != "approve" && decision != "block" {
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				HookEventName: "UserPromptSubmit",
				SystemMessage: "Invalid decision value: must be 'approve' or 'block'",
			}, nil
		}

		// Check for unsupported fields and log warnings to stderr
		checkUnsupportedFieldsUserPromptSubmit(stdout)

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
		decision := "approve" // default
		if action.Decision != nil {
			if *action.Decision != "approve" && *action.Decision != "block" {
				return &ActionOutput{
					Continue:      true,
					Decision:      "block",
					HookEventName: "UserPromptSubmit",
					SystemMessage: "Invalid decision value: must be 'approve' or 'block'",
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

// ExecutePreToolUseAction executes an action for the PreToolUse event and returns JSON output.
// This method implements Phase 3 JSON output functionality for PreToolUse hooks.
func (e *ActionExecutor) ExecutePreToolUseAction(action Action, input *PreToolUseInput, rawJSON interface{}) (*ActionOutput, error) {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		stdout, stderr, exitCode, err := e.runner.RunCommandWithOutput(cmd, action.UseStdin, rawJSON)

		// Command failed with non-zero exit code
		if exitCode != 0 {
			return &ActionOutput{
				Continue:           true,
				PermissionDecision: "deny",
				HookEventName:      "PreToolUse",
				SystemMessage:      fmt.Sprintf("Command failed with exit code %d: %s", exitCode, stderr),
			}, nil
		}

		// Empty stdout - Allow for validation-type CLI tools
		// Tools like linters exit 0 with no output when everything is OK.
		// In this case, we return continue: true with permissionDecision: allow to proceed.
		if strings.TrimSpace(stdout) == "" {
			return &ActionOutput{
				Continue:           true,
				PermissionDecision: "allow",
				HookEventName:      "PreToolUse",
			}, nil
		}

		// Parse JSON output
		var cmdOutput PreToolUseOutput
		if err := json.Unmarshal([]byte(stdout), &cmdOutput); err != nil {
			return &ActionOutput{
				Continue:           true,
				PermissionDecision: "deny",
				HookEventName:      "PreToolUse",
				SystemMessage:      fmt.Sprintf("Command output is not valid JSON: %s", stdout),
			}, nil
		}

		// Check for required field: hookSpecificOutput.hookEventName
		if cmdOutput.HookSpecificOutput == nil || cmdOutput.HookSpecificOutput.HookEventName == "" {
			return &ActionOutput{
				Continue:           true,
				PermissionDecision: "deny",
				HookEventName:      "PreToolUse",
				SystemMessage:      "Command output is missing required field: hookSpecificOutput.hookEventName",
			}, nil
		}

		// Validate hookEventName value
		if cmdOutput.HookSpecificOutput.HookEventName != "PreToolUse" {
			return &ActionOutput{
				Continue:           true,
				PermissionDecision: "deny",
				HookEventName:      "PreToolUse",
				SystemMessage:      fmt.Sprintf("Invalid hookEventName: expected 'PreToolUse', got '%s'", cmdOutput.HookSpecificOutput.HookEventName),
			}, nil
		}

		// Validate permissionDecision field (required field - fail-safe to "deny" if missing)
		permissionDecision := cmdOutput.HookSpecificOutput.PermissionDecision
		if permissionDecision == "" {
			// Fail-safe: Default to "deny" if permissionDecision is missing
			return &ActionOutput{
				Continue:           true,
				PermissionDecision: "deny",
				HookEventName:      "PreToolUse",
				SystemMessage:      "Missing required field 'permissionDecision' in command output",
			}, nil
		}

		if permissionDecision != "allow" && permissionDecision != "deny" && permissionDecision != "ask" {
			return &ActionOutput{
				Continue:           true,
				PermissionDecision: "deny",
				HookEventName:      "PreToolUse",
				SystemMessage:      "Invalid permissionDecision value: must be 'allow', 'deny', or 'ask'",
			}, nil
		}

		// Validate against JSON Schema
		// This checks:
		// - hookSpecificOutput exists (required field)
		// - hookEventName is "PreToolUse" (enum validation)
		// - permissionDecision is "allow", "deny", or "ask" (required field)
		// - All field types match the schema
		if err := validatePreToolUseOutput([]byte(stdout)); err != nil {
			return &ActionOutput{
				Continue:           true,
				PermissionDecision: "deny",
				HookEventName:      "PreToolUse",
				SystemMessage:      fmt.Sprintf("Command output validation failed: %s", err.Error()),
			}, nil
		}

		// Check for unsupported fields and log warnings to stderr
		checkUnsupportedFieldsPreToolUse(stdout)

		// Build ActionOutput from parsed JSON
		// After validation, hookSpecificOutput is guaranteed to exist
		result := &ActionOutput{
			Continue:                 true,
			PermissionDecision:       permissionDecision,
			PermissionDecisionReason: cmdOutput.HookSpecificOutput.PermissionDecisionReason,
			UpdatedInput:             cmdOutput.HookSpecificOutput.UpdatedInput,
			StopReason:               cmdOutput.StopReason,
			SuppressOutput:           cmdOutput.SuppressOutput,
			HookEventName:            cmdOutput.HookSpecificOutput.HookEventName,
			SystemMessage:            cmdOutput.SystemMessage,
		}

		return result, err

	case "output":
		processedMessage := unifiedTemplateReplace(action.Message, rawJSON)

		// Empty message check
		if strings.TrimSpace(processedMessage) == "" {
			return &ActionOutput{
				Continue:           true,
				PermissionDecision: "deny",
				HookEventName:      "PreToolUse",
				SystemMessage:      "Action output has no message",
			}, nil
		}

		// Validate action.PermissionDecision if set
		// Default to "deny" for backward compatibility (formerly exit status 2)
		permissionDecision := "deny"
		if action.PermissionDecision != nil {
			if *action.PermissionDecision != "allow" && *action.PermissionDecision != "deny" && *action.PermissionDecision != "ask" {
				return &ActionOutput{
					Continue:           true,
					PermissionDecision: "deny",
					HookEventName:      "PreToolUse",
					SystemMessage:      "Invalid permission_decision value in action config: must be 'allow', 'deny', or 'ask'",
				}, nil
			}
			permissionDecision = *action.PermissionDecision
		}

		return &ActionOutput{
			Continue:                 true,
			PermissionDecision:       permissionDecision,
			HookEventName:            "PreToolUse",
			PermissionDecisionReason: processedMessage,
		}, nil
	}

	return nil, nil
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

// checkUnsupportedFieldsUserPromptSubmit checks for unsupported fields in UserPromptSubmit JSON output
// and logs warnings to stderr for any fields that are not in the supported list.
func checkUnsupportedFieldsUserPromptSubmit(stdout string) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &data); err != nil {
		// JSON parsing failed - this will be caught by the main validation
		return
	}

	supportedFields := map[string]bool{
		"continue":           true,
		"decision":           true, // UserPromptSubmit specific
		"stopReason":         true,
		"suppressOutput":     true,
		"systemMessage":      true,
		"hookSpecificOutput": true,
	}

	for field := range data {
		if !supportedFields[field] {
			fmt.Fprintf(os.Stderr, "Warning: Field '%s' is not supported for UserPromptSubmit hooks\n", field)
		}
	}
}

// checkUnsupportedFieldsPreToolUse checks for unsupported fields in PreToolUse JSON output
// and logs warnings to stderr for any fields that are not in the supported list.
func checkUnsupportedFieldsPreToolUse(stdout string) {
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
		// Note: "permissionDecision" should be inside hookSpecificOutput, not at top level
	}

	for field := range data {
		if !supportedFields[field] {
			fmt.Fprintf(os.Stderr, "Warning: Field '%s' is not supported for PreToolUse hooks\n", field)
		}
	}
}
