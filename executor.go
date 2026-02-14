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

// ExecuteNotificationAction executes an action for the Notification event and returns JSON output.
// Similar to SessionStart, Notification uses hookSpecificOutput with additionalContext.
func (e *ActionExecutor) ExecuteNotificationAction(action Action, input *NotificationInput, rawJSON interface{}) (*ActionOutput, error) {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		stdout, stderr, exitCode, err := e.runner.RunCommandWithOutput(cmd, action.UseStdin, rawJSON)

		// Command failed with non-zero exit code
		if exitCode != 0 {
			errMsg := fmt.Sprintf("Command failed with exit code %d: %s", exitCode, stderr)
			if strings.TrimSpace(stderr) == "" && err != nil {
				errMsg = fmt.Sprintf("Command failed with exit code %d: %v", exitCode, err)
			}
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      false,
				SystemMessage: errMsg,
			}, nil
		}

		// Empty stdout - Allow for validation-type CLI tools
		if strings.TrimSpace(stdout) == "" {
			return &ActionOutput{
				Continue:      true,
				HookEventName: "Notification",
			}, nil
		}

		// Parse JSON output
		var cmdOutput NotificationOutput
		if err := json.Unmarshal([]byte(stdout), &cmdOutput); err != nil {
			errMsg := fmt.Sprintf("Command output is not valid JSON: %s", stdout)
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      false,
				SystemMessage: errMsg,
			}, nil
		}

		// Complement hookSpecificOutput with default if missing (spec marks it optional)
		if cmdOutput.HookSpecificOutput == nil {
			cmdOutput.HookSpecificOutput = &NotificationHookSpecificOutput{
				HookEventName: "Notification",
			}
		} else if cmdOutput.HookSpecificOutput.HookEventName == "" {
			// hookSpecificOutput exists but hookEventName is missing - this is invalid
			errMsg := "Command output has hookSpecificOutput but missing hookEventName"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      false,
				SystemMessage: errMsg,
			}, nil
		}

		// Validate against JSON Schema (using complemented data)
		complementedJSON, err := json.Marshal(cmdOutput)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to marshal complemented output: %s", err.Error())
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      false,
				SystemMessage: errMsg,
			}, nil
		}
		if err := validateNotificationOutput(complementedJSON); err != nil {
			errMsg := fmt.Sprintf("Command output validation failed: %s", err.Error())
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      false,
				SystemMessage: errMsg,
			}, nil
		}

		// Check for unsupported fields and log warnings to stderr
		checkUnsupportedFieldsNotification(stdout)

		// Build ActionOutput from parsed JSON
		result := &ActionOutput{
			Continue:       cmdOutput.Continue,
			HookEventName:  cmdOutput.HookSpecificOutput.HookEventName,
			SystemMessage:  cmdOutput.SystemMessage,
			StopReason:     cmdOutput.StopReason,
			SuppressOutput: cmdOutput.SuppressOutput,
		}

		// Set AdditionalContext
		result.AdditionalContext = cmdOutput.HookSpecificOutput.AdditionalContext

		return result, err

	case "output":
		processedMessage := unifiedTemplateReplace(action.Message, rawJSON)

		// Empty message check
		if strings.TrimSpace(processedMessage) == "" {
			errMsg := "Action output has no message"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      false,
				SystemMessage: errMsg,
			}, nil
		}

		// Determine continue value: default to true if unspecified
		continueValue := true
		if action.Continue != nil {
			continueValue = *action.Continue
		}

		return &ActionOutput{
			Continue:          continueValue,
			HookEventName:     "Notification",
			AdditionalContext: processedMessage,
		}, nil
	}

	return nil, nil
}

