package main

import (
	"fmt"
)

// runHooks parses input and executes hooks for the specified event type.
func runHooks(config *Config, eventType HookEventType) error {
	switch eventType {
	case PermissionRequest:
		return RunPermissionRequestHooks(config)
	case Notification:
		input, rawJSON, err := parseInput[*NotificationInput](eventType)
		if err != nil {
			return err
		}
		_, err = executeNotificationHooksJSON(config, input, rawJSON)
		return err
	case Stop:
		input, rawJSON, err := parseInput[*StopInput](eventType)
		if err != nil {
			return err
		}
		_, err = executeStopHooks(config, input, rawJSON)
		return err
	case SubagentStop:
		input, rawJSON, err := parseInput[*SubagentStopInput](eventType)
		if err != nil {
			return err
		}
		_, err = executeSubagentStopHooks(config, input, rawJSON)
		return err
	case SubagentStart:
		input, rawJSON, err := parseInput[*SubagentStartInput](eventType)
		if err != nil {
			return err
		}
		_, err = executeSubagentStartHooksJSON(config, input, rawJSON)
		return err
	case PreCompact:
		input, rawJSON, err := parseInput[*PreCompactInput](eventType)
		if err != nil {
			return err
		}
		_, err = executePreCompactHooksJSON(config, input, rawJSON)
		return err
	case SessionStart:
		input, rawJSON, err := parseInput[*SessionStartInput](eventType)
		if err != nil {
			return err
		}
		// TODO: Task 9 will handle JSON serialization and output
		_, err = executeSessionStartHooks(config, input, rawJSON)
		return err
	case UserPromptSubmit:
		input, rawJSON, err := parseInput[*UserPromptSubmitInput](eventType)
		if err != nil {
			return err
		}
		// TODO: Task 9 will handle JSON serialization and output
		_, err = executeUserPromptSubmitHooks(config, input, rawJSON)
		return err
	default:
		return fmt.Errorf("unsupported event type: %s", eventType)
	}
}

// RunSessionStartHooks executes SessionStart hooks and returns JSON output.
// This is a special exported function for SessionStart event handling in main.go.
func RunSessionStartHooks(config *Config) (*SessionStartOutput, error) {
	input, rawJSON, err := parseInput[*SessionStartInput](SessionStart)
	if err != nil {
		return nil, err
	}
	return executeSessionStartHooks(config, input, rawJSON)
}

// RunUserPromptSubmitHooks is a wrapper function that parses UserPromptSubmit input and executes hooks.
// This function is called from main.go to handle UserPromptSubmit events with JSON output.
func RunUserPromptSubmitHooks(config *Config) (*UserPromptSubmitOutput, error) {
	input, rawJSON, err := parseInput[*UserPromptSubmitInput](UserPromptSubmit)
	if err != nil {
		return nil, err
	}
	return executeUserPromptSubmitHooks(config, input, rawJSON)
}

// RunPreToolUseHooks parses input from stdin and executes PreToolUse hooks.
// Returns PreToolUseOutput for JSON serialization.
func RunPreToolUseHooks(config *Config) (*PreToolUseOutput, error) {
	input, rawJSON, err := parseInput[*PreToolUseInput](PreToolUse)
	if err != nil {
		return nil, err
	}
	return executePreToolUseHooksJSON(config, input, rawJSON)
}