// ExecuteSubagentStartAction executes an action for the SubagentStart event and returns JSON output.
// Similar to Notification, SubagentStart uses hookSpecificOutput with additionalContext.
func (e *ActionExecutor) ExecuteSubagentStartAction(action Action, input *SubagentStartInput, rawJSON interface{}) (*ActionOutput, error) {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		stdout, stderr, exitCode, err := e.runner.RunCommandWithOutput(cmd, action.UseStdin, rawJSON)

		// Command failed with non-zero exit code
		if exitCode != 0 {
			errMsg := fmt.Sprintf("Command failed with exit code %d: %s", exitCode, stderr)
			if strings.TrimSpace(stderr) == "" && err != nil {
				errMsg = fmt.Sprintf("Command failed with exit code %d: %v", exitCode, err)
			}
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      false,
				SystemMessage: errMsg,
			}, nil
		}

		// Empty stdout - Allow for validation-type CLI tools
		if strings.TrimSpace(stdout) == "" {
			return &ActionOutput{
				Continue:      true,
				HookEventName: "SubagentStart",
			}, nil
		}

		// Parse JSON output
		var cmdOutput SubagentStartOutput
		if err := json.Unmarshal([]byte(stdout), &cmdOutput); err != nil {
			errMsg := fmt.Sprintf("Command output is not valid JSON: %s", stdout)
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      false,
				SystemMessage: errMsg,
			}, nil
		}

		// Complement hookSpecificOutput with default if missing (spec marks it optional)
		if cmdOutput.HookSpecificOutput == nil {
			cmdOutput.HookSpecificOutput = &SubagentStartHookSpecificOutput{
				HookEventName: "SubagentStart",
			}
		} else if cmdOutput.HookSpecificOutput.HookEventName == "" {
			// hookSpecificOutput exists but hookEventName is missing - this is invalid
			errMsg := "Command output has hookSpecificOutput but missing hookEventName"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      false,
				SystemMessage: errMsg,
			}, nil
		}

		// Validate against JSON Schema (using complemented data)
		complementedJSON, err := json.Marshal(cmdOutput)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to marshal complemented output: %s", err.Error())
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      false,
				SystemMessage: errMsg,
			}, nil
		}
		if err := validateSubagentStartOutput(complementedJSON); err != nil {
			errMsg := fmt.Sprintf("Command output validation failed: %s", err.Error())
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      false,
				SystemMessage: errMsg,
			}, nil
		}

		// Check for unsupported fields and log warnings to stderr
		checkUnsupportedFieldsSubagentStart(stdout)

		// Build ActionOutput from parsed JSON
		result := &ActionOutput{
			Continue:       cmdOutput.Continue,
			HookEventName:  cmdOutput.HookSpecificOutput.HookEventName,
			SystemMessage:  cmdOutput.SystemMessage,
			StopReason:     cmdOutput.StopReason,
			SuppressOutput: cmdOutput.SuppressOutput,
		}

		// Set AdditionalContext
		result.AdditionalContext = cmdOutput.HookSpecificOutput.AdditionalContext

		return result, err

	case "output":
		processedMessage := unifiedTemplateReplace(action.Message, rawJSON)

		// Empty message check
		if strings.TrimSpace(processedMessage) == "" {
			errMsg := "Action output has no message"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      false,
				SystemMessage: errMsg,
			}, nil
		}

		// Determine continue value: default to true if unspecified
		continueValue := true
		if action.Continue != nil {
			continueValue = *action.Continue
		}

		return &ActionOutput{
			Continue:          continueValue,
			HookEventName:     "SubagentStart",
			AdditionalContext: processedMessage,
		}, nil
	}

	return nil, nil
}