// dryRunHooks parses input and performs a dry-run of hooks for the specified event type.
func dryRunHooks(config *Config, eventType HookEventType) error {
	switch eventType {
	case PreToolUse:
		input, rawJSON, err := parseInput[*PreToolUseInput](eventType)
		if err != nil {
			return err
		}
		return dryRunPreToolUseHooks(config, input, rawJSON)
	case PermissionRequest:
		input, rawJSON, err := parseInput[*PermissionRequestInput](eventType)
		if err != nil {
			return err
		}
		return dryRunPermissionRequestHooks(config, input, rawJSON)
	case PostToolUse:
		input, rawJSON, err := parseInput[*PostToolUseInput](eventType)
		if err != nil {
			return err
		}
		return dryRunPostToolUseHooks(config, input, rawJSON)
	case Notification:
		input, rawJSON, err := parseInput[*NotificationInput](eventType)
		if err != nil {
			return err
		}
		return dryRunNotificationHooks(config, input, rawJSON)
	case Stop:
		input, rawJSON, err := parseInput[*StopInput](eventType)
		if err != nil {
			return err
		}
		return dryRunStopHooks(config, input, rawJSON)
	case SubagentStop:
		input, rawJSON, err := parseInput[*SubagentStopInput](eventType)
		if err != nil {
			return err
		}
		return dryRunSubagentStopHooks(config, input, rawJSON)
	case SubagentStart:
		input, rawJSON, err := parseInput[*SubagentStartInput](eventType)
		if err != nil {
			return err
		}
		return dryRunSubagentStartHooks(config, input, rawJSON)
	case PreCompact:
		input, rawJSON, err := parseInput[*PreCompactInput](eventType)
		if err != nil {
			return err
		}
		return dryRunPreCompactHooks(config, input, rawJSON)
	case SessionStart:
		input, rawJSON, err := parseInput[*SessionStartInput](eventType)
		if err != nil {
			return err
		}
		return dryRunSessionStartHooks(config, input, rawJSON)
	case UserPromptSubmit:
		input, rawJSON, err := parseInput[*UserPromptSubmitInput](eventType)
		if err != nil {
			return err
		}
		return dryRunUserPromptSubmitHooks(config, input, rawJSON)
	case SessionEnd:
		input, rawJSON, err := parseInput[*SessionEndInput](eventType)
		if err != nil {
			return err
		}
		return dryRunSessionEndHooks(config, input, rawJSON)
	default:
		return fmt.Errorf("unsupported event type: %s", eventType)
	}
}

// RunNotificationHooks is the wrapper function called from main.go.
// It delegates to executeNotificationHooksJSON.
func RunNotificationHooks(config *Config) (*NotificationOutput, error) {
	input, rawJSON, err := parseInput[*NotificationInput](Notification)
	if err != nil {
		return nil, err
	}

	return executeNotificationHooksJSON(config, input, rawJSON)
}

// RunSubagentStartHooks is the wrapper function called from main.go.
// It delegates to executeSubagentStartHooksJSON.
func RunSubagentStartHooks(config *Config) (*SubagentStartOutput, error) {
	input, rawJSON, err := parseInput[*SubagentStartInput](SubagentStart)
	if err != nil {
		return nil, err
	}

	return executeSubagentStartHooksJSON(config, input, rawJSON)
}

// RunStopHooks parses input from stdin and executes Stop hooks.
// Returns StopOutput for JSON serialization.
func RunStopHooks(config *Config) (*StopOutput, error) {
	input, rawJSON, err := parseInput[*StopInput](Stop)
	if err != nil {
		return nil, err
	}
	return executeStopHooks(config, input, rawJSON)
}

// RunSubagentStopHooks parses input from stdin and executes SubagentStop hooks.
// Returns SubagentStopOutput for JSON serialization.
func RunSubagentStopHooks(config *Config) (*SubagentStopOutput, error) {
	input, rawJSON, err := parseInput[*SubagentStopInput](SubagentStop)
	if err != nil {
		return nil, err
	}
	return executeSubagentStopHooks(config, input, rawJSON)
}

// RunPostToolUseHooks parses input from stdin and executes PostToolUse hooks.
// Returns PostToolUseOutput for JSON serialization.
func RunPostToolUseHooks(config *Config) (*PostToolUseOutput, error) {
	input, rawJSON, err := parseInput[*PostToolUseInput](PostToolUse)
	if err != nil {
		return nil, err
	}
	return executePostToolUseHooksJSON(config, input, rawJSON)
}

// RunPreCompactHooks is the public wrapper for PreCompact hook execution.
// It parses input from stdin and executes matching hooks, returning PreCompactOutput for JSON serialization.
func RunPreCompactHooks(config *Config) (*PreCompactOutput, error) {
	input, rawJSON, err := parseInput[*PreCompactInput](PreCompact)
	if err != nil {
		return nil, err
	}
	return executePreCompactHooksJSON(config, input, rawJSON)
}

// RunSessionEndHooks is the wrapper function called from main.go.
// It delegates to executeSessionEndHooksJSON.
func RunSessionEndHooks(config *Config) (*SessionEndOutput, error) {
	input, rawJSON, err := parseInput[*SessionEndInput](SessionEnd)
	if err != nil {
		return nil, err
	}
	return executeSessionEndHooksJSON(config, input, rawJSON)
}