// ExecuteStopAction executes an action for the Stop event.
// Returns ActionOutput for JSON serialization with decision/reason fields.
// Stop hooks use top-level decision pattern (no hookSpecificOutput).
func (e *ActionExecutor) ExecuteStopAction(action Action, input *StopInput, rawJSON interface{}) (*ActionOutput, error) {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		stdout, stderr, exitCode, err := e.runner.RunCommandWithOutput(cmd, action.UseStdin, rawJSON)

		// Command failed with non-zero exit code
		if exitCode != 0 {
			errMsg := fmt.Sprintf("Command failed with exit code %d: %s", exitCode, stderr)
			if strings.TrimSpace(stderr) == "" && err != nil {
				errMsg = fmt.Sprintf("Command failed with exit code %d: %v", exitCode, err)
			}
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				Reason:        errMsg,
				SystemMessage: errMsg,
			}, nil
		}

		// Empty stdout - Allow stop (validation-type CLI tools: silence = OK)
		if strings.TrimSpace(stdout) == "" {
			return &ActionOutput{
				Continue: true,
				Decision: "", // Allow stop
			}, nil
		}

		// Parse JSON output (StopOutput has no hookSpecificOutput)
		var cmdOutput StopOutput
		if err := json.Unmarshal([]byte(stdout), &cmdOutput); err != nil {
			errMsg := fmt.Sprintf("Command output is not valid JSON: %s", stdout)
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				Reason:        errMsg,
				SystemMessage: errMsg,
			}, nil
		}

		// Validate decision field (optional: "block" only, or field must be omitted entirely)
		decision := cmdOutput.Decision
		if decision != "" && decision != "block" {
			errMsg := "Invalid decision value: must be 'block' or field must be omitted entirely"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				Reason:        errMsg,
				SystemMessage: errMsg,
			}, nil
		}

		// Validate reason: required when decision is "block"
		if decision == "block" && cmdOutput.Reason == "" {
			errMsg := "Missing required field 'reason' when decision is 'block'"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				Reason:        errMsg,
				SystemMessage: errMsg,
			}, nil
		}

		// Check for unsupported fields and log warnings to stderr
		checkUnsupportedFieldsStop(stdout)

		// Build ActionOutput from parsed JSON
		return &ActionOutput{
			Continue:       true,
			Decision:       decision,
			Reason:         cmdOutput.Reason,
			StopReason:     cmdOutput.StopReason,
			SuppressOutput: cmdOutput.SuppressOutput,
			SystemMessage:  cmdOutput.SystemMessage,
		}, nil

	case "output":
		processedMessage := unifiedTemplateReplace(action.Message, rawJSON)

		// Empty message check → fail-safe (decision: block)
		if strings.TrimSpace(processedMessage) == "" {
			errMsg := "Empty message in Stop action"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				Reason:        errMsg,
				SystemMessage: errMsg,
			}, nil
		}

		// Warn if exit_status is set (deprecated for Stop in JSON mode)
		if action.ExitStatus != nil {
			fmt.Fprintf(os.Stderr, "Warning: exit_status field is deprecated for Stop hooks and will be ignored. Use 'decision' field instead.\n")
		}

		// Validate action.Decision if set
		// Default to "" (allow stop) - Stop fires every turn end, blocking by default causes infinite loops
		decision := ""
		if action.Decision != nil {
			if *action.Decision != "" && *action.Decision != "block" {
				errMsg := "Invalid decision value in action config: must be 'block' or field must be omitted"
				fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
				return &ActionOutput{
					Continue:      true,
					Decision:      "block",
					Reason:        processedMessage,
					SystemMessage: errMsg,
				}, nil
			}
			decision = *action.Decision
		}

		// Determine reason with template expansion
		reason := processedMessage
		if action.Reason != nil {
			// Apply template expansion to reason
			reasonValue := unifiedTemplateReplace(*action.Reason, rawJSON)
			// Empty/whitespace reason with "block" decision should fallback to processedMessage
			if strings.TrimSpace(reasonValue) == "" && decision == "block" {
				reason = processedMessage
			} else {
				reason = reasonValue
			}
		}

		// For allow (decision=""), clear reason (not applicable)
		if decision == "" {
			reason = ""
		}

		// Final validation: decision "block" requires non-empty reason
		if decision == "block" && strings.TrimSpace(reason) == "" {
			errMsg := "Empty reason when decision is 'block' (reason is required for block)"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				Reason:        errMsg,
				SystemMessage: errMsg,
			}, nil
		}

		return &ActionOutput{
			Continue:      true,
			Decision:      decision,
			Reason:        reason,
			SystemMessage: processedMessage,
		}, nil
	}

	return nil, nil
}

// ExecuteSubagentStopAction executes an action for the SubagentStop event.
// Command failures result in exit status 2 to block the subagent stop operation.
func (e *ActionExecutor) ExecuteSubagentStopAction(action Action, input *SubagentStopInput, rawJSON interface{}) (*ActionOutput, error) {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		stdout, stderr, exitCode, err := e.runner.RunCommandWithOutput(cmd, action.UseStdin, rawJSON)

		// Command failed with non-zero exit code
		if exitCode != 0 {
			errMsg := fmt.Sprintf("Command failed with exit code %d: %s", exitCode, stderr)
			if strings.TrimSpace(stderr) == "" && err != nil {
				errMsg = fmt.Sprintf("Command failed with exit code %d: %v", exitCode, err)
			}
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				Reason:        errMsg,
				SystemMessage: errMsg,
			}, nil
		}

		// Empty stdout - Allow subagent stop (validation-type CLI tools: silence = OK)
		if strings.TrimSpace(stdout) == "" {
			return &ActionOutput{
				Continue: true,
				Decision: "", // Allow subagent stop
			}, nil
		}

		// Parse JSON output (SubagentStopOutput has no hookSpecificOutput, same schema as Stop)
		var cmdOutput SubagentStopOutput
		if err := json.Unmarshal([]byte(stdout), &cmdOutput); err != nil {
			errMsg := fmt.Sprintf("Command output is not valid JSON: %s", stdout)
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				Reason:        errMsg,
				SystemMessage: errMsg,
			}, nil
		}

		// Validate decision field (optional: "block" only, or field must be omitted entirely)
		decision := cmdOutput.Decision
		if decision != "" && decision != "block" {
			errMsg := "Invalid decision value: must be 'block' or field must be omitted entirely"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				Reason:        errMsg,
				SystemMessage: errMsg,
			}, nil
		}

		// Validate reason: required when decision is "block"
		if decision == "block" && cmdOutput.Reason == "" {
			errMsg := "Missing required field 'reason' when decision is 'block'"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				Reason:        errMsg,
				SystemMessage: errMsg,
			}, nil
		}

		// Check for unsupported fields and log warnings to stderr
		checkUnsupportedFieldsSubagentStop(stdout)

		// Build ActionOutput from parsed JSON
		return &ActionOutput{
			Continue:       true,
			Decision:       decision,
			Reason:         cmdOutput.Reason,
			StopReason:     cmdOutput.StopReason,
			SuppressOutput: cmdOutput.SuppressOutput,
			SystemMessage:  cmdOutput.SystemMessage,
		}, nil

	case "output":
		processedMessage := unifiedTemplateReplace(action.Message, rawJSON)

		// Empty message check → fail-safe (decision: block)
		if strings.TrimSpace(processedMessage) == "" {
			errMsg := "Empty message in SubagentStop action"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				Reason:        errMsg,
				SystemMessage: errMsg,
			}, nil
		}

		// Warn if exit_status is set (deprecated for SubagentStop in JSON mode)
		if action.ExitStatus != nil {
			fmt.Fprintf(os.Stderr, "Warning: exit_status field is deprecated for SubagentStop hooks and will be ignored. Use 'decision' field instead.\n")
		}

		// Validate action.Decision if set
		// Default to "" (allow subagent stop)
		decision := ""
		if action.Decision != nil {
			if *action.Decision != "" && *action.Decision != "block" {
				errMsg := "Invalid decision value in action config: must be 'block' or field must be omitted"
				fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
				return &ActionOutput{
					Continue:      true,
					Decision:      "block",
					Reason:        processedMessage,
					SystemMessage: errMsg,
				}, nil
			}
			decision = *action.Decision
		}

		// Determine reason with template expansion
		reason := processedMessage
		if action.Reason != nil {
			// Apply template expansion to reason
			reasonValue := unifiedTemplateReplace(*action.Reason, rawJSON)
			// Empty/whitespace reason with "block" decision should fallback to processedMessage
			if strings.TrimSpace(reasonValue) == "" && decision == "block" {
				reason = processedMessage
			} else {
				reason = reasonValue
			}
		}

		// For allow (decision=""), clear reason (not applicable)
		if decision == "" {
			reason = ""
		}

		// Final validation: decision "block" requires non-empty reason
		if decision == "block" && strings.TrimSpace(reason) == "" {
			errMsg := "Empty reason when decision is 'block' (reason is required for block)"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				Reason:        errMsg,
				SystemMessage: errMsg,
			}, nil
		}

		return &ActionOutput{
			Continue:      true,
			Decision:      decision,
			Reason:        reason,
			SystemMessage: processedMessage,
		}, nil
	}

	return nil, nil
}

// ExecutePreCompactAction executes an action for the PreCompact event.
// Supports command execution and output actions.
// ExecutePreCompactAction executes an action for the PreCompact event and returns ActionOutput.
// PreCompact always returns continue=true (fail-safe: compaction cannot be blocked).
// Errors are reported via systemMessage field, not by blocking execution.
func (e *ActionExecutor) ExecutePreCompactAction(action Action, input *PreCompactInput, rawJSON interface{}) (*ActionOutput, error) {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		stdout, stderr, exitCode, err := e.runner.RunCommandWithOutput(cmd, action.UseStdin, rawJSON)

		// Command failed with non-zero exit code
		if exitCode != 0 {
			errMsg := fmt.Sprintf("Command failed with exit code %d: %s", exitCode, stderr)
			if strings.TrimSpace(stderr) == "" && err != nil {
				errMsg = fmt.Sprintf("Command failed with exit code %d: %v", exitCode, err)
			}
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				SystemMessage: errMsg,
			}, nil
		}

		// Empty stdout - Allow compaction (validation-type CLI tools: silence = OK)
		if strings.TrimSpace(stdout) == "" {
			return &ActionOutput{
				Continue: true,
			}, nil
		}

		// Parse JSON output (PreCompactOutput has Common JSON Fields only)
		var cmdOutput PreCompactOutput
		if err := json.Unmarshal([]byte(stdout), &cmdOutput); err != nil {
			errMsg := fmt.Sprintf("Command output is not valid JSON: %s", stdout)
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				SystemMessage: errMsg,
			}, nil
		}

		// Check for unsupported fields and log warnings to stderr
		checkUnsupportedFieldsPreCompact(stdout)

		// Build ActionOutput from parsed JSON
		return &ActionOutput{
			Continue:       true,
			StopReason:     cmdOutput.StopReason,
			SuppressOutput: cmdOutput.SuppressOutput,
			SystemMessage:  cmdOutput.SystemMessage,
		}, nil

	case "output":
		processedMessage := unifiedTemplateReplace(action.Message, rawJSON)

		// Emit warning if exit_status is set (backward compatibility - no longer used)
		if action.ExitStatus != nil {
			fmt.Fprintf(os.Stderr, "Warning: exit_status field is ignored in PreCompact output actions (JSON output does not use exit codes)\n")
		}

		// Empty message is an error (fail-safe)
		if strings.TrimSpace(processedMessage) == "" {
			errMsg := "Empty message in PreCompact action"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				SystemMessage: errMsg,
			}, nil
		}

		// Output action: message maps to systemMessage
		return &ActionOutput{
			Continue:      true,
			SystemMessage: processedMessage,
		}, nil

	default:
		return nil, fmt.Errorf("unknown action type: %s", action.Type)
	}
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
			errMsg := fmt.Sprintf("Command failed with exit code %d: %s", exitCode, stderr)
			if strings.TrimSpace(stderr) == "" && err != nil {
				errMsg = fmt.Sprintf("Command failed with exit code %d: %v", exitCode, err)
			}
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      false,
				SystemMessage: errMsg,
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
			errMsg := fmt.Sprintf("Command output is not valid JSON: %s", stdout)
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      false,
				SystemMessage: errMsg,
			}, nil
		}

		// Check for required field: hookSpecificOutput.hookEventName (requirement 3.4)
		if cmdOutput.HookSpecificOutput == nil || cmdOutput.HookSpecificOutput.HookEventName == "" {
			errMsg := "Command output is missing required field: hookSpecificOutput.hookEventName"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      false,
				SystemMessage: errMsg,
			}, nil
		}

		// Validate against JSON Schema
		// This checks:
		// - hookSpecificOutput exists (required field)
		// - hookEventName is "SessionStart" (enum validation)
		// - All field types match the schema
		if err := validateSessionStartOutput([]byte(stdout)); err != nil {
			errMsg := fmt.Sprintf("Command output validation failed: %s", err.Error())
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      false,
				SystemMessage: errMsg,
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
			errMsg := "Action output has no message"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      false,
				SystemMessage: errMsg,
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
			errMsg := fmt.Sprintf("Command failed with exit code %d: %s", exitCode, stderr)
			if strings.TrimSpace(stderr) == "" && err != nil {
				errMsg = fmt.Sprintf("Command failed with exit code %d: %v", exitCode, err)
			}
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				HookEventName: "UserPromptSubmit",
				SystemMessage: errMsg,
			}, nil
		}

		// Empty stdout - Allow for validation-type CLI tools
		// Tools like linters exit 0 with no output when everything is OK.
		// In this case, we return continue: true with decision omitted (empty string) to proceed.
		if strings.TrimSpace(stdout) == "" {
			return &ActionOutput{
				Continue:      true,
				Decision:      "", // Empty string will be omitted from JSON (omitempty), allowing prompt
				HookEventName: "UserPromptSubmit",
			}, nil
		}

		// Parse JSON output
		var cmdOutput UserPromptSubmitOutput
		if err := json.Unmarshal([]byte(stdout), &cmdOutput); err != nil {
			errMsg := fmt.Sprintf("Command output is not valid JSON: %s", stdout)
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				HookEventName: "UserPromptSubmit",
				SystemMessage: errMsg,
			}, nil
		}

		// Check for required field: hookSpecificOutput.hookEventName
		if cmdOutput.HookSpecificOutput == nil || cmdOutput.HookSpecificOutput.HookEventName == "" {
			errMsg := "Command output is missing required field: hookSpecificOutput.hookEventName"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				HookEventName: "UserPromptSubmit",
				SystemMessage: errMsg,
			}, nil
		}

		// Validate hookEventName value
		if cmdOutput.HookSpecificOutput.HookEventName != "UserPromptSubmit" {
			errMsg := fmt.Sprintf("Invalid hookEventName: expected 'UserPromptSubmit', got '%s'", cmdOutput.HookSpecificOutput.HookEventName)
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				HookEventName: "UserPromptSubmit",
				SystemMessage: errMsg,
			}, nil
		}

		// Validate decision field (optional: "block" only, or field must be omitted entirely)
		decision := cmdOutput.Decision
		if decision != "" && decision != "block" {
			errMsg := "Invalid decision value: must be 'block' or field must be omitted entirely"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				HookEventName: "UserPromptSubmit",
				SystemMessage: errMsg,
			}, nil
		}

		// Validate against JSON Schema
		// This checks:
		// - hookSpecificOutput exists (required field)
		// - hookEventName is "UserPromptSubmit" (enum validation)
		// - All field types match the schema
		if err := validateUserPromptSubmitOutput([]byte(stdout)); err != nil {
			errMsg := fmt.Sprintf("Command output validation failed: %s", err.Error())
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				HookEventName: "UserPromptSubmit",
				SystemMessage: errMsg,
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
			errMsg := "Action output has no message"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				Decision:      "block",
				HookEventName: "UserPromptSubmit",
				SystemMessage: errMsg,
			}, nil
		}

		// Validate action.Decision if set
		decision := "" // default: empty string (will be omitted from JSON via omitempty)
		if action.Decision != nil {
			if *action.Decision != "" && *action.Decision != "block" {
				errMsg := "Invalid decision value in action config: must be 'block' or field must be omitted"
				fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
				return &ActionOutput{
					Continue:      true,
					Decision:      "block",
					HookEventName: "UserPromptSubmit",
					SystemMessage: errMsg,
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

// ExecuteSessionEndAction executes an action for the SessionEnd event and returns ActionOutput.
// SessionEnd always returns continue=true (fail-safe: session end cannot be blocked).
// Errors are reported via systemMessage field, not by blocking execution.
func (e *ActionExecutor) ExecuteSessionEndAction(action Action, input *SessionEndInput, rawJSON interface{}) (*ActionOutput, error) {
	switch action.Type {
	case "command":
		cmd := unifiedTemplateReplace(action.Command, rawJSON)
		stdout, stderr, exitCode, err := e.runner.RunCommandWithOutput(cmd, action.UseStdin, rawJSON)

		// Command failed with non-zero exit code
		if exitCode != 0 {
			errMsg := fmt.Sprintf("Command failed with exit code %d: %s", exitCode, stderr)
			if strings.TrimSpace(stderr) == "" && err != nil {
				errMsg = fmt.Sprintf("Command failed with exit code %d: %v", exitCode, err)
			}
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				SystemMessage: errMsg,
			}, nil
		}

		// Empty stdout - Allow session end (validation-type CLI tools: silence = OK)
		if strings.TrimSpace(stdout) == "" {
			return &ActionOutput{
				Continue: true,
			}, nil
		}

		// Parse JSON output (SessionEndOutput has Common JSON Fields only)
		var cmdOutput SessionEndOutput
		if err := json.Unmarshal([]byte(stdout), &cmdOutput); err != nil {
			errMsg := fmt.Sprintf("Command output is not valid JSON: %s", stdout)
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				SystemMessage: errMsg,
			}, nil
		}

		// Check for unsupported fields and log warnings to stderr
		checkUnsupportedFieldsSessionEnd(stdout)

		// Build ActionOutput from parsed JSON
		return &ActionOutput{
			Continue:       true,
			StopReason:     cmdOutput.StopReason,
			SuppressOutput: cmdOutput.SuppressOutput,
			SystemMessage:  cmdOutput.SystemMessage,
		}, nil

	case "output":
		processedMessage := unifiedTemplateReplace(action.Message, rawJSON)

		// Emit warning if exit_status is set (backward compatibility - no longer used)
		if action.ExitStatus != nil {
			fmt.Fprintf(os.Stderr, "Warning: exit_status field is ignored in SessionEnd output actions (JSON output does not use exit codes)\n")
		}

		// Empty message is an error (fail-safe)
		if strings.TrimSpace(processedMessage) == "" {
			errMsg := "Empty message in SessionEnd action"
			fmt.Fprintf(os.Stderr, "Warning: %s\n", errMsg)
			return &ActionOutput{
				Continue:      true,
				SystemMessage: errMsg,
			}, nil
		}

		// Output action: message maps to systemMessage
		return &ActionOutput{
			Continue:      true,
			SystemMessage: processedMessage,
		}, nil
	}

	return &ActionOutput{
		Continue: true,
	}, nil
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

// checkUnsupportedFieldsSessionEnd checks for unsupported fields in SessionEnd hook output
// SessionEnd uses Common JSON Fields only (no decision/reason/hookSpecificOutput)
func checkUnsupportedFieldsSessionEnd(stdout string) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &data); err != nil {
		return
	}

	supportedFields := map[string]bool{
		"continue":       true,
		"stopReason":     true,
		"suppressOutput": true,
		"systemMessage":  true,
		// Note: decision, reason, hookSpecificOutput are NOT supported for SessionEnd hooks
	}

	for field := range data {
		if !supportedFields[field] {
			fmt.Fprintf(os.Stderr, "Warning: Field '%s' is not supported for SessionEnd hooks\n", field)
		}
	}
}

//nolint:unused // Will be used in Step 3
func checkUnsupportedFieldsPreCompact(stdout string) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &data); err != nil {
		return
	}

	supportedFields := map[string]bool{
		"continue":       true,
		"stopReason":     true,
		"suppressOutput": true,
		"systemMessage":  true,
		// Note: decision, reason, hookSpecificOutput are NOT supported for PreCompact hooks
	}

	for field := range data {
		if !supportedFields[field] {
			fmt.Fprintf(os.Stderr, "Warning: Field '%s' is not supported for PreCompact hooks\n", field)
		}
	}
}

// checkUnsupportedFieldsNotification checks for unsupported fields in Notification JSON output
// and logs warnings to stderr for any fields that are not in the supported list.
func checkUnsupportedFieldsNotification(stdout string) {
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
			fmt.Fprintf(os.Stderr, "Warning: Field '%s' is not supported for Notification hooks\n", field)
		}
	}
}

// checkUnsupportedFieldsSubagentStart checks for unsupported fields in SubagentStart JSON output
// and logs warnings to stderr for any fields that are not in the supported list.
func checkUnsupportedFieldsSubagentStart(stdout string) {
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
			fmt.Fprintf(os.Stderr, "Warning: Field '%s' is not supported for SubagentStart hooks\n", field)
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

// checkUnsupportedFieldsStop checks for unsupported fields in Stop JSON output
// and logs warnings to stderr. Stop uses top-level decision/reason (no hookSpecificOutput).
func checkUnsupportedFieldsStop(stdout string) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &data); err != nil {
		return
	}

	supportedFields := map[string]bool{
		"continue":       true,
		"decision":       true, // Stop specific (top-level)
		"reason":         true, // Stop specific (top-level)
		"stopReason":     true,
		"suppressOutput": true,
		"systemMessage":  true,
		// Note: hookSpecificOutput is NOT supported for Stop hooks
	}

	for field := range data {
		if !supportedFields[field] {
			fmt.Fprintf(os.Stderr, "Warning: Field '%s' is not supported for Stop hooks\n", field)
		}
	}
}

// checkUnsupportedFieldsSubagentStop checks for unsupported fields in SubagentStop hook output
// SubagentStop uses the same schema as Stop (no hookSpecificOutput)
func checkUnsupportedFieldsSubagentStop(stdout string) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &data); err != nil {
		return
	}

	supportedFields := map[string]bool{
		"continue":       true,
		"decision":       true, // SubagentStop specific (top-level, same as Stop)
		"reason":         true, // SubagentStop specific (top-level, same as Stop)
		"stopReason":     true,
		"suppressOutput": true,
		"systemMessage":  true,
		// Note: hookSpecificOutput is NOT supported for SubagentStop hooks
	}

	for field := range data {
		if !supportedFields[field] {
			fmt.Fprintf(os.Stderr, "Warning: Field '%s' is not supported for SubagentStop hooks\n", field)
		}
	}
}
